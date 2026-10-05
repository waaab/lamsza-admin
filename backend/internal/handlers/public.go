package handlers

import (
	"backend/internal/auth"
	"backend/internal/db"
	"backend/internal/directory"
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/utils"

	"github.com/lib/pq"
)

func EntriesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")
	locationSlug := r.URL.Query().Get("location_slug")
	locationID := r.URL.Query().Get("location_id")
	countySlug := r.URL.Query().Get("county_slug")

	var rows *sql.Rows
	var err error

	params := []interface{}{}
	paramIdx := 1
	sqlQuery := ""

	normalizedQ := utils.Slugify(q)
	log.Printf("EntriesHandler: q=%q, normalizedQ=%q", q, normalizedQ)
	if q != "" && normalizedQ != "" {
		sqlQuery = `
			SELECT 
				e.id, COALESCE(typ.name, ''), COALESCE(ec.name, ''), e.name, e.slug, 
				COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(c.name, ''), COALESCE(c.slug, ''), COALESCE(s.type, ''), 
				COALESCE(s.name_ro, ''), COALESCE(s.name_de, ''),
				COALESCE(e.phone, ''), COALESCE(e.address, ''), COALESCE(e.notes, ''), 
				e.languages, COALESCE(e.url, ''),
				EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'), COALESCE(e.verified, false), COALESCE(e.hours, '{}'::jsonb), COALESCE(e.delivery_hours, '{}'::jsonb),
				COALESCE(e.hours_enabled, false), COALESCE(e.delivery_enabled, false), COALESCE(e.social_links, '[]'::jsonb), COALESCE(gl.latitude::text || ', ' || gl.longitude::text, ''),
				COALESCE(e.photos, '[]'::jsonb),
				CASE WHEN unaccent(LOWER(e.name)) = unaccent(LOWER($1)) THEN true ELSE false END as is_direct_match,
				ts_rank_cd(e.search_vector, plainto_tsquery('simple', $2)) as rank,
				COALESCE(array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL), ARRAY[]::text[]),
				COALESCE(e.ratings_enabled, false)
			FROM entries e
			JOIN entry_types typ ON typ.id = e.type_id
			LEFT JOIN settlements s ON e.location_id = s.id
			LEFT JOIN geo_locations gl ON gl.id = s.location_id
			LEFT JOIN counties c ON s.county_id = c.id
			LEFT JOIN entry_categories ec ON e.category_id = ec.id
			LEFT JOIN entry_categories ec_parent ON ec_parent.id = ec.parent_id
			LEFT JOIN entry_tags et ON e.id = et.entry_id
			LEFT JOIN tags t ON et.tag_id = t.id
			WHERE e.search_vector @@ plainto_tsquery('simple', $2)
				AND e.published = true
		`
		params = append(params, q, normalizedQ)
		paramIdx = 3
	} else {
		sqlQuery = `
			SELECT 
				e.id, COALESCE(typ.name, ''), COALESCE(ec.name, ''), e.name, e.slug, 
				COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(c.name, ''), COALESCE(c.slug, ''), COALESCE(s.type, ''), 
				COALESCE(s.name_ro, ''), COALESCE(s.name_de, ''),
				COALESCE(e.phone, ''), COALESCE(e.address, ''), COALESCE(e.notes, ''), 
				e.languages, COALESCE(e.url, ''),
				EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'), COALESCE(e.verified, false), COALESCE(e.hours, '{}'::jsonb), COALESCE(e.delivery_hours, '{}'::jsonb),
				COALESCE(e.hours_enabled, false), COALESCE(e.delivery_enabled, false), COALESCE(e.social_links, '[]'::jsonb), COALESCE(gl.latitude::text || ', ' || gl.longitude::text, ''),
				COALESCE(e.photos, '[]'::jsonb),
				CASE WHEN unaccent(LOWER(e.name)) = unaccent(LOWER($1)) THEN true ELSE false END as is_direct_match,
				0 as rank,
				COALESCE(array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL), ARRAY[]::text[]),
				COALESCE(e.ratings_enabled, false)
			FROM entries e
			JOIN entry_types typ ON typ.id = e.type_id
			LEFT JOIN settlements s ON e.location_id = s.id
			LEFT JOIN geo_locations gl ON gl.id = s.location_id
			LEFT JOIN counties c ON s.county_id = c.id
			LEFT JOIN entry_categories ec ON e.category_id = ec.id
			LEFT JOIN entry_categories ec_parent ON ec_parent.id = ec.parent_id
			LEFT JOIN entry_tags et ON e.id = et.entry_id
			LEFT JOIN tags t ON et.tag_id = t.id
			WHERE e.published = true
		`
		params = append(params, q)
		paramIdx = 2
	}

	if category != "" {
		sqlQuery += " AND (pg_slugify(ec.slug) = pg_slugify($" + fmt.Sprintf("%d", paramIdx) + ") OR pg_slugify(ec_parent.slug) = pg_slugify($" + fmt.Sprintf("%d", paramIdx) + ") OR unaccent(ec.name) ILIKE unaccent($" + fmt.Sprintf("%d", paramIdx) + ") OR EXISTS (SELECT 1 FROM entry_category_links l JOIN entry_categories lc ON lc.id = l.category_id LEFT JOIN entry_categories lp ON lp.id = lc.parent_id WHERE l.entry_id = e.id AND (pg_slugify(lc.slug) = pg_slugify($" + fmt.Sprintf("%d", paramIdx) + ") OR pg_slugify(lp.slug) = pg_slugify($" + fmt.Sprintf("%d", paramIdx) + ") OR unaccent(lc.name) ILIKE unaccent($" + fmt.Sprintf("%d", paramIdx) + "))))"
		params = append(params, category)
		paramIdx++
	}
	if tag != "" {
		sqlQuery += " AND unaccent(t.name) ILIKE unaccent($" + fmt.Sprintf("%d", paramIdx) + ")"
		params = append(params, tag)
		paramIdx++
	}
	if locationSlug != "" {
		sqlQuery += " AND s.slug = $" + fmt.Sprintf("%d", paramIdx)
		params = append(params, locationSlug)
		paramIdx++
	}
	if locationID != "" {
		sqlQuery += " AND e.location_id = $" + fmt.Sprintf("%d", paramIdx)
		params = append(params, locationID)
		paramIdx++
	}
	if countySlug != "" {
		sqlQuery += " AND c.slug = $" + fmt.Sprintf("%d", paramIdx)
		params = append(params, countySlug)
		paramIdx++
	}

	if q != "" && normalizedQ != "" {
		sqlQuery += " GROUP BY e.id, typ.name, ec.name, e.name, e.slug, s.id, s.name, s.slug, c.name, c.slug, s.type, s.name_ro, s.name_de, e.phone, e.address, e.notes, e.languages, e.url, e.verified, e.hours, e.delivery_hours, e.hours_enabled, e.delivery_enabled, e.social_links, gl.latitude, gl.longitude, e.photos, e.ratings_enabled ORDER BY is_direct_match DESC, rank DESC, btrim(e.name) ASC, e.id ASC"
	} else {
		sqlQuery += " GROUP BY e.id, typ.name, ec.name, e.name, e.slug, s.id, s.name, s.slug, c.name, c.slug, s.type, s.name_ro, s.name_de, e.phone, e.address, e.notes, e.languages, e.url, e.verified, e.hours, e.delivery_hours, e.hours_enabled, e.delivery_enabled, e.social_links, gl.latitude, gl.longitude, e.photos, e.ratings_enabled ORDER BY is_direct_match DESC, btrim(e.name) ASC, e.id ASC"
	}

	log.Printf("EntriesHandler query: %s", sqlQuery)
	log.Printf("EntriesHandler params: %v", params)

	rows, err = db.DB.Query(sqlQuery, params...)
	if err != nil {
		log.Printf("EntriesHandler query error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	entries := []models.Entry{}
	for rows.Next() {
		var e models.Entry
		var pqLanguages []string
		var pqTags []string
		var rank float64
		var hours, delivery, socialLinks, photos []byte
		if err := rows.Scan(&e.ID, &e.Type, &e.Category, &e.Name, &e.Slug, &e.Location, &e.LocationSlug, &e.LocationCounty, &e.CountySlug, &e.LocationType, &e.LocationRo, &e.LocationDe, &e.Phone, &e.Address, &e.Notes, pq.Array(&pqLanguages), &e.URL, &e.Claimed, &e.Verified, &hours, &delivery, &e.HoursEnabled, &e.DeliveryEnabled, &socialLinks, &e.LocationCoordinates, &photos, &e.IsDirectMatch, &rank, pq.Array(&pqTags), &e.RatingsEnabled); err != nil {
			log.Printf("EntriesHandler scan error: %v", err)
			continue
		}
		e.Name = strings.TrimSpace(e.Name)
		e.Languages = pqLanguages
		e.Type = utils.CanonicalEntryType(e.Type)
		e.Hours = jsonObjectOrEmpty(hours)
		e.DeliveryHours = jsonObjectOrEmpty(delivery)
		e.SocialLinks = jsonArrayOrEmpty(socialLinks)
		e.Photos = sanitizePhotos(photos)
		if pqTags != nil {
			e.Tags = pqTags
		} else {
			e.Tags = []string{}
		}
		entries = append(entries, e)
	}
	// encoding/json encodes nil []string as null; clients expect [] for "no tags".
	viewerUserID := 0
	if user, err := auth.UserFromRequest(r); err == nil && user != nil {
		viewerUserID = user.ID
	}
	entryIDs := make([]int, 0, len(entries))
	for i := range entries {
		if entries[i].Tags == nil {
			entries[i].Tags = []string{}
		}
		entries[i].Categories = []string{}
		if id, err := strconv.Atoi(entries[i].ID); err == nil {
			entryIDs = append(entryIDs, id)
		}
		ApplyPublicEntryExtrasMode(&entries[i], viewerUserID, true)
	}
	if names, err := directory.NamesForEntries(db.DB, entryIDs); err == nil {
		for i := range entries {
			id, convErr := strconv.Atoi(entries[i].ID)
			if convErr != nil {
				continue
			}
			got := names[id]
			if len(got) == 0 && entries[i].Category != "" {
				got = []string{entries[i].Category}
			}
			entries[i].Categories = got
		}
	}
	json.NewEncoder(w).Encode(entries)
	log.Printf("EntriesHandler found %d entries", len(entries))
}

func EntryDetailHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Missing slug", 400)
		return
	}

	var e models.Entry
	var pqLanguages []string
	var hours, delivery, socialLinks, photos []byte
	err := db.DB.QueryRow(`
		SELECT 
			e.id, COALESCE(typ.name, ''), COALESCE(ec.name, ''), e.name, e.slug, 
			s.name, s.slug, c.name, c.slug, s.type, 
			COALESCE(s.name_ro, ''), COALESCE(s.name_de, ''),
			COALESCE(e.phone, ''), COALESCE(e.address, ''), COALESCE(e.notes, ''), 
			e.languages, COALESCE(e.url, ''),
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'), COALESCE(e.verified, false), COALESCE(e.hours, '{}'::jsonb), COALESCE(e.delivery_hours, '{}'::jsonb),
			COALESCE(e.hours_enabled, false), COALESCE(e.delivery_enabled, false), COALESCE(e.social_links, '[]'::jsonb), COALESCE(gl.latitude::text || ', ' || gl.longitude::text, ''),
			COALESCE(e.photos, '[]'::jsonb),
			COALESCE(e.ratings_enabled, false),
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'pending'),
			EXISTS (SELECT 1 FROM entry_suggestions sg WHERE sg.entry_id = e.id AND sg.status = 'open')
		FROM entries e
		JOIN entry_types typ ON typ.id = e.type_id
		JOIN settlements s ON e.location_id = s.id
		LEFT JOIN geo_locations gl ON gl.id = s.location_id
		JOIN counties c ON s.county_id = c.id
		LEFT JOIN entry_categories ec ON e.category_id = ec.id
		WHERE e.slug = $1 AND e.published = true`, slug).Scan(&e.ID, &e.Type, &e.Category, &e.Name, &e.Slug, &e.Location, &e.LocationSlug, &e.LocationCounty, &e.CountySlug, &e.LocationType, &e.LocationRo, &e.LocationDe, &e.Phone, &e.Address, &e.Notes, pq.Array(&pqLanguages), &e.URL, &e.Claimed, &e.Verified, &hours, &delivery, &e.HoursEnabled, &e.DeliveryEnabled, &socialLinks, &e.LocationCoordinates, &photos, &e.RatingsEnabled, &e.ClaimPending, &e.SuggestionPending)

	if err != nil {
		http.Error(w, "Entry not found", 404)
		return
	}
	e.Name = strings.TrimSpace(e.Name)
	e.Languages = pqLanguages
	e.Type = utils.CanonicalEntryType(e.Type)
	e.Hours = jsonObjectOrEmpty(hours)
	e.DeliveryHours = jsonObjectOrEmpty(delivery)
	e.SocialLinks = jsonArrayOrEmpty(socialLinks)
	e.Photos = sanitizePhotos(photos)
	e.Categories = []string{}
	if id, convErr := strconv.Atoi(e.ID); convErr == nil {
		if names, nameErr := directory.NamesForEntries(db.DB, []int{id}); nameErr == nil {
			if got := names[id]; len(got) > 0 {
				e.Categories = got
			} else if e.Category != "" {
				e.Categories = []string{e.Category}
			}
		}
	}

	rows, _ := db.DB.Query("SELECT t.name FROM tags t JOIN entry_tags et ON t.id = et.tag_id WHERE et.entry_id = $1", e.ID)
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			e.Tags = append(e.Tags, tag)
		}
	}

	// Apply public entry extras based on claimed status
	viewerUserID := 0
	if user, err := auth.UserFromRequest(r); err == nil && user != nil {
		viewerUserID = user.ID
	}
	ApplyPublicEntryExtras(&e, viewerUserID)

	json.NewEncoder(w).Encode(e)
}

func HandlePublicEntryCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.DB.Query(`SELECT id, name, slug, parent_id, sort_order FROM entry_categories ORDER BY COALESCE(parent_id, id), sort_order, id`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type publicCategory struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		ParentID  *int   `json:"parent_id"`
		SortOrder int    `json:"sort_order"`
	}
	res := []publicCategory{}
	for rows.Next() {
		var item publicCategory
		var parentID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &parentID, &item.SortOrder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parentID.Valid {
			v := int(parentID.Int64)
			item.ParentID = &v
		}
		res = append(res, item)
	}
	json.NewEncoder(w).Encode(res)
}
