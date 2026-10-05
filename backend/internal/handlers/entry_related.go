package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"backend/internal/db"

	"github.com/lib/pq"
)

const relatedEntryLimit = 8

type relatedEntry struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Category     string          `json:"category"`
	Location     string          `json:"location"`
	LocationSlug string          `json:"location_slug"`
	CountySlug   string          `json:"county_slug"`
	Photos       json.RawMessage `json:"photos"`
	Claimed      bool            `json:"claimed"`
	Verified     bool            `json:"-"`
}

func HandleEntryRelated(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	var id, locationID, categoryID int
	var countySlug string
	err := db.DB.QueryRow(`
		SELECT e.id, e.location_id, COALESCE(e.category_id, 0), c.slug
		FROM entries e
		JOIN settlements s ON e.location_id = s.id
		JOIN counties c ON s.county_id = c.id
		WHERE e.slug = $1`, slug).Scan(&id, &locationID, &categoryID, &countySlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Entry not found", http.StatusNotFound)
		} else {
			log.Printf("HandleEntryRelated: QueryRow error for slug %q: %v", slug, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	nearby := queryRelatedEntries(`
		SELECT e.id::text, e.name, COALESCE(e.slug,''), COALESCE(ec.name,''),
			s.name, s.slug, c.slug, COALESCE(e.photos, '[]'::jsonb),
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'),
			COALESCE(e.verified, false)
		FROM entries e
		JOIN settlements s ON e.location_id = s.id
		JOIN counties c ON s.county_id = c.id
		LEFT JOIN entry_categories ec ON e.category_id = ec.id
		WHERE e.location_id = $1 AND e.id <> $2 AND COALESCE(e.slug,'') <> ''
		ORDER BY LOWER(e.name) ASC, e.id ASC
		LIMIT $3`, locationID, id, relatedEntryLimit)

	if len(nearby) < relatedEntryLimit {
		exclude := relatedIDs(nearby)
		exclude = append(exclude, id)
		pad := queryRelatedEntries(`
			SELECT e.id::text, e.name, COALESCE(e.slug,''), COALESCE(ec.name,''),
				s.name, s.slug, c.slug, COALESCE(e.photos, '[]'::jsonb),
				EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'),
				COALESCE(e.verified, false)
			FROM entries e
			JOIN settlements s ON e.location_id = s.id
			JOIN counties c ON s.county_id = c.id
			LEFT JOIN entry_categories ec ON e.category_id = ec.id
			WHERE c.slug = $1 AND e.id <> ALL($2) AND COALESCE(e.slug,'') <> ''
			ORDER BY LOWER(e.name) ASC, e.id ASC
			LIMIT $3`, countySlug, excludeIntArray(exclude), relatedEntryLimit-len(nearby))
		nearby = append(nearby, pad...)
	}

	relExclude := relatedIDs(nearby)
	relExclude = append(relExclude, id)
	related := []relatedEntry{}
	if categoryID > 0 {
		related = queryRelatedEntries(`
			SELECT e.id::text, e.name, COALESCE(e.slug,''), COALESCE(ec.name,''),
				s.name, s.slug, c.slug, COALESCE(e.photos, '[]'::jsonb),
				EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'),
				COALESCE(e.verified, false)
			FROM entries e
			JOIN settlements s ON e.location_id = s.id
			JOIN counties c ON s.county_id = c.id
			LEFT JOIN entry_categories ec ON e.category_id = ec.id
			WHERE e.category_id = $1 AND c.slug = $2 AND e.id <> ALL($3) AND COALESCE(e.slug,'') <> ''
			ORDER BY LOWER(e.name) ASC, e.id ASC
			LIMIT $4`, categoryID, countySlug, excludeIntArray(relExclude), relatedEntryLimit)
	}

	if nearby == nil {
		nearby = []relatedEntry{}
	}
	if related == nil {
		related = []relatedEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"nearby":  nearby,
		"related": related,
	})
}

func queryRelatedEntries(sql string, args ...any) []relatedEntry {
	rows, err := db.DB.Query(sql, args...)
	if err != nil {
		log.Printf("queryRelatedEntries: Query error: %v", err)
		return []relatedEntry{}
	}
	defer rows.Close()

	out := make([]relatedEntry, 0)
	for rows.Next() {
		var e relatedEntry
		var photos json.RawMessage
		if err := rows.Scan(&e.ID, &e.Name, &e.Slug, &e.Category, &e.Location, &e.LocationSlug, &e.CountySlug, &photos, &e.Claimed, &e.Verified); err != nil {
			log.Printf("queryRelatedEntries: Scan error: %v", err)
			continue
		}
		e.Photos = sanitizePhotos(photos)
		if !e.Claimed || !e.Verified {
			e.Photos = json.RawMessage("[]")
		}
		out = append(out, e)
	}
	return out
}

func relatedIDs(entries []relatedEntry) []int {
	ids := make([]int, 0, len(entries))
	for _, e := range entries {
		n, err := strconv.Atoi(e.ID)
		if err == nil && n > 0 {
			ids = append(ids, n)
		}
	}
	return ids
}

func excludeIntArray(ids []int) interface{} {
	if len(ids) == 0 {
		return pq.Array([]int{0})
	}
	return pq.Array(ids)
}
