package handlers

import (
	"encoding/json"
	"log"
	"strings"

	"backend/internal/db"
)

// MigrateEntryAvailability adds claimed / hours / delivery_hours / photos (idempotent).
func MigrateEntryAvailability() {
	stmts := []string{
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS claimed BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS hours JSONB NOT NULL DEFAULT '{}'::jsonb`,
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS delivery_hours JSONB NOT NULL DEFAULT '{}'::jsonb`,
		`ALTER TABLE entries ADD COLUMN IF NOT EXISTS photos JSONB NOT NULL DEFAULT '[]'::jsonb`,
	}
	for _, sql := range stmts {
		if _, err := db.DB.Exec(sql); err != nil {
			log.Printf("MigrateEntryAvailability: %v", err)
		}
	}
}

func jsonObjectOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`{}`)
	}
	if !json.Valid(b) {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

func jsonArrayOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`[]`)
	}
	if !json.Valid(b) || !strings.HasPrefix(s, "[") {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}
