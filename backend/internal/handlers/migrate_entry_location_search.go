package handlers

import (
	"log"

	"backend/internal/db"
)

// MigrateEntryLocationSearch keeps entries.loc_name in step with the listing's
// settlement so city-name search hits every directory entry in that city.
func MigrateEntryLocationSearch() {
	stmts := []string{
		`CREATE OR REPLACE FUNCTION sync_entry_location_names()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.location_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT s.name, COALESCE(s.name_ro, ''), COALESCE(s.name_de, '')
      INTO NEW.loc_name, NEW.loc_name_ro, NEW.loc_name_de
      FROM settlements s
     WHERE s.id = NEW.location_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS trg_entries_sync_location_names ON entries`,
		`CREATE TRIGGER trg_entries_sync_location_names
BEFORE INSERT OR UPDATE OF location_id ON entries
FOR EACH ROW
EXECUTE FUNCTION sync_entry_location_names()`,
		`UPDATE entries e
SET loc_name = s.name,
    loc_name_ro = COALESCE(s.name_ro, ''),
    loc_name_de = COALESCE(s.name_de, '')
FROM settlements s
WHERE e.location_id = s.id
  AND (
    e.loc_name IS DISTINCT FROM s.name
    OR COALESCE(e.loc_name_ro, '') IS DISTINCT FROM COALESCE(s.name_ro, '')
    OR COALESCE(e.loc_name_de, '') IS DISTINCT FROM COALESCE(s.name_de, '')
  )`,
	}
	for _, sql := range stmts {
		if _, err := db.DB.Exec(sql); err != nil {
			log.Printf("MigrateEntryLocationSearch: %v", err)
		}
	}
}
