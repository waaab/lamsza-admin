package venues

import (
	"backend/internal/db"
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// HandlePublicVenueTypes GET /api/venue_types - ordered list for public UIs
func HandlePublicVenueTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := fetchAllVenueTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// HandleAdminVenueTypes GET/POST/PUT/DELETE /api/admin/venue_types
func HandleAdminVenueTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := fetchAllVenueTypes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		createVenueType(w, r)
	case http.MethodPut:
		updateVenueType(w, r)
	case http.MethodDelete:
		deleteVenueType(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func fetchAllVenueTypes() ([]models.VenueType, error) {
	rows, err := db.DB.Query(`SELECT id, slug, label_hu FROM venue_types ORDER BY LOWER(label_hu) ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.VenueType
	for rows.Next() {
		var t models.VenueType
		if err := rows.Scan(&t.ID, &t.Slug, &t.LabelHu); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []models.VenueType{}
	}
	return out, nil
}

// nextUniqueVenueTypeSlug picks a unique slug; excludeID lets the row keep its slug when the label still maps to the same base.
func nextUniqueVenueTypeSlug(base string, excludeID int) (string, error) {
	if strings.TrimSpace(base) == "" {
		base = "tipus"
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		var existingID int
		err := db.DB.QueryRow(`SELECT id FROM venue_types WHERE slug = $1`, candidate).Scan(&existingID)
		if err == sql.ErrNoRows {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if excludeID > 0 && existingID == excludeID {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("nem sikerült egyedi slugot találni")
}

func createVenueType(w http.ResponseWriter, r *http.Request) {
	var p struct {
		LabelHu string `json:"label_hu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(p.LabelHu) == "" {
		http.Error(w, "megnevezés kötelező", http.StatusBadRequest)
		return
	}
	base := normalizeSlug(p.LabelHu, "")
	slug, err := nextUniqueVenueTypeSlug(base, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = db.DB.Exec(
		`INSERT INTO venue_types (slug, label_hu) VALUES ($1, $2)`,
		slug, strings.TrimSpace(p.LabelHu),
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			http.Error(w, "ilyen slug már létezik", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func updateVenueType(w http.ResponseWriter, r *http.Request) {
	var p struct {
		ID      int    `json:"id"`
		LabelHu string `json:"label_hu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.ID < 1 || strings.TrimSpace(p.LabelHu) == "" {
		http.Error(w, "id és megnevezés kötelező", http.StatusBadRequest)
		return
	}
	var oldSlug string
	err := db.DB.QueryRow(`SELECT slug FROM venue_types WHERE id = $1`, p.ID).Scan(&oldSlug)
	if err == sql.ErrNoRows {
		http.Error(w, "nem található", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	base := normalizeSlug(p.LabelHu, "")
	newSlug, err := nextUniqueVenueTypeSlug(base, p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if newSlug != oldSlug {
		if _, err := tx.Exec(`UPDATE venues SET kind = $1 WHERE kind = $2`, newSlug, oldSlug); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := tx.Exec(
			`UPDATE venue_types SET slug = $1, label_hu = $2 WHERE id = $3`,
			newSlug, strings.TrimSpace(p.LabelHu), p.ID,
		); err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				http.Error(w, "ilyen slug már létezik", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := tx.Exec(
			`UPDATE venue_types SET label_hu = $1 WHERE id = $2`,
			strings.TrimSpace(p.LabelHu), p.ID,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteVenueType(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "id kötelező", http.StatusBadRequest)
		return
	}
	var slug string
	err = db.DB.QueryRow(`SELECT slug FROM venue_types WHERE id = $1`, id).Scan(&slug)
	if err == sql.ErrNoRows {
		http.Error(w, "nem található", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var n int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM venues WHERE kind = $1`, slug).Scan(&n); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Error(w, "nem törölhető: helyszínek ezt a típust használják", http.StatusBadRequest)
		return
	}
	res, err := db.DB.Exec(`DELETE FROM venue_types WHERE id = $1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nn, _ := res.RowsAffected()
	if nn == 0 {
		http.Error(w, "nem található", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
