package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const historyStoreMax = 12

type HistoryItem struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}

type historyBody struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}

func NormalizeHistoryItem(slug, name, category, location, photo string) (HistoryItem, error) {
	slug = strings.TrimSpace(slug)
	name = strings.TrimSpace(name)
	if slug == "" {
		return HistoryItem{}, fmt.Errorf("slug required")
	}
	if name == "" {
		return HistoryItem{}, fmt.Errorf("name required")
	}
	return HistoryItem{
		Slug:     slug,
		Name:     name,
		Category: strings.TrimSpace(category),
		Location: strings.TrimSpace(location),
		Photo:    strings.TrimSpace(photo),
	}, nil
}

func migrateUserHistory() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_entry_history (
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			slug VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			category VARCHAR(255) NOT NULL DEFAULT '',
			location VARCHAR(255) NOT NULL DEFAULT '',
			photo TEXT NOT NULL DEFAULT '',
			viewed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, slug)
		)
	`)
	if err != nil {
		log.Printf("user_entry_history table: %v", err)
	}
}

func listUserHistory(userID int) ([]HistoryItem, error) {
	rows, err := db.DB.Query(`
		SELECT slug, name, category, location, photo
		FROM user_entry_history
		WHERE user_id = $1
		ORDER BY viewed_at DESC
		LIMIT $2
	`, userID, historyStoreMax)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		if err := rows.Scan(&item.Slug, &item.Name, &item.Category, &item.Location, &item.Photo); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []HistoryItem{}
	}
	return items, nil
}

func writeUserHistory(w http.ResponseWriter, items []HistoryItem) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func trimHistoryBeyondCap(execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, userID int) error {
	_, err := execer.Exec(`
		DELETE FROM user_entry_history
		WHERE user_id = $1 AND slug NOT IN (
			SELECT slug FROM user_entry_history WHERE user_id = $1 ORDER BY viewed_at DESC LIMIT $2
		)
	`, userID, historyStoreMax)
	return err
}

func upsertHistory(execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, userID int, item HistoryItem) error {
	_, err := execer.Exec(`
		INSERT INTO user_entry_history (user_id, slug, name, category, location, photo, viewed_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, slug) DO UPDATE SET
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			location = EXCLUDED.location,
			photo = EXCLUDED.photo,
			viewed_at = NOW()
	`, userID, item.Slug, item.Name, item.Category, item.Location, item.Photo)
	if err != nil {
		return err
	}
	return trimHistoryBeyondCap(execer, userID)
}

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := listUserHistory(u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeUserHistory(w, items)

	case http.MethodPost:
		var body historyBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		item, err := NormalizeHistoryItem(body.Slug, body.Name, body.Category, body.Location, body.Photo)
		if err != nil {
			http.Error(w, "invalid history item", http.StatusBadRequest)
			return
		}
		if err := upsertHistory(db.DB, u.ID, item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items, err := listUserHistory(u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeUserHistory(w, items)

	case http.MethodDelete:
		slug := strings.TrimSpace(r.URL.Query().Get("slug"))
		if slug == "" {
			_, err := db.DB.Exec(`DELETE FROM user_entry_history WHERE user_id = $1`, u.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			_, err := db.DB.Exec(`DELETE FROM user_entry_history WHERE user_id = $1 AND slug = $2`, u.ID, slug)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeOK(w)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type importHistoryItem struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}

func insertImportHistory(tx *sql.Tx, userID int, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload []importHistoryItem
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	inserted := 0
	seen := make(map[string]bool)
	for i, row := range payload {
		if inserted >= historyStoreMax {
			break
		}
		item, err := NormalizeHistoryItem(row.Slug, row.Name, row.Category, row.Location, row.Photo)
		if err != nil {
			continue
		}
		if seen[item.Slug] {
			continue
		}
		seen[item.Slug] = true
		if _, err := tx.Exec(`
			INSERT INTO user_entry_history (user_id, slug, name, category, location, photo, viewed_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW() - ($7 * INTERVAL '1 second'))
		`, userID, item.Slug, item.Name, item.Category, item.Location, item.Photo, i); err != nil {
			return err
		}
		inserted++
	}
	return nil
}
