package handlers

import (
	"backend/internal/db"
	"backend/internal/models"
	"encoding/json"
	"strconv"
)

// ApplyPublicEntryExtras enforces public listing vs claimed-extras on an Entry.
// Unclaimed public listings must not expose photos, hours, or ratings even if admin data exists.
// Claimed listings keep stored extras and gain ratings_enabled from the entry struct.
// ratings_enabled is pre-loaded by the entry SELECT in public.go (list + detail queries).
func ApplyPublicEntryExtras(e *models.Entry, viewerUserID int) {
	ApplyPublicEntryExtrasMode(e, viewerUserID, false)
}

// ApplyPublicEntryExtrasMode enforces public listing vs claimed-extras with optional lite mode.
// When liteMode is true, only loads rating and review_count (no reviews, no my_review).
// Use lite mode for list/related views to avoid loading 50 reviews per card.
func ApplyPublicEntryExtrasMode(e *models.Entry, viewerUserID int, liteMode bool) {
	if !e.Claimed || !e.Verified {
		e.Photos = json.RawMessage("[]")
	}
	if !e.Claimed {
		e.Phone = ""
		e.SocialLinks = json.RawMessage("[]")
		e.RatingsEnabled = false
	}
	if !e.HoursEnabled {
		e.Hours = json.RawMessage("{}")
	}
	if !e.DeliveryEnabled {
		e.DeliveryHours = json.RawMessage("{}")
	}
	if !e.Claimed {
		return
	}

	if !e.RatingsEnabled {
		return
	}

	entryIDInt, _ := strconv.Atoi(e.ID)

	// Load rating and review count
	var avgRating *float64
	var reviewCount int
	err := db.DB.QueryRow(`
		SELECT 
			ROUND(AVG(score)::numeric, 1),
			COUNT(*)
		FROM entry_reviews
		WHERE entry_id = $1
	`, entryIDInt).Scan(&avgRating, &reviewCount)
	if err == nil {
		e.Rating = avgRating
		e.ReviewCount = reviewCount
	}

	if liteMode {
		if reviewCount > 0 {
			var body string
			err = db.DB.QueryRow(`
				SELECT COALESCE(body, '')
				FROM entry_reviews
				WHERE entry_id = $1 AND btrim(body) <> ''
				ORDER BY created_at DESC
				LIMIT 1
			`, entryIDInt).Scan(&body)
			if err == nil {
				e.FeaturedReview = body
			}
		}
		return
	}

	// Load up to 50 reviews, newest first
	rows, err := db.DB.Query(`
		SELECT 
			r.id,
			COALESCE(NULLIF(u.name, ''), 'Felhasználó'),
			COALESCE(u.picture, ''),
			r.score,
			r.body,
			r.created_at,
			r.updated_at
		FROM entry_reviews r
		JOIN users u ON u.id = r.user_id
		WHERE r.entry_id = $1
		ORDER BY r.created_at DESC
		LIMIT 50
	`, entryIDInt)
	reviews := []models.PublicReview{}
	if err != nil {
		e.Reviews = &reviews
		return
	}
	defer rows.Close()

	for rows.Next() {
		var r models.PublicReview
		var id int
		if err := rows.Scan(&id, &r.AuthorName, &r.AuthorPhoto, &r.Score, &r.Text, &r.CreatedAt, &r.UpdatedAt); err == nil {
			r.ID = strconv.Itoa(id)
			reviews = append(reviews, r)
		}
	}
	e.Reviews = &reviews

	// Load my_review if viewerUserID > 0
	if viewerUserID > 0 {
		var r models.PublicReview
		var id int
		err = db.DB.QueryRow(`
			SELECT 
				r.id,
				COALESCE(NULLIF(u.name, ''), 'Felhasználó'),
				COALESCE(u.picture, ''),
				r.score,
				r.body,
				r.created_at,
				r.updated_at
			FROM entry_reviews r
			JOIN users u ON u.id = r.user_id
			WHERE r.entry_id = $1 AND r.user_id = $2
		`, entryIDInt, viewerUserID).Scan(&id, &r.AuthorName, &r.AuthorPhoto, &r.Score, &r.Text, &r.CreatedAt, &r.UpdatedAt)
		if err == nil {
			r.ID = strconv.Itoa(id)
			e.MyReview = &r
		}
	}
}
