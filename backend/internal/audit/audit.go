// Package audit records every write the admin API makes (BOG-48).
//
// Why it exists: nothing recorded which admin changed what. This app is the
// only writer of the directory the public site reads, and there was no way at
// all to answer "who deleted this entry".
//
// Why it is a wrapper and not a call inside each handler: coverage. There are
// thirty-odd mutating admin routes across twelve packages. Instrumentation
// added by hand is instrumentation that is missing from the route somebody adds
// next month. `main.go` registers every admin route through one function that
// applies `Wrap`, `Wrap` refuses to wire a route that has no declared resource
// (see resources.go), and `audit_routes_test.go` in package main re-derives the
// route list from the source and fails if either rule is broken. So "every
// mutating admin route writes a record" is a property of the wiring, not of
// anybody's discipline.
//
// Append-only. The only statement this package runs against `admin_audit_log`
// besides a read is INSERT. No admin route updates or deletes a record, and
// `append_only_test.go` fails if one appears. Pruning is an operator action
// against the database, documented in
// `lamsza/backend/migrations/admin_audit_log.sql`.
//
// Schema: the table is created by the main `lamsza` backend, which owns the
// shared database. This process runs no DDL (BOG-39).
package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"backend/internal/auth"
)

// Byte caps. The app bounds the row; nothing bounds the table (see the
// retention note in the migration). An oversized value is replaced by a marker
// rather than cut mid-JSON, so every stored value stays valid JSON.
const (
	maxStateBytes = 16 * 1024  // payload / before_state / after_state / diff
	maxBodyBytes  = 512 * 1024 // request body we are willing to buffer at all
	maxReplyBytes = 64 * 1024  // response body read back to learn a new row's id
)

// Record is one row of admin_audit_log.
type Record struct {
	OccurredAt  time.Time
	ActorUserID int // 0 when unknown
	ActorEmail  string
	Resource    string
	ResourceID  string
	Action      string // create | update | delete | other
	Method      string
	Route       string
	StatusCode  int
	Payload     json.RawMessage
	BeforeState json.RawMessage
	AfterState  json.RawMessage
	Diff        json.RawMessage
}

// Sink is where records go and where row snapshots come from. Tests replace it.
type Sink interface {
	Append(Record) error
	// Snapshot returns the row of `table` whose id is `id`, as a decoded JSON
	// object, or nil when there is no such row.
	Snapshot(table, id string) (map[string]any, error)
}

// Store is the live sink. Set it in tests.
var Store Sink = PostgresSink{}

// Wrap returns h with the audit trail applied.
//
// It panics when the route has no declared resource. That is deliberate: a
// missing entry is a wiring mistake, and a process that refuses to start is
// cheaper than an admin route that writes the directory invisibly.
func Wrap(route string, h http.HandlerFunc) http.HandlerFunc {
	res, ok := Lookup(route)
	if !ok {
		panic("audit: route " + route + " has no declared resource; add it to internal/audit/resources.go")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isWrite(r.Method) {
			h(w, r)
			return
		}

		payload, bodyJSON := capturePayload(r)
		id := resourceID(r, res, bodyJSON)
		recordedRoute := route
		if res.Subtree {
			recordedRoute = r.URL.Path
			if id == "" {
				id = strings.TrimPrefix(r.URL.Path, route)
			}
		}
		before := snapshot(res, id)

		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		if r.Method == http.MethodPost && len(res.Tables) > 0 {
			rec.capture = true
		}
		h(rec, r)

		// A create has no id until the handler has made the row; the reply
		// carries it.
		if id == "" {
			id = idFromReply(rec.body.Bytes())
		}
		after := snapshot(res, id)

		write(Record{
			OccurredAt:  time.Now().UTC(),
			ActorUserID: actorID(r),
			ActorEmail:  actorEmail(r),
			Resource:    res.Name,
			ResourceID:  id,
			Action:      action(r.Method),
			Method:      r.Method,
			Route:       recordedRoute,
			StatusCode:  rec.status,
			Payload:     payload,
			BeforeState: marshalState(before),
			AfterState:  marshalState(after),
			Diff:        marshalState(diff(before, after)),
		})
	}
}

// write sends the record and shouts if it cannot.
//
// It does not fail the request. The record is written after the handler has
// already committed, so refusing the response would not undo the change - it
// would only hide a completed write behind a 500. A dropped record is instead
// made loud in the log, prefixed so it is greppable.
func write(rec Record) {
	if err := Store.Append(rec); err != nil {
		log.Printf("AUDIT WRITE FAILED: %s %s resource=%s id=%s actor=%s: %v",
			rec.Method, rec.Route, rec.Resource, rec.ResourceID, rec.ActorEmail, err)
	}
}

func isWrite(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

func action(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	}
	return "other"
}

func actorID(r *http.Request) int {
	if u := auth.UserFromContext(r.Context()); u != nil {
		return u.ID
	}
	return 0
}

func actorEmail(r *http.Request) string {
	if u := auth.UserFromContext(r.Context()); u != nil {
		return u.Email
	}
	return ""
}

