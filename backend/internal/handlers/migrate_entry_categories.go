package handlers

import (
	"backend/internal/db"
	"backend/internal/directory"
	"backend/internal/utils"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// MigrateEntryCategories is a legacy migrator. It no-ops once directory_catalog_v2 is set.
func MigrateEntryCategories() {
	var flag string
	err := db.DB.QueryRow(`SELECT value FROM site_settings WHERE key = $1`, directoryCatalogV2Key).Scan(&flag)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("MigrateEntryCategories (flag check): %v", err)
		return
	}
	if err == nil && flag == "1" {
		log.Println("skipped: directory_catalog_v2")
		return
	}
}

func resolveEntryCategoryID(id *int, name string) (int, string, error) {
	if id != nil && *id > 0 {
		var n string
		var parentID sql.NullInt64
		err := db.DB.QueryRow(`SELECT name, parent_id FROM entry_categories WHERE id = $1`, *id).Scan(&n, &parentID)
		if err == nil {
			return leafCategory(*id, n, parentID)
		}
	}
	canon := utils.CanonicalEntryCategory(name)
	if canon != "" {
		var cid int
		var parentID sql.NullInt64
		err := db.DB.QueryRow(`SELECT id, parent_id FROM entry_categories WHERE name = $1`, canon).Scan(&cid, &parentID)
		if err == nil {
			return leafCategory(cid, canon, parentID)
		}
	}
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		var cid int
		var parentID sql.NullInt64
		err := db.DB.QueryRow(`SELECT id, parent_id FROM entry_categories WHERE name = $1`, trimmed).Scan(&cid, &parentID)
		if err == nil {
			return leafCategory(cid, trimmed, parentID)
		}
	}
	return 0, "", fmt.Errorf("unknown entry category: %s", strings.TrimSpace(name))
}

var errCategoryNotLeaf = errors.New("category must be a subcategory")

func leafCategory(id int, name string, parentID sql.NullInt64) (int, string, error) {
	if !parentID.Valid {
		return 0, "", errCategoryNotLeaf
	}
	return id, name, nil
}

func writeCategoryResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, errCategoryNotLeaf) || errors.Is(err, directory.ErrCategorySet) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
