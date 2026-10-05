package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

type adminTag struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Usage int    `json:"usage"`
}

func HandleAdminTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listAdminTags(w)
	case http.MethodPost:
		createAdminTag(w, r)
	case http.MethodPut:
		updateAdminTag(w, r)
	case http.MethodDelete:
		deleteAdminTag(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAdminTags(w http.ResponseWriter) {
	rows, err := db.DB.Query(`
		SELECT t.id, t.name, COUNT(et.entry_id)
		FROM tags t
		LEFT JOIN entry_tags et ON et.tag_id = t.id
		GROUP BY t.id, t.name
		ORDER BY LOWER(t.name) ASC, t.id ASC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []adminTag{}
	for rows.Next() {
		var tag adminTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Usage); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, tag)
	}
	json.NewEncoder(w).Encode(out)
}

func createAdminTag(w http.ResponseWriter, r *http.Request) {
	var body adminTag
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name, ok := cleanTagName(w, body.Name)
	if !ok {
		return
	}
	var tag adminTag
	err := db.DB.QueryRow(`INSERT INTO tags (name) VALUES ($1) RETURNING id, name`, name).Scan(&tag.ID, &tag.Name)
	if err != nil {
		writeTagWriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func updateAdminTag(w http.ResponseWriter, r *http.Request) {
	var body adminTag
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.ID <= 0 {
		http.Error(w, "A név kötelező.", http.StatusBadRequest)
		return
	}
	name, ok := cleanTagName(w, body.Name)
	if !ok {
		return
	}
	var tag adminTag
	err := db.DB.QueryRow(`UPDATE tags SET name = $1 WHERE id = $2 RETURNING id, name`, name, body.ID).Scan(&tag.ID, &tag.Name)
	if err != nil {
		writeTagWriteError(w, err)
		return
	}
	json.NewEncoder(w).Encode(tag)
}

func deleteAdminTag(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Hiányzó azonosító.", http.StatusBadRequest)
		return
	}
	res, err := db.DB.Exec(`DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "A címke nem található.", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func cleanTagName(w http.ResponseWriter, raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" {
		http.Error(w, "A név kötelező.", http.StatusBadRequest)
		return "", false
	}
	slug := utils.Slugify(name)
	var n int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM entry_categories WHERE slug = $1`, slug).Scan(&n); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", false
	}
	if n > 0 {
		http.Error(w, "A címke nem ismételheti a kategória nevét.", http.StatusBadRequest)
		return "", false
	}
	return name, true
}

func writeTagWriteError(w http.ResponseWriter, err error) {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		http.Error(w, "Ez a címke már létezik.", http.StatusConflict)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "A címke nem található.", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
