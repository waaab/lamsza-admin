package main

// The validation the admin write surface owes the public app (BOG-55).
//
// Three write paths reject more than a missing field, and each rejection is the
// only thing standing between the admin UI and a row the public app cannot
// render: an entry filed under a branch category instead of a leaf, an event
// missing a date, a schedule day attached to an event that is gone.
//
// These are the cases the old lamsza suite covered in directory_catalog_test.go
// and lost on BOG-42.

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"backend/internal/db"
)

// TestAdminEntriesRejectNonLeafCategory is the directory-catalog rule: an entry
// hangs off a leaf, never off a branch. The public app's category pages list
// leaves, so an entry on a branch is invisible there while looking fine in the
// admin.
func TestAdminEntriesRejectNonLeafCategory(t *testing.T) {
	requireDB(t)

	rr := asAdmin(t, http.MethodPost, "/api/admin/entries", map[string]interface{}{
		"name":        testName("entry-branch"),
		"location_id": someSettlementID(t),
		"type":        "Vállalkozás",
		"category_id": topLevelCategoryID(t),
	})
	mustStatus(t, rr, http.StatusBadRequest, "POST an entry under a branch category")

	// The refusal is plain text here and names the rule - unlike the websites
	// route, which answers a JSON {"error","field"} the SPA can attach to an
	// input (see TestWebsiteModerationApproveWithCategory). The two shapes are
	// pinned so the difference is a decision somebody made rather than a
	// surprise the next person debugs twice.
	if got := strings.TrimSpace(rr.Body.String()); !strings.Contains(got, "subcategory") {
		t.Errorf("refusal reads %q; it should name the leaf rule", got)
	}
}

// TestAdminEntriesRejectUnknownType pins the other closed list the entry write
// path validates against. entry_types is not editable from the admin
// (TestAdminEntryTypesIsAClosedList), so an unknown type is always a client bug.
func TestAdminEntriesRejectUnknownType(t *testing.T) {
	requireDB(t)

	rr := asAdmin(t, http.MethodPost, "/api/admin/entries", map[string]interface{}{
		"name":        testName("entry-badtype"),
		"location_id": someSettlementID(t),
		"type":        "BOG-55 Nem létező típus",
		"category_id": leafCategoryID(t),
	})
	if rr.Code == http.StatusOK {
		t.Fatalf("POST an entry with an unknown type was accepted: %s", rr.Body.String())
	}
}

// TestAdminEntriesCarryTheirCategorySet covers the multi-category write that
// BOG-42's directory work introduced: category_id is the primary one and
// category_ids is the whole set, and both have to land.
func TestAdminEntriesCarryTheirCategorySet(t *testing.T) {
	requireDB(t)

	primary := leafCategoryID(t)
	extra := secondLeafCategoryID(t, primary)

	rr := asAdmin(t, http.MethodPost, "/api/admin/entries", map[string]interface{}{
		"name":         testName("entry-catset"),
		"location_id":  someSettlementID(t),
		"type":         "Vállalkozás",
		"category_id":  primary,
		"category_ids": []int{primary, extra},
	})
	mustOK(t, rr, "POST an entry with two categories")
	id := idOf(t, decodeObject(t, rr, "POST an entry with two categories"), "POST an entry with two categories")
	t.Cleanup(func() { deleteEntry(id) })

	got := entryCategoryIDs(t, id)
	if len(got) != 2 || !containsInt(got, primary) || !containsInt(got, extra) {
		t.Fatalf("entry_category_links for %d = %v, want both %d and %d", id, got, primary, extra)
	}

	// Narrowing the set removes the link rather than leaving an orphan: the
	// handler replaces the set, it does not add to it.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/entries", map[string]interface{}{
		"id":           id,
		"name":         testName("entry-catset"),
		"location_id":  someSettlementID(t),
		"type":         "Vállalkozás",
		"category_id":  primary,
		"category_ids": []int{primary},
	}), "PUT the entry with one category")

	if got := entryCategoryIDs(t, id); len(got) != 1 || got[0] != primary {
		t.Errorf("entry_category_links for %d = %v after narrowing, want [%d]", id, got, primary)
	}
}

// TestAdminEventsRejectIncomplete walks every required field. An event with a
// missing date or time renders as a blank row on the public events page, and
// the only guard is this handler.
func TestAdminEventsRejectIncomplete(t *testing.T) {
	requireDB(t)

	complete := func() map[string]interface{} {
		return map[string]interface{}{
			"title":         testName("event-validation"),
			"location_id":   someSettlementID(t),
			"start_date":    "2099-07-01",
			"start_time":    "18:00",
			"end_date":      "2099-07-01",
			"end_time":      "20:00",
			"event_type_id": someCatalogEventTypeID(t),
		}
	}

	for _, field := range []string{"title", "location_id", "start_date", "start_time", "end_date", "end_time", "event_type_id"} {
		t.Run("without "+field, func(t *testing.T) {
			body := complete()
			delete(body, field)
			mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/events", body),
				http.StatusBadRequest, "POST an event with no "+field)
		})
	}

	t.Run("with an unknown event_type slug", func(t *testing.T) {
		body := complete()
		delete(body, "event_type_id")
		body["event_type"] = "bog55-nincs-ilyen-tipus"
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/events", body),
			http.StatusBadRequest, "POST an event with an unknown event_type slug")
	})

	t.Run("with a venue in another settlement", func(t *testing.T) {
		// EnsureVenueBelongsToSettlement is the cross-table rule: an event's
		// default venue must be in the event's own settlement, or the public
		// page shows a venue in the wrong town.
		venueID, venueSettlement := someVenueWithSettlement(t)
		other := settlementOtherThan(t, venueSettlement)

		body := complete()
		body["location_id"] = other
		body["default_venue_id"] = venueID
		mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/events", body),
			http.StatusBadRequest, "POST an event whose venue is in another settlement")
	})
}

