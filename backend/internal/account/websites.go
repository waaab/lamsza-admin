package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"backend/internal/directory"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// WebsiteHit is a public website search result.
type WebsiteHit struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	URL         string `json:"url"`
	Claimed     bool   `json:"claimed"`
}

type websiteListItem struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Domain      string   `json:"domain"`
	URL         string   `json:"url"`
	Claimed     bool     `json:"claimed"`
	CategoryID  int      `json:"category_id"`
	Category    string   `json:"category"`
	Categories  []string `json:"categories"`
	EntryID     int      `json:"entry_id"`
	EntrySlug   string   `json:"entry_slug,omitempty"`
}

type websitesListResponse struct {
	Websites []websiteListItem `json:"websites"`
}

type submitWebsiteBody struct {
	Domain      string `json:"domain"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CategoryID  int    `json:"category_id"`
	CategoryIDs []int  `json:"category_ids"`
}

type submitWebsiteResponse struct {
	ID          int    `json:"id"`
	Domain      string `json:"domain"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type accountWebsiteItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	Claimed     bool   `json:"claimed"`
}

type websiteLookup struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	Claimed     bool   `json:"claimed"`
	EntryID     int    `json:"entry_id"`
	Published   bool   `json:"published"`
	Membership  string `json:"membership"`
}

func writeWebsiteFieldError(w http.ResponseWriter, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": "invalid", "field": field})
}

type adminWebsiteBody struct {
	ID          int    `json:"id"`
	Action      string `json:"action"`
	CategoryID  int    `json:"category_id"`
	CategoryIDs []int  `json:"category_ids"`
}

func HandleAdminWebsite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminWebsitesList(w, r)
	case http.MethodPost:
		handleAdminWebsiteAction(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAdminWebsitesList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT w.id, w.domain_key, w.title, w.description, w.status,
			COALESCE(u.email, ''),
			w.created_at,
			w.approved_at,
			COALESCE(approver.email, ''),
			EXISTS (
				SELECT 1 FROM entry_members m
				WHERE m.entry_id = w.entry_id AND m.role = 'owner' AND m.status = 'active'
			)
		FROM websites w
		LEFT JOIN users u ON u.id = w.user_id
		LEFT JOIN users approver ON approver.id = w.approved_by
		ORDER BY w.title ASC, w.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type adminWebsiteRow struct {
		ID          int    `json:"id"`
		Domain      string `json:"domain"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Submitter   string `json:"submitter"`
		SubmittedAt string `json:"submitted_at"`
		ApprovedAt  string `json:"approved_at"`
		Approver    string `json:"approver"`
		Claimed     bool   `json:"claimed"`
		URL         string `json:"url"`
	}
	list := []adminWebsiteRow{}
	for rows.Next() {
		var item adminWebsiteRow
		var submittedAt, approvedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.Domain, &item.Title, &item.Description, &item.Status,
			&item.Submitter, &submittedAt, &approvedAt, &item.Approver, &item.Claimed,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		item.URL = "https://" + item.Domain
		if submittedAt.Valid {
			item.SubmittedAt = submittedAt.Time.Format(time.RFC3339)
		}
		if approvedAt.Valid {
			item.ApprovedAt = approvedAt.Time.Format(time.RFC3339)
		}
		list = append(list, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"websites": list})
}

func handleAdminWebsiteAction(w http.ResponseWriter, r *http.Request) {

	var body adminWebsiteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.ID <= 0 {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "approve":
		adminUser := auth.UserFromContext(r.Context())
		if adminUser == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if body.CategoryID > 0 {
			categoryIDs, err := directory.NormalizeCategoryIDs(body.CategoryID, body.CategoryIDs)
			if err != nil {
				writeWebsiteFieldError(w, "category_id")
				return
			}
			if _, err = directory.ValidateLeaves(db.DB, categoryIDs); err != nil {
				writeWebsiteFieldError(w, "category_id")
				return
			}
			res, err := db.DB.Exec(`
				UPDATE websites
				SET status = 'approved', approved_at = NOW(), approved_by = $2, category_id = $3
				WHERE id = $1 AND status = 'pending'
			`, body.ID, adminUser.ID, body.CategoryID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if err = directory.ReplaceWebsiteCategories(db.DB, body.ID, categoryIDs); err != nil {
				writeWebsiteFieldError(w, "category_id")
				return
			}
		} else {
			res, err := db.DB.Exec(`
				UPDATE websites
				SET status = 'approved', approved_at = NOW(), approved_by = $2
				WHERE id = $1 AND status = 'pending'
			`, body.ID, adminUser.ID)
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
	case "reject":
		res, err := db.DB.Exec(`
			DELETE FROM websites WHERE id = $1 AND status = 'pending'
		`, body.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	case "ban":
		tx, err := db.DB.Begin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var userID sql.NullInt64
		err = tx.QueryRow(`
			SELECT user_id FROM websites WHERE id = $1 AND status = 'pending'
		`, body.ID).Scan(&userID)
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !userID.Valid {
			http.Error(w, "user required", http.StatusBadRequest)
			return
		}

		res, err := tx.Exec(`DELETE FROM websites WHERE id = $1 AND status = 'pending'`, body.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		_, err = tx.Exec(`UPDATE users SET website_banned = true WHERE id = $1`, userID.Int64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "action must be approve, reject, or ban", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
