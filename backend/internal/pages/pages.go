package pages

import (
	"backend/internal/db"
	"encoding/json"
	"net/http"
	"strconv"
)

type Page struct {
	ID        int    `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Greeting  string `json:"greeting"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

// HandleAdminPages: GET returns all pages, PUT updates a page
func HandleAdminPages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query("SELECT id, slug, title, greeting, content, updated_at::text FROM pages ORDER BY LOWER(title) ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var result []Page
		for rows.Next() {
			var p Page
			if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Greeting, &p.Content, &p.UpdatedAt); err == nil {
				result = append(result, p)
			}
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result == nil {
			result = []Page{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	case http.MethodPut:
		var p Page
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if p.ID == 0 {
			http.Error(w, "Missing page id", http.StatusBadRequest)
			return
		}
		_, err := db.DB.Exec(
			`UPDATE pages SET title = $1, greeting = $2, content = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`,
			p.Title, p.Greeting, p.Content, p.ID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "Missing id", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid id", http.StatusBadRequest)
			return
		}
		_, _ = db.DB.Exec("DELETE FROM pages WHERE id = $1", id)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
