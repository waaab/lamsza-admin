package account

import (
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"net/http"
)

type queueUnpublished struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	OwnerEmail string `json:"owner_email"`
}

type queueMember struct {
	EntryID   int    `json:"entry_id"`
	EntryName string `json:"entry_name"`
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
}

type queueWebsite struct {
	ID          int    `json:"id"`
	Domain      string `json:"domain"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Submitter   string `json:"submitter"`
}

type queueClaim struct {
	EntryID   int    `json:"entry_id"`
	EntryName string `json:"entry_name"`
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
}

type queueSuggestion struct {
	ID        int             `json:"id"`
	EntryID   int             `json:"entry_id"`
	EntryName string          `json:"entry_name"`
	Email     string          `json:"email"`
	Changes   json.RawMessage `json:"changes"`
	Note      string          `json:"note"`
}

type queueResponse struct {
	Unpublished []queueUnpublished `json:"unpublished"`
	Members     []queueMember      `json:"members"`
	Claims      []queueClaim       `json:"claims"`
	Suggestions []queueSuggestion  `json:"suggestions"`
	Websites    []queueWebsite     `json:"websites"`
}

func HandleListingQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := queueResponse{
		Unpublished: []queueUnpublished{},
		Members:     []queueMember{},
		Claims:      []queueClaim{},
		Suggestions: []queueSuggestion{},
		Websites:    []queueWebsite{},
	}

	rows, err := db.DB.Query(`
		SELECT e.id, e.name, COALESCE(e.slug, ''), COALESCE(u.email, '')
		FROM entries e
		LEFT JOIN entry_members m ON m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'
		LEFT JOIN users u ON u.id = m.user_id
		WHERE e.published = false
		ORDER BY e.name ASC, e.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item queueUnpublished
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.OwnerEmail); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Unpublished = append(resp.Unpublished, item)
	}

	rows, err = db.DB.Query(`
		SELECT m.entry_id, e.name, m.user_id, u.email
		FROM entry_members m
		JOIN entries e ON e.id = m.entry_id
		JOIN users u ON u.id = m.user_id
		WHERE m.status = 'pending' AND m.role = 'member'
		ORDER BY e.name ASC, m.user_id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item queueMember
		if err := rows.Scan(&item.EntryID, &item.EntryName, &item.UserID, &item.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Members = append(resp.Members, item)
	}

	rows, err = db.DB.Query(`
		SELECT m.entry_id, e.name, m.user_id, u.email
		FROM entry_members m
		JOIN entries e ON e.id = m.entry_id
		JOIN users u ON u.id = m.user_id
		WHERE m.status = 'pending' AND m.role = 'owner'
		ORDER BY e.name ASC, m.user_id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item queueClaim
		if err := rows.Scan(&item.EntryID, &item.EntryName, &item.UserID, &item.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Claims = append(resp.Claims, item)
	}

	rows, err = db.DB.Query(`
		SELECT s.id, s.entry_id, e.name, COALESCE(u.email, ''), s.changes, COALESCE(s.note, '')
		FROM entry_suggestions s
		JOIN entries e ON e.id = s.entry_id
		JOIN users u ON u.id = s.user_id
		WHERE s.status = 'open'
		ORDER BY e.name ASC, s.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item queueSuggestion
		if err := rows.Scan(&item.ID, &item.EntryID, &item.EntryName, &item.Email, &item.Changes, &item.Note); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Suggestions = append(resp.Suggestions, item)
	}

	rows, err = db.DB.Query(`
		SELECT w.id, w.domain_key, w.title, w.description, COALESCE(u.email, '')
		FROM websites w
		LEFT JOIN users u ON u.id = w.user_id
		WHERE w.status = 'pending'
		ORDER BY w.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item queueWebsite
		if err := rows.Scan(&item.ID, &item.Domain, &item.Title, &item.Description, &item.Submitter); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Websites = append(resp.Websites, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type publishBody struct {
	EntryID int `json:"entry_id"`
}

func HandleListingQueuePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body publishBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.EntryID <= 0 {
		http.Error(w, "entry_id required", http.StatusBadRequest)
		return
	}

	res, err := db.DB.Exec(`UPDATE entries SET published = true WHERE id = $1 AND published = false`, body.EntryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists bool
		err = db.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM entries WHERE id = $1)`, body.EntryID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

type memberActionBody struct {
	EntryID int    `json:"entry_id"`
	UserID  int    `json:"user_id"`
	Action  string `json:"action"`
}

func HandleListingQueueMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body memberActionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.EntryID <= 0 || body.UserID <= 0 {
		http.Error(w, "entry_id and user_id required", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "approve":
		res, err := db.DB.Exec(`
			UPDATE entry_members SET status = 'active'
			WHERE entry_id = $1 AND user_id = $2 AND role = 'member' AND status = 'pending'
		`, body.EntryID, body.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	case "reject":
		res, err := db.DB.Exec(`
			DELETE FROM entry_members
			WHERE entry_id = $1 AND user_id = $2 AND role = 'member' AND status = 'pending'
		`, body.EntryID, body.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	default:
		http.Error(w, "action must be approve or reject", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandleListingQueueClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body memberActionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.EntryID <= 0 || body.UserID <= 0 {
		http.Error(w, "entry_id and user_id required", http.StatusBadRequest)
		return
	}
	if body.Action != "accept" && body.Action != "deny" {
		http.Error(w, "action must be accept or deny", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var lockedID int
	err = tx.QueryRow(`SELECT id FROM entries WHERE id = $1 FOR UPDATE`, body.EntryID).Scan(&lockedID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if body.Action == "accept" {
		var activeOwner bool
		err = tx.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM entry_members
				WHERE entry_id = $1 AND role = 'owner' AND status = 'active'
			)
		`, body.EntryID).Scan(&activeOwner)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if activeOwner {
			http.Error(w, "already claimed", http.StatusConflict)
			return
		}
		res, err := tx.Exec(`
			UPDATE entry_members SET status = 'active'
			WHERE entry_id = $1 AND user_id = $2 AND role = 'owner' AND status = 'pending'
		`, body.EntryID, body.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	} else {
		res, err := tx.Exec(`
			DELETE FROM entry_members
			WHERE entry_id = $1 AND user_id = $2 AND role = 'owner' AND status = 'pending'
		`, body.EntryID, body.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
