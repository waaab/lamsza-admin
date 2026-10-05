package events

import (
	"backend/internal/db"
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// HandleAdminCatalogEventTypes CRUD /api/admin/catalog_event_types
func HandleAdminCatalogEventTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query(`
			SELECT id, slug, label_hu, sort_order FROM catalog_event_types ORDER BY sort_order ASC, label_hu ASC`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var out []models.CatalogEventType
		for rows.Next() {
			var t models.CatalogEventType
			if err := rows.Scan(&t.ID, &t.Slug, &t.LabelHu, &t.SortOrder); err != nil {
				log.Printf("catalog_event_types scan: %v", err)
				continue
			}
			out = append(out, t)
		}
		json.NewEncoder(w).Encode(out)

	case http.MethodPost:
		var body models.CatalogEventType
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		body.Slug = strings.TrimSpace(strings.ToLower(body.Slug))
		body.LabelHu = strings.TrimSpace(body.LabelHu)
		if body.Slug == "" || body.LabelHu == "" {
			http.Error(w, "slug és label_hu kötelező", http.StatusBadRequest)
			return
		}
		err := db.DB.QueryRow(`
			INSERT INTO catalog_event_types (slug, label_hu, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			body.Slug, body.LabelHu, body.SortOrder).Scan(&body.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(body)

	case http.MethodPut:
		var body models.CatalogEventType
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
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
		_, err := db.DB.Exec(`UPDATE catalog_event_types SET label_hu=$1, sort_order=$2 WHERE id=$3`,
			body.LabelHu, body.SortOrder, body.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		var n int
		err := db.DB.QueryRow(`SELECT COUNT(*) FROM events WHERE event_type_id = $1`, id).Scan(&n)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if n > 0 {
			http.Error(w, fmt.Sprintf("Nem törölhető: %d esemény használja ezt a típust.", n), http.StatusConflict)
			return
		}
		_, err = db.DB.Exec(`DELETE FROM catalog_event_subtypes WHERE event_type_id = $1`, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = db.DB.Exec(`DELETE FROM catalog_event_types WHERE id = $1`, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAdminCatalogEventSubtypes CRUD /api/admin/catalog_event_subtypes?event_type_id=
func HandleAdminCatalogEventSubtypes(w http.ResponseWriter, r *http.Request) {
	etID := r.URL.Query().Get("event_type_id")

	switch r.Method {
	case http.MethodGet:
		var rows *sql.Rows
		var err error
		if etID != "" {
			id, conv := strconv.Atoi(etID)
			if conv != nil || id < 1 {
				http.Error(w, "invalid event_type_id", http.StatusBadRequest)
				return
			}
			rows, err = db.DB.Query(`
				SELECT id, event_type_id, slug, label_hu, sort_order FROM catalog_event_subtypes
				WHERE event_type_id = $1 ORDER BY sort_order ASC, label_hu ASC`, id)
		} else {
			rows, err = db.DB.Query(`
				SELECT id, event_type_id, slug, label_hu, sort_order FROM catalog_event_subtypes
				ORDER BY event_type_id, sort_order ASC, label_hu ASC`)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var out []models.CatalogEventSubtype
		for rows.Next() {
			var s models.CatalogEventSubtype
			if err := rows.Scan(&s.ID, &s.EventTypeID, &s.Slug, &s.LabelHu, &s.SortOrder); err != nil {
				continue
			}
			out = append(out, s)
		}
		json.NewEncoder(w).Encode(out)

	case http.MethodPost:
		var body models.CatalogEventSubtype
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		body.Slug = strings.TrimSpace(strings.ToLower(body.Slug))
		body.LabelHu = strings.TrimSpace(body.LabelHu)
		if body.EventTypeID < 1 || body.Slug == "" || body.LabelHu == "" {
			http.Error(w, "event_type_id, slug és label_hu kötelező", http.StatusBadRequest)
			return
		}
		err := db.DB.QueryRow(`
			INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order) VALUES ($1, $2, $3, $4) RETURNING id`,
			body.EventTypeID, body.Slug, body.LabelHu, body.SortOrder).Scan(&body.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(body)

	case http.MethodPut:
		var body models.CatalogEventSubtype
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
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
		_, err := db.DB.Exec(`UPDATE catalog_event_subtypes SET label_hu=$1, sort_order=$2 WHERE id=$3`,
			body.LabelHu, body.SortOrder, body.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		var n int
		err := db.DB.QueryRow(`SELECT COUNT(*) FROM events WHERE event_subtype_id = $1`, id).Scan(&n)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if n > 0 {
			http.Error(w, fmt.Sprintf("Nem törölhető: %d esemény használja ezt az altípust.", n), http.StatusConflict)
			return
		}
		_, err = db.DB.Exec(`DELETE FROM catalog_event_subtypes WHERE id = $1`, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
