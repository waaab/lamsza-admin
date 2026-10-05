package venues

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const venueSelectCols = `v.id, v.settlement_id, v.name, COALESCE(v.name_ro, ''), COALESCE(v.name_de, ''), v.slug, v.kind,
		COALESCE(v.address, ''), COALESCE(v.notes, ''),
		v.latitude, v.longitude, v.seating_capacity, COALESCE(v.description, ''),
		COALESCE(vt.label_hu, v.kind),
		s.name, s.slug, c.name, c.slug`

const venueJoins = `FROM venues v
		JOIN settlements s ON v.settlement_id = s.id
		JOIN counties c ON s.county_id = c.id
		LEFT JOIN venue_types vt ON vt.slug = v.kind`

func scanVenueRow(scanner interface {
	Scan(dest ...interface{}) error
}) (models.Venue, error) {
	var v models.Venue
	var lat, lng sql.NullFloat64
	var seat sql.NullInt64
	err := scanner.Scan(
		&v.ID, &v.SettlementID, &v.Name, &v.NameRO, &v.NameDE, &v.Slug, &v.Kind,
		&v.Address, &v.Notes,
		&lat, &lng, &seat, &v.Description,
		&v.KindLabel,
		&v.SettlementName, &v.SettlementSlug, &v.CountyName, &v.CountySlug,
	)
	if err != nil {
		return v, err
	}
	if lat.Valid {
		x := lat.Float64
		v.Latitude = &x
	}
	if lng.Valid {
		x := lng.Float64
		v.Longitude = &x
	}
	if seat.Valid {
		x := int(seat.Int64)
		v.SeatingCapacity = &x
	}
	return v, nil
}

// HandlePublic GET /api/venues?settlement_id= | ?county_slug=&settlement_slug=&venue_slug=
func HandlePublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	if q.Get("venue_slug") != "" && q.Get("settlement_slug") != "" && q.Get("county_slug") != "" {
		getVenueBySlugs(w,
			strings.TrimSpace(q.Get("county_slug")),
			strings.TrimSpace(q.Get("settlement_slug")),
			strings.TrimSpace(q.Get("venue_slug")),
		)
		return
	}
	listVenues(w, q.Get("settlement_id"))
}

