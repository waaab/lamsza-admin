package main

// The admin listing queue, end to end (BOG-55).
//
// These five routes are the ones that write rows the public app reads: publish
// makes a listing visible on lamsza.com, an accepted claim makes somebody its
// owner, an accepted suggestion rewrites its fields. Nothing covered them after
// BOG-42 moved them out of `lamsza`.
//
// The old lamsza tests drove the pending side through /api/account/listings.
// That route is not here, so the fixtures insert the pending rows (see
// fixtures_test.go) and the assertions read the stored state back rather than
// trusting the 200.

import (
	"net/http"
	"testing"
)

// TestListingQueueShowsEveryPendingKind proves the queue reports all five
// buckets, each from its own table. A query that silently returned nothing -
// the easy regression, since every bucket is a separate SELECT - would leave
// the admin with an empty page and no error.
func TestListingQueueShowsEveryPendingKind(t *testing.T) {
	requireDB(t)

	entryID, _ := createAdminEntry(t, "queue-all")
	unpublishEntry(t, entryID)

	claimant := createTestUser(t, "claimant")
	joiner := createTestUser(t, "joiner")
	suggester := createTestUser(t, "suggester")
	submitter := createTestUser(t, "submitter")

	addEntryMember(t, entryID, claimant, "owner", "pending")
	addEntryMember(t, entryID, joiner, "member", "pending")
	suggestionID := addOpenSuggestion(t, entryID, suggester, map[string]interface{}{"notes": "BOG-55 proposed note"})
	websiteID, _ := addPendingWebsite(t, submitter, "queue-site")

	rr := asAdmin(t, http.MethodGet, "/api/admin/listing-queue", nil)
	mustOK(t, rr, "GET /api/admin/listing-queue")
	queue := decodeObject(t, rr, "GET /api/admin/listing-queue")

	if findByID(bucket(t, queue, "unpublished"), entryID) == nil {
		t.Errorf("unpublished bucket is missing entry %d", entryID)
	}
	if findByEntryAndUser(bucket(t, queue, "claims"), entryID, claimant) == nil {
		t.Errorf("claims bucket is missing the pending owner for entry %d", entryID)
	}
	if findByEntryAndUser(bucket(t, queue, "members"), entryID, joiner) == nil {
		t.Errorf("members bucket is missing the pending member for entry %d", entryID)
	}
	if findByID(bucket(t, queue, "suggestions"), suggestionID) == nil {
		t.Errorf("suggestions bucket is missing suggestion %d", suggestionID)
	}
	if findByID(bucket(t, queue, "websites"), websiteID) == nil {
		t.Errorf("websites bucket is missing website %d", websiteID)
	}
}

// TestListingQueuePublish is the state change the public site depends on most:
// published = false -> true. Asserted on the row, not on the response code.
func TestListingQueuePublish(t *testing.T) {
	requireDB(t)

	entryID, _ := createAdminEntry(t, "publish")
	unpublishEntry(t, entryID)
	if entryPublished(t, entryID) {
		t.Fatalf("fixture entry %d should start unpublished", entryID)
	}

	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/publish",
		map[string]int{"entry_id": entryID}), "publish entry")

	if !entryPublished(t, entryID) {
		t.Fatalf("entry %d is still unpublished after publish returned 200", entryID)
	}
	if findByID(queueBucket(t, "unpublished"), entryID) != nil {
		t.Errorf("entry %d is still in the unpublished queue after publish", entryID)
	}

	// Publishing twice is not an error - the admin UI can double-click - but it
	// must not report success for an id that does not exist.
	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/publish",
		map[string]int{"entry_id": entryID}), "publish an already-published entry")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/publish",
		map[string]int{"entry_id": 2147483646}), http.StatusNotFound, "publish a missing entry")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/publish",
		map[string]int{"entry_id": 0}), http.StatusBadRequest, "publish with no entry_id")
	mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/listing-queue/publish", nil),
		http.StatusMethodNotAllowed, "GET the publish route")
}

