package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"backend/internal/directory"
	"backend/internal/utils"
	"backend/internal/webdomain"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
)

func migrateEntryMembers() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS entry_members (
			entry_id INT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(16) NOT NULL,
			status VARCHAR(16) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (entry_id, user_id)
		)
	`)
	if err != nil {
		log.Printf("entry_members table: %v", err)
		return
	}
	_, err = db.DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS entry_members_one_active_owner
		ON entry_members (entry_id)
		WHERE role = 'owner' AND status = 'active'
	`)
	if err != nil {
		log.Printf("entry_members unique index: %v", err)
	}
}

type membershipResponse struct {
	EntryID int    `json:"entry_id"`
	Role    string `json:"role"`
	Status  string `json:"status"`
}

// jsonInt accepts a JSON number or a numeric string. Public entries expose id as a string.
type jsonInt int

func (n *jsonInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return fmt.Errorf("invalid number")
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		*n = jsonInt(v)
		return nil
	}
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = jsonInt(v)
	return nil
}

type claimBody struct {
	EntryID jsonInt `json:"entry_id"`
}

type createListingBody struct {
	WebsiteID       int             `json:"website_id"`
	Name            string          `json:"name"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	Photos          json.RawMessage `json:"photos"`
	Tags            []string        `json:"tags"`
}

type updateListingBody struct {
	Name            string          `json:"name"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	SocialLinks     json.RawMessage `json:"social_links"`
	Photos          json.RawMessage `json:"photos"`
	RatingsEnabled  bool            `json:"ratings_enabled"`
	Tags            []string        `json:"tags"`
}

type listingItem struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Published  bool   `json:"published"`
	Verified   bool   `json:"verified"`
	LocationID int    `json:"location_id"`
	CategoryID int    `json:"category_id"`
	TypeID     int    `json:"type_id"`
}

type listingsResponse struct {
	Owned       []listingItem `json:"owned"`
	Member      []listingItem `json:"member"`
	Pending     []listingItem `json:"pending"`
	Unpublished []listingItem `json:"unpublished"`
}

func scanMembershipDB(entryID, userID int) (membershipResponse, bool, error) {
	var resp membershipResponse
	err := db.DB.QueryRow(`
		SELECT entry_id, role, status
		FROM entry_members
		WHERE entry_id = $1 AND user_id = $2
	`, entryID, userID).Scan(&resp.EntryID, &resp.Role, &resp.Status)
	if err == sql.ErrNoRows {
		return resp, false, nil
	}
	if err != nil {
		return resp, false, err
	}
	return resp, true, nil
}

func canEditListing(role, status string) bool {
	return status == "active" && (role == "owner" || role == "member")
}

func isActiveOwner(role, status string) bool {
	return role == "owner" && status == "active"
}

const maxListingPhotos = 24
const maxListingPhotoTextRunes = 500
const listingImageURLPrefix = "/api/media/entry-images/"
const defaultListingPhotoWidth = 1600
const defaultListingPhotoHeight = 1200

