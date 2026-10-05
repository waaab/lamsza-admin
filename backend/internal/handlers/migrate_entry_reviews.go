package handlers

import (
	"backend/internal/db"
	"log"
)

func MigrateEntryReviews() {
	stmts := []string{
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS ratings_enabled BOOLEAN NOT NULL DEFAULT false`,
		`CREATE TABLE IF NOT EXISTS entry_reviews (
			id SERIAL PRIMARY KEY,
			entry_id INT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			score SMALLINT NOT NULL CHECK (score >= 1 AND score <= 5),
			body TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (entry_id, user_id)
		)`,
	}
	for _, sql := range stmts {
		if _, err := db.DB.Exec(sql); err != nil {
			log.Printf("MigrateEntryReviews: %v", err)
		}
	}
}
