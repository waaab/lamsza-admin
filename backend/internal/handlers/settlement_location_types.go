package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"backend/internal/utils"
)

func fetchAllSettlementLocationTypes() ([]models.SettlementLocationType, error) {
	rows, err := db.DB.Query(`
		SELECT id, slug, label_hu, sort_order FROM settlement_location_types
		ORDER BY sort_order ASC, LOWER(label_hu) ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SettlementLocationType
	for rows.Next() {
		var t models.SettlementLocationType
		if err := rows.Scan(&t.ID, &t.Slug, &t.LabelHu, &t.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []models.SettlementLocationType{}
	}
	return out, nil
}

// HandleAdminSettlementLocationTypes CRUD /api/admin/settlement_location_types
func HandleAdminSettlementLocationTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := fetchAllSettlementLocationTypes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		var body struct {
			Slug      string `json:"slug"`
			LabelHu   string `json:"label_hu"`
			SortOrder int    `json:"sort_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body.LabelHu = strings.TrimSpace(body.LabelHu)
		slug := strings.TrimSpace(strings.ToLower(body.Slug))
		if slug == "" {
			slug = utils.Slugify(body.LabelHu)
		}
		if slug == "" || body.LabelHu == "" {
			http.Error(w, "slug és label_hu kötelező", http.StatusBadRequest)
			return
		}
		var id int
		err := db.DB.QueryRow(`
			INSERT INTO settlement_location_types (slug, label_hu, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			slug, body.LabelHu, body.SortOrder).Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(models.SettlementLocationType{ID: id, Slug: slug, LabelHu: body.LabelHu, SortOrder: body.SortOrder})

	case http.MethodPut:
		var body models.SettlementLocationType
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.ID < 1 {
			http.Error(w, "id kötelező", http.StatusBadRequest)
			return
		}
		body.LabelHu = strings.TrimSpace(body.LabelHu)
		if body.LabelHu == "" {
			http.Error(w, "label_hu kötelező", http.StatusBadRequest)
			return
		}
		_, err := db.DB.Exec(`UPDATE settlement_location_types SET label_hu=$1, sort_order=$2 WHERE id=$3`,
			body.LabelHu, body.SortOrder, body.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		var slug string
		err := db.DB.QueryRow(`SELECT slug FROM settlement_location_types WHERE id = $1`, idStr).Scan(&slug)
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var n int
		err = db.DB.QueryRow(
			`SELECT COUNT(*) FROM settlements WHERE LOWER(TRIM("type"::text)) = LOWER(TRIM($1::text))`, slug).Scan(&n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if n > 0 {
			http.Error(w, fmt.Sprintf("Nem törölhető: %d település használja ezt a típust.", n), http.StatusConflict)
			return
		}
		_, err = db.DB.Exec(`DELETE FROM settlement_location_types WHERE id = $1`, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