func getVenueBySlugs(w http.ResponseWriter, countySlug, settlementSlug, venueSlug string) {
	countySlug = strings.ToLower(countySlug)
	settlementSlug = strings.ToLower(settlementSlug)
	venueSlug = strings.ToLower(venueSlug)
	if countySlug == "" || settlementSlug == "" || venueSlug == "" {
		http.Error(w, "county_slug, settlement_slug és venue_slug kötelező", http.StatusBadRequest)
		return
	}
	row := db.DB.QueryRow(`
		SELECT `+venueSelectCols+` `+venueJoins+`
		WHERE lower(c.slug) = lower($1) AND lower(s.slug) = lower($2) AND lower(v.slug) = lower($3)`,
		countySlug, settlementSlug, venueSlug)
	v, err := scanVenueRow(row)
	if err == sql.ErrNoRows {
		http.Error(w, "nem található", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// HandleAdmin GET/POST/PUT/DELETE /api/admin/venues
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listVenues(w, r.URL.Query().Get("settlement_id"))
	case http.MethodPost:
		createVenue(w, r)
	case http.MethodPut:
		updateVenue(w, r)
	case http.MethodDelete:
		deleteVenue(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listVenues(w http.ResponseWriter, settlementIDStr string) {
	var rows *sql.Rows
	var err error
	base := `SELECT ` + venueSelectCols + ` ` + venueJoins + ` `
	if strings.TrimSpace(settlementIDStr) != "" {
		sid, err := strconv.Atoi(settlementIDStr)
		if err != nil || sid < 1 {
			http.Error(w, "érvénytelen settlement_id", http.StatusBadRequest)
			return
		}
		rows, err = db.DB.Query(base+`WHERE v.settlement_id = $1 ORDER BY v.name`, sid)
	} else {
		rows, err = db.DB.Query(base + `ORDER BY c.name, s.name, v.name`)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := []models.Venue{}
	for rows.Next() {
		v, err := scanVenueRow(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, v)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type venuePayload struct {
	SettlementID    int     `json:"settlement_id"`
	Name            string  `json:"name"`
	NameRO          string  `json:"name_ro"`
	NameDE          string  `json:"name_de"`
	Slug            string  `json:"slug"`
	Kind            string  `json:"kind"`
	Address         string  `json:"address"`
	Notes           string  `json:"notes"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	SeatingCapacity *int    `json:"seating_capacity"`
	Description     string  `json:"description"`
}

func normalizeKind(k string) string {
	k = strings.TrimSpace(strings.ToLower(k))
	if k == "" {
		return defaultVenueKindSlug()
	}
	var n int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM venue_types WHERE slug = $1`, k).Scan(&n)
	if err == nil && n > 0 {
		return k
	}
	return defaultVenueKindSlug()
}

func defaultVenueKindSlug() string {
	var s string
	err := db.DB.QueryRow(`SELECT slug FROM venue_types ORDER BY LOWER(label_hu) ASC, id ASC LIMIT 1`).Scan(&s)
	if err != nil || strings.TrimSpace(s) == "" {
		return "other"
	}
	return s
}

func normalizeSlug(name, slug string) string {
	if strings.TrimSpace(slug) != "" {
		return utils.Slugify(slug)
	}
	base := utils.Slugify(strings.TrimSpace(name))
	if base == "" {
		base = "helyszin"
	}
	return base
}

func createVenue(w http.ResponseWriter, r *http.Request) {
	var p venuePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.SettlementID < 1 || strings.TrimSpace(p.Name) == "" {
		http.Error(w, "settlement_id és név kötelező", http.StatusBadRequest)
		return
	}
	slug := normalizeSlug(p.Name, p.Slug)
	kind := normalizeKind(p.Kind)
	var id int
	err := db.DB.QueryRow(`
		INSERT INTO venues (settlement_id, name, name_ro, name_de, slug, kind, address, notes, latitude, longitude, seating_capacity, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		p.SettlementID, strings.TrimSpace(p.Name), strings.TrimSpace(p.NameRO), strings.TrimSpace(p.NameDE),
		slug, kind, strings.TrimSpace(p.Address), strings.TrimSpace(p.Notes),
		nullFloat64Ptr(p.Latitude), nullFloat64Ptr(p.Longitude), nullIntPtr(p.SeatingCapacity),
		strings.TrimSpace(p.Description)).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			http.Error(w, "ilyen slug már létezik ennél a településnél", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func nullFloat64Ptr(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullIntPtr(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func updateVenue(w http.ResponseWriter, r *http.Request) {
	var p struct {
		ID              int      `json:"id"`
		SettlementID    int      `json:"settlement_id"`
		Name            string   `json:"name"`
		NameRO          string   `json:"name_ro"`
		NameDE          string   `json:"name_de"`
		Slug            string   `json:"slug"`
		Kind            string   `json:"kind"`
		Address         string   `json:"address"`
		Notes           string   `json:"notes"`
		Latitude        *float64 `json:"latitude"`
		Longitude       *float64 `json:"longitude"`
		SeatingCapacity *int     `json:"seating_capacity"`
		Description     string   `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.ID < 1 || p.SettlementID < 1 || strings.TrimSpace(p.Name) == "" {
		http.Error(w, "id, settlement_id és név kötelező", http.StatusBadRequest)
		return
	}
	slug := normalizeSlug(p.Name, p.Slug)
	kind := normalizeKind(p.Kind)
	_, err := db.DB.Exec(`
		UPDATE venues SET settlement_id = $1, name = $2, name_ro = $3, name_de = $4, slug = $5, kind = $6, address = $7, notes = $8,
			latitude = $9, longitude = $10, seating_capacity = $11, description = $12
		WHERE id = $13`,
		p.SettlementID, strings.TrimSpace(p.Name), strings.TrimSpace(p.NameRO), strings.TrimSpace(p.NameDE),
		slug, kind, strings.TrimSpace(p.Address), strings.TrimSpace(p.Notes),
		nullFloat64Ptr(p.Latitude), nullFloat64Ptr(p.Longitude), nullIntPtr(p.SeatingCapacity),
		strings.TrimSpace(p.Description), p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteVenue(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "id kötelező", http.StatusBadRequest)
		return
	}
	res, err := db.DB.Exec(`DELETE FROM venues WHERE id = $1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "nem található", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// EnsureVenueBelongsToSettlement returns nil if venueID is nil/0 or matches settlement.
func EnsureVenueBelongsToSettlement(venueID *int, settlementID int) error {
	if venueID == nil || *venueID < 1 {
		return nil
	}
	var sid int
	err := db.DB.QueryRow(`SELECT settlement_id FROM venues WHERE id = $1`, *venueID).Scan(&sid)
	if err == sql.ErrNoRows {
		return fmt.Errorf("ismeretlen helyszín (épület)")
	}
	if err != nil {
		return err
	}
	if sid != settlementID {
		return fmt.Errorf("a helyszín nem ehhez a településhez tartozik")
	}
	return nil
}
