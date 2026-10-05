package events

import (
	"backend/internal/db"
	"log"
)

// Migrate ensures catalog tables, event columns, and backfill from legacy event_type (idempotent).
func Migrate() {
	_, err := db.DB.Exec(`
		ALTER TABLE events ADD COLUMN IF NOT EXISTS featured_image VARCHAR(1024) DEFAULT ''`)
	if err != nil {
		log.Printf("events.Migrate (featured_image): %v", err)
	}
	_, err = db.DB.Exec(`
		ALTER TABLE events ADD COLUMN IF NOT EXISTS entry_price VARCHAR(128) DEFAULT ''`)
	if err != nil {
		log.Printf("events.Migrate (entry_price): %v", err)
	}

	_, err = db.DB.Exec(`
CREATE TABLE IF NOT EXISTS catalog_event_types (
	id SERIAL PRIMARY KEY,
	slug VARCHAR(64) NOT NULL UNIQUE,
	label_hu TEXT NOT NULL,
	sort_order INT NOT NULL DEFAULT 0
)`)
	if err != nil {
		log.Printf("events.Migrate (catalog_event_types): %v", err)
	}

	_, err = db.DB.Exec(`
CREATE TABLE IF NOT EXISTS catalog_event_subtypes (
	id SERIAL PRIMARY KEY,
	event_type_id INT NOT NULL REFERENCES catalog_event_types(id) ON DELETE CASCADE,
	slug VARCHAR(64) NOT NULL,
	label_hu TEXT NOT NULL,
	sort_order INT NOT NULL DEFAULT 0,
	UNIQUE(event_type_id, slug)
)`)
	if err != nil {
		log.Printf("events.Migrate (catalog_event_subtypes): %v", err)
	}

	_, err = db.DB.Exec(`
INSERT INTO catalog_event_types (slug, label_hu, sort_order) VALUES
	('cultural', 'Kulturális', 1),
	('sports', 'Sport', 2),
	('festival', 'Fesztivál', 3),
	('religious', 'Vallási', 4),
	('other', 'Egyéb', 5)
ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		log.Printf("events.Migrate (seed types): %v", err)
	}

	// Sub-types (examples per type)
	seedSub := []string{
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'hockey', 'Jégkorong', 1 FROM catalog_event_types t WHERE t.slug = 'sports'
		 ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'football', 'Futball', 2 FROM catalog_event_types t WHERE t.slug = 'sports' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'golf', 'Golf', 3 FROM catalog_event_types t WHERE t.slug = 'sports' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'tennis', 'Tenisz', 4 FROM catalog_event_types t WHERE t.slug = 'sports' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'handball', 'Kézilabda', 5 FROM catalog_event_types t WHERE t.slug = 'sports' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'concert', 'Koncert', 1 FROM catalog_event_types t WHERE t.slug = 'cultural' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'theatre', 'Színház', 2 FROM catalog_event_types t WHERE t.slug = 'cultural' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'exhibition', 'Kiállítás', 3 FROM catalog_event_types t WHERE t.slug = 'cultural' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'cinema', 'Mozi', 4 FROM catalog_event_types t WHERE t.slug = 'cultural' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'music', 'Zene', 1 FROM catalog_event_types t WHERE t.slug = 'festival' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'folk', 'Népi', 2 FROM catalog_event_types t WHERE t.slug = 'festival' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'wine', 'Bor', 3 FROM catalog_event_types t WHERE t.slug = 'festival' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'mass', 'Mise', 1 FROM catalog_event_types t WHERE t.slug = 'religious' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'pilgrimage', 'Zarándoklat', 2 FROM catalog_event_types t WHERE t.slug = 'religious' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'community', 'Közösségi', 1 FROM catalog_event_types t WHERE t.slug = 'other' ON CONFLICT (event_type_id, slug) DO NOTHING`,
		`INSERT INTO catalog_event_subtypes (event_type_id, slug, label_hu, sort_order)
		 SELECT t.id, 'charity', 'Jótékonysági', 2 FROM catalog_event_types t WHERE t.slug = 'other' ON CONFLICT (event_type_id, slug) DO NOTHING`,
	}
	for _, q := range seedSub {
		if _, e := db.DB.Exec(q); e != nil {
			log.Printf("events.Migrate (seed subtype): %v", e)
		}
	}

	_, err = db.DB.Exec(`
ALTER TABLE events ADD COLUMN IF NOT EXISTS access_type VARCHAR(32) DEFAULT 'public'`)
	if err != nil {
		log.Printf("events.Migrate (access_type): %v", err)
	}
	_, err = db.DB.Exec(`
ALTER TABLE events ADD COLUMN IF NOT EXISTS event_type_id INTEGER`)
	if err != nil {
		log.Printf("events.Migrate (event_type_id add): %v", err)
	}
	_, err = db.DB.Exec(`
ALTER TABLE events ADD COLUMN IF NOT EXISTS event_subtype_id INTEGER`)
	if err != nil {
		log.Printf("events.Migrate (event_subtype_id add): %v", err)
	}

	_, err = db.DB.Exec(`
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'events' AND column_name = 'event_type'
	) THEN
		UPDATE events e
		SET event_type_id = ct.id
		FROM catalog_event_types ct
		WHERE e.event_type = ct.slug AND e.event_type_id IS NULL;
		UPDATE events
		SET event_type_id = (SELECT id FROM catalog_event_types WHERE slug = 'other' LIMIT 1)
		WHERE event_type_id IS NULL;
		ALTER TABLE events DROP COLUMN event_type;
	END IF;
END $$`)
	if err != nil {
		log.Printf("events.Migrate (backfill/drop legacy event_type): %v", err)
	}

	_, err = db.DB.Exec(`
UPDATE events SET event_type_id = (SELECT id FROM catalog_event_types WHERE slug = 'other' LIMIT 1) WHERE event_type_id IS NULL`)
	if err != nil {
		log.Printf("events.Migrate (null event_type_id): %v", err)
	}

	_, err = db.DB.Exec(`ALTER TABLE events ALTER COLUMN event_type_id SET NOT NULL`)
	if err != nil {
		log.Printf("events.Migrate (event_type_id NOT NULL): %v", err)
	}

	_, err = db.DB.Exec(`
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_event_type_id_fkey`)
	if err != nil {
		log.Printf("events.Migrate (drop fk event_type_id): %v", err)
	}
	_, err = db.DB.Exec(`
ALTER TABLE events ADD CONSTRAINT events_event_type_id_fkey
	FOREIGN KEY (event_type_id) REFERENCES catalog_event_types(id) ON DELETE RESTRICT`)
	if err != nil {
		log.Printf("events.Migrate (fk event_type_id): %v", err)
	}

	_, err = db.DB.Exec(`
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_event_subtype_id_fkey`)
	if err != nil {
		log.Printf("events.Migrate (drop fk event_subtype_id): %v", err)
	}
	_, err = db.DB.Exec(`
ALTER TABLE events ADD CONSTRAINT events_event_subtype_id_fkey
	FOREIGN KEY (event_subtype_id) REFERENCES catalog_event_subtypes(id) ON DELETE SET NULL`)
	if err != nil {
		log.Printf("events.Migrate (fk event_subtype_id): %v", err)
	}

	_, err = db.DB.Exec(`
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_access_type_check`)
	if err != nil {
		log.Printf("events.Migrate (drop access check): %v", err)
	}
	_, err = db.DB.Exec(`
ALTER TABLE events ADD CONSTRAINT events_access_type_check
	CHECK (access_type IN ('public', 'members_only', 'invitation_only'))`)
	if err != nil {
		log.Printf("events.Migrate (access_type check): %v", err)
	}

	_, err = db.DB.Exec(`UPDATE events SET access_type = 'public' WHERE access_type IS NULL OR access_type = ''`)
	if err != nil {
		log.Printf("events.Migrate (access_type default): %v", err)
	}

	_, err = db.DB.Exec(`ALTER TABLE events ADD COLUMN IF NOT EXISTS attraction_id INTEGER REFERENCES attractions(id) ON DELETE SET NULL`)
	if err != nil {
		log.Printf("events.Migrate (attraction_id): %v", err)
	}
	_, err = db.DB.Exec(`ALTER TABLE events ADD COLUMN IF NOT EXISTS featured_image_copyright TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		log.Printf("events.Migrate (featured_image_copyright): %v", err)
	}
}
