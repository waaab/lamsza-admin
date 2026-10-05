package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type FavoriteItem struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Slug       *string  `json:"slug"`
	CountySlug *string  `json:"county_slug"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	CreatedAt  string   `json:"created_at"`
}

type favoritesResponse struct {
	Settlements []FavoriteItem `json:"settlements"`
	Attractions []FavoriteItem `json:"attractions"`
	Entries     []FavoriteItem `json:"entries"`
	Events      []FavoriteItem `json:"events"`
}

type favoriteBody struct {
	Type string  `json:"type"`
	ID   jsonInt `json:"id"`
}

type favoriteRow struct {
	Type      string `json:"type"`
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
}

func NormalizeFavorite(entityType string, entityID int) (string, int, error) {
	switch entityType {
	case "settlement", "attraction", "entry", "event":
		if entityID <= 0 {
			return "", 0, fmt.Errorf("invalid id")
		}
		return entityType, entityID, nil
	default:
		return "", 0, fmt.Errorf("invalid type")
	}
}

func migrateUserFavorites() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_favorites (
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			entity_type VARCHAR(20) NOT NULL,
			entity_id INT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, entity_type, entity_id)
		)
	`)
	if err != nil {
		log.Printf("user_favorites table: %v", err)
	}
}

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func scanFavoriteItem(id int, name string, slug, countySlug sql.NullString, lat, lon sql.NullFloat64, createdAt time.Time) FavoriteItem {
	item := FavoriteItem{
		ID:        id,
		Name:      name,
		CreatedAt: formatTimestamp(createdAt),
	}
	if slug.Valid {
		item.Slug = &slug.String
	}
	if countySlug.Valid {
		item.CountySlug = &countySlug.String
	}
	if lat.Valid {
		item.Latitude = &lat.Float64
	}
	if lon.Valid {
		item.Longitude = &lon.Float64
	}
	return item
}

func listFavoriteSettlements(userID int) ([]FavoriteItem, error) {
	rows, err := db.DB.Query(`
		SELECT s.id, s.name, s.slug, c.slug, gl.latitude, gl.longitude, uf.created_at
		FROM user_favorites uf
		LEFT JOIN settlements s ON uf.entity_id = s.id
		LEFT JOIN counties c ON s.county_id = c.id
		LEFT JOIN geo_locations gl ON s.location_id = gl.id
		WHERE uf.user_id = $1 AND uf.entity_type = 'settlement' AND s.id IS NOT NULL
		ORDER BY uf.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FavoriteItem
	for rows.Next() {
		var id int
		var name string
		var slug, countySlug sql.NullString
		var lat, lon sql.NullFloat64
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &countySlug, &lat, &lon, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, scanFavoriteItem(id, name, slug, countySlug, lat, lon, createdAt))
	}
	if items == nil {
		items = []FavoriteItem{}
	}
	return items, nil
}

func listFavoriteAttractions(userID int) ([]FavoriteItem, error) {
	rows, err := db.DB.Query(`
		SELECT a.id, a.name, a.slug, c.slug, gl.latitude, gl.longitude, uf.created_at
		FROM user_favorites uf
		LEFT JOIN attractions a ON uf.entity_id = a.id
		LEFT JOIN counties c ON a.county_id = c.id
		LEFT JOIN geo_locations gl ON a.location_id = gl.id
		WHERE uf.user_id = $1 AND uf.entity_type = 'attraction' AND a.id IS NOT NULL
		ORDER BY uf.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FavoriteItem
	for rows.Next() {
		var id int
		var name string
		var slug, countySlug sql.NullString
		var lat, lon sql.NullFloat64
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &countySlug, &lat, &lon, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, scanFavoriteItem(id, name, slug, countySlug, lat, lon, createdAt))
	}
	if items == nil {
		items = []FavoriteItem{}
	}
	return items, nil
}

