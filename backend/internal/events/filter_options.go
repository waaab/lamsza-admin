package events

import (
	"backend/internal/db"
	"encoding/json"
	"net/http"
)

// eventSubtypeOption is a distinct (type slug, subtype slug) among upcoming events.
type eventSubtypeOption struct {
	TypeSlug string `json:"type_slug"`
	Slug     string `json:"slug"`
	LabelHu  string `json:"label_hu"`
}

// filterOptionsResponse drives the /esemenyek filters (types, locations, month list).
type filterOptionsResponse struct {
	EventTypes    []string             `json:"event_types"`
	EventSubtypes []eventSubtypeOption `json:"event_subtypes"`
	Locations     []locationOption     `json:"locations"`
	// Months are YYYY-MM values that have at least one upcoming event (by start_date).
	Months []string `json:"months"`
	// EventDays are YYYY-MM-DD start dates of upcoming events (distinct calendar days).
	EventDays []string `json:"event_days"`
	// ScheduleEventIDs lists upcoming events that have at least one napi program day (for list UI fallback).
	ScheduleEventIDs []int `json:"schedule_event_ids"`
}

type locationOption struct {
	Name         string `json:"name"`
	LocationSlug string `json:"location_slug"`
	CountySlug   string `json:"county_slug"`
	CountyName   string `json:"county_name"`
}

// HandleEventFilterOptions GET - distinct types and locations among upcoming events.
func HandleEventFilterOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var out filterOptionsResponse

	rows, err := db.DB.Query(`
		SELECT DISTINCT et.slug
		FROM events e
		JOIN settlements s ON e.location_id = s.id
		JOIN counties c ON s.county_id = c.id
		JOIN catalog_event_types et ON e.event_type_id = et.id
		WHERE e.end_date >= CURRENT_DATE
		ORDER BY et.slug`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if t != "" {
			out.EventTypes = append(out.EventTypes, t)
		}
	}
	rows.Close()

	// ORDER BY must use columns from the SELECT list when DISTINCT is used (PostgreSQL 42P10).
	rowsSub, err := db.DB.Query(`
		SELECT DISTINCT et.slug, es.slug, es.label_hu
		FROM events e
		JOIN catalog_event_types et ON e.event_type_id = et.id
		JOIN catalog_event_subtypes es ON e.event_subtype_id = es.id
		WHERE e.end_date >= CURRENT_DATE
		ORDER BY et.slug, es.label_hu`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rowsSub.Next() {
		var o eventSubtypeOption
		if err := rowsSub.Scan(&o.TypeSlug, &o.Slug, &o.LabelHu); err != nil {
			rowsSub.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out.EventSubtypes = append(out.EventSubtypes, o)
	}
	rowsSub.Close()

	rows2, err := db.DB.Query(`
		SELECT DISTINCT s.name, s.slug, c.slug, c.name
		FROM events e
		JOIN settlements s ON e.location_id = s.id
		JOIN counties c ON s.county_id = c.id
		WHERE e.end_date >= CURRENT_DATE
		ORDER BY c.name, s.name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var loc locationOption
		if err := rows2.Scan(&loc.Name, &loc.LocationSlug, &loc.CountySlug, &loc.CountyName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out.Locations = append(out.Locations, loc)
	}

	rows3, err := db.DB.Query(`
		SELECT DISTINCT ym FROM (
			SELECT to_char(e.start_date::date + gs.day_offset * interval '1 day', 'YYYY-MM') AS ym
			FROM events e
			CROSS JOIN LATERAL generate_series(
				0,
				GREATEST(0, (e.end_date::date - e.start_date::date)),
				1
			) AS gs(day_offset)
			WHERE e.end_date >= CURRENT_DATE
			UNION
			SELECT to_char(esd.schedule_date, 'YYYY-MM') AS ym
			FROM event_schedule_days esd
			INNER JOIN events e ON e.id = esd.event_id
			WHERE e.end_date >= CURRENT_DATE
		) t
		WHERE ym IS NOT NULL AND ym != ''
		ORDER BY 1`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows3.Close()
	for rows3.Next() {
		var ym string
		if err := rows3.Scan(&ym); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ym != "" {
			out.Months = append(out.Months, ym)
		}
	}

	rows4, err := db.DB.Query(`
		SELECT DISTINCT d FROM (
			SELECT to_char(e.start_date::date + gs.day_offset * interval '1 day', 'YYYY-MM-DD') AS d
			FROM events e
			CROSS JOIN LATERAL generate_series(
				0,
				GREATEST(0, (e.end_date::date - e.start_date::date)),
				1
			) AS gs(day_offset)
			WHERE e.end_date >= CURRENT_DATE
			UNION
			SELECT to_char(esd.schedule_date, 'YYYY-MM-DD') AS d
			FROM event_schedule_days esd
			INNER JOIN events e ON e.id = esd.event_id
			WHERE e.end_date >= CURRENT_DATE
		) x
		WHERE d IS NOT NULL AND d != ''
		ORDER BY 1`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows4.Close()
	for rows4.Next() {
		var d string
		if err := rows4.Scan(&d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if d != "" {
			out.EventDays = append(out.EventDays, d)
		}
	}

	rows5, err := db.DB.Query(`
		SELECT DISTINCT esd.event_id
		FROM event_schedule_days esd
		INNER JOIN events e ON e.id = esd.event_id
		WHERE e.end_date >= CURRENT_DATE
		ORDER BY 1`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows5.Close()
	for rows5.Next() {
		var eid int
		if err := rows5.Scan(&eid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if eid > 0 {
			out.ScheduleEventIDs = append(out.ScheduleEventIDs, eid)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
