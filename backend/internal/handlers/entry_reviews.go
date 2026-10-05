package handlers

import (
	"backend/internal/auth"
	"backend/internal/db"
	"backend/internal/models"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type reviewPostBody struct {
	Slug  string `json:"slug"`
	Score int    `json:"score"`
	Text  string `json:"text"`
}

func HandleEntryReviews(w http.ResponseWriter, r *http.Request) {
	user, err := auth.UserFromRequest(r)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var slug string
	if r.Method == http.MethodPost {
		var body reviewPostBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		slug = body.Slug

		if body.Score < 1 || body.Score > 5 {
			http.Error(w, "score must be 1-5", http.StatusBadRequest)
			return
		}

		sanitized := sanitizeReviewText(body.Text)
		if utf8.RuneCountInString(sanitized) > 2000 {
			http.Error(w, "text too long", http.StatusBadRequest)
			return
		}

		entryID, claimed, ratingsEnabled, published, isMember, err := loadEntryForReview(slug, user.ID)
		if err != nil {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		if !published && !isMember {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		if !claimed || !ratingsEnabled {
			http.Error(w, "reviews not enabled", http.StatusForbidden)
			return
		}

		_, err = db.DB.Exec(`
			INSERT INTO entry_reviews (entry_id, user_id, score, body)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (entry_id, user_id)
			DO UPDATE SET score = EXCLUDED.score, body = EXCLUDED.body, updated_at = NOW()
		`, entryID, user.ID, body.Score, sanitized)
		if err != nil {
			http.Error(w, "failed to save review", http.StatusInternalServerError)
			return
		}

		writeReviewSummary(w, entryID, user.ID)
		return
	}

	if r.Method == http.MethodDelete {
		slug = r.URL.Query().Get("slug")
		if slug == "" {
			http.Error(w, "missing slug", http.StatusBadRequest)
			return
		}

		entryID, claimed, ratingsEnabled, published, isMember, err := loadEntryForReview(slug, user.ID)
		if err != nil {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		if !published && !isMember {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		if !claimed || !ratingsEnabled {
			http.Error(w, "reviews not enabled", http.StatusForbidden)
			return
		}

		_, err = db.DB.Exec(`
			DELETE FROM entry_reviews
			WHERE entry_id = $1 AND user_id = $2
		`, entryID, user.ID)
		if err != nil {
			http.Error(w, "failed to delete review", http.StatusInternalServerError)
			return
		}

		writeReviewSummary(w, entryID, user.ID)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func loadEntryForReview(slug string, userID int) (entryID int, claimed, ratingsEnabled, published, isMember bool, err error) {
	err = db.DB.QueryRow(`
		SELECT 
			e.id,
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active'),
			COALESCE(e.ratings_enabled, false),
			e.published,
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.user_id = $2 AND m.status = 'active')
		FROM entries e
		WHERE e.slug = $1
	`, slug, userID).Scan(&entryID, &claimed, &ratingsEnabled, &published, &isMember)
	return
}

func sanitizeReviewText(text string) string {
	text = strings.TrimSpace(text)

	// Strip HTML tags
	htmlTag := regexp.MustCompile(`<[^>]*>`)
	text = htmlTag.ReplaceAllString(text, "")

	// Remove javascript: protocol
	jsProtocol := regexp.MustCompile(`(?i)javascript:`)
	text = jsProtocol.ReplaceAllString(text, "")

	return text
}

func writeReviewSummary(w http.ResponseWriter, entryID, viewerUserID int) {
	// Load the full entry data needed for ApplyPublicEntryExtras
	var e models.Entry
	var hours, delivery, photos []byte
	var idInt int
	err := db.DB.QueryRow(`
		SELECT 
			e.id,
			COALESCE(e.hours, '{}'::jsonb),
			COALESCE(e.delivery_hours, '{}'::jsonb),
			COALESCE(e.photos, '[]'::jsonb),
			COALESCE(e.ratings_enabled, false),
			EXISTS (SELECT 1 FROM entry_members m WHERE m.entry_id = e.id AND m.role = 'owner' AND m.status = 'active')
		FROM entries e
		WHERE e.id = $1
	`, entryID).Scan(&idInt, &hours, &delivery, &photos, &e.RatingsEnabled, &e.Claimed)
	
	if err != nil {
		http.Error(w, "entry not found", http.StatusNotFound)
		return
	}

	e.ID = strconv.Itoa(idInt)
	e.Hours = hours
	e.DeliveryHours = delivery
	e.Photos = photos

	ApplyPublicEntryExtras(&e, viewerUserID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"my_review":    e.MyReview,
		"rating":       e.Rating,
		"review_count": e.ReviewCount,
		"reviews":      e.Reviews,
	})
}
