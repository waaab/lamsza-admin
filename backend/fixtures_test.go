package main

// Fixtures for the admin handler suite (BOG-55).
//
// The rows the admin write surface moderates - an unpublished listing, a
// pending claim, an open suggestion, a pending website - are all created by the
// *public* app's /api/account/* endpoints, and those live in `lamsza`. This
// process has no route that makes any of them. So they are inserted here
// directly, the mirror image of what lamsza/backend/fixtures_test.go had to do
// after BOG-42 moved the admin side out.
//
// Keep this the only file that writes the directory tables from a test. An
// admin handler under test must never need it to reach its own happy path - if
// one does, that is a missing route, not a missing fixture.
//
// Every fixture registers its own t.Cleanup, so a failed test still leaves the
// shared database as it found it. See the isolation rule in testsupport_test.go.

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"backend/internal/db"
)

// leafCategoryID returns a category with a parent. The entry write path rejects
// a top-level category (directory.ValidateLeaves), so the fixture must pick a
// leaf rather than entry_categories' first row.
func leafCategoryID(t *testing.T) int {
	t.Helper()
	var id int
	err := db.DB.QueryRow(`
		SELECT c.id FROM entry_categories c
		WHERE c.parent_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM entry_categories k WHERE k.parent_id = c.id)
		ORDER BY c.id ASC LIMIT 1`).Scan(&id)
	if err != nil {
		t.Skipf("no leaf entry_categories row: %v (the main lamsza backend seeds the catalog)", err)
	}
	return id
}

// topLevelCategoryID returns a category with no parent - the one shape the
// write paths must refuse (directory.ValidateLeaves). The negative cases need a
// category that genuinely exists, so "0" or a made-up id would not prove it.
func topLevelCategoryID(t *testing.T) int {
	t.Helper()
	var id int
	err := db.DB.QueryRow(`
		SELECT id FROM entry_categories
		WHERE parent_id IS NULL
		  AND EXISTS (SELECT 1 FROM entry_categories k WHERE k.parent_id = entry_categories.id)
		ORDER BY id ASC LIMIT 1`).Scan(&id)
	if err != nil {
		t.Skipf("no top-level entry_categories row with children: %v", err)
	}
	return id
}

// someSettlementID returns a settlement to hang fixtures off.
func someSettlementID(t *testing.T) int {
	t.Helper()
	var id int
	if err := db.DB.QueryRow(`SELECT id FROM settlements ORDER BY id ASC LIMIT 1`).Scan(&id); err != nil {
		t.Skipf("no settlements row: %v", err)
	}
	return id
}

// createAdminEntry creates one directory entry through the handler under test,
// POST /api/admin/entries, and removes it afterwards. Using the real route
// rather than SQL keeps the fixture honest about what the admin API accepts;
// TestAdminEntriesCRUD is what proves the route itself works.
func createAdminEntry(t *testing.T, label string) (id int, slug string) {
	t.Helper()
	name := testName(label)
	rr := asAdmin(t, http.MethodPost, "/api/admin/entries", map[string]interface{}{
		"name":        name,
		"location_id": someSettlementID(t),
		"type":        "Vállalkozás",
		"category_id": leafCategoryID(t),
		"phone":       "0700-000-000",
		"address":     "BOG-55 Test Address 1",
		"notes":       "BOG-55 fixture",
		"languages":   []string{"HU"},
	})
	mustOK(t, rr, "create entry fixture "+name)
	created := decodeObject(t, rr, "create entry fixture")
	id = idOf(t, created, "create entry fixture")
	slug, _ = created["slug"].(string)

	t.Cleanup(func() { deleteEntry(id) })
	return id, slug
}

// deleteEntry removes an entry and the rows that reference it. websites.entry_id
// is ON DELETE SET NULL, so an approved site would survive its entry and stay
// in the admin list for good.
func deleteEntry(id int) {
	db.DB.Exec(`DELETE FROM websites WHERE entry_id = $1`, id)
	db.DB.Exec(`DELETE FROM entries WHERE id = $1`, id)
}

// unpublishEntry is the state the public app's listing-create path leaves an
// entry in: created by its owner, waiting for the admin queue. There is no
// route here that produces it.
func unpublishEntry(t *testing.T, entryID int) {
	t.Helper()
	if _, err := db.DB.Exec(`UPDATE entries SET published = false WHERE id = $1`, entryID); err != nil {
		t.Fatalf("unpublish entry %d: %v", entryID, err)
	}
}

// entryNotes reads entries.notes. An accepted suggestion is only proved by the
// entry row changing, which is why the suggestion tests read this back rather
// than trusting the 200. It scans through sql.NullString because the column is
// nullable for rows older than the admin form; a failed query means the entry
// row is gone, which is a real failure and not an empty note.
func entryNotes(t *testing.T, entryID int) string {
	t.Helper()
	var notes sql.NullString
	if err := db.DB.QueryRow(`SELECT notes FROM entries WHERE id = $1`, entryID).Scan(&notes); err != nil {
		t.Fatalf("read notes for entry %d: %v", entryID, err)
	}
	return notes.String
}

