package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

type AttractionImage struct {
	URL       string `json:"url"`
	Copyright string `json:"copyright,omitempty"`
}

type Attraction struct {
	ID                     int                     `json:"id"`
	CountyID               int                     `json:"county_id"`
	CountySlug             string                  `json:"county_slug"`
	CountyName             string                  `json:"county_name"`
	Name                   string                  `json:"name"`
	NameRo                 string                  `json:"name_ro"`
	NameDe                 string                  `json:"name_de"`
	Slug                   string                  `json:"slug"`
	Description            string                  `json:"description"`
	Latitude               float64                 `json:"latitude,omitempty"`
	Longitude              float64                 `json:"longitude,omitempty"`
	FeaturedImage          string                  `json:"featured_image,omitempty"`
	FeaturedImageCopyright string                  `json:"featured_image_copyright,omitempty"`
	Content                string                  `json:"content,omitempty"`
	Activities             []string                `json:"activities,omitempty"`
	Prohibitions           []string                `json:"prohibitions,omitempty"`
	Images                 []AttractionImage       `json:"images,omitempty"`
	SuggestionPending      bool                    `json:"suggestion_pending"`
	Contributors           []AttractionContributor `json:"contributors"`
}

type HistoricalSeat struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	NameRo  string `json:"name_ro"`
	NameDe  string `json:"name_de"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

type County struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	NameRo  string `json:"name_ro"`
	NameDe  string `json:"name_de"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

func HandleAttractions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	countySlug := strings.TrimSpace(strings.ToLower(q.Get("county_slug")))
	slug := strings.TrimSpace(strings.ToLower(q.Get("slug")))
	if nearID := strings.TrimSpace(q.Get("near_id")); nearID != "" {
		writeNearbyAttractions(w, nearID)
		return
	}

	if slug != "" && countySlug != "" {
		// Single attraction detail
		var a Attraction
		var activitiesText, prohibitionsText string
		err := db.DB.QueryRow(`
			SELECT a.id, a.county_id, c.slug, c.name, a.name, COALESCE(a.name_ro,''), COALESCE(a.name_de,''),
				a.slug, COALESCE(a.description,''), COALESCE(a.featured_image,''), COALESCE(a.featured_image_copyright,''),
				COALESCE(a.content,''), COALESCE(a.activities,''), COALESCE(a.prohibitions,''),
				gl.latitude, gl.longitude
			FROM attractions a
			JOIN counties c ON a.county_id = c.id
			LEFT JOIN geo_locations gl ON a.location_id = gl.id
			WHERE LOWER(a.slug) = $1 AND LOWER(c.slug) = $2
		`, slug, countySlug).Scan(&a.ID, &a.CountyID, &a.CountySlug, &a.CountyName, &a.Name, &a.NameRo, &a.NameDe,
			&a.Slug, &a.Description, &a.FeaturedImage, &a.FeaturedImageCopyright, &a.Content, &activitiesText, &prohibitionsText, &a.Latitude, &a.Longitude)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		a.Activities = splitActivities(activitiesText)
		a.Prohibitions = splitActivities(prohibitionsText)
		a.Images = loadAttractionImages(a.ID)
		attachAttractionPublicExtras(&a)
		json.NewEncoder(w).Encode(a)
		return
	}

	// List attractions (optionally filtered by county)
	query := `
		SELECT a.id, a.county_id, c.slug, c.name, a.name, COALESCE(a.name_ro,''), COALESCE(a.name_de,''),
			a.slug, COALESCE(a.description,''), COALESCE(a.featured_image,''), COALESCE(a.featured_image_copyright,''),
			COALESCE(a.content,''), COALESCE(a.activities,''), COALESCE(a.prohibitions,''),
			gl.latitude, gl.longitude
		FROM attractions a
		JOIN counties c ON a.county_id = c.id
		LEFT JOIN geo_locations gl ON a.location_id = gl.id
	`
	args := []interface{}{}
	if countySlug != "" {
		query += " WHERE LOWER(c.slug) = $1"
		args = append(args, countySlug)
	}
	query += " ORDER BY a.name ASC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []Attraction
	for rows.Next() {
		var a Attraction
		var activitiesText, prohibitionsText string
		if err := rows.Scan(&a.ID, &a.CountyID, &a.CountySlug, &a.CountyName, &a.Name, &a.NameRo, &a.NameDe,
			&a.Slug, &a.Description, &a.FeaturedImage, &a.FeaturedImageCopyright, &a.Content, &activitiesText, &prohibitionsText, &a.Latitude, &a.Longitude); err == nil {
			a.Activities = splitActivities(activitiesText)
			a.Prohibitions = splitActivities(prohibitionsText)
			list = append(list, a)
		}
	}
	if list == nil {
		list = []Attraction{}
	}
	json.NewEncoder(w).Encode(list)
}