// TestListingQueueClaimAcceptAndDeny covers both halves of the claim decision.
// Accept promotes the pending owner; deny deletes the row. They are different
// outcomes, and a handler that treated deny as accept would hand a stranger a
// listing.
func TestListingQueueClaimAcceptAndDeny(t *testing.T) {
	requireDB(t)

	t.Run("accept", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "claim-accept")
		claimant := createTestUser(t, "claim-accept-user")
		addEntryMember(t, entryID, claimant, "owner", "pending")

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": entryID, "user_id": claimant, "action": "accept",
		}), "accept claim")

		role, status := memberState(t, entryID, claimant)
		if role != "owner" || status != "active" {
			t.Fatalf("after accept, membership is %q/%q; want owner/active", role, status)
		}
	})

	t.Run("deny", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "claim-deny")
		claimant := createTestUser(t, "claim-deny-user")
		addEntryMember(t, entryID, claimant, "owner", "pending")

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": entryID, "user_id": claimant, "action": "deny",
		}), "deny claim")

		role, status := memberState(t, entryID, claimant)
		if role != "" || status != "" {
			t.Fatalf("after deny, membership is still %q/%q; the row should be gone", role, status)
		}
	})

	t.Run("second claim on an owned listing is refused", func(t *testing.T) {
		// entry_members_one_active_owner enforces this in the schema too, but
		// the handler must answer 409 rather than let the constraint surface as
		// a 500.
		entryID, _ := createAdminEntry(t, "claim-conflict")
		first := createTestUser(t, "claim-first")
		second := createTestUser(t, "claim-second")
		addEntryMember(t, entryID, first, "owner", "active")
		addEntryMember(t, entryID, second, "owner", "pending")

		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": entryID, "user_id": second, "action": "accept",
		}), http.StatusConflict, "accept a second claim")

		if _, status := memberState(t, entryID, second); status != "pending" {
			t.Errorf("the refused claim is now %q; it should stay pending", status)
		}
		if _, status := memberState(t, entryID, first); status != "active" {
			t.Errorf("the existing owner is now %q; it should stay active", status)
		}
	})

	t.Run("bad input", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "claim-bad")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": entryID, "user_id": 1, "action": "whatever",
		}), http.StatusBadRequest, "claim with an unknown action")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": entryID, "action": "accept",
		}), http.StatusBadRequest, "claim with no user_id")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/claim", map[string]interface{}{
			"entry_id": 2147483646, "user_id": 1, "action": "accept",
		}), http.StatusNotFound, "claim on a missing entry")
	})
}

// TestListingQueueMemberApproveAndReject is the same decision for a join
// request: approve activates, reject deletes.
func TestListingQueueMemberApproveAndReject(t *testing.T) {
	requireDB(t)

	t.Run("approve", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "member-approve")
		joiner := createTestUser(t, "member-approve-user")
		addEntryMember(t, entryID, joiner, "member", "pending")

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/member", map[string]interface{}{
			"entry_id": entryID, "user_id": joiner, "action": "approve",
		}), "approve member")

		role, status := memberState(t, entryID, joiner)
		if role != "member" || status != "active" {
			t.Fatalf("after approve, membership is %q/%q; want member/active", role, status)
		}
	})

	t.Run("reject", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "member-reject")
		joiner := createTestUser(t, "member-reject-user")
		addEntryMember(t, entryID, joiner, "member", "pending")

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/member", map[string]interface{}{
			"entry_id": entryID, "user_id": joiner, "action": "reject",
		}), "reject member")

		if role, status := memberState(t, entryID, joiner); role != "" || status != "" {
			t.Fatalf("after reject, membership is still %q/%q; the row should be gone", role, status)
		}
	})

	t.Run("a pending owner is not a member", func(t *testing.T) {
		// The member route and the claim route read the same table and differ
		// only by role. Approving a *claim* through the member route would
		// bypass the already-claimed check in the claim handler.
		entryID, _ := createAdminEntry(t, "member-role-split")
		claimant := createTestUser(t, "member-role-split-user")
		addEntryMember(t, entryID, claimant, "owner", "pending")

		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/member", map[string]interface{}{
			"entry_id": entryID, "user_id": claimant, "action": "approve",
		}), http.StatusNotFound, "approve a pending owner through the member route")

		if _, status := memberState(t, entryID, claimant); status != "pending" {
			t.Errorf("the pending claim is now %q; the member route must not touch it", status)
		}
	})

	t.Run("bad input", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "member-bad")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/member", map[string]interface{}{
			"entry_id": entryID, "user_id": 1, "action": "accept",
		}), http.StatusBadRequest, "member with the claim route's action name")
		mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/listing-queue/member", nil),
			http.StatusMethodNotAllowed, "GET the member route")
	})
}

