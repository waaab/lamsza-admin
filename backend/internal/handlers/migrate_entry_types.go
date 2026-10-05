package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// MigrateEntryTypes is a legacy migrator. It no-ops once directory_catalog_v2 is set.
func MigrateEntryTypes() {
	var flag string
	err := db.DB.QueryRow(`SELECT value FROM site_settings WHERE key = $1`, directoryCatalogV2Key).Scan(&flag)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("MigrateEntryTypes (flag check): %v", err)
		return
	}
	if err == nil && flag == "1" {
		log.Println("skipped: directory_catalog_v2")
		return
	}
}

// resolveEntryTypeID maps a client type name to a catalog row.
func resolveEntryTypeID(typeName string) (int, string, error) {
	name := utils.CanonicalEntryType(typeName)
	if name == "" {
		trimmed := strings.TrimSpace(typeName)
		if trimmed != "" {
			var id int
			err := db.DB.QueryRow(`SELECT id FROM entry_types WHERE name = $1`, trimmed).Scan(&id)
			if err == nil {
				return id, trimmed, nil
			}
		}
		return 0, "", fmt.Errorf("unknown entry type: %s", strings.TrimSpace(typeName))
	}
	var id int
	err := db.DB.QueryRow(`SELECT id FROM entry_types WHERE name = $1`, name).Scan(&id)
	if err != nil {
		return 0, "", fmt.Errorf("unknown entry type: %s", name)
	}
	return id, name, nil
}