func HandleHistoricalSeats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("slug")))
	if slug != "" {
		var h HistoricalSeat
		err := db.DB.QueryRow(
			`SELECT id, name, COALESCE(name_ro,''), COALESCE(name_de,''), slug, COALESCE(content,'') FROM historical_seats WHERE LOWER(slug) = $1`,
			slug,
		).Scan(&h.ID, &h.Name, &h.NameRo, &h.NameDe, &h.Slug, &h.Content)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(h)
		return
	}
	rows, err := db.DB.Query("SELECT id, name, COALESCE(name_ro,''), COALESCE(name_de,''), slug, COALESCE(content,'') FROM historical_seats ORDER BY name ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []HistoricalSeat
	for rows.Next() {
		var h HistoricalSeat
		if err := rows.Scan(&h.ID, &h.Name, &h.NameRo, &h.NameDe, &h.Slug, &h.Content); err == nil {
			list = append(list, h)
		}
	}
	if list == nil {
		list = []HistoricalSeat{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// HandleAdminCounties updates county fields (PUT JSON: id, name, name_ro, name_de, slug, content).
func HandleAdminCounties(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		NameRo  string `json:"name_ro"`
		NameDe  string `json:"name_de"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.ID == 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	body.Slug = strings.TrimSpace(strings.ToLower(body.Slug))
	if body.Slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	res, err := db.DB.Exec(
		`UPDATE counties SET name = $1, name_ro = $2, name_de = $3, slug = $4, content = $5 WHERE id = $6`,
		strings.TrimSpace(body.Name),
		strings.TrimSpace(body.NameRo),
		strings.TrimSpace(body.NameDe),
		body.Slug,
		body.Content,
		body.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleAdminHistoricalSeats updates historical seat fields (PUT JSON: id, name, name_ro, name_de, slug, content).
func HandleAdminHistoricalSeats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		NameRo  string `json:"name_ro"`
		NameDe  string `json:"name_de"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.ID == 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	body.Slug = strings.TrimSpace(strings.ToLower(body.Slug))
	if body.Slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	res, err := db.DB.Exec(
		`UPDATE historical_seats SET name = $1, name_ro = $2, name_de = $3, slug = $4, content = $5 WHERE id = $6`,
		strings.TrimSpace(body.Name),
		strings.TrimSpace(body.NameRo),
		strings.TrimSpace(body.NameDe),
		body.Slug,
		body.Content,
		body.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func HandleCounties(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.DB.Query("SELECT id, name, COALESCE(name_ro,''), COALESCE(name_de,''), slug, COALESCE(content,'') FROM counties ORDER BY name ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []County
	for rows.Next() {
		var c County
		if err := rows.Scan(&c.ID, &c.Name, &c.NameRo, &c.NameDe, &c.Slug, &c.Content); err == nil {
			list = append(list, c)
		}
	}
	if list == nil {
		list = []County{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// HandleAdminAttractions: GET list, POST create, PUT update, DELETE
func HandleAdminAttractions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		q := r.URL.Query()
		countySlug := strings.TrimSpace(strings.ToLower(q.Get("county_slug")))
		query := `
			SELECT a.id, a.county_id, c.slug, c.name, a.name, COALESCE(a.name_ro,''), COALESCE(a.name_de,''),
				a.slug, COALESCE(a.description,''), COALESCE(a.featured_image,''), COALESCE(a.featured_image_copyright,''),
				COALESCE(a.content,''), COALESCE(a.activities,''), COALESCE(a.prohibitions,''),
				gl.latitude, gl.longitude
			FROM attractions a
			JOIN counties c ON a.county_id = c.id
			LEFT JOIN geo_locations gl ON a.location_id = gl.id
		`
		args := []interface{}{}
		if countySlug != "" {
			query += " WHERE LOWER(c.slug) = $1"
			args = append(args, countySlug)
		}
		query += " ORDER BY a.name ASC"
		rows, err := db.DB.Query(query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var list []Attraction
		for rows.Next() {
			var a Attraction
			var activitiesText, prohibitionsText string
			if err := rows.Scan(&a.ID, &a.CountyID, &a.CountySlug, &a.CountyName, &a.Name, &a.NameRo, &a.NameDe,
				&a.Slug, &a.Description, &a.FeaturedImage, &a.FeaturedImageCopyright, &a.Content, &activitiesText, &prohibitionsText, &a.Latitude, &a.Longitude); err == nil {
				a.Activities = splitActivities(activitiesText)
				a.Prohibitions = splitActivities(prohibitionsText)
				a.Images = loadAttractionImages(a.ID)
				list = append(list, a)
			}
		}
		if list == nil {
			list = []Attraction{}
		}
		json.NewEncoder(w).Encode(list)

	case "POST":
		var a struct {
			CountySlug             string          `json:"county_slug"`
			Name                   string          `json:"name"`
			NameRo                 string          `json:"name_ro"`
			NameDe                 string          `json:"name_de"`
			Slug                   string          `json:"slug"`
			Description            string          `json:"description"`
			Latitude               float64         `json:"latitude"`
			Longitude              float64         `json:"longitude"`
			FeaturedImage          string          `json:"featured_image"`
			FeaturedImageCopyright string          `json:"featured_image_copyright"`
			Content                string          `json:"content"`
			Activities             []string        `json:"activities"`
			Prohibitions           []string        `json:"prohibitions"`
			Images                 json.RawMessage `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slug := utils.Slugify(a.Name)
		if a.Slug != "" {
			slug = utils.Slugify(a.Slug)
		}
		var countyID int
		if err := db.DB.QueryRow("SELECT id FROM counties WHERE slug = $1", strings.ToLower(a.CountySlug)).Scan(&countyID); err != nil {
			http.Error(w, "County not found", http.StatusBadRequest)
			return
		}
		var glID *int
		if a.Latitude != 0 || a.Longitude != 0 {
			var id int
			if err := db.DB.QueryRow("INSERT INTO geo_locations (latitude, longitude) VALUES ($1, $2) RETURNING id", a.Latitude, a.Longitude).Scan(&id); err == nil {
				glID = &id
			}
		}
		var attID int
		err := db.DB.QueryRow(`
			INSERT INTO attractions (county_id, name, name_ro, name_de, slug, description, location_id, featured_image, featured_image_copyright, content, activities, prohibitions)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id
		`, countyID, a.Name, a.NameRo, a.NameDe, slug, a.Description, glID, a.FeaturedImage, strings.TrimSpace(a.FeaturedImageCopyright), a.Content, joinActivities(a.Activities), joinActivities(a.Prohibitions)).Scan(&attID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i, img := range parseAttractionImages(a.Images) {
			if img.URL != "" {
				db.DB.Exec("INSERT INTO attraction_images (attraction_id, url, copyright, sort_order) VALUES ($1, $2, $3, $4)", attID, img.URL, img.Copyright, i)
			}
		}
		rememberAttractionEditor(r, attID)
		json.NewEncoder(w).Encode(map[string]int{"id": attID})

	case "PUT":
		var a struct {
			ID                     int             `json:"id"`
			CountySlug             string          `json:"county_slug"`
			Name                   string          `json:"name"`
			NameRo                 string          `json:"name_ro"`
			NameDe                 string          `json:"name_de"`
			Slug                   string          `json:"slug"`
			Description            string          `json:"description"`
			Latitude               float64         `json:"latitude"`
			Longitude              float64         `json:"longitude"`
			FeaturedImage          string          `json:"featured_image"`
			FeaturedImageCopyright string          `json:"featured_image_copyright"`
			Content                string          `json:"content"`
			Activities             []string        `json:"activities"`
			Prohibitions           []string        `json:"prohibitions"`
			Images                 json.RawMessage `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slug := utils.Slugify(a.Name)
		if a.Slug != "" {
			slug = utils.Slugify(a.Slug)
		}
		var countyID int
		if err := db.DB.QueryRow("SELECT id FROM counties WHERE slug = $1", strings.ToLower(a.CountySlug)).Scan(&countyID); err != nil {
			http.Error(w, "County not found", http.StatusBadRequest)
			return
		}
		var glID *int
		if a.Latitude != 0 || a.Longitude != 0 {
			var oldID *int
			db.DB.QueryRow("SELECT location_id FROM attractions WHERE id = $1", a.ID).Scan(&oldID)
			if oldID != nil {
				db.DB.Exec("UPDATE geo_locations SET latitude=$1, longitude=$2 WHERE id=$3", a.Latitude, a.Longitude, *oldID)
				glID = oldID
			} else {
				var id int
				if err := db.DB.QueryRow("INSERT INTO geo_locations (latitude, longitude) VALUES ($1, $2) RETURNING id", a.Latitude, a.Longitude).Scan(&id); err == nil {
					glID = &id
				}
			}
		}
		_, err := db.DB.Exec(`
			UPDATE attractions SET county_id=$1, name=$2, name_ro=$3, name_de=$4, slug=$5, description=$6, location_id=$7, featured_image=$8, featured_image_copyright=$9, content=$10, activities=$11, prohibitions=$12
			WHERE id=$13
		`, countyID, a.Name, a.NameRo, a.NameDe, slug, a.Description, glID, a.FeaturedImage, strings.TrimSpace(a.FeaturedImageCopyright), a.Content, joinActivities(a.Activities), joinActivities(a.Prohibitions), a.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		db.DB.Exec("DELETE FROM attraction_images WHERE attraction_id = $1", a.ID)
		for i, img := range parseAttractionImages(a.Images) {
			if img.URL != "" {
				db.DB.Exec("INSERT INTO attraction_images (attraction_id, url, copyright, sort_order) VALUES ($1, $2, $3, $4)", a.ID, img.URL, img.Copyright, i)
			}
		}
		rememberAttractionEditor(r, a.ID)
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		var locID *int
		db.DB.QueryRow("SELECT location_id FROM attractions WHERE id = $1", id).Scan(&locID)
		db.DB.Exec("DELETE FROM attraction_images WHERE attraction_id = $1", id)
		db.DB.Exec("DELETE FROM attractions WHERE id = $1", id)
		if locID != nil {
			db.DB.Exec("DELETE FROM geo_locations WHERE id = $1", *locID)
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func MigrateAttractions() {
	statements := []string{
		`ALTER TABLE attractions ADD COLUMN IF NOT EXISTS activities TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE attractions ADD COLUMN IF NOT EXISTS prohibitions TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE attractions ADD COLUMN IF NOT EXISTS featured_image_copyright TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE attraction_images ADD COLUMN IF NOT EXISTS copyright TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS attraction_suggestions (
			id SERIAL PRIMARY KEY,
			attraction_id INT NOT NULL REFERENCES attractions(id) ON DELETE CASCADE,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			changes JSONB NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'denied')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS attraction_suggestions_one_open
			ON attraction_suggestions (attraction_id)
			WHERE status = 'open'`,
		`CREATE TABLE IF NOT EXISTS attraction_contributors (
			attraction_id INT NOT NULL REFERENCES attractions(id) ON DELETE CASCADE,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (attraction_id, user_id)
		)`,
	}
	for _, q := range statements {
		if _, err := db.DB.Exec(q); err != nil {
			log.Printf("MigrateAttractions: %v", err)
		}
	}
}

func splitActivities(raw string) []string {
	out := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func joinActivities(lines []string) string {
	return strings.Join(splitActivities(strings.Join(lines, "\n")), "\n")
}

// Nearby rings widen until at least one other attraction is found.
// Settlements have no coordinates, so villages, towns, and cities are distance steps, then the county.
var nearbyAttractionTiers = []struct {
	km    float64
	scope string
}{
	{30, "near"},
	{60, "vicinity"},
	{100, "area"},
}

type nearbyAttraction struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description,omitempty"`
	CountySlug  string  `json:"county_slug"`
	CountyName  string  `json:"county_name"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

func writeNearbyAttractions(w http.ResponseWriter, idRaw string) {
	id, err := strconv.Atoi(idRaw)
	if err != nil || id <= 0 {
		http.Error(w, "near_id required", http.StatusBadRequest)
		return
	}
	var originCounty int
	var originLat, originLon sql.NullFloat64
	err = db.DB.QueryRow(`
		SELECT a.county_id, gl.latitude, gl.longitude
		FROM attractions a
		LEFT JOIN geo_locations gl ON a.location_id = gl.id
		WHERE a.id = $1
	`, id).Scan(&originCounty, &originLat, &originLon)
	if err == sql.ErrNoRows {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := db.DB.Query(`
		SELECT a.id, a.county_id, c.slug, c.name, a.name, a.slug, COALESCE(a.description, ''),
			gl.latitude, gl.longitude
		FROM attractions a
		JOIN counties c ON a.county_id = c.id
		LEFT JOIN geo_locations gl ON a.location_id = gl.id
		WHERE a.id <> $1
		ORDER BY a.name ASC
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var all []candidateNearby
	originHasCoords := originLat.Valid && originLon.Valid
	for rows.Next() {
		var item nearbyAttraction
		var county int
		var lat, lon sql.NullFloat64
		if err := rows.Scan(&item.ID, &county, &item.CountySlug, &item.CountyName, &item.Name, &item.Slug, &item.Description, &lat, &lon); err != nil {
			continue
		}
		c := candidateNearby{item: item, county: county}
		if originHasCoords && lat.Valid && lon.Valid {
			item.Latitude = lat.Float64
			item.Longitude = lon.Float64
			c.item = item
			c.hasDist = true
			c.distKm = haversineKm(originLat.Float64, originLon.Float64, lat.Float64, lon.Float64)
		}
		all = append(all, c)
	}

	scope := "county"
	chosen := []candidateNearby{}
	if originHasCoords {
		for _, tier := range nearbyAttractionTiers {
			var matched []candidateNearby
			for _, c := range all {
				if c.hasDist && c.distKm <= tier.km {
					matched = append(matched, c)
				}
			}
			if len(matched) > 0 {
				scope = tier.scope
				chosen = matched
				break
			}
		}
	}
	if len(chosen) == 0 {
		scope = "county"
		for _, c := range all {
			if c.county == originCounty {
				chosen = append(chosen, c)
			}
		}
	}
	sortNearby(chosen)

	out := make([]nearbyAttraction, 0, len(chosen))
	for _, c := range chosen {
		if len(out) >= 12 {
			break
		}
		out = append(out, c.item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"scope":       scope,
		"attractions": out,
	})
}

func sortNearby(items []candidateNearby) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && nearbyLess(items[j], items[j-1]) {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

type candidateNearby struct {
	item    nearbyAttraction
	county  int
	hasDist bool
	distKm  float64
}

func nearbyLess(a, b candidateNearby) bool {
	if a.hasDist != b.hasDist {
		return a.hasDist
	}
	if a.hasDist && b.hasDist && a.distKm != b.distKm {
		return a.distKm < b.distKm
	}
	return a.item.Name < b.item.Name
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthKm = 6371.0
	r1 := lat1 * math.Pi / 180
	r2 := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(r1)*math.Cos(r2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthKm * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

func loadAttractionImages(id int) []AttractionImage {
	out := []AttractionImage{}
	rows, err := db.DB.Query(`SELECT url, COALESCE(copyright, '') FROM attraction_images WHERE attraction_id = $1 ORDER BY sort_order, id`, id)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var img AttractionImage
		if rows.Scan(&img.URL, &img.Copyright) == nil && strings.TrimSpace(img.URL) != "" {
			out = append(out, img)
		}
	}
	return out
}

func parseAttractionImages(raw json.RawMessage) []AttractionImage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var objects []AttractionImage
	if err := json.Unmarshal(raw, &objects); err == nil {
		out := make([]AttractionImage, 0, len(objects))
		for _, img := range objects {
			url := strings.TrimSpace(img.URL)
			if url == "" {
				continue
			}
			out = append(out, AttractionImage{URL: url, Copyright: clipPhotoCredit(img.Copyright)})
		}
		if len(out) > 0 || strings.Contains(string(raw), "{") {
			return out
		}
	}
	var urls []string
	if err := json.Unmarshal(raw, &urls); err != nil {
		return nil
	}
	out := make([]AttractionImage, 0, len(urls))
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url != "" {
			out = append(out, AttractionImage{URL: url})
		}
	}
	return out
}

func clipPhotoCredit(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "©"))
	s = strings.TrimSpace(s)
	const max = 500
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
