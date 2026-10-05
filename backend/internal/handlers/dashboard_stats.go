package handlers

import (
	"backend/internal/db"
	"encoding/json"
	"log"
	"net/http"
)

// dashboardCountTables maps dashboard card id → physical table name (fixed whitelist only).
var dashboardCountTables = []struct {
	Key   string
	Table string
}{
	{"mondasok", "mondasok"},
	{"quicklinks", "quick_links"},
	{"newsfeeds", "news_feeds"},
	{"locations", "locations"},
	{"counties", "counties"},
	{"venues", "venues"},
	{"attractions", "attractions"},
	{"events", "events"},
	{"websites", "websites"},
	{"entries", "entries"},
	{"entry_categories", "entry_categories"},
	{"entry_types", "entry_types"},
	{"tags", "tags"},
	{"pages", "pages"},
	{"page_faq", "page_faq_sections"},
	{"weather_translations", "weather_desc_translations"},
	{"users", "users"},
	{"settings", "site_settings"},
}

// HandleAdminDashboardStats GET /api/admin/dashboard_stats - row counts for admin welcome cards.
func HandleAdminDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	out := make(map[string]int64, len(dashboardCountTables))
	for _, row := range dashboardCountTables {
		var n int64
		var err error
		if row.Key == "locations" {
			// Település-sorok: legacy `locations` + új `settlements` (a GET/POST admin is mindkettőt használja).
			err = db.DB.QueryRow(`
				SELECT COALESCE((SELECT COUNT(*) FROM locations), 0) + COALESCE((SELECT COUNT(*) FROM settlements), 0)
			`).Scan(&n)
		} else {
			err = db.DB.QueryRow("SELECT COUNT(*) FROM " + row.Table).Scan(&n)
		}
		if err != nil {
			log.Printf("dashboard_stats COUNT %s: %v", row.Table, err)
			out[row.Key] = 0
			continue
		}
		out[row.Key] = n
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(out)
}