// TestAdminEventScheduleWrite covers the PUT half of the schedule route: it
// replaces the whole schedule for one event, which is both the feature and the
// risk - a wrong event_id would wipe a different event's programme.
func TestAdminEventScheduleWrite(t *testing.T) {
	requireDB(t)

	eventID := createTestEvent(t, "schedule")
	other := createTestEvent(t, "schedule-bystander")

	// Give the bystander a schedule first, so a PUT that deleted too much would
	// show up as its programme disappearing.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/events/schedule", scheduleBody(other, "BOG-55 bystander day")),
		"PUT the bystander's schedule")

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/events/schedule", scheduleBody(eventID, "BOG-55 első nap")),
		"PUT a schedule")

	days := scheduleDays(t, eventID)
	if len(days) != 1 {
		t.Fatalf("event %d has %d schedule days, want 1", eventID, len(days))
	}
	if got := days[0]["notes"]; got != "BOG-55 első nap" {
		t.Errorf("schedule day notes = %#v, want %q", got, "BOG-55 első nap")
	}
	acts, _ := days[0]["activities"].([]interface{})
	if len(acts) != 1 {
		t.Fatalf("schedule day has %d activities, want 1: %#v", len(acts), days[0]["activities"])
	}
	if act, _ := acts[0].(map[string]interface{}); act["title"] != "BOG-55 program" {
		t.Errorf("activity = %#v, want title BOG-55 program", acts[0])
	}

	// A second PUT replaces rather than appends.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/events/schedule", scheduleBody(eventID, "BOG-55 átírt nap")),
		"PUT the schedule again")
	days = scheduleDays(t, eventID)
	if len(days) != 1 {
		t.Fatalf("event %d has %d schedule days after a second PUT, want 1 (it replaces, not appends)", eventID, len(days))
	}
	if got := days[0]["notes"]; got != "BOG-55 átírt nap" {
		t.Errorf("schedule day notes = %#v after the second PUT, want the new value", got)
	}

	// And it touched nothing else.
	if n := len(scheduleDays(t, other)); n != 1 {
		t.Errorf("the bystander event now has %d schedule days, want 1; the PUT reached outside its own event", n)
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/events/schedule", map[string]interface{}{"days": []interface{}{}}),
		http.StatusBadRequest, "PUT a schedule with no event_id")
	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/events/schedule",
		map[string]interface{}{"event_id": 2147483646, "days": []interface{}{}}),
		http.StatusNotFound, "PUT a schedule for an event that does not exist")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/events/schedule", nil),
		http.StatusMethodNotAllowed, "POST to the schedule route")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func scheduleBody(eventID int, notes string) map[string]interface{} {
	return map[string]interface{}{
		"event_id": eventID,
		"days": []map[string]interface{}{{
			"schedule_date": "2099-07-01",
			"notes":         notes,
			"activities": []map[string]interface{}{{
				"activity_type": "program",
				"starts_at":     "18:00",
				"ends_at":       "19:00",
				"title":         "BOG-55 program",
				"sort_order":    0,
			}},
		}},
	}
}

func scheduleDays(t *testing.T, eventID int) []map[string]interface{} {
	t.Helper()
	rr := asAdmin(t, http.MethodGet, "/api/admin/events/schedule?event_id="+strconv.Itoa(eventID), nil)
	mustOK(t, rr, "GET the schedule")
	raw, _ := decodeObject(t, rr, "GET the schedule")["days"].([]interface{})
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

func entryCategoryIDs(t *testing.T, entryID int) []int {
	t.Helper()
	rows, err := db.DB.Query(
		`SELECT category_id FROM entry_category_links WHERE entry_id = $1 ORDER BY category_id`, entryID)
	if err != nil {
		t.Fatalf("read entry_category_links for %d: %v", entryID, err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan entry_category_links: %v", err)
		}
		out = append(out, id)
	}
	return out
}

// secondLeafCategoryID returns a leaf category other than the one given, so the
// multi-category test has a genuinely different id to add.
func secondLeafCategoryID(t *testing.T, notThis int) int {
	t.Helper()
	var id int
	err := db.DB.QueryRow(`
		SELECT c.id FROM entry_categories c
		WHERE c.parent_id IS NOT NULL AND c.id <> $1
		  AND NOT EXISTS (SELECT 1 FROM entry_categories k WHERE k.parent_id = c.id)
		ORDER BY c.id ASC LIMIT 1`, notThis).Scan(&id)
	if err != nil {
		t.Skipf("only one leaf entry_categories row: %v", err)
	}
	return id
}

// someVenueWithSettlement creates a venue for this test, through the admin API,
// and returns it with its settlement. The scratch database starts with none.
func someVenueWithSettlement(t *testing.T) (venueID, settlementID int) {
	t.Helper()
	settlementID = someSettlementID(t)
	rr := asAdmin(t, http.MethodPost, "/api/admin/venues", map[string]interface{}{
		"settlement_id": settlementID,
		"name":          testName("venue-for-event"),
		"kind":          "egyeb",
	})
	mustOK(t, rr, "POST a venue")
	venueID = idOf(t, decodeObject(t, rr, "POST a venue"), "POST a venue")
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM venues WHERE id = $1`, venueID) })
	return venueID, settlementID
}

func settlementOtherThan(t *testing.T, notThis int) int {
	t.Helper()
	var id int
	if err := db.DB.QueryRow(
		`SELECT id FROM settlements WHERE id <> $1 ORDER BY id ASC LIMIT 1`, notThis).Scan(&id); err != nil {
		t.Skipf("only one settlements row: %v", err)
	}
	return id
}

func containsInt(list []int, want int) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