// TestListingQueueSuggestionAcceptAndDeny is the one queue action that rewrites
// the entry: accepting a suggestion applies the visitor's proposed change. Deny
// must leave the entry alone.
func TestListingQueueSuggestionAcceptAndDeny(t *testing.T) {
	requireDB(t)

	t.Run("accept applies the change", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "suggestion-accept")
		suggester := createTestUser(t, "suggestion-accept-user")
		const proposed = "BOG-55 accepted note"
		id := addOpenSuggestion(t, entryID, suggester, map[string]interface{}{"notes": proposed})

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": id, "action": "accept",
		}), "accept suggestion")

		if got := suggestionStatus(t, id); got != "accepted" {
			t.Fatalf("suggestion status = %q, want accepted", got)
		}
		if got := entryNotes(t, entryID); got != proposed {
			t.Fatalf("entry notes = %q; the accepted suggestion was not applied (want %q)", got, proposed)
		}
	})

	t.Run("deny leaves the entry alone", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "suggestion-deny")
		before := entryNotes(t, entryID)
		suggester := createTestUser(t, "suggestion-deny-user")
		id := addOpenSuggestion(t, entryID, suggester, map[string]interface{}{"notes": "BOG-55 denied note"})

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": id, "action": "deny",
		}), "deny suggestion")

		if got := suggestionStatus(t, id); got != "denied" {
			t.Fatalf("suggestion status = %q, want denied", got)
		}
		if got := entryNotes(t, entryID); got != before {
			t.Fatalf("entry notes changed to %q on a denied suggestion (was %q)", got, before)
		}
	})

	t.Run("a resolved suggestion cannot be decided twice", func(t *testing.T) {
		entryID, _ := createAdminEntry(t, "suggestion-twice")
		suggester := createTestUser(t, "suggestion-twice-user")
		id := addOpenSuggestion(t, entryID, suggester, map[string]interface{}{"notes": "BOG-55 once"})

		mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": id, "action": "deny",
		}), "deny suggestion")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": id, "action": "accept",
		}), http.StatusNotFound, "accept an already-denied suggestion")

		if got := suggestionStatus(t, id); got != "denied" {
			t.Errorf("suggestion status = %q after a refused second decision, want denied", got)
		}
	})

	t.Run("bad input", func(t *testing.T) {
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": 2147483646, "action": "accept",
		}), http.StatusNotFound, "accept a missing suggestion")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"action": "accept",
		}), http.StatusBadRequest, "accept with no id")
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/listing-queue/suggestion", map[string]interface{}{
			"id": 1, "action": "approve",
		}), http.StatusBadRequest, "suggestion with an unknown action")
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func queueBucket(t *testing.T, name string) []map[string]interface{} {
	t.Helper()
	rr := asAdmin(t, http.MethodGet, "/api/admin/listing-queue", nil)
	mustOK(t, rr, "GET /api/admin/listing-queue")
	return bucket(t, decodeObject(t, rr, "GET /api/admin/listing-queue"), name)
}

func bucket(t *testing.T, queue map[string]interface{}, name string) []map[string]interface{} {
	t.Helper()
	raw, ok := queue[name]
	if !ok {
		t.Fatalf("listing-queue has no %q bucket: keys %v", name, objectKeys(queue))
	}
	list, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("listing-queue %q is %T, want an array", name, raw)
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// findByEntryAndUser is findByID for the two buckets keyed by a pair rather
// than by an id of their own.
func findByEntryAndUser(list []map[string]interface{}, entryID, userID int) map[string]interface{} {
	for _, item := range list {
		e, okE := item["entry_id"].(float64)
		u, okU := item["user_id"].(float64)
		if okE && okU && int(e) == entryID && int(u) == userID {
			return item
		}
	}
	return nil
}