type listingPhoto struct {
	URL         string `json:"url"`
	Alt         string `json:"alt"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Copyright   string `json:"copyright"`
	Uploader    string `json:"uploader"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type listingSocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func listingDeliveryEnabled(catName, name string, requested bool) bool {
	if !requested {
		return false
	}
	if strings.TrimSpace(name) == "Lámsza.com" {
		return false
	}
	return utils.CategoryOffersDelivery(catName)
}

func sanitizeSocialLinks(raw json.RawMessage) json.RawMessage {
	raw = photosArrayOrEmpty(raw)
	var in []listingSocialLink
	if err := json.Unmarshal(raw, &in); err != nil {
		return json.RawMessage(`[]`)
	}
	out := make([]listingSocialLink, 0, len(in))
	for _, link := range in {
		url := strings.TrimSpace(link.URL)
		lower := strings.ToLower(url)
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			continue
		}
		out = append(out, listingSocialLink{
			Label: strings.TrimSpace(link.Label),
			URL:   url,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

func photosOrEmpty(raw json.RawMessage) string {
	return string(sanitizeListingPhotos(raw))
}

func sanitizeListingPhotos(raw json.RawMessage) json.RawMessage {
	return sanitizeListingPhotosLimit(raw, maxListingPhotos)
}

func sanitizeListingPhotosLimit(raw json.RawMessage, limit int) json.RawMessage {
	if limit < 1 {
		limit = 1
	}
	raw = photosArrayOrEmpty(raw)
	var in []listingPhoto
	if err := json.Unmarshal(raw, &in); err != nil {
		return json.RawMessage(`[]`)
	}
	out := make([]listingPhoto, 0, len(in))
	for _, p := range in {
		if len(out) >= limit {
			break
		}
		url := strings.TrimSpace(p.URL)
		if !validListingPhotoURL(url) {
			continue
		}
		w, h := p.Width, p.Height
		if w < 1 || w > 10000 {
			w = defaultListingPhotoWidth
		}
		if h < 1 || h > 10000 {
			h = defaultListingPhotoHeight
		}
		out = append(out, listingPhoto{
			URL:         url,
			Alt:         clipListingPhotoRunes(strings.TrimSpace(p.Alt), maxListingPhotoTextRunes),
			Title:       clipListingPhotoRunes(strings.TrimSpace(p.Title), maxListingPhotoTextRunes),
			Description: clipListingPhotoRunes(strings.TrimSpace(p.Description), maxListingPhotoTextRunes),
			Copyright:   clipListingPhotoRunes(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p.Copyright), "©")), maxListingPhotoTextRunes),
			Uploader:    clipListingPhotoRunes(strings.TrimSpace(p.Uploader), maxListingPhotoTextRunes),
			Width:       w,
			Height:      h,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

func normalizeListingLanguages(in []string) []string {
	allowed := map[string]struct{}{"RO": {}, "HU": {}, "DE": {}, "EN": {}}
	seen := map[string]struct{}{}
	for _, part := range in {
		code := strings.ToUpper(strings.TrimSpace(part))
		if _, ok := allowed[code]; ok {
			seen[code] = struct{}{}
		}
	}
	out := make([]string, 0, 4)
	for _, code := range []string{"RO", "HU", "DE", "EN"} {
		if _, ok := seen[code]; ok {
			out = append(out, code)
		}
	}
	if len(out) == 0 {
		return []string{"HU"}
	}
	return out
}

func listingClaimed(entryID int) (bool, error) {
	var claimed bool
	err := db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM entry_members
			WHERE entry_id = $1 AND role = 'owner' AND status = 'active'
		)
	`, entryID).Scan(&claimed)
	return claimed, err
}

func photosArrayOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`[]`)
	}
	if !json.Valid(b) || !strings.HasPrefix(s, "[") {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

func validListingURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return true
	}
	lower := strings.ToLower(u)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func nextUniqueEntrySlug(tx *sql.Tx, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("empty slug")
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		var existingID int
		err := tx.QueryRow(`SELECT id FROM entries WHERE slug = $1`, candidate).Scan(&existingID)
		if err == sql.ErrNoRows {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate unique slug")
}

func validListingPhotoURL(u string) bool {
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.Contains(u, "://") && !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	if strings.HasPrefix(u, listingImageURLPrefix) {
		base := u[len(listingImageURLPrefix):]
		return base != "" && !strings.Contains(base, "/") && !strings.Contains(base, "..")
	}
	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}

func clipListingPhotoRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func scanMembership(tx *sql.Tx, entryID, userID int) (membershipResponse, bool, error) {
	var resp membershipResponse
	err := tx.QueryRow(`
		SELECT entry_id, role, status
		FROM entry_members
		WHERE entry_id = $1 AND user_id = $2
	`, entryID, userID).Scan(&resp.EntryID, &resp.Role, &resp.Status)
	if err == sql.ErrNoRows {
		return resp, false, nil
	}
	if err != nil {
		return resp, false, err
	}
	return resp, true, nil
}

func hasActiveOwner(tx *sql.Tx, entryID int) (bool, error) {
	var exists bool
	err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM entry_members
			WHERE entry_id = $1 AND role = 'owner' AND status = 'active'
		)
	`, entryID).Scan(&exists)
	return exists, err
}

func insertMembership(tx *sql.Tx, entryID, userID int, role, status string) (membershipResponse, error) {
	resp := membershipResponse{EntryID: entryID, Role: role, Status: status}
	_, err := tx.Exec(`
		INSERT INTO entry_members (entry_id, user_id, role, status)
		VALUES ($1, $2, $3, $4)
	`, entryID, userID, role, status)
	return resp, err
}

func writeMembership(w http.ResponseWriter, resp membershipResponse) {
	json.NewEncoder(w).Encode(resp)
}

func HandleClaimListing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body claimBody
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

	var published bool
	err = tx.QueryRow(`
		SELECT published FROM entries WHERE id = $1 FOR UPDATE
	`, entryID).Scan(&published)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !published {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var pendingClaim bool
	err = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM entry_members
			WHERE entry_id = $1 AND role = 'owner' AND status = 'pending'
		)
	`, entryID).Scan(&pendingClaim)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pendingClaim {
		writeListingConflict(w, http.StatusConflict, "claim_pending")
		return
	}

	existing, found, err := scanMembership(tx, entryID, u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if found {
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeMembership(w, existing)
		return
	}

	ownerExists, err := hasActiveOwner(tx, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp membershipResponse
	if !ownerExists {
		resp, err = insertMembership(tx, entryID, u.ID, "owner", "pending")
	} else {
		resp, err = insertMembership(tx, entryID, u.ID, "member", "pending")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeMembership(w, resp)
}

func HandleListings(w http.ResponseWriter, r *http.Request) {
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if idStr := strings.TrimSpace(r.URL.Query().Get("id")); idStr != "" {
			handleGetListingDetail(w, r, u.ID, idStr)
		} else {
			handleListListings(w, u.ID)
		}
	case http.MethodPost:
		handleCreateListing(w, r, u.ID)
	case http.MethodPatch:
		handleUpdateListing(w, r, u.ID)
	case http.MethodDelete:
		handleDeleteListing(w, r, u.ID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type catalogItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id,omitempty"`
}

type listingCatalogResponse struct {
	Categories []catalogItem `json:"categories"`
	Types      []catalogItem `json:"types"`
}

func HandleListingCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp := listingCatalogResponse{
		Categories: []catalogItem{},
		Types:      []catalogItem{},
	}

	rows, err := db.DB.Query(`SELECT id, name, parent_id FROM entry_categories ORDER BY COALESCE(parent_id, id), sort_order, id`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var item catalogItem
		var parentID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &parentID); err != nil {
			rows.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			item.ParentID = &pid
		}
		resp.Categories = append(resp.Categories, item)
	}
	rows.Close()

	rows, err = db.DB.Query(`SELECT id, name FROM entry_types ORDER BY name ASC, id ASC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var item catalogItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Types = append(resp.Types, item)
	}
	rows.Close()

	json.NewEncoder(w).Encode(resp)
}

type listingDetailResponse struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	SocialLinks     json.RawMessage `json:"social_links"`
	Photos          json.RawMessage `json:"photos"`
	Published       bool            `json:"published"`
	RatingsEnabled  bool            `json:"ratings_enabled"`
	Tags            []string        `json:"tags"`
}

func handleGetListingDetail(w http.ResponseWriter, r *http.Request, userID int, idStr string) {
	entryID, err := strconv.Atoi(idStr)
	if err != nil || entryID <= 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	membership, found, err := scanMembershipDB(entryID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !canEditListing(membership.Role, membership.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var detail listingDetailResponse
	var pqLanguages []string
	var hours, delivery, socialLinks, photos []byte
	err = db.DB.QueryRow(`
		SELECT e.id, e.name, COALESCE(e.slug, ''), e.location_id, e.category_id, e.type_id,
			COALESCE(e.url, ''), COALESCE(e.phone, ''), COALESCE(e.address, ''), COALESCE(e.notes, ''),
			e.languages, COALESCE(e.hours, '{}'::jsonb), COALESCE(e.hours_enabled, false),
			COALESCE(e.delivery_hours, '{}'::jsonb), COALESCE(e.delivery_enabled, false),
			COALESCE(e.social_links, '[]'::jsonb),
			COALESCE(e.photos, '[]'::jsonb), e.published, COALESCE(e.ratings_enabled, false)
		FROM entries e
		WHERE e.id = $1
	`, entryID).Scan(&detail.ID, &detail.Name, &detail.Slug, &detail.LocationID, &detail.CategoryID, &detail.TypeID,
		&detail.URL, &detail.Phone, &detail.Address, &detail.Notes, pq.Array(&pqLanguages),
		&hours, &detail.HoursEnabled, &delivery, &detail.DeliveryEnabled, &socialLinks,
		&photos, &detail.Published, &detail.RatingsEnabled)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(pqLanguages) == 0 {
		detail.Languages = []string{"HU"}
	} else {
		detail.Languages = pqLanguages
	}
	detail.Hours = json.RawMessage(jsonObjectOrEmptyListing(hours))
	detail.DeliveryHours = json.RawMessage(jsonObjectOrEmptyListing(delivery))
	detail.SocialLinks = photosArrayOrEmpty(socialLinks)
	detail.Photos = sanitizeListingPhotos(photos)
	detail.Tags, err = loadSuggestionTags(db.DB, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail.Tags == nil {
		detail.Tags = []string{}
	}
	detail.CategoryIDs, err = directory.LoadEntryCategoryIDs(db.DB, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(detail.CategoryIDs) == 0 && detail.CategoryID > 0 {
		detail.CategoryIDs = []int{detail.CategoryID}
	}
	json.NewEncoder(w).Encode(detail)
}

type listingMemberRow struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

func HandleListingMembers(w http.ResponseWriter, r *http.Request) {
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleListListingMembers(w, r, u.ID)
	case http.MethodPost:
		handleDecideListingMember(w, r, u.ID)
	case http.MethodDelete:
		handleDeleteListingMember(w, r, u.ID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListListingMembers(w http.ResponseWriter, r *http.Request, ownerUserID int) {
	entryID, err := strconv.Atoi(r.URL.Query().Get("entry_id"))
	if err != nil || entryID <= 0 {
		http.Error(w, "entry_id required", http.StatusBadRequest)
		return
	}

	owner, found, err := scanMembershipDB(entryID, ownerUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !isActiveOwner(owner.Role, owner.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := db.DB.Query(`
		SELECT m.user_id, COALESCE(u.email, ''), m.role, m.status
		FROM entry_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.entry_id = $1 AND m.role = 'member' AND m.status IN ('pending', 'active')
		ORDER BY u.email ASC, m.user_id ASC
	`, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := []listingMemberRow{}
	for rows.Next() {
		var row listingMemberRow
		if err := rows.Scan(&row.UserID, &row.Email, &row.Role, &row.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, row)
	}
	json.NewEncoder(w).Encode(out)
}

func handleDecideListingMember(w http.ResponseWriter, r *http.Request, ownerUserID int) {
	var body struct {
		EntryID int    `json:"entry_id"`
		UserID  int    `json:"user_id"`
		Action  string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.EntryID <= 0 || body.UserID <= 0 {
		http.Error(w, "entry_id and user_id required", http.StatusBadRequest)
		return
	}
	if body.Action != "accept" && body.Action != "deny" {
		http.Error(w, "action must be accept or deny", http.StatusBadRequest)
		return
	}

	owner, found, err := scanMembershipDB(body.EntryID, ownerUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !isActiveOwner(owner.Role, owner.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var targetRole, targetStatus string
	err = db.DB.QueryRow(`
		SELECT role, status FROM entry_members WHERE entry_id = $1 AND user_id = $2
	`, body.EntryID, body.UserID).Scan(&targetRole, &targetStatus)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if targetRole != "member" || targetStatus != "pending" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if body.Action == "accept" {
		_, err = db.DB.Exec(`
			UPDATE entry_members SET status = 'active'
			WHERE entry_id = $1 AND user_id = $2 AND role = 'member' AND status = 'pending'
		`, body.EntryID, body.UserID)
	} else {
		_, err = db.DB.Exec(`
			DELETE FROM entry_members
			WHERE entry_id = $1 AND user_id = $2 AND role = 'member' AND status = 'pending'
		`, body.EntryID, body.UserID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeOK(w)
}

func handleDeleteListingMember(w http.ResponseWriter, r *http.Request, ownerUserID int) {
	entryID, err := strconv.Atoi(r.URL.Query().Get("entry_id"))
	if err != nil || entryID <= 0 {
		http.Error(w, "entry_id required", http.StatusBadRequest)
		return
	}
	targetUserID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil || targetUserID <= 0 {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	owner, found, err := scanMembershipDB(entryID, ownerUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !isActiveOwner(owner.Role, owner.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var targetRole string
	err = db.DB.QueryRow(`
		SELECT role FROM entry_members WHERE entry_id = $1 AND user_id = $2
	`, entryID, targetUserID).Scan(&targetRole)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if targetRole == "owner" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = db.DB.Exec(`DELETE FROM entry_members WHERE entry_id = $1 AND user_id = $2`, entryID, targetUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeOK(w)
}

func handleListListings(w http.ResponseWriter, userID int) {
	resp := listingsResponse{
		Owned:       []listingItem{},
		Member:      []listingItem{},
		Pending:     []listingItem{},
		Unpublished: []listingItem{},
	}
	rows, err := db.DB.Query(`
		SELECT e.id, e.name, COALESCE(e.slug, ''), m.role, m.status, e.published, COALESCE(e.verified, false),
			e.location_id, e.category_id, e.type_id
		FROM entry_members m
		JOIN entries e ON e.id = m.entry_id
		WHERE m.user_id = $1
		ORDER BY e.name ASC, e.id ASC
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item listingItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Role, &item.Status, &item.Published, &item.Verified,
			&item.LocationID, &item.CategoryID, &item.TypeID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch {
		case item.Role == "owner" && item.Status == "active" && !item.Published:
			resp.Unpublished = append(resp.Unpublished, item)
		case item.Role == "owner" && item.Status == "active":
			resp.Owned = append(resp.Owned, item)
		case item.Status == "pending":
			resp.Pending = append(resp.Pending, item)
		case item.Role == "member" && item.Status == "active":
			resp.Member = append(resp.Member, item)
		}
	}
	json.NewEncoder(w).Encode(resp)
}