func listFavoriteEntries(userID int) ([]FavoriteItem, error) {
	rows, err := db.DB.Query(`
		SELECT e.id, e.name, e.slug, c.slug, uf.created_at
		FROM user_favorites uf
		LEFT JOIN entries e ON uf.entity_id = e.id
		LEFT JOIN settlements s ON e.location_id = s.id
		LEFT JOIN counties c ON s.county_id = c.id
		WHERE uf.user_id = $1 AND uf.entity_type = 'entry' AND e.id IS NOT NULL
		ORDER BY uf.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FavoriteItem
	for rows.Next() {
		var id int
		var name string
		var slug, countySlug sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &countySlug, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, scanFavoriteItem(id, name, slug, countySlug, sql.NullFloat64{}, sql.NullFloat64{}, createdAt))
	}
	if items == nil {
		items = []FavoriteItem{}
	}
	return items, nil
}

func listFavoriteEvents(userID int) ([]FavoriteItem, error) {
	rows, err := db.DB.Query(`
		SELECT ev.id, ev.title, uf.created_at
		FROM user_favorites uf
		LEFT JOIN events ev ON uf.entity_id = ev.id
		WHERE uf.user_id = $1 AND uf.entity_type = 'event' AND ev.id IS NOT NULL
		ORDER BY uf.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FavoriteItem
	for rows.Next() {
		var id int
		var title string
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, FavoriteItem{
			ID:        id,
			Name:      title,
			CreatedAt: formatTimestamp(createdAt),
		})
	}
	if items == nil {
		items = []FavoriteItem{}
	}
	return items, nil
}

func listFavorites(userID int) (favoritesResponse, error) {
	settlements, err := listFavoriteSettlements(userID)
	if err != nil {
		return favoritesResponse{}, err
	}
	attractions, err := listFavoriteAttractions(userID)
	if err != nil {
		return favoritesResponse{}, err
	}
	entries, err := listFavoriteEntries(userID)
	if err != nil {
		return favoritesResponse{}, err
	}
	events, err := listFavoriteEvents(userID)
	if err != nil {
		return favoritesResponse{}, err
	}
	return favoritesResponse{
		Settlements: settlements,
		Attractions: attractions,
		Entries:     entries,
		Events:      events,
	}, nil
}

func favoriteTargetExists(entityType string, entityID int) (bool, error) {
	var table string
	switch entityType {
	case "settlement":
		table = "settlements"
	case "attraction":
		table = "attractions"
	case "entry":
		table = "entries"
	case "event":
		table = "events"
	default:
		return false, fmt.Errorf("invalid type")
	}
	var id int
	err := db.DB.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE id = $1", table), entityID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func selectFavoriteRow(userID int, entityType string, entityID int) (favoriteRow, error) {
	var row favoriteRow
	var createdAt time.Time
	err := db.DB.QueryRow(`
		SELECT entity_type, entity_id, created_at
		FROM user_favorites
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID).Scan(&row.Type, &row.ID, &createdAt)
	if err != nil {
		return favoriteRow{}, err
	}
	row.CreatedAt = formatTimestamp(createdAt)
	return row, nil
}

func writeFavorites(w http.ResponseWriter, payload favoritesResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func HandleFavorites(w http.ResponseWriter, r *http.Request) {
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		payload, err := listFavorites(u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeFavorites(w, payload)

	case http.MethodPost:
		var body favoriteBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		entityType, entityID, err := NormalizeFavorite(body.Type, int(body.ID))
		if err != nil {
			http.Error(w, "invalid favorite", http.StatusBadRequest)
			return
		}
		exists, err := favoriteTargetExists(entityType, entityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "target not found", http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec(`
			INSERT INTO user_favorites (user_id, entity_type, entity_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, u.ID, entityType, entityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		row, err := selectFavoriteRow(u.ID, entityType, entityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(row)

	case http.MethodDelete:
		typeStr := r.URL.Query().Get("type")
		idStr := r.URL.Query().Get("id")
		if typeStr == "" || idStr == "" {
			http.Error(w, "type and id required", http.StatusBadRequest)
			return
		}
		entityID, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		entityType, entityID, err := NormalizeFavorite(typeStr, entityID)
		if err != nil {
			http.Error(w, "invalid favorite", http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec(`
			DELETE FROM user_favorites
			WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
		`, u.ID, entityType, entityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeOK(w)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
