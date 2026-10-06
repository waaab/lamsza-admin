package account

import (
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lib/pq"
)

var suggestionFieldKeys = map[string]bool{
	"name": true, "tags": true, "notes": true, "location_id": true,
	"address": true, "hours": true, "delivery_hours": true, "url": true,
	"phone": true, "social_links": true, "languages": true,
}

type suggestionEntry struct {
	ID              int
	Name            string
	Slug            string
	Category        string
	LocationID      int
	Address         string
	Notes           string
	Phone           string
	URL             string
	Languages       []string
	Tags            []string
	Hours           json.RawMessage
	DeliveryHours   json.RawMessage
	SocialLinks     json.RawMessage
	HoursEnabled    bool
	DeliveryEnabled bool
	Published       bool
}

type suggestionFormResponse struct {
	EntryID         int             `json:"entry_id"`
	Name            string          `json:"name"`
	Tags            []string        `json:"tags"`
	Notes           string          `json:"notes"`
	LocationID      int             `json:"location_id"`
	Address         string          `json:"address"`
	Hours           json.RawMessage `json:"hours"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	SocialLinks     json.RawMessage `json:"social_links"`
	Languages       []string        `json:"languages"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	Category        string          `json:"category"`
}

func HandleListingQueueSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID     int    `json:"id"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.ID <= 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if body.Action != "accept" && body.Action != "deny" {
		http.Error(w, "action must be accept or deny", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var entryID int
	var changesRaw []byte
	var status string
	err = tx.QueryRow(`
		SELECT entry_id, changes, status FROM entry_suggestions WHERE id = $1 FOR UPDATE
	`, body.ID).Scan(&entryID, &changesRaw, &status)
	if err == sql.ErrNoRows || (err == nil && status != "open") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if body.Action == "deny" {
		_, err = tx.Exec(`UPDATE entry_suggestions SET status = 'denied' WHERE id = $1`, body.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	var changes map[string]any
	if err := json.Unmarshal(changesRaw, &changes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var lockedID int
	if err = tx.QueryRow(`SELECT id FROM entries WHERE id = $1 FOR UPDATE`, entryID).Scan(&lockedID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := applySuggestionChanges(tx, entryID, changes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = tx.Exec(`UPDATE entry_suggestions SET status = 'accepted' WHERE id = $1`, body.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type suggestionQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

type suggestionTagQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

type suggestionError string

func applySuggestionChanges(tx *sql.Tx, entryID int, changes map[string]any) error {
	sets := []string{}
	args := []any{}
	n := 1
	add := func(expr string, val any) {
		sets = append(sets, expr)
		args = append(args, val)
		n++
	}
	if v, ok := changes["name"]; ok {
		add("name = $"+itoa(n), strings.TrimSpace(suggestionString(v)))
	}
	if v, ok := changes["notes"]; ok {
		add("notes = $"+itoa(n), suggestionString(v))
	}
	if v, ok := changes["address"]; ok {
		add("address = $"+itoa(n), suggestionString(v))
	}
	if v, ok := changes["phone"]; ok {
		add("phone = $"+itoa(n), suggestionString(v))
	}
	if v, ok := changes["url"]; ok {
		add("url = $"+itoa(n), suggestionString(v))
	}
	if v, ok := changes["location_id"]; ok {
		add("location_id = $"+itoa(n), suggestionInt(v))
	}
	if v, ok := changes["languages"]; ok {
		add("languages = $"+itoa(n), pq.Array(suggestionStrings(v)))
	}
	if v, ok := changes["hours"]; ok {
		raw, _ := json.Marshal(v)
		add("hours = $"+itoa(n)+"::jsonb", string(raw))
	}
	if v, ok := changes["delivery_hours"]; ok {
		raw, _ := json.Marshal(v)
		add("delivery_hours = $"+itoa(n)+"::jsonb", string(raw))
	}
	if v, ok := changes["social_links"]; ok {
		raw, _ := json.Marshal(v)
		add("social_links = $"+itoa(n)+"::jsonb", string(sanitizeSocialLinks(raw)))
	}
	if len(sets) > 0 {
		args = append(args, entryID)
		_, err := tx.Exec(`UPDATE entries SET `+strings.Join(sets, ", ")+` WHERE id = $`+itoa(n), args...)
		if err != nil {
			return err
		}
	}
	if v, ok := changes["tags"]; ok {
		if err := replaceSuggestionTags(tx, entryID, suggestionStrings(v)); err != nil {
			return err
		}
	}
	return nil
}

func replaceSuggestionTags(tx *sql.Tx, entryID int, tags []string) error {
	if _, err := tx.Exec(`DELETE FROM entry_tags WHERE entry_id = $1`, entryID); err != nil {
		return err
	}
	for _, tagName := range tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		var tagID int
		err := tx.QueryRow(`
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tagName).Scan(&tagID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)`, entryID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func suggestionString(v any) string {
	s, _ := v.(string)
	return s
}

func suggestionInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func suggestionStrings(v any) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return []string{}
	}
}

func itoa(n int) string {
	return strings.TrimPrefix(strings.ReplaceAll(jsonNumber(n), " ", ""), "+")
}

func jsonNumber(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
