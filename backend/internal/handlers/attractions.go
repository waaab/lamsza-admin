package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
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
	ElevationM             *float64                `json:"elevation_m"`
	AreaKm2                *float64                `json:"area_km2"`
	DepthM                 *float64                `json:"depth_m"`
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

// attractionFactsError: the attraction's area and depth cannot be negative
// (the column checks in lamsza's schema say the same). Elevation can be, for
// a hollow below sea level. Null means "not set" and is always fine.
func attractionFactsError(areaKm2, depthM *float64) string {
	if areaKm2 != nil && *areaKm2 < 0 {
		return "A felszín nem lehet negatív."
	}
	if depthM != nil && *depthM < 0 {
		return "A mélység nem lehet negatív."
	}
	return ""
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
				gl.latitude, gl.longitude, a.elevation_m, a.area_km2, a.depth_m
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
			// latitude/longitude come from a LEFT JOIN on geo_locations and are
			// NULL for an attraction saved without coordinates. Scanning those
			// straight into float64 fails, and the error used to be swallowed
			// by an `err == nil` guard, so an attraction with no coordinates
			// was stored and then invisible in the admin list, with no way to
			// edit or delete it from the UI. Found on BOG-55; guarded by
			// TestAdminAttractionsListIncludesOneWithoutCoordinates.
			var lat, lon sql.NullFloat64
			if err := rows.Scan(&a.ID, &a.CountyID, &a.CountySlug, &a.CountyName, &a.Name, &a.NameRo, &a.NameDe,
				&a.Slug, &a.Description, &a.FeaturedImage, &a.FeaturedImageCopyright, &a.Content, &activitiesText, &prohibitionsText, &lat, &lon,
				&a.ElevationM, &a.AreaKm2, &a.DepthM); err != nil {
				log.Printf("admin attractions scan: %v", err)
				continue
			}
			a.Latitude = lat.Float64
			a.Longitude = lon.Float64
			a.Activities = splitActivities(activitiesText)
			a.Prohibitions = splitActivities(prohibitionsText)
			a.Images = loadAttractionImages(a.ID)
			list = append(list, a)
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
			ElevationM             *float64        `json:"elevation_m"`
			AreaKm2                *float64        `json:"area_km2"`
			DepthM                 *float64        `json:"depth_m"`
			Images                 json.RawMessage `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if msg := attractionFactsError(a.AreaKm2, a.DepthM); msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
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
			INSERT INTO attractions (county_id, name, name_ro, name_de, slug, description, location_id, featured_image, featured_image_copyright, content, activities, prohibitions, elevation_m, area_km2, depth_m)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id
		`, countyID, a.Name, a.NameRo, a.NameDe, slug, a.Description, glID, a.FeaturedImage, strings.TrimSpace(a.FeaturedImageCopyright), a.Content, joinActivities(a.Activities), joinActivities(a.Prohibitions), a.ElevationM, a.AreaKm2, a.DepthM).Scan(&attID)
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
			ElevationM             *float64        `json:"elevation_m"`
			AreaKm2                *float64        `json:"area_km2"`
			DepthM                 *float64        `json:"depth_m"`
			Images                 json.RawMessage `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if msg := attractionFactsError(a.AreaKm2, a.DepthM); msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
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
			UPDATE attractions SET county_id=$1, name=$2, name_ro=$3, name_de=$4, slug=$5, description=$6, location_id=$7, featured_image=$8, featured_image_copyright=$9, content=$10, activities=$11, prohibitions=$12,
				elevation_m=$14, area_km2=$15, depth_m=$16
			WHERE id=$13
		`, countyID, a.Name, a.NameRo, a.NameDe, slug, a.Description, glID, a.FeaturedImage, strings.TrimSpace(a.FeaturedImageCopyright), a.Content, joinActivities(a.Activities), joinActivities(a.Prohibitions), a.ID,
			a.ElevationM, a.AreaKm2, a.DepthM)
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

type candidateNearby struct {
	item    nearbyAttraction
	county  int
	hasDist bool
	distKm  float64
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