func jsonObjectOrEmptyListing(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func writeListingConflict(w http.ResponseWriter, code int, errorCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": errorCode})
}

func writeListingFieldError(w http.ResponseWriter, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": "invalid", "field": field})
}

func validateLeafCategoryID(categoryID int) error {
	if categoryID <= 0 {
		return sql.ErrNoRows
	}
	var parentID sql.NullInt64
	err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, categoryID).Scan(&parentID)
	if err != nil {
		return err
	}
	if !parentID.Valid {
		return sql.ErrNoRows
	}
	return nil
}

func validateEntryTypeID(typeID int) error {
	if typeID <= 0 {
		return sql.ErrNoRows
	}
	var exists bool
	err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM entry_types WHERE id = $1)`, typeID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return sql.ErrNoRows
	}
	return nil
}

func validateListingTags(categoryID int, tags []string) error {
	for _, tag := range tags {
		tagSlug := utils.Slugify(tag)
		if tagSlug == "" {
			continue
		}
		var exists bool
		err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM entry_categories WHERE slug = $1)`, tagSlug).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("tag matches category slug")
		}
	}
	return nil
}

func listingLocationValue(locationID int) any {
	if locationID <= 0 {
		return nil
	}
	return locationID
}

func listingURLDomainConflict(entryID int, newURL string) (bool, error) {
	newURL = strings.TrimSpace(newURL)
	var newKey string
	if newURL != "" {
		key, err := webdomain.CanonicalDomain(newURL)
		if err != nil {
			// Keep the linked website URL; do not treat a parse miss as a takeover.
			return false, nil
		}
		newKey = key
	}

	var linkedKey sql.NullString
	err := db.DB.QueryRow(`SELECT domain_key FROM websites WHERE entry_id = $1`, entryID).Scan(&linkedKey)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		return false, err
	}
	if linkedKey.Valid {
		if newURL == "" {
			return true, nil
		}
		if linkedKey.String == newKey {
			return false, nil
		}
		return true, nil
	}
	if newURL == "" || newKey == "" {
		return false, nil
	}

	var existingEntryID sql.NullInt64
	err = db.DB.QueryRow(`SELECT entry_id FROM websites WHERE domain_key = $1`, newKey).Scan(&existingEntryID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !existingEntryID.Valid || int(existingEntryID.Int64) != entryID {
		return true, nil
	}
	return false, nil
}

