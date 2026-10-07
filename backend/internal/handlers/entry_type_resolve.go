package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"fmt"
	"strings"
)

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
