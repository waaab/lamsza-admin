package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func HandleSetCountySeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		LocationID int `json:"location_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LocationID == 0 {
		http.Error(w, "Invalid request: location_id required", http.StatusBadRequest)
		return
	}

	var countyID int
	err := db.DB.QueryRow("SELECT county_id FROM settlements WHERE id = $1", body.LocationID).Scan(&countyID)
	if err != nil {
		http.Error(w, "Settlement not found", http.StatusNotFound)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec("UPDATE settlements SET is_county_seat = false WHERE county_id = $1", countyID); err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("UPDATE settlements SET is_county_seat = true WHERE id = $1", body.LocationID); err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("County seat set: settlement_id=%d, county_id=%d", body.LocationID, countyID)
	w.WriteHeader(http.StatusOK)
}

func HandleAdminLocations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		q := r.URL.Query()
		typeFilter := q.Get("type")
		countySlugFilter := q.Get("county_slug")

		baseQuery := "SELECT id, name, COALESCE(name_ro, ''), COALESCE(name_de, ''), COALESCE(county, ''), COALESCE(county_slug, ''), COALESCE(type, ''), COALESCE(slug, ''), COALESCE(post_code, ''), COALESCE(coordinates, ''), COALESCE(population, ''), COALESCE(area, ''), COALESCE(crest, ''), parent_id, COALESCE(is_county_seat, false) FROM locations"
		args := []interface{}{}
		argIdx := 1
		var conditions []string

		if typeFilter != "" {
			conditions = append(conditions, fmt.Sprintf("LOWER(type) = LOWER($%d)", argIdx))
			args = append(args, typeFilter)
			argIdx++
		}
		if countySlugFilter != "" {
			conditions = append(conditions, fmt.Sprintf("LOWER(county_slug) = LOWER($%d)", argIdx))
			args = append(args, countySlugFilter)
			argIdx++
		}
		if len(conditions) > 0 {
			baseQuery += " WHERE " + conditions[0]
			for i := 1; i < len(conditions); i++ {
				baseQuery += " AND " + conditions[i]
			}
		}
		baseQuery += " ORDER BY LOWER(name) ASC, id ASC"

		rows, err := db.DB.Query(baseQuery, args...)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.Location
		for rows.Next() {
			var loc models.Location
			if err := rows.Scan(&loc.ID, &loc.Name, &loc.NameRo, &loc.NameDe, &loc.County, &loc.CountySlug, &loc.Type, &loc.Slug, &loc.PostCode, &loc.Coordinates, &loc.Population, &loc.Area, &loc.Crest, &loc.ParentID, &loc.IsCountySeat); err == nil {
				res = append(res, loc)
			}
		}
		if res == nil {
			res = []models.Location{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var l models.Location
		if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		l.Slug = utils.Slugify(l.Name)
		l.CountySlug = utils.Slugify(l.County)
		if l.Type == "megye" {
			err := db.DB.QueryRow("INSERT INTO counties (name, name_ro, name_de, slug) VALUES ($1, $2, $3, $4) RETURNING id",
				l.Name, l.NameRo, l.NameDe, l.Slug).Scan(&l.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		} else {
			var countyID int
			if err := db.DB.QueryRow("SELECT id FROM counties WHERE slug = $1", l.CountySlug).Scan(&countyID); err != nil {
				db.DB.QueryRow("SELECT id FROM counties LIMIT 1").Scan(&countyID)
			}
			var glID *int
			if l.Coordinates != "" {
				var id int
				if err := db.DB.QueryRow("INSERT INTO geo_locations (latitude, longitude) VALUES (trim(split_part($1, ',', 1))::float, trim(split_part($1, ',', 2))::float) RETURNING id", l.Coordinates).Scan(&id); err == nil {
					glID = &id
				}
			}
			err := db.DB.QueryRow("INSERT INTO settlements (county_id, name, name_ro, name_de, slug, type, location_id, parent_id, post_code, population, area, crest, is_county_seat) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id",
				countyID, l.Name, l.NameRo, l.NameDe, l.Slug, l.Type, glID, l.ParentID, l.PostCode, l.Population, l.Area, l.Crest, l.IsCountySeat).Scan(&l.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		json.NewEncoder(w).Encode(l)

	case "PUT":
		var l models.Location
		if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		l.Slug = utils.Slugify(l.Name)
		l.CountySlug = utils.Slugify(l.County)
		if l.Type == "megye" {
			_, err := db.DB.Exec("UPDATE counties SET name=$1, name_ro=$2, name_de=$3, slug=$4 WHERE id=$5",
				l.Name, l.NameRo, l.NameDe, l.Slug, l.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		} else {
			_, err := db.DB.Exec("UPDATE settlements SET name=$1, name_ro=$2, name_de=$3, slug=$4, post_code=$5, population=$6, area=$7, crest=$8, parent_id=$9, is_county_seat=$10 WHERE id=$11",
				l.Name, l.NameRo, l.NameDe, l.Slug, l.PostCode, l.Population, l.Area, l.Crest, l.ParentID, l.IsCountySeat, l.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			res, _ := db.DB.Exec("DELETE FROM counties WHERE id = $1", id)
			if n, _ := res.RowsAffected(); n == 0 {
				db.DB.Exec("DELETE FROM settlements WHERE id = $1", id)
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