func handleCreateListing(w http.ResponseWriter, r *http.Request, userID int) {
	var body createListingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.TypeID <= 0 {
		http.Error(w, "name, category_id, and type_id required", http.StatusBadRequest)
		return
	}

	if body.WebsiteID > 0 {
		var banned bool
		err := db.DB.QueryRow(`SELECT website_banned FROM users WHERE id = $1`, userID).Scan(&banned)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if banned {
			writeListingConflict(w, http.StatusForbidden, "website_banned")
			return
		}
		if body.CategoryID <= 0 {
			var websiteCategoryID sql.NullInt64
			var status string
			err = db.DB.QueryRow(`SELECT category_id, status FROM websites WHERE id = $1`, body.WebsiteID).Scan(&websiteCategoryID, &status)
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if status != "approved" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if !websiteCategoryID.Valid || websiteCategoryID.Int64 <= 0 {
				writeListingFieldError(w, "category_id")
				return
			}
			body.CategoryID = int(websiteCategoryID.Int64)
		}
	}

	if body.CategoryID <= 0 {
		http.Error(w, "name, category_id, and type_id required", http.StatusBadRequest)
		return
	}
	categoryIDs, err := directory.NormalizeCategoryIDs(body.CategoryID, body.CategoryIDs)
	if err != nil {
		writeListingFieldError(w, "category_id")
		return
	}
	if _, err = directory.ValidateLeaves(db.DB, categoryIDs); err != nil {
		writeListingFieldError(w, "category_id")
		return
	}
	if err := validateEntryTypeID(body.TypeID); err != nil {
		writeListingFieldError(w, "type_id")
		return
	}
	if len(body.Tags) > 0 {
		if err := validateListingTags(body.CategoryID, normalizeListingTags(body.Tags)); err != nil {
			writeListingFieldError(w, "tags")
			return
		}
	}
	body.Languages = normalizeListingLanguages(body.Languages)
	if !validListingURL(body.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	if body.WebsiteID <= 0 && strings.TrimSpace(body.URL) != "" {
		domainKey, err := webdomain.CanonicalDomain(body.URL)
		if err == nil {
			var exists bool
			err = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM websites WHERE domain_key = $1)`, domainKey).Scan(&exists)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if exists {
				writeListingConflict(w, http.StatusConflict, "domain_taken")
				return
			}
		}
	}

	var catName string
	err = db.DB.QueryRow(`SELECT name FROM entry_categories WHERE id = $1`, body.CategoryID).Scan(&catName)
	if err == sql.ErrNoRows {
		writeListingFieldError(w, "category_id")
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseSlug := utils.Slugify(body.Name)
	if baseSlug == "" {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	hours := jsonObjectOrEmptyListing(body.Hours)
	delivery := jsonObjectOrEmptyListing(body.DeliveryHours)
	deliveryEnabled := listingDeliveryEnabled(catName, body.Name, body.DeliveryEnabled)
	photos := string(sanitizeListingPhotosLimit(body.Photos, 1))

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	listingURL := strings.TrimSpace(body.URL)
	if body.WebsiteID > 0 {
		var domainKey, status string
		var linkedEntryID sql.NullInt64
		err = tx.QueryRow(`
			SELECT domain_key, status, entry_id FROM websites WHERE id = $1 FOR UPDATE
		`, body.WebsiteID).Scan(&domainKey, &status, &linkedEntryID)
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status != "approved" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if linkedEntryID.Valid {
			writeListingConflict(w, http.StatusConflict, "domain_taken")
			return
		}
		listingURL = "https://" + domainKey
	}

	slug, err := nextUniqueEntrySlug(tx, baseSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var entryID int
	err = tx.QueryRow(`
		INSERT INTO entries (type_id, location_id, category_id, cat_name, name, slug, url, phone, address, notes, languages, verified, published, hours, hours_enabled, delivery_hours, delivery_enabled, photos, ratings_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, false, false, $12::jsonb, $13, $14::jsonb, $15, $16::jsonb, false)
		RETURNING id
	`, body.TypeID, listingLocationValue(body.LocationID), body.CategoryID, catName, body.Name, slug, listingURL, body.Phone, body.Address, body.Notes, pq.Array(body.Languages), hours, body.HoursEnabled, delivery, deliveryEnabled, photos).Scan(&entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = directory.ReplaceEntryCategories(tx, entryID, categoryIDs); err != nil {
		writeListingFieldError(w, "category_id")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO entry_members (entry_id, user_id, role, status)
		VALUES ($1, $2, 'owner', 'active')
	`, entryID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if body.WebsiteID > 0 {
		res, err := tx.Exec(`
			UPDATE websites SET entry_id = $1 WHERE id = $2 AND entry_id IS NULL
		`, entryID, body.WebsiteID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			writeListingConflict(w, http.StatusConflict, "domain_taken")
			return
		}
		if err = directory.ReplaceWebsiteCategories(tx, body.WebsiteID, categoryIDs); err != nil {
			writeListingFieldError(w, "category_id")
			return
		}
	} else if listingURL != "" {
		domainKey, err := webdomain.CanonicalDomain(listingURL)
		if err == nil {
			submittedHostValue := domainKey
			if host, err := submittedHost(listingURL); err == nil {
				submittedHostValue = host
			}
			var websiteID int
			err = tx.QueryRow(`
				INSERT INTO websites (domain_key, submitted_host, title, description, status, user_id, entry_id, category_id)
				VALUES ($1, $2, $3, $4, 'approved', $5, $6, $7)
				RETURNING id
			`, domainKey, submittedHostValue, websiteTitle(body.Name), websiteDescription(body.Notes), userID, entryID, body.CategoryID).Scan(&websiteID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = directory.ReplaceWebsiteCategories(tx, websiteID, categoryIDs); err != nil {
				writeListingFieldError(w, "category_id")
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(body.Tags) > 0 {
		if err := replaceListingTags(entryID, normalizeListingTags(body.Tags)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          entryID,
		"name":        body.Name,
		"slug":        slug,
		"location_id": body.LocationID,
		"category_id": body.CategoryID,
		"type_id":     body.TypeID,
		"published":   false,
		"verified":    false,
		"role":        "owner",
		"status":      "active",
	})
}

func handleUpdateListing(w http.ResponseWriter, r *http.Request, userID int) {
	entryID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || entryID <= 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	membership, found, err := scanMembershipDB(entryID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !canEditListing(membership.Role, membership.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var body updateListingBody
	if err := json.Unmarshal(rawBody, &body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &rawFields); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	_, hasSocialLinks := rawFields["social_links"]
	_, hasTags := rawFields["tags"]
	if body.Name == "" || body.CategoryID <= 0 || body.TypeID <= 0 {
		http.Error(w, "name, category_id, and type_id required", http.StatusBadRequest)
		return
	}
	categoryIDs, err := directory.NormalizeCategoryIDs(body.CategoryID, body.CategoryIDs)
	if err != nil {
		writeListingFieldError(w, "category_id")
		return
	}
	if _, err = directory.ValidateLeaves(db.DB, categoryIDs); err != nil {
		writeListingFieldError(w, "category_id")
		return
	}
	if err := validateEntryTypeID(body.TypeID); err != nil {
		writeListingFieldError(w, "type_id")
		return
	}
	body.Languages = normalizeListingLanguages(body.Languages)
	if !validListingURL(body.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	var catName string
	err = db.DB.QueryRow(`SELECT name FROM entry_categories WHERE id = $1`, body.CategoryID).Scan(&catName)
	if err == sql.ErrNoRows {
		writeListingFieldError(w, "category_id")
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hours := jsonObjectOrEmptyListing(body.Hours)
	delivery := jsonObjectOrEmptyListing(body.DeliveryHours)
	body.DeliveryEnabled = listingDeliveryEnabled(catName, body.Name, body.DeliveryEnabled)
	claimed, err := listingClaimed(entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	photoLimit := 1
	if claimed {
		photoLimit = maxListingPhotos
	} else {
		body.RatingsEnabled = false
	}
	photos := string(sanitizeListingPhotosLimit(body.Photos, photoLimit))
	listingURL := strings.TrimSpace(body.URL)

	conflict, err := listingURLDomainConflict(entryID, listingURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflict {
		writeListingConflict(w, http.StatusConflict, "domain_taken")
		return
	}

	if hasTags {
		if err := validateListingTags(body.CategoryID, normalizeListingTags(body.Tags)); err != nil {
			writeListingFieldError(w, "tags")
			return
		}
	}

	locationValue := listingLocationValue(body.LocationID)
	if hasSocialLinks {
		socialLinks := string(sanitizeSocialLinks(body.SocialLinks))
		_, err = db.DB.Exec(`
			UPDATE entries SET
				type_id = $1, location_id = $2, category_id = $3, cat_name = $4, name = $5,
				url = $6, phone = $7, address = $8, notes = $9, languages = $10,
				hours = $11::jsonb, hours_enabled = $12, delivery_hours = $13::jsonb, delivery_enabled = $14,
				social_links = $15::jsonb, photos = $16::jsonb, ratings_enabled = $17
			WHERE id = $18
		`, body.TypeID, locationValue, body.CategoryID, catName, body.Name,
			listingURL, body.Phone, body.Address, body.Notes, pq.Array(body.Languages),
			hours, body.HoursEnabled, delivery, body.DeliveryEnabled, socialLinks, photos, body.RatingsEnabled, entryID)
	} else {
		_, err = db.DB.Exec(`
			UPDATE entries SET
				type_id = $1, location_id = $2, category_id = $3, cat_name = $4, name = $5,
				url = $6, phone = $7, address = $8, notes = $9, languages = $10,
				hours = $11::jsonb, hours_enabled = $12, delivery_hours = $13::jsonb, delivery_enabled = $14,
				photos = $15::jsonb, ratings_enabled = $16
			WHERE id = $17
		`, body.TypeID, locationValue, body.CategoryID, catName, body.Name,
			listingURL, body.Phone, body.Address, body.Notes, pq.Array(body.Languages),
			hours, body.HoursEnabled, delivery, body.DeliveryEnabled, photos, body.RatingsEnabled, entryID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = directory.ReplaceEntryCategories(db.DB, entryID, categoryIDs); err != nil {
		writeListingFieldError(w, "category_id")
		return
	}
	var linkedWebsiteID int
	err = db.DB.QueryRow(`SELECT id FROM websites WHERE entry_id = $1`, entryID).Scan(&linkedWebsiteID)
	if err == nil {
		if err = directory.ReplaceWebsiteCategories(db.DB, linkedWebsiteID, categoryIDs); err != nil {
			writeListingFieldError(w, "category_id")
			return
		}
	} else if err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hasTags {
		if err := replaceListingTags(entryID, normalizeListingTags(body.Tags)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeOK(w)
}

func normalizeListingTags(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, tag := range in {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "#")
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func replaceListingTags(entryID int, tags []string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := replaceSuggestionTags(tx, entryID, tags); err != nil {
		return err
	}
	return tx.Commit()
}

func handleDeleteListing(w http.ResponseWriter, r *http.Request, userID int) {
	entryID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || entryID <= 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	membership, found, err := scanMembershipDB(entryID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found || !isActiveOwner(membership.Role, membership.Status) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = db.DB.Exec(`DELETE FROM websites WHERE entry_id = $1`, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = db.DB.Exec(`DELETE FROM entries WHERE id = $1`, entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeOK(w)
}
