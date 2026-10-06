package audit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/db"
)

// PostgresSink is the live store: `admin_audit_log` in the shared `lamsza`
// database.
//
// INSERT and SELECT only. There is no UPDATE and no DELETE here, and
// `append_only_test.go` fails if one is added — "append-only from the app's
// perspective" has to be a property of the code, not a habit.
//
// The table is created by the main `lamsza` backend
// (`internal/auth.migrateAdminAuditLog`, explicit form in
// `backend/migrations/admin_audit_log.sql`). This process runs no DDL.
type PostgresSink struct{}

func (PostgresSink) Append(rec Record) error {
	if db.DB == nil {
		return errors.New("no database connection")
	}
	_, err := db.DB.Exec(`
		INSERT INTO admin_audit_log
			(occurred_at, actor_user_id, actor_email, resource, resource_id,
			 action, method, route, status_code, payload, before_state, after_state, diff)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		rec.OccurredAt,
		nullableID(rec.ActorUserID),
		rec.ActorEmail,
		rec.Resource,
		rec.ResourceID,
		rec.Action,
		rec.Method,
		rec.Route,
		rec.StatusCode,
		nullableJSON(rec.Payload),
		nullableJSON(rec.BeforeState),
		nullableJSON(rec.AfterState),
		nullableJSON(rec.Diff),
	)
	return err
}

// Snapshot reads one row as JSON.
//
// The table name is interpolated because a table name cannot be a bind
// parameter. It is safe here and only here: every value reaching this argument
// comes from `resources.go`, a map of compile-time constants. `allowedTables`
// re-checks that at run time so a future caller cannot turn this into an
// injection by passing a request value.
//
// `id::text` because ids are compared as text: the column is an integer on
// every table today, but the id arrives from a query string.
func (PostgresSink) Snapshot(table, id string) (map[string]any, error) {
	if db.DB == nil {
		return nil, errors.New("no database connection")
	}
	if !allowedTables[table] {
		return nil, fmt.Errorf("audit: table %q is not declared in resources.go", table)
	}
	var raw []byte
	err := db.DB.QueryRow(
		`SELECT to_jsonb(t) FROM `+table+` t WHERE t.id::text = $1`, id,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return nil, err
	}
	return redact(row), nil
}

// allowedTables is the set every Resource declares, built once at init so the
// run-time check above cannot drift from the registry.
var allowedTables = func() map[string]bool {
	out := map[string]bool{}
	for _, res := range resources {
		for _, t := range res.Tables {
			out[t] = true
		}
	}
	return out
}()

func nullableID(id int) any {
	if id == 0 {
		return nil
	}
	return id
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

// --- the read side ---------------------------------------------------------

type logRow struct {
	ID          int64           `json:"id"`
	OccurredAt  string          `json:"occurred_at"`
	ActorUserID *int            `json:"actor_user_id"`
	ActorEmail  string          `json:"actor_email"`
	Resource    string          `json:"resource"`
	ResourceID  string          `json:"resource_id"`
	Action      string          `json:"action"`
	Method      string          `json:"method"`
	Route       string          `json:"route"`
	StatusCode  int             `json:"status_code"`
	Payload     json.RawMessage `json:"payload"`
	BeforeState json.RawMessage `json:"before_state"`
	AfterState  json.RawMessage `json:"after_state"`
	Diff        json.RawMessage `json:"diff"`
}

// HandleAdminAuditLog makes the trail queryable: newest first, filtered by
// resource, resource id or actor.
//
// GET only. There is deliberately no route that edits or removes a record —
// see the package comment.
func HandleAdminAuditLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if v := strings.TrimSpace(q.Get("resource")); v != "" {
		add("resource = $%d", v)
	}
	if v := strings.TrimSpace(q.Get("resource_id")); v != "" {
		add("resource_id = $%d", v)
	}
	if v := strings.TrimSpace(q.Get("actor_email")); v != "" {
		add("LOWER(actor_email) = LOWER($%d)", v)
	}
	if v := strings.TrimSpace(q.Get("action")); v != "" {
		add("action = $%d", v)
	}

	limit := 100
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 500 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		offset = n
	}

	sqlText := `
		SELECT id, occurred_at, actor_user_id, actor_email, resource, resource_id,
		       action, method, route, status_code,
		       COALESCE(payload, 'null'::jsonb), COALESCE(before_state, 'null'::jsonb),
		       COALESCE(after_state, 'null'::jsonb), COALESCE(diff, 'null'::jsonb)
		FROM admin_audit_log`
	if len(where) > 0 {
		sqlText += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, limit, offset)
	sqlText += fmt.Sprintf(" ORDER BY occurred_at DESC, id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := db.DB.Query(sqlText, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := []logRow{}
	for rows.Next() {
		var e logRow
		if err := rows.Scan(
			&e.ID, &e.OccurredAt, &e.ActorUserID, &e.ActorEmail, &e.Resource, &e.ResourceID,
			&e.Action, &e.Method, &e.Route, &e.StatusCode,
			&e.Payload, &e.BeforeState, &e.AfterState, &e.Diff,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
