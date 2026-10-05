package handlers

import (
	"backend/internal/db"
	"log"
)

func MigrateEntrySuggestions() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS entry_suggestions (
			id SERIAL PRIMARY KEY,
			entry_id INT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			changes JSONB NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'denied')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS entry_suggestions_one_open
			ON entry_suggestions (entry_id)
			WHERE status = 'open'`,
	}
	for _, sql := range stmts {
		if _, err := db.DB.Exec(sql); err != nil {
			log.Printf("MigrateEntrySuggestions: %v", err)
		}
	}
}
