package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/auth"
)

// memorySink stands in for admin_audit_log so these tests need no database and
// run in the normal `npm test`.
type memorySink struct {
	records []Record
	rows    map[string]map[string]any // "table#id" -> row, mutated by the fake handler
	fail    error
}

func newSink() *memorySink { return &memorySink{rows: map[string]map[string]any{}} }

func (s *memorySink) Append(rec Record) error {
	if s.fail != nil {
		return s.fail
	}
	s.records = append(s.records, rec)
	return nil
}

func (s *memorySink) Snapshot(table, id string) (map[string]any, error) {
	row, ok := s.rows[table+"#"+id]
	if !ok {
		return nil, nil
	}
	out := map[string]any{}
	for k, v := range row {
		out[k] = v
	}
	return out, nil
}

func (s *memorySink) put(table, id string, row map[string]any) { s.rows[table+"#"+id] = row }
func (s *memorySink) drop(table, id string)                    { delete(s.rows, table+"#"+id) }

func useSink(t *testing.T) *memorySink {
	t.Helper()
	previous := Store
	sink := newSink()
	Store = sink
	t.Cleanup(func() { Store = previous })
	return sink
}

var testActor = &auth.User{ID: 7, Email: "Admin@Lamsza.Example", Name: "Teszt Admin"}

// call runs one request through the wrapper with a signed-in admin on the
// context, the way RequireAdmin leaves it.
func call(t *testing.T, route, method, target, body string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	r := httptest.NewRequest(method, target, reader)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	r = r.WithContext(auth.WithUser(r.Context(), testActor))
	w := httptest.NewRecorder()
	Wrap(route, h)(w, r)
	return w
}

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

// TestEveryRouteAndWriteMethodRecordsExactlyOne is the route-group assertion
// the issue asks for, derived from the registry rather than from a hand-written
// list — so a route added to resources.go is covered the moment it is added,
// and a route that stops recording goes red here.
func TestEveryRouteAndWriteMethodRecordsExactlyOne(t *testing.T) {
	for _, route := range Routes() {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			t.Run(strings.TrimPrefix(route, "/api/admin/")+" "+method, func(t *testing.T) {
				sink := useSink(t)
				call(t, route, method, route+"?id=42", `{"id":42}`, okHandler)

				if len(sink.records) != 1 {
					t.Fatalf("%s %s wrote %d audit records, want exactly 1", method, route, len(sink.records))
				}
				rec := sink.records[0]
				res, _ := Lookup(route)
				switch {
				case rec.Resource != res.Name:
					t.Errorf("resource = %q, want %q", rec.Resource, res.Name)
				case rec.Method != method:
					t.Errorf("method = %q, want %q", rec.Method, method)
				case rec.Route != route:
					t.Errorf("route = %q, want %q", rec.Route, route)
				case rec.ActorUserID != testActor.ID:
					t.Errorf("actor id = %d, want %d", rec.ActorUserID, testActor.ID)
				case rec.ActorEmail != testActor.Email:
					t.Errorf("actor email = %q, want %q", rec.ActorEmail, testActor.Email)
				case rec.OccurredAt.IsZero():
					t.Error("occurred_at is zero")
				case rec.Action == "":
					t.Error("action is empty")
				}
			})
		}
	}
}

// TestReadMethodsRecordNothing — the trail is the write trail. A GET on every
// admin route would bury the writes it exists to surface.
func TestReadMethodsRecordNothing(t *testing.T) {
	for _, route := range Routes() {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
			sink := useSink(t)
			call(t, route, method, route, "", okHandler)
			if len(sink.records) != 0 {
				t.Errorf("%s %s wrote %d records, want 0", method, route, len(sink.records))
			}
		}
	}
}

func TestWrapPanicsOnUndeclaredRoute(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Wrap accepted a route with no declared resource; an unaudited admin route could be wired")
		}
	}()
	Wrap("/api/admin/not-declared-anywhere", okHandler)
}

func TestActionPerMethod(t *testing.T) {
	for method, want := range map[string]string{
		http.MethodPost:   "create",
		http.MethodPut:    "update",
		http.MethodPatch:  "update",
		http.MethodDelete: "delete",
	} {
		sink := useSink(t)
		call(t, "/api/admin/tags", method, "/api/admin/tags?id=3", `{"name":"x"}`, okHandler)
		if got := sink.records[0].Action; got != want {
			t.Errorf("%s -> action %q, want %q", method, got, want)
		}
	}
}

