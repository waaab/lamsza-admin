package main

import (
	"backend/internal/db"
	"fmt"
	"net/http"
	"testing"
)

// The attraction's facts (elevation, area, depth) are saved on create, listed
// back, cleared by null on update, and a negative area or depth is refused
// (lamsza UI_BASELINE "szf-pages").
func TestAdminAttractionFacts(t *testing.T) {
	requireDB(t)
	county := someCountySlug(t)
	name := testName("attraction-facts")

	rr := asAdmin(t, http.MethodPost, "/api/admin/attractions", map[string]interface{}{
		"county_slug": county, "name": name, "description": "facts",
		"elevation_m": 946, "area_km2": 0.22, "depth_m": 7,
	})
	mustStatus(t, rr, http.StatusOK, "POST with facts")
	id := idOf(t, decodeObject(t, rr, "POST"), "POST")
	t.Cleanup(func() { _, _ = db.DB.Exec(`DELETE FROM attractions WHERE id = $1`, id) })

	row := findInAdminListBy(t, "/api/admin/attractions", "name", name)
	if row == nil {
		t.Fatal("created attraction is not listed")
	}
	if fmt.Sprint(row["elevation_m"], row["area_km2"], row["depth_m"]) != "946 0.22 7" {
		t.Fatalf("listed facts: %v %v %v", row["elevation_m"], row["area_km2"], row["depth_m"])
	}

	rr = asAdmin(t, http.MethodPut, "/api/admin/attractions", map[string]interface{}{
		"id": id, "county_slug": county, "name": name, "description": "facts",
		"elevation_m": 946, "area_km2": nil, "depth_m": -1,
	})
	mustStatus(t, rr, http.StatusBadRequest, "PUT with a negative depth")

	rr = asAdmin(t, http.MethodPut, "/api/admin/attractions", map[string]interface{}{
		"id": id, "county_slug": county, "name": name, "description": "facts",
		"elevation_m": -12.5, "area_km2": nil, "depth_m": nil,
	})
	mustStatus(t, rr, http.StatusOK, "PUT clearing area and depth")
	var elev, area, depth *float64
	if err := db.DB.QueryRow(`SELECT elevation_m, area_km2, depth_m FROM attractions WHERE id = $1`, id).Scan(&elev, &area, &depth); err != nil {
		t.Fatal(err)
	}
	if elev == nil || *elev != -12.5 || area != nil || depth != nil {
		t.Fatalf("after clearing: %v %v %v", elev, area, depth)
	}
}