func entryPublished(t *testing.T, entryID int) bool {
	t.Helper()
	var published bool
	if err := db.DB.QueryRow(`SELECT published FROM entries WHERE id = $1`, entryID).Scan(&published); err != nil {
		t.Fatalf("read published for entry %d: %v", entryID, err)
	}
	return published
}

// createTestUser makes an ordinary (non-admin) account for fixtures that need a
// claimant, a suggester or a website submitter.
func createTestUser(t *testing.T, label string) int {
	t.Helper()
	email := testName(label) + "@test.lamsza"
	id, err := upsertTestUser(email)
	if err != nil {
		t.Fatalf("create test user %s: %v", email, err)
	}
	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM admin_sessions WHERE user_id = $1`, id)
		db.DB.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// addEntryMember inserts one entry_members row in a given role and status. The
// public app writes these when a visitor claims a listing ('owner','pending')
// or asks to join one ('member','pending').
func addEntryMember(t *testing.T, entryID, userID int, role, status string) {
	t.Helper()
	_, err := db.DB.Exec(`
		INSERT INTO entry_members (entry_id, user_id, role, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (entry_id, user_id) DO UPDATE SET role = EXCLUDED.role, status = EXCLUDED.status`,
		entryID, userID, role, status)
	if err != nil {
		t.Fatalf("add entry_members %s/%s for entry %d: %v", role, status, entryID, err)
	}
	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM entry_members WHERE entry_id = $1 AND user_id = $2`, entryID, userID)
	})
}

// memberState reports the role and status stored for one membership, or
// ("", "") when the row is gone. The queue actions either promote a row or
// delete it, and "deleted" is a distinct outcome from "still pending".
func memberState(t *testing.T, entryID, userID int) (role, status string) {
	t.Helper()
	err := db.DB.QueryRow(
		`SELECT role, status FROM entry_members WHERE entry_id = $1 AND user_id = $2`,
		entryID, userID,
	).Scan(&role, &status)
	if err != nil {
		return "", ""
	}
	return role, status
}

// addOpenSuggestion inserts an open entry_suggestions row. changes is the
// visitor's proposed edit, in the shape the public suggestion form submits.
func addOpenSuggestion(t *testing.T, entryID, userID int, changes map[string]interface{}) int {
	t.Helper()
	raw, err := json.Marshal(changes)
	if err != nil {
		t.Fatalf("marshal suggestion changes: %v", err)
	}
	var id int
	err = db.DB.QueryRow(`
		INSERT INTO entry_suggestions (entry_id, user_id, changes, note, status)
		VALUES ($1, $2, $3::jsonb, $4, 'open') RETURNING id`,
		entryID, userID, string(raw), "BOG-55 suggestion",
	).Scan(&id)
	if err != nil {
		t.Fatalf("add entry_suggestions for entry %d: %v", entryID, err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM entry_suggestions WHERE id = $1`, id) })
	return id
}

func suggestionStatus(t *testing.T, id int) string {
	t.Helper()
	var status string
	if err := db.DB.QueryRow(`SELECT status FROM entry_suggestions WHERE id = $1`, id).Scan(&status); err != nil {
		return ""
	}
	return status
}

// addPendingWebsite inserts a pending website submission - what a visitor
// creates through the public app's /api/account/websites.
func addPendingWebsite(t *testing.T, userID int, label string) (id int, domain string) {
	t.Helper()
	domain = testName(label) + ".test"
	err := db.DB.QueryRow(`
		INSERT INTO websites (domain_key, submitted_host, title, description, status, user_id)
		VALUES ($1, $1, $2, $3, 'pending', $4) RETURNING id`,
		domain, testName(label), "BOG-55 pending website", userID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("add pending website %s: %v", domain, err)
	}
	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM website_category_links WHERE website_id = $1`, id)
		db.DB.Exec(`DELETE FROM websites WHERE id = $1`, id)
	})
	return id, domain
}

// websiteState returns a website's status and approver id. An empty status
// means the row is gone, which is what reject and ban do.
func websiteState(t *testing.T, id int) (status string, approvedBy int) {
	t.Helper()
	var approver *int
	err := db.DB.QueryRow(`SELECT status, approved_by FROM websites WHERE id = $1`, id).Scan(&status, &approver)
	if err != nil {
		return "", 0
	}
	if approver != nil {
		approvedBy = *approver
	}
	return status, approvedBy
}

func userWebsiteBanned(t *testing.T, userID int) bool {
	t.Helper()
	var banned bool
	if err := db.DB.QueryRow(`SELECT website_banned FROM users WHERE id = $1`, userID).Scan(&banned); err != nil {
		t.Fatalf("read website_banned for user %d: %v", userID, err)
	}
	return banned
}
