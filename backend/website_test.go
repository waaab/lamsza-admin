package main

// Website moderation, end to end (BOG-55).
//
// One route, /api/admin/websites, carries the whole state machine: a visitor's
// pending submission is either approved (it becomes a row the public app lists),
// rejected (deleted), or banned (deleted and the submitter blocked from
// submitting again). Nothing covered it after BOG-42 moved it out of `lamsza`.
//
// Each action is asserted on the stored row, not on the 200: approve and reject
// both answer 200 and differ only in what the database holds afterwards, which
// is exactly the pair a wrong `switch` branch would swap.

import (
	"net/http"
	"testing"

	"backend/internal/db"
)

// TestWebsiteModerationApprove: pending -> approved, stamped with the admin who
// did it. approved_by is what the admin list shows as the approver, so an
// approval that left it null would read as "approved by nobody".
func TestWebsiteModerationApprove(t *testing.T) {
	requireDB(t)

	submitter := createTestUser(t, "site-approve-user")
	websiteID, domain := addPendingWebsite(t, submitter, "site-approve")

	// It starts in the admin list as pending, which is what makes it a queue.
	if row := findByID(adminWebsites(t), websiteID); row == nil {
		t.Fatalf("pending website %d (%s) is not in the admin list", websiteID, domain)
	} else if row["status"] != "pending" {
		t.Fatalf("website %d starts as %#v, want pending", websiteID, row["status"])
	}

	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "approve",
	}), "approve website")

	status, approvedBy := websiteState(t, websiteID)
	if status != "approved" {
		t.Fatalf("website %d status = %q after approve, want approved", websiteID, status)
	}
	adminID, err := userIDByEmail(testAdminEmail)
	if err != nil {
		t.Fatalf("admin user: %v", err)
	}
	if approvedBy != adminID {
		t.Errorf("website %d approved_by = %d, want the signed-in admin %d", websiteID, approvedBy, adminID)
	}

	// Approving twice is a 404, not a second approval: the UPDATE is guarded on
	// status = 'pending'. Without that guard a stale admin tab could re-stamp a
	// site somebody else had already decided.
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "approve",
	}), http.StatusNotFound, "approve an already-approved website")
}

// TestWebsiteModerationApproveWithCategory covers the other approve branch: the
// admin files the site under a directory category at the same time. It takes a
// different SQL path, writes website_category_links, and validates the category
// - three things the plain approve above does not touch.
func TestWebsiteModerationApproveWithCategory(t *testing.T) {
	requireDB(t)

	categoryID := leafCategoryID(t)
	submitter := createTestUser(t, "site-cat-user")
	websiteID, _ := addPendingWebsite(t, submitter, "site-cat")

	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "approve", "category_id": categoryID,
	}), "approve website with a category")

	if status, _ := websiteState(t, websiteID); status != "approved" {
		t.Fatalf("website %d status = %q, want approved", websiteID, status)
	}
	if got := websiteCategoryIDs(t, websiteID); len(got) != 1 || got[0] != categoryID {
		t.Errorf("website_category_links for %d = %v, want [%d]", websiteID, got, categoryID)
	}

	// A top-level category is refused by directory.ValidateLeaves, and the
	// refusal must name the field so the SPA can mark the right input.
	other, _ := addPendingWebsite(t, submitter, "site-cat-bad")
	rr := asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": other, "action": "approve", "category_id": topLevelCategoryID(t),
	})
	mustStatus(t, rr, http.StatusBadRequest, "approve under a non-leaf category")
	if field := decodeObject(t, rr, "approve under a non-leaf category")["field"]; field != "category_id" {
		t.Errorf("refusal names field %#v, want category_id", field)
	}
	if status, _ := websiteState(t, other); status != "pending" {
		t.Errorf("website %d is now %q; a refused approval must leave it pending", other, status)
	}
}

// TestWebsiteModerationReject deletes the submission and leaves the submitter
// free to try again. That last part is the whole difference from ban.
func TestWebsiteModerationReject(t *testing.T) {
	requireDB(t)

	submitter := createTestUser(t, "site-reject-user")
	websiteID, _ := addPendingWebsite(t, submitter, "site-reject")

	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "reject",
	}), "reject website")

	if status, _ := websiteState(t, websiteID); status != "" {
		t.Fatalf("website %d is still %q after reject; the row should be gone", websiteID, status)
	}
	if userWebsiteBanned(t, submitter) {
		t.Errorf("user %d was banned by a reject; only ban does that", submitter)
	}
	if findByID(adminWebsites(t), websiteID) != nil {
		t.Errorf("website %d is still in the admin list after reject", websiteID)
	}

	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "reject",
	}), http.StatusNotFound, "reject an already-rejected website")
}

// TestWebsiteModerationBan deletes the submission *and* sets
// users.website_banned. Both halves are in one transaction, so a ban that only
// did one of them would be the bug worth catching: a deleted site with an
// un-banned spammer, or a banned user whose spam is still listed.
func TestWebsiteModerationBan(t *testing.T) {
	requireDB(t)

	submitter := createTestUser(t, "site-ban-user")
	websiteID, _ := addPendingWebsite(t, submitter, "site-ban")
	if userWebsiteBanned(t, submitter) {
		t.Fatalf("fixture user %d starts banned", submitter)
	}

	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": websiteID, "action": "ban",
	}), "ban website submitter")

	if status, _ := websiteState(t, websiteID); status != "" {
		t.Errorf("website %d is still %q after ban; the row should be gone", websiteID, status)
	}
	if !userWebsiteBanned(t, submitter) {
		t.Errorf("user %d is not banned after ban returned 200", submitter)
	}

	// An approved site cannot be banned: every action is guarded on
	// status = 'pending'.
	approved, _ := addPendingWebsite(t, submitter, "site-ban-approved")
	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": approved, "action": "approve",
	}), "approve the second website")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": approved, "action": "ban",
	}), http.StatusNotFound, "ban an approved website")
	if status, _ := websiteState(t, approved); status != "approved" {
		t.Errorf("website %d is now %q; the refused ban must have changed nothing", approved, status)
	}
}

func TestWebsiteModerationBadInput(t *testing.T) {
	requireDB(t)

	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": 2147483646, "action": "approve",
	}), http.StatusNotFound, "approve a website id that does not exist")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"action": "approve",
	}), http.StatusBadRequest, "approve with no id")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", map[string]interface{}{
		"id": 1, "action": "delete",
	}), http.StatusBadRequest, "an action outside approve/reject/ban")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/websites", "{not json"),
		http.StatusBadRequest, "a malformed body")
	mustStatus(t, asAdmin(t, http.MethodDelete, "/api/admin/websites", nil),
		http.StatusMethodNotAllowed, "DELETE the websites route")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// adminWebsites returns the admin list. It is an object with a `websites` key
// rather than a bare array, which is why it cannot reuse decodeArray.
func adminWebsites(t *testing.T) []map[string]interface{} {
	t.Helper()
	rr := asAdmin(t, http.MethodGet, "/api/admin/websites", nil)
	mustOK(t, rr, "GET /api/admin/websites")
	return bucket(t, decodeObject(t, rr, "GET /api/admin/websites"), "websites")
}

func websiteCategoryIDs(t *testing.T, websiteID int) []int {
	t.Helper()
	rows, err := db.DB.Query(
		`SELECT category_id FROM website_category_links WHERE website_id = $1 ORDER BY category_id`, websiteID)
	if err != nil {
		t.Fatalf("read website_category_links for %d: %v", websiteID, err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan website_category_links: %v", err)
		}
		out = append(out, id)
	}
	return out
}
