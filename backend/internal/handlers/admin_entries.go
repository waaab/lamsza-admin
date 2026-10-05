package handlers

import (
	"backend/internal/db"
	"backend/internal/directory"
	"backend/internal/models"
	"backend/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

func adminDeliveryEnabled(catName, name string, requested bool) bool {
	if !requested {
		return false
	}
	if strings.TrimSpace(name) == "Lámsza.com" {
		return false
	}
	return utils.CategoryOffersDelivery(catName)
}

func HandleAdminEntries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sqlQuery := `
			SELECT 
				e.id, COALESCE(typ.name, ''), e.location_id, e.category_id, COALESCE(cat.name, ''), e.name, COALESCE(e.slug, ''),
				COALESCE(e.url, ''), COALESCE(e.phone, ''), COALESCE(e.address, ''), COALESCE(e.notes, ''), 
				e.languages,
				COALESCE(e.verified, false), COALESCE(e.hours, '{}'::jsonb), COALESCE(e.hours_enabled, false),
				COALESCE(e.delivery_hours, '{}'::jsonb), COALESCE(e.delivery_enabled, false),
				COALESCE(e.photos, '[]'::jsonb),
				COALESCE(array_agg(t.name) FILTER (WHERE t.name IS NOT NULL), ARRAY[]::VARCHAR[]) as tags
			FROM entries e
			JOIN entry_types typ ON typ.id = e.type_id
			LEFT JOIN entry_categories cat ON cat.id = e.category_id
			LEFT JOIN entry_tags et ON e.id = et.entry_id
			LEFT JOIN tags t ON et.tag_id = t.id
			GROUP BY e.id, typ.name, e.location_id, e.category_id, cat.name, e.name, e.url, e.phone, e.address, e.notes, e.languages, e.verified, e.hours, e.hours_enabled, e.delivery_hours, e.delivery_enabled, e.photos
			ORDER BY LOWER(e.name) ASC, e.id ASC
		`
		rows, err := db.DB.Query(sqlQuery)
		if err != nil {
			log.Println("handleAdminEntries GET error:", err)
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.AdminEntry
		for rows.Next() {
			var s models.AdminEntry
			var pqLanguages []string
			var pqTags []string
			var locID sql.NullInt64
			var hours, delivery, photos []byte
			if err := rows.Scan(&s.ID, &s.Type, &locID, &s.CategoryID, &s.Category, &s.Name, &s.Slug, &s.URL, &s.Phone, &s.Address, &s.Notes, pq.Array(&pqLanguages), &s.Verified, &hours, &s.HoursEnabled, &delivery, &s.DeliveryEnabled, &photos, pq.Array(&pqTags)); err == nil {
				if locID.Valid {
					v := int(locID.Int64)
					s.LocationID = &v
				}
				s.Languages = pqLanguages
				s.Tags = pqTags
				s.Hours = jsonObjectOrEmpty(hours)
				s.DeliveryHours = jsonObjectOrEmpty(delivery)
				s.Photos = sanitizePhotos(photos)
				s.Type = utils.CanonicalEntryType(s.Type)
				res = append(res, s)
			} else {
				log.Println("handleAdminEntries rows.Scan error:", err)
			}
		}
		if res == nil {
			res = []models.AdminEntry{}
		}
		for i := range res {
			got, loadErr := directory.LoadEntryCategoryIDs(db.DB, res[i].ID)
			if loadErr != nil || len(got) == 0 {
				if res[i].CategoryID != nil {
					res[i].CategoryIDs = []int{*res[i].CategoryID}
				} else {
					res[i].CategoryIDs = []int{}
				}
				continue
			}
			res[i].CategoryIDs = got
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var s models.AdminEntry
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		typeID, typeName, err := resolveEntryTypeID(s.Type)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.Type = typeName
		catID, catName, catErr := resolveEntryCategoryID(s.CategoryID, s.Category)
		if catErr != nil {
			writeCategoryResolveError(w, catErr)
			return
		}
		s.CategoryID = &catID
		s.Category = catName
		if len(s.Languages) == 0 {
			s.Languages = []string{"HU"}
		}

		s.Slug = utils.Slugify(s.Name)
		s.Hours = jsonObjectOrEmpty(s.Hours)
		s.DeliveryHours = jsonObjectOrEmpty(s.DeliveryHours)
		s.DeliveryEnabled = adminDeliveryEnabled(catName, s.Name, s.DeliveryEnabled)
		s.Photos = sanitizePhotos(s.Photos)
		categoryIDs, catSetErr := directory.NormalizeCategoryIDs(catID, s.CategoryIDs)
		if catSetErr != nil {
			writeCategoryResolveError(w, catSetErr)
			return
		}
		err = db.DB.QueryRow("INSERT INTO entries (type_id, location_id, category_id, cat_name, name, slug, url, phone, address, notes, languages, verified, hours, hours_enabled, delivery_hours, delivery_enabled, photos) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15::jsonb, $16, $17::jsonb) RETURNING id",
			typeID, s.LocationID, catID, catName, s.Name, s.Slug, s.URL, s.Phone, s.Address, s.Notes, pq.Array(s.Languages), s.Verified, string(s.Hours), s.HoursEnabled, string(s.DeliveryHours), s.DeliveryEnabled, string(s.Photos)).Scan(&s.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err = directory.ReplaceEntryCategories(db.DB, s.ID, categoryIDs); err != nil {
			writeCategoryResolveError(w, err)
			return
		}
		s.CategoryIDs = categoryIDs

		for _, tagName := range s.Tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			var tagID int
			err = db.DB.QueryRow("INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id", tagName).Scan(&tagID)
			if err == nil {
				db.DB.Exec("INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)", s.ID, tagID)
			}
		}

		json.NewEncoder(w).Encode(s)

	case "PUT":
		var s models.AdminEntry
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		typeID, typeName, err := resolveEntryTypeID(s.Type)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.Type = typeName
		catID, catName, catErr := resolveEntryCategoryID(s.CategoryID, s.Category)
		if catErr != nil {
			writeCategoryResolveError(w, catErr)
			return
		}
		s.CategoryID = &catID
		s.Category = catName

		s.Slug = utils.Slugify(s.Name)
		s.Hours = jsonObjectOrEmpty(s.Hours)
		s.DeliveryHours = jsonObjectOrEmpty(s.DeliveryHours)
		s.DeliveryEnabled = adminDeliveryEnabled(catName, s.Name, s.DeliveryEnabled)
		s.Photos = sanitizePhotos(s.Photos)
		categoryIDs, catSetErr := directory.NormalizeCategoryIDs(catID, s.CategoryIDs)
		if catSetErr != nil {
			writeCategoryResolveError(w, catSetErr)
			return
		}
		_, err = db.DB.Exec(`UPDATE entries SET 
            type_id=$1, location_id=$2, category_id=$3, cat_name=$4, name=$5, slug=$6, url=$7, 
            phone=$8, address=$9, notes=$10, languages=$11, verified=$12, hours=$13::jsonb, hours_enabled=$14, delivery_hours=$15::jsonb, delivery_enabled=$16, photos=$17::jsonb 
            WHERE id=$18`,
			typeID, s.LocationID, catID, catName, s.Name, s.Slug, s.URL, s.Phone, s.Address, s.Notes, pq.Array(s.Languages), s.Verified, string(s.Hours), s.HoursEnabled, string(s.DeliveryHours), s.DeliveryEnabled, string(s.Photos), s.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err = directory.ReplaceEntryCategories(db.DB, s.ID, categoryIDs); err != nil {
			writeCategoryResolveError(w, err)
			return
		}
		s.CategoryIDs = categoryIDs

		db.DB.Exec("DELETE FROM entry_tags WHERE entry_id = $1", s.ID)
		for _, tagName := range s.Tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			var tagID int
			err = db.DB.QueryRow("INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id", tagName).Scan(&tagID)
			if err == nil {
				db.DB.Exec("INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)", s.ID, tagID)
			}
		}

		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			db.DB.Exec("DELETE FROM websites WHERE entry_id = $1", id)
			db.DB.Exec("DELETE FROM entries WHERE id = $1", id)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func scanEntryCategory(row interface {
	Scan(dest ...interface{}) error
}) (models.EntryCategory, error) {
	var sc models.EntryCategory
	var parentID sql.NullInt64
	if err := row.Scan(&sc.ID, &sc.Name, &sc.Slug, &parentID, &sc.SortOrder); err != nil {
		return sc, err
	}
	if parentID.Valid {
		v := int(parentID.Int64)
		sc.ParentID = &v
	}
	return sc, nil
}

func validateCategoryParent(parentID *int) error {
	if parentID == nil {
		return nil
	}
	var grandparent sql.NullInt64
	err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, *parentID).Scan(&grandparent)
	if err == sql.ErrNoRows {
		return fmt.Errorf("parent not found")
	}
	if err != nil {
		return err
	}
	if grandparent.Valid {
		return fmt.Errorf("third level not allowed")
	}
	return nil
}

func siblingSortOrder(parentID *int) (int, error) {
	var sortOrder int
	err := db.DB.QueryRow(`
		SELECT COALESCE(MAX(sort_order), 0) + 1 FROM entry_categories
		WHERE (($1::int IS NULL AND parent_id IS NULL) OR parent_id = $1)`,
		parentID).Scan(&sortOrder)
	return sortOrder, err
}

func HandleAdminEntryCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query(`SELECT id, name, slug, parent_id, sort_order FROM entry_categories ORDER BY COALESCE(parent_id, id), sort_order, id`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.EntryCategory
		for rows.Next() {
			sc, err := scanEntryCategory(rows)
			if err == nil {
				res = append(res, sc)
			}
		}
		if res == nil {
			res = []models.EntryCategory{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var sc models.EntryCategory
		if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sc.Name = strings.TrimSpace(sc.Name)
		if sc.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if err := validateCategoryParent(sc.ParentID); err != nil {
			if err.Error() == "parent not found" {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err.Error() == "third level not allowed" {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		sortOrder, err := siblingSortOrder(sc.ParentID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sc.Slug = utils.Slugify(sc.Name)
		err = db.DB.QueryRow(
			`INSERT INTO entry_categories (name, slug, parent_id, sort_order) VALUES ($1, $2, $3, $4) RETURNING id, sort_order`,
			sc.Name, sc.Slug, sc.ParentID, sortOrder,
		).Scan(&sc.ID, &sc.SortOrder)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(sc)

	case "PUT":
		var sc models.EntryCategory
		if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sc.Name = strings.TrimSpace(sc.Name)
		if sc.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if err := validateCategoryParent(sc.ParentID); err != nil {
			if err.Error() == "parent not found" || err.Error() == "third level not allowed" {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		if sc.ParentID != nil {
			var children int
			if err := db.DB.QueryRow(`SELECT COUNT(*) FROM entry_categories WHERE parent_id = $1`, sc.ID).Scan(&children); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if children > 0 {
				http.Error(w, "parent with children cannot be moved", http.StatusBadRequest)
				return
			}
		}
		sc.Slug = utils.Slugify(sc.Name)
		_, err := db.DB.Exec(
			`UPDATE entry_categories SET name=$1, slug=$2, parent_id=$3 WHERE id=$4`,
			sc.Name, sc.Slug, sc.ParentID, sc.ID,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if _, err := db.DB.Exec(`UPDATE entries SET cat_name = $1 WHERE category_id = $2`, sc.Name, sc.ID); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		moveToStr := r.URL.Query().Get("move_to")
		if moveToStr != "" {
			moveTo, err := strconv.Atoi(moveToStr)
			if err != nil {
				http.Error(w, "invalid move_to", http.StatusBadRequest)
				return
			}
			if moveTo == id {
				http.Error(w, "move_to cannot equal id", http.StatusBadRequest)
				return
			}
			var parentID sql.NullInt64
			if err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, id).Scan(&parentID); err == sql.ErrNoRows {
				http.Error(w, "category not found", http.StatusNotFound)
				return
			} else if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !parentID.Valid {
				var moveParent sql.NullInt64
				if err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, moveTo).Scan(&moveParent); err == sql.ErrNoRows {
					http.Error(w, "move_to not found", http.StatusBadRequest)
					return
				} else if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				if moveParent.Valid {
					http.Error(w, "move_to must be a parent", http.StatusBadRequest)
					return
				}
			} else {
				var moveParent sql.NullInt64
				if err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, moveTo).Scan(&moveParent); err == sql.ErrNoRows {
					http.Error(w, "move_to not found", http.StatusBadRequest)
					return
				} else if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				if !moveParent.Valid {
					http.Error(w, "move_to must be a child", http.StatusBadRequest)
					return
				}
			}
			tx, err := db.DB.Begin()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !parentID.Valid {
				if _, err := tx.Exec(`UPDATE entry_categories SET parent_id = $1 WHERE parent_id = $2`, moveTo, id); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
				var entries int
				if err := tx.QueryRow(`SELECT COUNT(*) FROM entry_category_links WHERE category_id = $1`, id).Scan(&entries); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
				var websites int
				if err := tx.QueryRow(`SELECT COUNT(*) FROM website_category_links WHERE category_id = $1`, id).Scan(&websites); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
				if entries > 0 || websites > 0 {
					tx.Rollback()
					http.Error(w, "Előbb helyezd át a bejegyzéseket.", http.StatusConflict)
					return
				}
			} else {
				if err := directory.MoveCategoryLinks(tx, id, moveTo); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
				if _, err := tx.Exec(`
					UPDATE entries SET category_id = $1, cat_name = (SELECT name FROM entry_categories WHERE id = $1)
					WHERE category_id = $2`, moveTo, id); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
				if _, err := tx.Exec(`UPDATE websites SET category_id = $1 WHERE category_id = $2`, moveTo, id); err != nil {
					tx.Rollback()
					http.Error(w, err.Error(), 500)
					return
				}
			}
			if _, err := tx.Exec(`DELETE FROM entry_categories WHERE id = $1`, id); err != nil {
				tx.Rollback()
				http.Error(w, err.Error(), 500)
				return
			}
			if err := tx.Commit(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		var parentID sql.NullInt64
		if err := db.DB.QueryRow(`SELECT parent_id FROM entry_categories WHERE id = $1`, id).Scan(&parentID); err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !parentID.Valid {
			var children int
			if err := db.DB.QueryRow(`SELECT COUNT(*) FROM entry_categories WHERE parent_id = $1`, id).Scan(&children); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if children > 0 {
				http.Error(w, "Előbb helyezd át vagy töröld az alkategóriákat.", http.StatusConflict)
				return
			}
		}
		var entries int
		if err := db.DB.QueryRow(`SELECT COUNT(*) FROM entry_category_links WHERE category_id = $1`, id).Scan(&entries); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var websites int
		if err := db.DB.QueryRow(`SELECT COUNT(*) FROM website_category_links WHERE category_id = $1`, id).Scan(&websites); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if entries > 0 || websites > 0 {
			http.Error(w, "Előbb helyezd át a bejegyzéseket.", http.StatusConflict)
			return
		}
		if _, err := db.DB.Exec(`DELETE FROM entry_categories WHERE id = $1`, id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func HandleAdminEntryTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query("SELECT id, name FROM entry_types ORDER BY name ASC")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.EntryType
		for rows.Next() {
			var et models.EntryType
			if err := rows.Scan(&et.ID, &et.Name); err == nil {
				res = append(res, et)
			}
		}
		if res == nil {
			res = []models.EntryType{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		http.Error(w, "A típuslista zárt.", http.StatusForbidden)
		return

	case "PUT":
		http.Error(w, "A típuslista zárt.", http.StatusForbidden)
		return

	case "DELETE":
		http.Error(w, "A típuslista zárt.", http.StatusForbidden)
		return
	}
}
