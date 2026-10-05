package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/lib/pq"
)

func diffSuggestion(before, after map[string]any) map[string]any {
	out := map[string]any{}
	for key, next := range after {
		prev, ok := before[key]
		if !ok || !suggestionValuesEqual(prev, next) {
			out[key] = next
		}
	}
	return out
}

func suggestionValuesEqual(prev, next any) bool {
	a, errA := json.Marshal(prev)
	b, errB := json.Marshal(next)
	if errA != nil || errB != nil {
		return false
	}
	return string(a) == string(b)
}

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

func HandleSuggestionForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	entry, err := loadSuggestionEntryBySlug(slug)
	if err == sql.ErrNoRows || (err == nil && !entry.Published) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if suggestionBlocked(entry.ID, u.ID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestionFormResponse{
		EntryID:         entry.ID,
		Name:            entry.Name,
		Tags:            nonNilStrings(entry.Tags),
		Notes:           entry.Notes,
		LocationID:      entry.LocationID,
		Address:         entry.Address,
		Hours:           jsonObjectOrEmptyListingRaw(entry.Hours),
		DeliveryHours:   jsonObjectOrEmptyListingRaw(entry.DeliveryHours),
		URL:             entry.URL,
		Phone:           entry.Phone,
		SocialLinks:     jsonArrayOrEmptyListingRaw(entry.SocialLinks),
		Languages:       nonNilStrings(entry.Languages),
		HoursEnabled:    entry.HoursEnabled,
		DeliveryEnabled: entry.DeliveryEnabled,
		Category:        entry.Category,
	})
}

func HandleEntrySuggestions(w http.ResponseWriter, r *http.Request) {
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
		EntryID jsonInt                    `json:"entry_id"`
		Fields  map[string]json.RawMessage `json:"fields"`
		Note    string                     `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	entryID := int(body.EntryID)
	if entryID <= 0 {
		http.Error(w, "entry_id required", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	entry, err := loadSuggestionEntryTx(tx, entryID)
	if err == sql.ErrNoRows || (err == nil && !entry.Published) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	blocked, err := suggestionBlockedTx(tx, entry.ID, u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if blocked {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var open bool
	err = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM entry_suggestions WHERE entry_id = $1 AND status = 'open'
		)
	`, entry.ID).Scan(&open)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if open {
		writeListingConflict(w, http.StatusConflict, "suggestion_pending")
		return
	}

	after, err := normalizeSuggestionFields(body.Fields, entry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	changes := diffSuggestion(suggestionBefore(entry, after), after)
	if len(changes) == 0 {
		writeListingConflict(w, http.StatusBadRequest, "empty_diff")
		return
	}
	encoded, err := json.Marshal(changes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var id int
	err = tx.QueryRow(`
		INSERT INTO entry_suggestions (entry_id, user_id, changes, note, status)
		VALUES ($1, $2, $3::jsonb, $4, 'open')
		RETURNING id
	`, entry.ID, u.ID, string(encoded), strings.TrimSpace(body.Note)).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "entry_suggestions_one_open") {
			writeListingConflict(w, http.StatusConflict, "suggestion_pending")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "open"})
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

func suggestionBlocked(entryID, userID int) bool {
	blocked, err := suggestionBlockedTx(db.DB, entryID, userID)
	return err == nil && blocked
}

type suggestionQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