// capturePayload records the request body and hands the handler an unread copy.
//
// Only JSON bodies are buffered. An image upload is megabytes of bytes that say
// nothing an audit reader wants, and buffering it would double the memory of
// every upload; those records carry a marker naming the content type and length
// instead, and the request body is left untouched.
func capturePayload(r *http.Request) (json.RawMessage, map[string]any) {
	ct := r.Header.Get("Content-Type")
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}
	if !strings.Contains(strings.ToLower(ct), "json") {
		return marshalCapped(map[string]any{
			"_not_captured":  "body is not JSON",
			"content_type":   ct,
			"content_length": r.ContentLength,
		}), nil
	}
	if r.ContentLength > maxBodyBytes {
		return marshalCapped(map[string]any{
			"_not_captured":  "body over cap",
			"content_length": r.ContentLength,
			"cap_bytes":      maxBodyBytes,
		}), nil
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	r.Body.Close()
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		return marshalCapped(map[string]any{"_not_captured": "body read failed"}), nil
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if len(raw) > maxBodyBytes {
		return marshalCapped(map[string]any{
			"_not_captured": "body over cap",
			"cap_bytes":     maxBodyBytes,
		}), nil
	}

	// Decoded so credential-shaped keys can be redacted and so the body's id
	// field is available. A body that is not an object (an array, a bare
	// value) is stored as-is and yields no id.
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		if json.Valid(raw) {
			return capRaw(raw), nil
		}
		return marshalCapped(map[string]any{"_not_captured": "body is not valid JSON"}), nil
	}
	return marshalCapped(redact(obj)), obj
}

// resourceID finds the id of the row being written: the query parameter first,
// because that is what the delete and update routes use, then the body field.
func resourceID(r *http.Request, res Resource, body map[string]any) string {
	if v := strings.TrimSpace(r.URL.Query().Get(res.idParam())); v != "" {
		return v
	}
	if body != nil {
		if s := scalarString(body[res.idField()]); s != "" {
			return s
		}
	}
	return ""
}

func idFromReply(reply []byte) string {
	if len(reply) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(reply, &obj); err != nil {
		return ""
	}
	return scalarString(obj["id"])
}

// scalarString renders an id that arrived as a JSON number, string or bool.
// JSON has no integers, so an id decoded into `any` is a float64.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "0" {
			return ""
		}
		return s
	case float64:
		if t == 0 {
			return ""
		}
		if t == float64(int64(t)) {
			return formatInt(int64(t))
		}
		b, _ := json.Marshal(t)
		return string(b)
	case json.Number:
		return t.String()
	}
	return ""
}

func formatInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// snapshot reads the resource's row from every table it may live in, keyed by
// table name. A route with no tables, or a write with no id yet, snapshots
// nothing - see the note in resources.go.
func snapshot(res Resource, id string) map[string]any {
	if id == "" || len(res.Tables) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, table := range res.Tables {
		row, err := Store.Snapshot(table, id)
		if err != nil {
			log.Printf("audit: snapshot %s#%s: %v", table, id, err)
			continue
		}
		if row != nil {
			out[table] = row
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// diff reports the changed columns per table, as {table: {column: {from, to}}}.
// A table that appears on one side only is reported whole, which is what makes
// a delete reversible from the record alone.
func diff(before, after map[string]any) map[string]any {
	if before == nil && after == nil {
		return nil
	}
	out := map[string]any{}
	for _, table := range union(before, after) {
		b, _ := before[table].(map[string]any)
		a, _ := after[table].(map[string]any)
		if d := diffRow(b, a); len(d) > 0 {
			out[table] = d
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func diffRow(before, after map[string]any) map[string]any {
	out := map[string]any{}
	for _, col := range union(before, after) {
		b, bOK := before[col]
		a, aOK := after[col]
		if bOK && aOK && sameJSON(b, a) {
			continue
		}
		change := map[string]any{}
		if bOK {
			change["from"] = b
		}
		if aOK {
			change["to"] = a
		}
		out[col] = change
	}
	return out
}

func union(a, b map[string]any) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range []map[string]any{a, b} {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sameJSON(a, b any) bool {
	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}

// redactedKeys matches field names whose value must never reach the log. The
// audit trail is read by more people than the secrets are shared with, and a
// record of a settings write is exactly where a key would otherwise land.
var redactedKeys = []string{"password", "passwd", "secret", "token", "credential", "apikey", "api_key", "private_key", "client_secret"}

func redact(obj map[string]any) map[string]any {
	out := make(map[string]any, len(obj))
	for k, v := range obj {
		if isRedacted(k) {
			out[k] = "[redacted]"
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			out[k] = redact(nested)
			continue
		}
		out[k] = v
	}
	return out
}

func isRedacted(key string) bool {
	k := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, needle := range redactedKeys {
		if strings.Contains(k, needle) {
			return true
		}
	}
	return false
}

// marshalState leaves the column NULL for an absent snapshot rather than
// storing the JSON literal `null`. "There was no row" and "the row was null"
// read the same in JSONB otherwise, and the first is what a delete means.
//
// It is a separate function from marshalCapped on purpose: a nil map inside an
// `any` is not `== nil`, so the generic path would quietly marshal `null`.
func marshalState(m map[string]any) json.RawMessage {
	if len(m) == 0 {
		return nil
	}
	return marshalCapped(m)
}

func marshalCapped(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"_not_captured":"marshal failed"}`)
	}
	return capRaw(raw)
}

func capRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) <= maxStateBytes {
		return raw
	}
	// Replaced whole rather than cut: a truncated JSON value is not JSON, and
	// the column is JSONB.
	return json.RawMessage(`{"_truncated":true,"_bytes":` + formatInt(int64(len(raw))) + `,"_cap_bytes":` + formatInt(maxStateBytes) + `}`)
}

// recorder captures the status code, and optionally the reply, without
// changing what the handler writes.
type recorder struct {
	http.ResponseWriter
	status  int
	wrote   bool
	capture bool
	body    bytes.Buffer
}

func (rc *recorder) WriteHeader(code int) {
	if !rc.wrote {
		rc.status = code
		rc.wrote = true
	}
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *recorder) Write(b []byte) (int, error) {
	rc.wrote = true
	if rc.capture && rc.body.Len() < maxReplyBytes {
		rc.body.Write(b[:min(len(b), maxReplyBytes-rc.body.Len())])
	}
	return rc.ResponseWriter.Write(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
