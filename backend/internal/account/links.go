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
	"strings"
)

const defaultLinkColor = "#e6f0ff"

type UserLink struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	BgColor  string `json:"bg_color"`
	Position int    `json:"position"`
}

type linkBody struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	BgColor string `json:"bg_color"`
}

type linksPutBody struct {
	Links []UserLink `json:"links"`
}

func NormalizeLink(title, url, color string) (string, string, string, error) {
	title = strings.TrimSpace(title)
	url = strings.TrimSpace(url)
	color = strings.TrimSpace(color)
	if title == "" {
		return "", "", "", fmt.Errorf("title required")
	}
	if url == "" {
		return "", "", "", fmt.Errorf("url required")
	}
	if color == "" {
		color = defaultLinkColor
	}
	return title, url, color, nil
}

func migrateUserLinks() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_links (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(100) NOT NULL,
			url VARCHAR(255) NOT NULL,
			bg_color VARCHAR(50) NOT NULL DEFAULT '#e6f0ff',
			position INT NOT NULL
		)
	`)
	if err != nil {
		log.Printf("user_links table: %v", err)
	}
}

func listUserLinks(userID int) ([]UserLink, error) {
	rows, err := db.DB.Query(`
		SELECT id, title, url, bg_color, position
		FROM user_links
		WHERE user_id = $1
		ORDER BY position, id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []UserLink
	for rows.Next() {
		var l UserLink
		if err := rows.Scan(&l.ID, &l.Title, &l.URL, &l.BgColor, &l.Position); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	if links == nil {
		links = []UserLink{}
	}
	return links, nil
}

func writeUserLinks(w http.ResponseWriter, links []UserLink) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}

func HandleLinks(w http.ResponseWriter, r *http.Request) {
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		links, err := listUserLinks(u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeUserLinks(w, links)

	case http.MethodPost:
		var body linkBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		title, url, color, err := NormalizeLink(body.Title, body.URL, body.BgColor)
		if err != nil {
			http.Error(w, "invalid link", http.StatusBadRequest)
			return
		}
		var maxPos int
		err = db.DB.QueryRow(`
			SELECT COALESCE(MAX(position), -1) FROM user_links WHERE user_id = $1
		`, u.ID).Scan(&maxPos)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var link UserLink
		err = db.DB.QueryRow(`
			INSERT INTO user_links (user_id, title, url, bg_color, position)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, title, url, bg_color, position
		`, u.ID, title, url, color, maxPos+1).Scan(&link.ID, &link.Title, &link.URL, &link.BgColor, &link.Position)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(link)

	case http.MethodPut:
		var body linksPutBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		for i, item := range body.Links {
			title, url, color, err := NormalizeLink(item.Title, item.URL, item.BgColor)
			if err != nil {
				http.Error(w, "invalid link", http.StatusBadRequest)
				return
			}
			_, err = db.DB.Exec(`
				UPDATE user_links
				SET title = $1, url = $2, bg_color = $3, position = $4
				WHERE id = $5 AND user_id = $6
			`, title, url, color, i, item.ID, u.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		links, err := listUserLinks(u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeUserLinks(w, links)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec(`DELETE FROM user_links WHERE id = $1 AND user_id = $2`, id, u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeOK(w)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type importLink struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	BgColor string `json:"bg_color"`
}

func insertImportLinks(tx *sql.Tx, userID int, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload []importLink
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	for i, item := range payload {
		title, url, color, err := NormalizeLink(item.Title, item.URL, item.BgColor)
		if err != nil {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO user_links (user_id, title, url, bg_color, position)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, title, url, color, i); err != nil {
			return err
		}
	}
	return nil
}
