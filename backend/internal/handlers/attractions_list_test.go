package handlers

import (
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// withScratchDB points db.DB at the scratch test database for one test, or
// skips. It reads only TEST_DATABASE_URL; `npm run test:backend` sets it.
func withScratchDB(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL is not set; run `npm run test:backend` to use a scratch database")
	}
	url, err := db.TestDatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		t.Fatalf("scratch database not reachable: %v", err)
	}
	prev := db.DB
	db.DB = conn
	t.Cleanup(func() {
		db.DB = prev
		conn.Close()
	})
}

// An attraction saved without coordinates has no geo_locations row, so the
// list query's LEFT JOIN returns NULL latitude and longitude. Scanning those
// into float64 failed, the row was skipped without a word, and the attraction
// could no longer be edited or deleted from the admin UI.
func TestAdminAttractionsListIncludesOneWithoutCoordinates(t *testing.T) {
	withScratchDB(t)

	var countyID int
	if err := db.DB.QueryRow(`SELECT id FROM counties ORDER BY id LIMIT 1`).Scan(&countyID); err != nil {
		t.Fatalf("no county in the scratch database: %v", err)
	}
	slug := fmt.Sprintf("koordinata-nelkul-%d", time.Now().UnixNano())
	var id int
	if err := db.DB.QueryRow(`
		INSERT INTO attractions (county_id, name, slug, location_id)
		VALUES ($1, 'Koordináta nélkül', $2, NULL) RETURNING id`, countyID, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.DB.Exec(`DELETE FROM attractions WHERE id = $1`, id) })

	rec := httptest.NewRecorder()
	HandleAdminAttractions(rec, httptest.NewRequest(http.MethodGet, "/api/admin/attractions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/admin/attractions: %d %s", rec.Code, rec.Body.String())
	}
	var list []Attraction
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == id {
			if a.Latitude != 0 || a.Longitude != 0 {
				t.Errorf("coordinates = %v,%v, want 0,0 for an attraction with none", a.Latitude, a.Longitude)
			}
			return
		}
	}
	t.Fatalf("attraction %d without coordinates is missing from the admin list (%d listed)", id, len(list))
}