// TestUpdateRecordsBeforeAfterAndDiff — "enough of a before/after to undo a
// mistake".
func TestUpdateRecordsBeforeAfterAndDiff(t *testing.T) {
	sink := useSink(t)
	sink.put("tags", "3", map[string]any{"id": float64(3), "name": "régi"})

	call(t, "/api/admin/tags", http.MethodPut, "/api/admin/tags?id=3", `{"id":3,"name":"új"}`,
		func(w http.ResponseWriter, r *http.Request) {
			sink.put("tags", "3", map[string]any{"id": float64(3), "name": "új"})
			w.WriteHeader(http.StatusOK)
		})

	rec := sink.records[0]
	if rec.ResourceID != "3" {
		t.Fatalf("resource id = %q, want \"3\"", rec.ResourceID)
	}
	assertJSON(t, "before_state", rec.BeforeState, `{"tags":{"id":3,"name":"régi"}}`)
	assertJSON(t, "after_state", rec.AfterState, `{"tags":{"id":3,"name":"új"}}`)
	assertJSON(t, "diff", rec.Diff, `{"tags":{"name":{"from":"régi","to":"új"}}}`)
}

// TestDeleteKeepsTheWholeRow — a delete whose record only said "deleted #3" is
// not undoable, which is the case the issue names.
func TestDeleteKeepsTheWholeRow(t *testing.T) {
	sink := useSink(t)
	sink.put("entries", "88", map[string]any{"id": float64(88), "name": "Vendéglő", "published": true})

	call(t, "/api/admin/entries", http.MethodDelete, "/api/admin/entries?id=88", "",
		func(w http.ResponseWriter, r *http.Request) {
			sink.drop("entries", "88")
			w.WriteHeader(http.StatusOK)
		})

	rec := sink.records[0]
	assertJSON(t, "before_state", rec.BeforeState, `{"entries":{"id":88,"name":"Vendéglő","published":true}}`)
	if len(rec.AfterState) != 0 {
		t.Errorf("after_state = %s, want empty after a delete", rec.AfterState)
	}
	assertJSON(t, "diff", rec.Diff,
		`{"entries":{"id":{"from":88},"name":{"from":"Vendéglő"},"published":{"from":true}}}`)
}

// TestCreateLearnsTheIDFromTheReply — a create has no id in the request.
func TestCreateLearnsTheIDFromTheReply(t *testing.T) {
	sink := useSink(t)
	call(t, "/api/admin/tags", http.MethodPost, "/api/admin/tags", `{"name":"friss"}`,
		func(w http.ResponseWriter, r *http.Request) {
			sink.put("tags", "12", map[string]any{"id": float64(12), "name": "friss"})
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 12, "name": "friss"})
		})

	rec := sink.records[0]
	if rec.ResourceID != "12" {
		t.Fatalf("resource id = %q, want \"12\" from the reply body", rec.ResourceID)
	}
	if rec.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.StatusCode)
	}
	assertJSON(t, "after_state", rec.AfterState, `{"tags":{"id":12,"name":"friss"}}`)
}

// TestResourceIDFromBodyField covers the routes that carry the id in the JSON
// body under a name of their own.
func TestResourceIDFromBodyField(t *testing.T) {
	sink := useSink(t)
	call(t, "/api/admin/county_seat", http.MethodPut, "/api/admin/county_seat", `{"location_id":55}`, okHandler)
	if got := sink.records[0].ResourceID; got != "55" {
		t.Errorf("resource id = %q, want \"55\" from the body's location_id", got)
	}
}

// TestFailedWriteIsStillRecorded — a refused write is part of the trail. The
// status code says what happened.
func TestFailedWriteIsStillRecorded(t *testing.T) {
	sink := useSink(t)
	call(t, "/api/admin/entry_types", http.MethodPost, "/api/admin/entry_types", `{"name":"x"}`,
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "A típuslista zárt.", http.StatusForbidden)
		})
	if len(sink.records) != 1 {
		t.Fatalf("got %d records, want 1", len(sink.records))
	}
	if got := sink.records[0].StatusCode; got != http.StatusForbidden {
		t.Errorf("status = %d, want 403", got)
	}
}

