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