func suggestionBlockedTx(q suggestionQueryer, entryID, userID int) (bool, error) {
	var blocked bool
	err := q.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM entry_members
			WHERE entry_id = $1 AND user_id = $2 AND status = 'active'
			  AND role IN ('owner', 'member')
		)
	`, entryID, userID).Scan(&blocked)
	return blocked, err
}

func loadSuggestionEntryBySlug(slug string) (suggestionEntry, error) {
	var entry suggestionEntry
	var languages []string
	var hours, delivery, social []byte
	err := db.DB.QueryRow(`
		SELECT e.id, e.name, COALESCE(e.slug, ''), COALESCE(e.cat_name, ''), e.location_id,
			COALESCE(e.address, ''), COALESCE(e.notes, ''), COALESCE(e.phone, ''), COALESCE(e.url, ''),
			e.languages, COALESCE(e.hours, '{}'::jsonb), COALESCE(e.delivery_hours, '{}'::jsonb),
			COALESCE(e.social_links, '[]'::jsonb), COALESCE(e.hours_enabled, false),
			COALESCE(e.delivery_enabled, false), e.published
		FROM entries e
		WHERE e.slug = $1
	`, slug).Scan(
		&entry.ID, &entry.Name, &entry.Slug, &entry.Category, &entry.LocationID,
		&entry.Address, &entry.Notes, &entry.Phone, &entry.URL, pq.Array(&languages),
		&hours, &delivery, &social, &entry.HoursEnabled, &entry.DeliveryEnabled, &entry.Published,
	)
	if err != nil {
		return entry, err
	}
	entry.Languages = languages
	entry.Hours = hours
	entry.DeliveryHours = delivery
	entry.SocialLinks = social
	entry.Tags, err = loadSuggestionTags(db.DB, entry.ID)
	return entry, err
}

func loadSuggestionEntryTx(tx *sql.Tx, entryID int) (suggestionEntry, error) {
	var entry suggestionEntry
	var languages []string
	var hours, delivery, social []byte
	err := tx.QueryRow(`
		SELECT e.id, e.name, COALESCE(e.slug, ''), COALESCE(e.cat_name, ''), e.location_id,
			COALESCE(e.address, ''), COALESCE(e.notes, ''), COALESCE(e.phone, ''), COALESCE(e.url, ''),
			e.languages, COALESCE(e.hours, '{}'::jsonb), COALESCE(e.delivery_hours, '{}'::jsonb),
			COALESCE(e.social_links, '[]'::jsonb), COALESCE(e.hours_enabled, false),
			COALESCE(e.delivery_enabled, false), e.published
		FROM entries e
		WHERE e.id = $1
		FOR UPDATE
	`, entryID).Scan(
		&entry.ID, &entry.Name, &entry.Slug, &entry.Category, &entry.LocationID,
		&entry.Address, &entry.Notes, &entry.Phone, &entry.URL, pq.Array(&languages),
		&hours, &delivery, &social, &entry.HoursEnabled, &entry.DeliveryEnabled, &entry.Published,
	)
	if err != nil {
		return entry, err
	}
	entry.Languages = languages
	entry.Hours = hours
	entry.DeliveryHours = delivery
	entry.SocialLinks = social
	entry.Tags, err = loadSuggestionTags(tx, entry.ID)
	return entry, err
}

type suggestionTagQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func loadSuggestionTags(q suggestionTagQueryer, entryID int) ([]string, error) {
	rows, err := q.Query(`
		SELECT t.name FROM tags t
		JOIN entry_tags et ON et.tag_id = t.id
		WHERE et.entry_id = $1
		ORDER BY t.name ASC
	`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

func suggestionOffersDelivery(entry suggestionEntry) bool {
	if strings.TrimSpace(entry.Name) == "Lámsza.com" {
		return false
	}
	return entry.DeliveryEnabled && utils.CategoryOffersDelivery(entry.Category)
}

func normalizeSuggestionFields(raw map[string]json.RawMessage, entry suggestionEntry) (map[string]any, error) {
	out := map[string]any{}
	for key, val := range raw {
		if !suggestionFieldKeys[key] {
			continue
		}
		if key == "hours" && !entry.HoursEnabled {
			continue
		}
		if key == "delivery_hours" && !suggestionOffersDelivery(entry) {
			continue
		}
		switch key {
		case "name", "notes", "address", "phone":
			var s string
			if err := json.Unmarshal(val, &s); err != nil {
				return nil, err
			}
			out[key] = strings.TrimSpace(s)
		case "url":
			var s string
			if err := json.Unmarshal(val, &s); err != nil {
				return nil, err
			}
			s = strings.TrimSpace(s)
			if !validListingURL(s) {
				return nil, errInvalidSuggestionURL
			}
			out[key] = s
		case "location_id":
			var id int
			if err := json.Unmarshal(val, &id); err != nil {
				return nil, err
			}
			if id <= 0 {
				return nil, errInvalidSuggestionLocation
			}
			out[key] = id
		case "tags":
			var tags []string
			if err := json.Unmarshal(val, &tags); err != nil {
				return nil, err
			}
			clean := []string{}
			seen := map[string]bool{}
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag == "" || seen[tag] {
					continue
				}
				seen[tag] = true
				clean = append(clean, tag)
			}
			sort.Strings(clean)
			out[key] = clean
		case "languages":
			var langs []string
			if err := json.Unmarshal(val, &langs); err != nil {
				return nil, err
			}
			out[key] = normalizeListingLanguages(langs)
		case "hours", "delivery_hours":
			out[key] = suggestionJSONValue(jsonObjectOrEmptyListingRaw(val))
		case "social_links":
			out[key] = suggestionJSONValue(sanitizeSocialLinks(val))
		}
	}
	return out, nil
}

type suggestionError string

func (e suggestionError) Error() string { return string(e) }

const (
	errInvalidSuggestionURL      suggestionError = "invalid url"
	errInvalidSuggestionLocation suggestionError = "invalid location_id"
)

func suggestionBefore(entry suggestionEntry, after map[string]any) map[string]any {
	current := map[string]any{
		"name":           strings.TrimSpace(entry.Name),
		"tags":           nonNilStrings(entry.Tags),
		"notes":          entry.Notes,
		"location_id":    entry.LocationID,
		"address":        entry.Address,
		"hours":          suggestionJSONValue(jsonObjectOrEmptyListingRaw(entry.Hours)),
		"delivery_hours": suggestionJSONValue(jsonObjectOrEmptyListingRaw(entry.DeliveryHours)),
		"url":            strings.TrimSpace(entry.URL),
		"phone":          strings.TrimSpace(entry.Phone),
		"social_links":   suggestionJSONValue(sanitizeSocialLinks(entry.SocialLinks)),
		"languages":      normalizeListingLanguages(entry.Languages),
	}
	before := map[string]any{}
	for key := range after {
		before[key] = current[key]
	}
	return before
}

func suggestionJSONValue(raw json.RawMessage) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return map[string]any{}
	}
	return v
}

func jsonObjectOrEmptyListingRaw(raw json.RawMessage) json.RawMessage {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || !json.Valid(raw) || !strings.HasPrefix(s, "{") {
		return json.RawMessage(`{}`)
	}
	return raw
}

func jsonArrayOrEmptyListingRaw(raw json.RawMessage) json.RawMessage {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || !json.Valid(raw) || !strings.HasPrefix(s, "[") {
		return json.RawMessage(`[]`)
	}
	return raw
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

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
