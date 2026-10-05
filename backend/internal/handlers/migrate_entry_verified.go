package handlers

import (
	"log"

	"backend/internal/db"
)

// MigrateEntryVerified adds verified / published and backfills verified from claimed (idempotent).
func MigrateEntryVerified() {
	stmts := []string{
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS published BOOLEAN NOT NULL DEFAULT true`,
		`UPDATE entries SET verified = claimed WHERE verified = false AND claimed = true`,
	}
	for _, sql := range stmts {
		if _, err := db.DB.Exec(sql); err != nil {
			log.Printf("MigrateEntryVerified: %v", err)
		}
	}
}
