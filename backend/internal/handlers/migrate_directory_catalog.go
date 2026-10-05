package handlers

import (
	"backend/internal/db"
	"backend/internal/utils"
	"database/sql"
	"fmt"
	"log"
)

const directoryCatalogV2Key = "directory_catalog_v2"

// Tables wiped by the catalog migration or emptied via CASCADE. entry_tags and
// entry_members use composite primary keys and have no serial sequence.
var directorySerialTables = []string{
	"entries",
	"tags",
	"entry_reviews",
	"entry_suggestions",
	"entry_types",
	"entry_categories",
}

func alignDirectorySequences(exec interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}) error {
	for _, table := range directorySerialTables {
		stmt := fmt.Sprintf(`
DO $migrate$
BEGIN
  IF pg_get_serial_sequence('%s', 'id') IS NOT NULL THEN
    IF (SELECT COUNT(*) FROM %s) = 0 THEN
      PERFORM setval(pg_get_serial_sequence('%s', 'id'), 1, false);
    ELSE
      PERFORM setval(pg_get_serial_sequence('%s', 'id'), (SELECT MAX(id) FROM %s), true);
    END IF;
  END IF;
END
$migrate$`, table, table, table, table, table)
		if _, err := exec.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
	}
	return nil
}

// MigrateDirectoryCatalog wipes directory rows once and seeds the two-level
// category tree and three entry types. Guarded by site_settings.directory_catalog_v2.
func MigrateDirectoryCatalog() {
	schemaStmts := []string{
		`ALTER TABLE entry_categories ADD COLUMN IF NOT EXISTS parent_id INTEGER REFERENCES entry_categories(id)`,
		`ALTER TABLE entry_categories ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE websites ADD COLUMN IF NOT EXISTS category_id INTEGER REFERENCES entry_categories(id)`,
		`CREATE TABLE IF NOT EXISTS entry_category_links (
			entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			category_id INTEGER NOT NULL REFERENCES entry_categories(id),
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			PRIMARY KEY (entry_id, category_id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS entry_category_links_one_primary ON entry_category_links (entry_id) WHERE is_primary`,
		`CREATE TABLE IF NOT EXISTS website_category_links (
			website_id INTEGER NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
			category_id INTEGER NOT NULL REFERENCES entry_categories(id),
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			PRIMARY KEY (website_id, category_id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS website_category_links_one_primary ON website_category_links (website_id) WHERE is_primary`,
	}
	for _, stmt := range schemaStmts {
		if _, err := db.DB.Exec(stmt); err != nil {
			log.Printf("MigrateDirectoryCatalog (schema): %v", err)
			return
		}
	}

	var flag string
	err := db.DB.QueryRow(`SELECT value FROM site_settings WHERE key = $1`, directoryCatalogV2Key).Scan(&flag)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("MigrateDirectoryCatalog (flag check): %v", err)
		return
	}
	alreadyDone := err == nil && flag == "1"

	if !alreadyDone {
		tx, err := db.DB.Begin()
		if err != nil {
			log.Printf("MigrateDirectoryCatalog (begin): %v", err)
			return
		}
		defer tx.Rollback()

		wipeStmts := []string{
			`DELETE FROM entries`,
			`DELETE FROM tags`,
			`DELETE FROM entry_categories`,
			`DELETE FROM entry_types`,
		}
		for _, stmt := range wipeStmts {
			if _, err := tx.Exec(stmt); err != nil {
				log.Printf("MigrateDirectoryCatalog (wipe): %v", err)
				return
			}
		}

		for _, t := range utils.DirectoryTypes() {
			if _, err := tx.Exec(
				`INSERT INTO entry_types (id, name) VALUES ($1, $2)`,
				t.ID, t.Name,
			); err != nil {
				log.Printf("MigrateDirectoryCatalog (type %s): %v", t.Name, err)
				return
			}
		}

		insertCat := func(node utils.DirectoryNode) error {
			slug := utils.Slugify(node.Name)
			_, err := tx.Exec(
				`INSERT INTO entry_categories (id, name, slug, parent_id, sort_order) VALUES ($1, $2, $3, $4, $5)`,
				node.ID, node.Name, slug, node.ParentID, node.SortOrder,
			)
			return err
		}

		for _, node := range utils.DirectoryParents() {
			if err := insertCat(node); err != nil {
				log.Printf("MigrateDirectoryCatalog (parent %s): %v", node.Name, err)
				return
			}
		}
		for _, node := range utils.DirectoryChildren() {
			if err := insertCat(node); err != nil {
				log.Printf("MigrateDirectoryCatalog (child %s): %v", node.Name, err)
				return
			}
		}

		if _, err := tx.Exec(
			`INSERT INTO site_settings (key, value) VALUES ($1, '1')
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
			directoryCatalogV2Key,
		); err != nil {
			log.Printf("MigrateDirectoryCatalog (flag): %v", err)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("MigrateDirectoryCatalog (commit): %v", err)
			return
		}

		log.Println("Directory catalog v2 seeded")
	}

	if err := alignDirectorySequences(db.DB); err != nil {
		log.Printf("MigrateDirectoryCatalog (align sequences): %v", err)
		return
	}
	dropVenueOnlySportpalya()
	ensureDirectoryNodes()
	backfillCategoryLinks()
}

// backfillCategoryLinks copies the single category_id into the membership table.
func backfillCategoryLinks() {
	stmts := []string{
		`INSERT INTO entry_category_links (entry_id, category_id, is_primary)
		 SELECT id, category_id, TRUE FROM entries
		 WHERE category_id IS NOT NULL
		 ON CONFLICT (entry_id, category_id) DO NOTHING`,
		`INSERT INTO website_category_links (website_id, category_id, is_primary)
		 SELECT id, category_id, TRUE FROM websites
		 WHERE category_id IS NOT NULL
		 ON CONFLICT (website_id, category_id) DO NOTHING`,
		`CREATE OR REPLACE FUNCTION sync_entry_category()
		 RETURNS TRIGGER AS $sync$
		 BEGIN
		   UPDATE entries e
		   SET cat_name = (
		     SELECT string_agg(c.name, ' ' ORDER BY l.is_primary DESC, c.sort_order, c.name)
		     FROM entry_category_links l
		     JOIN entry_categories c ON c.id = l.category_id
		     WHERE l.entry_id = e.id
		   )
		   WHERE e.id IN (
		     SELECT entry_id FROM entry_category_links WHERE category_id = NEW.id
		   );
		   RETURN NEW;
		 END;
		 $sync$ LANGUAGE plpgsql`,
	}
	for _, stmt := range stmts {
		if _, err := db.DB.Exec(stmt); err != nil {
			log.Printf("MigrateDirectoryCatalog (category links): %v", err)
			return
		}
	}
}

// ensureDirectoryNodes inserts seed rows that are not in the database yet.
// An existing id or slug is left alone, so an admin rename is kept.
func ensureDirectoryNodes() {
	insert := func(node utils.DirectoryNode) {
		slug := utils.Slugify(node.Name)
		var existing int
		err := db.DB.QueryRow(
			`SELECT id FROM entry_categories WHERE id = $1 OR slug = $2`,
			node.ID, slug,
		).Scan(&existing)
		if err == nil {
			return
		}
		if err != sql.ErrNoRows {
			log.Printf("MigrateDirectoryCatalog (lookup %s): %v", node.Name, err)
			return
		}
		if _, err := db.DB.Exec(
			`INSERT INTO entry_categories (id, name, slug, parent_id, sort_order) VALUES ($1, $2, $3, $4, $5)`,
			node.ID, node.Name, slug, node.ParentID, node.SortOrder,
		); err != nil {
			log.Printf("MigrateDirectoryCatalog (insert %s): %v", node.Name, err)
		}
	}
	for _, node := range utils.DirectoryParents() {
		insert(node)
	}
	for _, node := range utils.DirectoryChildren() {
		insert(node)
	}
}

// dropVenueOnlySportpalya removes the seeded directory row. Sportpálya is a venue type.
func dropVenueOnlySportpalya() {
	var name string
	err := db.DB.QueryRow(`SELECT name FROM entry_categories WHERE id = 66`).Scan(&name)
	if err != nil || name != "Sportpálya" {
		return
	}
	var used int
	err = db.DB.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM entry_category_links WHERE category_id = 66) +
			(SELECT COUNT(*) FROM website_category_links WHERE category_id = 66) +
			(SELECT COUNT(*) FROM entries WHERE category_id = 66) +
			(SELECT COUNT(*) FROM websites WHERE category_id = 66) +
			(SELECT COUNT(*) FROM entry_categories WHERE parent_id = 66)
	`).Scan(&used)
	if err != nil || used > 0 {
		if err != nil {
			log.Printf("MigrateDirectoryCatalog (sportpalya check): %v", err)
		}
		return
	}
	if _, err := db.DB.Exec(`DELETE FROM entry_categories WHERE id = 66 AND name = 'Sportpálya'`); err != nil {
		log.Printf("MigrateDirectoryCatalog (drop sportpalya): %v", err)
	}
}