// TestHandlerStillReadsTheBody — the wrapper consumes the body to record it, so
// the handler must get an unread copy or every write breaks.
func TestHandlerStillReadsTheBody(t *testing.T) {
	useSink(t)
	var seen string
	w := call(t, "/api/admin/tags", http.MethodPost, "/api/admin/tags", `{"name":"átment"}`,
		func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("handler could not decode the body: %v", err)
			}
			seen = body.Name
			w.WriteHeader(http.StatusCreated)
		})
	if seen != "átment" {
		t.Errorf("handler saw name %q, want %q", seen, "átment")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status passed through as %d, want 201", w.Code)
	}
}

func TestCredentialShapedKeysAreRedacted(t *testing.T) {
	sink := useSink(t)
	call(t, "/api/admin/settings", http.MethodPut, "/api/admin/settings",
		`{"site_name":"Lámsza","google_client_secret":"super-secret","nested":{"api_key":"abc"}}`, okHandler)

	payload := string(sink.records[0].Payload)
	for _, leaked := range []string{"super-secret", "abc"} {
		if strings.Contains(payload, leaked) {
			t.Errorf("payload leaked %q: %s", leaked, payload)
		}
	}
	if !strings.Contains(payload, "Lámsza") {
		t.Errorf("payload dropped an ordinary field: %s", payload)
	}
}

func TestNonJSONBodyIsNotBuffered(t *testing.T) {
	sink := useSink(t)
	r := httptest.NewRequest(http.MethodPost, "/api/admin/entry-images?entry_id=4", strings.NewReader("\x89PNG....binary...."))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	r = r.WithContext(auth.WithUser(r.Context(), testActor))
	Wrap("/api/admin/entry-images", okHandler)(httptest.NewRecorder(), r)

	payload := string(sink.records[0].Payload)
	if !strings.Contains(payload, "_not_captured") {
		t.Errorf("payload = %s, want a _not_captured marker for a binary body", payload)
	}
	if strings.Contains(payload, "PNG") {
		t.Errorf("payload buffered binary upload bytes: %s", payload)
	}
}

func TestOversizedValuesAreReplacedNotTruncated(t *testing.T) {
	sink := useSink(t)
	big := strings.Repeat("ü", maxStateBytes)
	call(t, "/api/admin/pages", http.MethodPut, "/api/admin/pages?id=1",
		fmt.Sprintf(`{"id":1,"body":%q}`, big), okHandler)

	payload := sink.records[0].Payload
	if len(payload) > maxStateBytes {
		t.Errorf("payload is %d bytes, over the %d cap", len(payload), maxStateBytes)
	}
	if !json.Valid(payload) {
		t.Errorf("capped payload is not valid JSON, so JSONB would reject it: %s", payload)
	}
	if !strings.Contains(string(payload), "_truncated") {
		t.Errorf("capped payload does not say it was capped: %s", payload)
	}
}

// TestSinkFailureDoesNotFailTheRequest — the record is written after the
// handler has committed, so a 500 here would hide a change that already landed.
func TestSinkFailureDoesNotFailTheRequest(t *testing.T) {
	sink := useSink(t)
	sink.fail = fmt.Errorf("database is down")
	w := call(t, "/api/admin/tags", http.MethodDelete, "/api/admin/tags?id=1", "", okHandler)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 — a failed audit write must not fail the request", w.Code)
	}
}

func assertJSON(t *testing.T, field string, got json.RawMessage, want string) {
	t.Helper()
	var gotV, wantV any
	if err := json.Unmarshal(got, &gotV); err != nil {
		t.Fatalf("%s is not valid JSON (%v): %s", field, err, got)
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		t.Fatalf("test's want for %s is not valid JSON: %v", field, err)
	}
	gotB, _ := json.Marshal(gotV)
	wantB, _ := json.Marshal(wantV)
	if !bytes.Equal(gotB, wantB) {
		t.Errorf("%s =\n  %s\nwant\n  %s", field, gotB, wantB)
	}
}
