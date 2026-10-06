package events

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/venues"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type paginatedResponse struct {
	Events []models.Event `json:"events"`
	Total  int            `json:"total"`
}

func HandleAdminEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query(`
			SELECT e.id, e.location_id, e.default_venue_id, e.attraction_id, COALESCE(vdef.name, ''),
				e.title, e.description, COALESCE(e.featured_image, ''), COALESCE(e.featured_image_copyright, ''), e.start_date::text, e.start_time::text, e.end_date::text, e.end_time::text,
				e.event_type_id, e.event_subtype_id, COALESCE(et.slug, ''), COALESCE(es.slug, ''), COALESCE(e.access_type, 'public'), e.organizer, COALESCE(e.entry_price, '')
			FROM events e
			LEFT JOIN venues vdef ON e.default_venue_id = vdef.id
			JOIN catalog_event_types et ON e.event_type_id = et.id
			LEFT JOIN catalog_event_subtypes es ON e.event_subtype_id = es.id
			ORDER BY LOWER(e.title) ASC, e.id ASC`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		events := []models.AdminEvent{}
		for rows.Next() {
			var ev models.AdminEvent
			var locID, defVID, subID, attractionID sql.NullInt64
			var desc, st, et, org sql.NullString
			var feat, featCredit string
			var entryPrice string
			if err := rows.Scan(&ev.ID, &locID, &defVID, &attractionID, &ev.DefaultVenueName, &ev.Title, &desc, &feat, &featCredit, &ev.StartDate, &st, &ev.EndDate, &et, &ev.EventTypeID, &subID, &ev.EventType, &ev.EventSubtype, &ev.AccessType, &org, &entryPrice); err != nil {
				log.Printf("handleAdminEvents rows.Scan: %v", err)
				continue
			}
			if locID.Valid {
				v := int(locID.Int64)
				ev.LocationID = &v
			}
			if defVID.Valid {
				v := int(defVID.Int64)
				ev.DefaultVenueID = &v
			}
			if attractionID.Valid {
				v := int(attractionID.Int64)
				ev.AttractionID = &v
			}
			if subID.Valid {
				v := int(subID.Int64)
				ev.EventSubtypeID = &v
			}
			if desc.Valid {
				ev.Description = desc.String
			}
			ev.FeaturedImage = feat
			ev.FeaturedImageCopyright = featCredit
			if st.Valid {
				ev.StartTime = st.String
			}
			if et.Valid {
				ev.EndTime = et.String
			}
			if org.Valid {
				ev.Organizer = org.String
			}
			ev.EntryPrice = entryPrice
			events = append(events, ev)
		}
		json.NewEncoder(w).Encode(events)

	case "POST":
		var ev models.AdminEvent
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := resolveAdminEventIDs(&ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ev.AccessType = normalizeAccessType(ev.AccessType)
		if err := validateAdminEvent(&ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := venues.EnsureVenueBelongsToSettlement(ev.DefaultVenueID, *ev.LocationID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := db.DB.QueryRow(`INSERT INTO events (location_id, title, description, featured_image, featured_image_copyright, attraction_id, start_date, start_time, end_date, end_time, event_type_id, event_subtype_id, access_type, organizer, default_venue_id, entry_price) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`,
			ev.LocationID, ev.Title, ev.Description, ev.FeaturedImage, strings.TrimSpace(ev.FeaturedImageCopyright), nullIntPtr(ev.AttractionID), ev.StartDate, ev.StartTime, ev.EndDate, ev.EndTime, ev.EventTypeID, nullIntPtr(ev.EventSubtypeID), ev.AccessType, ev.Organizer, nullIntPtr(ev.DefaultVenueID), strings.TrimSpace(ev.EntryPrice)).Scan(&ev.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(ev)

	case "PUT":
		var ev models.AdminEvent
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := resolveAdminEventIDs(&ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ev.AccessType = normalizeAccessType(ev.AccessType)
		if err := validateAdminEvent(&ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := venues.EnsureVenueBelongsToSettlement(ev.DefaultVenueID, *ev.LocationID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, err := db.DB.Exec(`UPDATE events SET location_id=$1, title=$2, description=$3, featured_image=$4, featured_image_copyright=$5, attraction_id=$6, start_date=$7, start_time=$8, end_date=$9, end_time=$10, event_type_id=$11, event_subtype_id=$12, access_type=$13, organizer=$14, default_venue_id=$15, entry_price=$16 WHERE id=$17`,
			ev.LocationID, ev.Title, ev.Description, ev.FeaturedImage, strings.TrimSpace(ev.FeaturedImageCopyright), nullIntPtr(ev.AttractionID), ev.StartDate, ev.StartTime, ev.EndDate, ev.EndTime, ev.EventTypeID, nullIntPtr(ev.EventSubtypeID), ev.AccessType, ev.Organizer, nullIntPtr(ev.DefaultVenueID), strings.TrimSpace(ev.EntryPrice), ev.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			db.DB.Exec("DELETE FROM events WHERE id = $1", id)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func validateAdminEvent(ev *models.AdminEvent) error {
	if strings.TrimSpace(ev.Title) == "" {
		return fmt.Errorf("Az esemény címe kötelező.")
	}
	if ev.LocationID == nil || *ev.LocationID == 0 {
		return fmt.Errorf("Település / helyszín kötelező.")
	}
	if strings.TrimSpace(ev.StartDate) == "" || strings.TrimSpace(ev.EndDate) == "" {
		return fmt.Errorf("Kezdő és befejező dátum kötelező.")
	}
	if strings.TrimSpace(ev.StartTime) == "" || strings.TrimSpace(ev.EndTime) == "" {
		return fmt.Errorf("Kezdő és befejező időpont (óra:perc) kötelező.")
	}
	if ev.EventTypeID < 1 {
		return fmt.Errorf("Eseménytípus kötelező.")
	}
	return nil
}

func normalizeAccessType(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "members_only", "invitation_only":
		return s
	default:
		return "public"
	}
}

func resolveAdminEventIDs(ev *models.AdminEvent) error {
	if ev.EventTypeID < 1 && strings.TrimSpace(ev.EventType) != "" {
		slug := strings.TrimSpace(strings.ToLower(ev.EventType))
		err := db.DB.QueryRow(`SELECT id FROM catalog_event_types WHERE slug = $1`, slug).Scan(&ev.EventTypeID)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Ismeretlen eseménytípus.")
			}
			return err
		}
	}
	if ev.EventTypeID < 1 {
		return fmt.Errorf("Eseménytípus kötelező.")
	}
	if ev.EventSubtypeID != nil && *ev.EventSubtypeID > 0 {
		var etid int
		err := db.DB.QueryRow(`SELECT event_type_id FROM catalog_event_subtypes WHERE id = $1`, *ev.EventSubtypeID).Scan(&etid)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Érvénytelen altípus.")
			}
			return err
		}
		if etid != ev.EventTypeID {
			return fmt.Errorf("Az altípus nem ehhez a típushoz tartozik.")
		}
	}
	return nil
}

func nullIntPtr(p *int) interface{} {
	if p == nil || *p < 1 {
		return nil
	}
	return *p
}
