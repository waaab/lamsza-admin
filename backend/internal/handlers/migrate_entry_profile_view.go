package handlers

import (
	"backend/internal/db"
	"database/sql"
	"log"
	"os"
)

const entryProfileHoursBackfillKey = "entry_profile_hours_enabled_backfill_v1"

// MigrateEntryProfileView adds hours/delivery switches and social_links (idempotent).
func MigrateEntryProfileView() {
	sqlBytes, err := os.ReadFile("migrations/entry_profile_view.sql")
	if err != nil {
		log.Printf("MigrateEntryProfileView (read sql): %v", err)
		return
	}
	if _, err := db.DB.Exec(string(sqlBytes)); err != nil {
		log.Printf("MigrateEntryProfileView: %v", err)
	}
	backfillEntryProfileHoursEnabledOnce()
}

func backfillEntryProfileHoursEnabledOnce() {
	_, _ = db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS site_settings (
			key VARCHAR(100) PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)

	var marker string
	err := db.DB.QueryRow(`SELECT value FROM site_settings WHERE key = $1`, entryProfileHoursBackfillKey).Scan(&marker)
	if err == nil && marker == "done" {
		return
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("MigrateEntryProfileView (backfill marker): %v", err)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE entries
		SET hours_enabled = true
		WHERE hours_enabled = false
		  AND EXISTS (
			SELECT 1
			FROM jsonb_each(hours) AS day(key, val)
			WHERE COALESCE((val->>'closed')::boolean, false) = true
			   OR (
				 NULLIF(BTRIM(val->>'open'), '') IS NOT NULL
				 AND NULLIF(BTRIM(val->>'close'), '') IS NOT NULL
			   )
		  )
	`)
	if err != nil {
		log.Printf("MigrateEntryProfileView (backfill): %v", err)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO site_settings (key, value, updated_at)
		VALUES ($1, 'done', CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, entryProfileHoursBackfillKey)
	if err != nil {
		log.Printf("MigrateEntryProfileView (backfill marker write): %v", err)
	}
}
