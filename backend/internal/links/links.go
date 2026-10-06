package links

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/settings"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lib/pq"
)

func HandleAdminQuickLinks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query("SELECT id, title, url, COALESCE(bg_color, '#ffffff') FROM quick_links ORDER BY LOWER(title) ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.QuickLink
		for rows.Next() {
			var ql models.QuickLink
			if err := rows.Scan(&ql.ID, &ql.Title, &ql.URL, &ql.BgColor); err == nil {
				res = append(res, ql)
			}
		}
		if res == nil {
			res = []models.QuickLink{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var ql models.QuickLink
		if err := json.NewDecoder(r.Body).Decode(&ql); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		err := db.DB.QueryRow("INSERT INTO quick_links (title, url, bg_color) VALUES ($1, $2, $3) RETURNING id", ql.Title, ql.URL, ql.BgColor).Scan(&ql.ID)
		if err != nil {
			var pgErr *pq.Error
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				http.Error(w, "Ez az URL már szerepel a gyorslinkek között.", 400)
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		settings.IncrementQuickLinksVersion()
		json.NewEncoder(w).Encode(ql)

	case "PUT":
		var ql models.QuickLink
		if err := json.NewDecoder(r.Body).Decode(&ql); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_, err := db.DB.Exec("UPDATE quick_links SET title=$1, url=$2, bg_color=$3 WHERE id=$4",
			ql.Title, ql.URL, ql.BgColor, ql.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		settings.IncrementQuickLinksVersion()
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			db.DB.Exec("DELETE FROM quick_links WHERE id = $1", id)
			settings.IncrementQuickLinksVersion()
		}
		w.WriteHeader(http.StatusOK)
	}
}
