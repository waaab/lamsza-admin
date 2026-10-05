package mondasok

import (
	"backend/internal/db"
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var budapestLoc *time.Location

func init() {
	loc, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		budapestLoc = time.UTC
		return
	}
	budapestLoc = loc
}

func todayBudapest() string {
	return time.Now().In(budapestLoc).Format("2006-01-02")
}

// Migrate ensures mondasok.display_date exists and is populated (matches migrations/mondasok_display_date.sql).
// Call once after db.InitDB() so existing deployments without a manual migration keep working.
func Migrate() {
	_, err := db.DB.Exec(`ALTER TABLE mondasok ADD COLUMN IF NOT EXISTS display_date DATE`)
	if err != nil {
		log.Printf("mondasok Migrate (add display_date): %v", err)
		return
	}
	_, _ = db.DB.Exec(`
		UPDATE mondasok
		SET display_date = COALESCE((created_at AT TIME ZONE 'UTC')::date, CURRENT_DATE)
		WHERE display_date IS NULL`)
	_, _ = db.DB.Exec(`UPDATE mondasok SET display_date = CURRENT_DATE WHERE display_date IS NULL`)
	_, err = db.DB.Exec(`ALTER TABLE mondasok ALTER COLUMN display_date SET NOT NULL`)
	if err != nil {
		log.Printf("mondasok Migrate (display_date NOT NULL): %v", err)
	}
	_, _ = db.DB.Exec(`ALTER TABLE mondasok ALTER COLUMN display_date SET DEFAULT CURRENT_DATE`)
	_, _ = db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_mondasok_display_date ON mondasok (display_date)`)
	log.Println("mondasok table ready (display_date)")
}

// normalizeYMD trims ISO date strings from <input type="date"> or JSON (YYYY-MM-DD prefix).
func normalizeYMD(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// HandlePublicMondasok GET /api/mondasok - quotes for a calendar day.
// Optional ?date=YYYY-MM-DD uses that day (browser should pass local calendar date, same as homepage #datetime).
// If date is missing or invalid, falls back to “today” in Europe/Budapest.
func HandlePublicMondasok(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	today := todayBudapest()
	if q := strings.TrimSpace(r.URL.Query().Get("date")); q != "" {
		n := normalizeYMD(q)
		if n != "" {
			if _, err := time.Parse("2006-01-02", n); err == nil {
				today = n
			}
		}
	}
	rows, err := db.DB.Query(
		`SELECT id, text, display_date::text, created_at::text FROM mondasok WHERE display_date = $1::date ORDER BY id ASC`,
		today,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var res []models.Mondas
	for rows.Next() {
		var m models.Mondas
		var dd sql.NullString
		if err := rows.Scan(&m.ID, &m.Text, &dd, &m.CreatedAt); err != nil {
			continue
		}
		if dd.Valid {
			m.DisplayDate = dd.String
		}
		res = append(res, m)
	}
	if res == nil {
		res = []models.Mondas{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func HandleAdminMondasok(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query("SELECT id, text, display_date::text, created_at::text FROM mondasok ORDER BY LOWER(text) ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.Mondas
		for rows.Next() {
			var m models.Mondas
			var dd sql.NullString
			if err := rows.Scan(&m.ID, &m.Text, &dd, &m.CreatedAt); err != nil {
				continue
			}
			if dd.Valid {
				m.DisplayDate = dd.String
			}
			res = append(res, m)
		}
		if res == nil {
			res = []models.Mondas{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var m models.Mondas
		var alt struct {
			DisplayDateCamel string `json:"displayDate"`
		}
		raw, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, errRead.Error(), 400)
			return
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.Unmarshal(raw, &alt)
		if strings.TrimSpace(m.DisplayDate) == "" && alt.DisplayDateCamel != "" {
			m.DisplayDate = alt.DisplayDateCamel
		}
		m.DisplayDate = normalizeYMD(m.DisplayDate)
		if strings.TrimSpace(m.Text) == "" || m.DisplayDate == "" {
			http.Error(w, "szöveg és megjelenés napja kötelező (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		if _, err := time.Parse("2006-01-02", m.DisplayDate); err != nil {
			http.Error(w, "érvénytelen dátum (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		err := db.DB.QueryRow(
			`INSERT INTO mondasok (text, display_date) VALUES ($1, $2::date) RETURNING id, display_date::text, created_at::text`,
			strings.TrimSpace(m.Text),
			m.DisplayDate,
		).Scan(&m.ID, &m.DisplayDate, &m.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(m)

	case "PUT":
		var m models.Mondas
		var alt struct {
			DisplayDateCamel string `json:"displayDate"`
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.Unmarshal(raw, &alt)
		if strings.TrimSpace(m.DisplayDate) == "" && alt.DisplayDateCamel != "" {
			m.DisplayDate = alt.DisplayDateCamel
		}
		m.DisplayDate = normalizeYMD(m.DisplayDate)
		if m.ID < 1 || strings.TrimSpace(m.Text) == "" || m.DisplayDate == "" {
			http.Error(w, "id, szöveg és megjelenés napja kötelező", http.StatusBadRequest)
			return
		}
		if _, err := time.Parse("2006-01-02", m.DisplayDate); err != nil {
			http.Error(w, "érvénytelen dátum (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec(
			`UPDATE mondasok SET text=$1, display_date=$2::date WHERE id=$3`,
			strings.TrimSpace(m.Text),
			m.DisplayDate,
			m.ID,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			db.DB.Exec("DELETE FROM mondasok WHERE id = $1", id)
		}
		w.WriteHeader(http.StatusOK)
	}
}
