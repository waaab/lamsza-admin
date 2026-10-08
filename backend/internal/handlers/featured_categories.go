package handlers

import (
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"net/http"
)

// maxFeaturedCategories matches lamsza's schema: entry_categories.featured_order
// is 1-6 with a unique index, one category per chip.
const maxFeaturedCategories = 6

type featuredCategory struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ParentID      *int   `json:"parent_id"`
	FeaturedOrder int    `json:"featured_order"`
}

// HandleAdminFeaturedCategories reads and replaces the home page's category
// chips. GET lists them in chip order; PUT {"ids": [...]} sets the whole list
// at once (at most six, no repeats, every id an existing category), in one
// transaction, so the public page never sees half an edit.
func HandleAdminFeaturedCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := featuredCategories()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)

	case http.MethodPut:
		var body struct {
			IDs []int `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body.IDs) > maxFeaturedCategories {
			http.Error(w, "at most 6 featured categories", http.StatusBadRequest)
			return
		}
		seen := map[int]bool{}
		for _, id := range body.IDs {
			if seen[id] {
				http.Error(w, "a category appears twice", http.StatusBadRequest)
				return
			}
			seen[id] = true
		}
		tx, err := db.DB.Begin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`UPDATE entry_categories SET featured_order = NULL WHERE featured_order IS NOT NULL`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i, id := range body.IDs {
			res, err := tx.Exec(`UPDATE entry_categories SET featured_order = $1 WHERE id = $2`, i+1, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				http.Error(w, "category not found", http.StatusBadRequest)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list, err := featuredCategories()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func featuredCategories() ([]featuredCategory, error) {
	rows, err := db.DB.Query(`SELECT id, name, slug, parent_id, featured_order FROM entry_categories WHERE featured_order IS NOT NULL ORDER BY featured_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []featuredCategory{}
	for rows.Next() {
		var c featuredCategory
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &parentID, &c.FeaturedOrder); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := int(parentID.Int64)
			c.ParentID = &v
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
