package handlers

import (
	"backend/internal/auth"
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

type AttractionContributor struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func attachAttractionPublicExtras(a *Attraction) {
	a.Contributors = loadAttractionContributors(a.ID)
	a.SuggestionPending = attractionSuggestionOpen(a.ID)
}

func loadAttractionContributors(attractionID int) []AttractionContributor {
	out := []AttractionContributor{}
	rows, err := db.DB.Query(`
		SELECT u.id, COALESCE(NULLIF(btrim(u.display_name), ''), NULLIF(btrim(u.name), ''), u.email), COALESCE(u.picture, '')
		FROM attraction_contributors c
		JOIN users u ON u.id = c.user_id
		WHERE c.attraction_id = $1
		ORDER BY c.created_at ASC, u.id ASC
	`, attractionID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var person AttractionContributor
		if rows.Scan(&person.ID, &person.Name, &person.Picture) == nil {
			out = append(out, person)
		}
	}
	return out
}

func attractionSuggestionOpen(attractionID int) bool {
	var open bool
	_ = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM attraction_suggestions WHERE attraction_id = $1 AND status = 'open'
		)
	`, attractionID).Scan(&open)
	return open
}

func rememberAttractionEditor(r *http.Request, attractionID int) {
	if attractionID <= 0 || r == nil {
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		return
	}
	addAttractionContributor(attractionID, u.ID)
}

func addAttractionContributor(attractionID, userID int) {
	if attractionID <= 0 || userID <= 0 {
		return
	}
	_, _ = db.DB.Exec(`
		INSERT INTO attraction_contributors (attraction_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (attraction_id, user_id) DO NOTHING
	`, attractionID, userID)
}

func HandleAttractionSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		AttractionID int            `json:"attraction_id"`
		Fields       map[string]any `json:"fields"`
		Note         string         `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.AttractionID <= 0 {
		http.Error(w, "attraction_id required", http.StatusBadRequest)
		return
	}
	current, err := loadAttractionSuggestionSource(body.AttractionID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	changes := diffAttractionFields(current, body.Fields)
	if len(changes) == 0 {
		writeAttractionJSONError(w, http.StatusBadRequest, "empty_diff")
		return
	}
	encoded, err := json.Marshal(changes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var id int
	err = db.DB.QueryRow(`
		INSERT INTO attraction_suggestions (attraction_id, user_id, changes, note, status)
		VALUES ($1, $2, $3::jsonb, $4, 'open')
		RETURNING id
	`, body.AttractionID, u.ID, string(encoded), strings.TrimSpace(body.Note)).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "attraction_suggestions_one_open") || strings.Contains(err.Error(), "duplicate key") {
			writeAttractionJSONError(w, http.StatusConflict, "suggestion_pending")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "open"})
}

func HandleAdminAttractionSuggestions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query(`
			SELECT s.id, s.attraction_id, a.name, u.id, COALESCE(NULLIF(btrim(u.display_name), ''), NULLIF(btrim(u.name), ''), u.email),
				s.changes, COALESCE(s.note, ''), s.created_at
			FROM attraction_suggestions s
			JOIN attractions a ON a.id = s.attraction_id
			JOIN users u ON u.id = s.user_id
			WHERE s.status = 'open'
			ORDER BY s.created_at ASC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type row struct {
			ID             int             `json:"id"`
			AttractionID   int             `json:"attraction_id"`
			AttractionName string          `json:"attraction_name"`
			UserID         int             `json:"user_id"`
			UserName       string          `json:"user_name"`
			Changes        json.RawMessage `json:"changes"`
			Note           string          `json:"note"`
			CreatedAt      string          `json:"created_at"`
		}
		out := []row{}
		for rows.Next() {
			var item row
			var createdAt sql.NullTime
			if err := rows.Scan(&item.ID, &item.AttractionID, &item.AttractionName, &item.UserID, &item.UserName, &item.Changes, &item.Note, &createdAt); err != nil {
				continue
			}
			if createdAt.Valid {
				item.CreatedAt = createdAt.Time.Format("2006-01-02 15:04")
			}
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	case http.MethodPost:
		var body struct {
			ID     int    `json:"id"`
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.ID <= 0 || (body.Action != "accept" && body.Action != "deny") {
			http.Error(w, "id and action required", http.StatusBadRequest)
			return
		}
		if err := reviewAttractionSuggestion(r, body.ID, body.Action); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func reviewAttractionSuggestion(r *http.Request, id int, action string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var attractionID, userID int
	var changesRaw []byte
	var status string
	err = tx.QueryRow(`
		SELECT attraction_id, user_id, changes, status FROM attraction_suggestions WHERE id = $1 FOR UPDATE
	`, id).Scan(&attractionID, &userID, &changesRaw, &status)
	if err != nil {
		return err
	}
	if status != "open" {
		return sql.ErrNoRows
	}
	if action == "deny" {
		if _, err = tx.Exec(`UPDATE attraction_suggestions SET status = 'denied' WHERE id = $1`, id); err != nil {
			return err
		}
		return tx.Commit()
	}
	var changes map[string]any
	if err := json.Unmarshal(changesRaw, &changes); err != nil {
		return err
	}
	if err := applyAttractionSuggestion(tx, attractionID, changes); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE attraction_suggestions SET status = 'accepted' WHERE id = $1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		INSERT INTO attraction_contributors (attraction_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (attraction_id, user_id) DO NOTHING
	`, attractionID, userID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	rememberAttractionEditor(r, attractionID)
	return nil
}

func applyAttractionSuggestion(tx *sql.Tx, attractionID int, changes map[string]any) error {
	if text, ok := stringField(changes, "description"); ok {
		if _, err := tx.Exec(`UPDATE attractions SET description = $1 WHERE id = $2`, text, attractionID); err != nil {
			return err
		}
	}
	if text, ok := stringField(changes, "content"); ok {
		if _, err := tx.Exec(`UPDATE attractions SET content = $1 WHERE id = $2`, text, attractionID); err != nil {
			return err
		}
	}
	if text, ok := stringField(changes, "name_ro"); ok {
		if _, err := tx.Exec(`UPDATE attractions SET name_ro = $1 WHERE id = $2`, text, attractionID); err != nil {
			return err
		}
	}
	if text, ok := stringField(changes, "name_de"); ok {
		if _, err := tx.Exec(`UPDATE attractions SET name_de = $1 WHERE id = $2`, text, attractionID); err != nil {
			return err
		}
	}
	if lines, ok := changes["activities"]; ok {
		if _, err := tx.Exec(`UPDATE attractions SET activities = $1 WHERE id = $2`, joinActivities(activityLinesFromAny(lines)), attractionID); err != nil {
			return err
		}
	}
	if lines, ok := changes["prohibitions"]; ok {
		if _, err := tx.Exec(`UPDATE attractions SET prohibitions = $1 WHERE id = $2`, joinActivities(activityLinesFromAny(lines)), attractionID); err != nil {
			return err
		}
	}
	return nil
}

func stringField(changes map[string]any, key string) (string, bool) {
	raw, ok := changes[key]
	if !ok || raw == nil {
		return "", false
	}
	text, ok := raw.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func activityLinesFromAny(raw any) []string {
	switch lines := raw.(type) {
	case []string:
		return lines
	case []any:
		out := make([]string, 0, len(lines))
		for _, line := range lines {
			if text, ok := line.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

type attractionSuggestionSource struct {
	Description  string
	Content      string
	NameRo       string
	NameDe       string
	Activities   []string
	Prohibitions []string
}

func loadAttractionSuggestionSource(id int) (attractionSuggestionSource, error) {
	var src attractionSuggestionSource
	var activities, prohibitions string
	err := db.DB.QueryRow(`
		SELECT COALESCE(description, ''), COALESCE(content, ''), COALESCE(name_ro, ''), COALESCE(name_de, ''),
			COALESCE(activities, ''), COALESCE(prohibitions, '')
		FROM attractions WHERE id = $1
	`, id).Scan(&src.Description, &src.Content, &src.NameRo, &src.NameDe, &activities, &prohibitions)
	src.Activities = splitActivities(activities)
	src.Prohibitions = splitActivities(prohibitions)
	return src, err
}

func diffAttractionFields(current attractionSuggestionSource, fields map[string]any) map[string]any {
	out := map[string]any{}
	if fields == nil {
		return out
	}
	if text, ok := stringField(fields, "description"); ok && text != strings.TrimSpace(current.Description) {
		out["description"] = text
	}
	if text, ok := stringField(fields, "content"); ok && text != strings.TrimSpace(current.Content) {
		out["content"] = text
	}
	if text, ok := stringField(fields, "name_ro"); ok && text != strings.TrimSpace(current.NameRo) {
		out["name_ro"] = text
	}
	if text, ok := stringField(fields, "name_de"); ok && text != strings.TrimSpace(current.NameDe) {
		out["name_de"] = text
	}
	if _, ok := fields["activities"]; ok {
		next := activityLinesFromAny(fields["activities"])
		if strings.Join(splitActivities(strings.Join(next, "\n")), "\n") != strings.Join(current.Activities, "\n") {
			out["activities"] = splitActivities(strings.Join(next, "\n"))
		}
	}
	if _, ok := fields["prohibitions"]; ok {
		next := activityLinesFromAny(fields["prohibitions"])
		if strings.Join(splitActivities(strings.Join(next, "\n")), "\n") != strings.Join(current.Prohibitions, "\n") {
			out["prohibitions"] = splitActivities(strings.Join(next, "\n"))
		}
	}
	return out
}

func writeAttractionJSONError(w http.ResponseWriter, code int, errorCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": errorCode})
}
