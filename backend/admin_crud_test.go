package main

// CRUD for the admin catalog routes (BOG-55).
//
// Eleven routes share one shape: GET lists, POST creates and answers an object
// with an id, PUT updates from a body carrying that id, DELETE takes it as
// ?id=N. The old lamsza suite wrote that round trip out by hand per route, and
// the routes added after it was written never got one.
//
// So it is a table here. One case per route, each naming the field its update
// moves, and the round trip is asserted against the list the admin SPA actually
// reads - create, find, update, see the change, delete, no longer find it.
// A handler that answered 200 and wrote nothing passes a status-code test and
// fails this one.
//
// Routes whose shape is genuinely different (entries, events, the singleton
// settings/pages documents, the county and location updates, the two image
// uploads) are not in the table; they have their own tests.

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"backend/internal/db"
)

// crudCase describes one route's round trip.
type crudCase struct {
	// path is the admin route and the name of the subtest.
	path string
	// create builds the POST body. It is a func, not a literal, because most
	// cases need an id looked up from the shared database - and looking those
	// up at table-build time would run before requireDB.
	create func(t *testing.T) map[string]interface{}
	// updateField is the field PUT moves, and updateValue is what it moves to.
	// The assertion reads it back off the list, so it must be a field the list
	// returns under the same name the body sends.
	updateField string
	updateValue string
	// createStatus is what POST answers. Zero means 200. The routes are split
	// between 200 and 201 and that is not worth changing from a test - but it
	// has to be written down, because "201" is indistinguishable from a broken
	// handler to anyone reading only mustOK.
	createStatus int
	// identityField names a field to find the created row by, for the routes
	// whose POST does not answer with the new id. Empty means the id comes
	// straight out of the create response.
	identityField string
	// cleanupSQL deletes the created row by id, so a failed assertion still
	// leaves the shared database as it found it.
	cleanupSQL string
}

func crudCases() []crudCase {
	return []crudCase{
		{
			path:         "/api/admin/tags",
			create:       func(t *testing.T) map[string]interface{} { return map[string]interface{}{"name": testName("tag")} },
			createStatus: http.StatusCreated,
			updateField:  "name",
			updateValue:  testName("tag-renamed"),
			cleanupSQL:   `DELETE FROM tags WHERE id = $1`,
		},
		{
			path: "/api/admin/settlement_location_types",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{"slug": testName("loctype"), "label_hu": "BOG-55 Településtípus", "sort_order": 900}
			},
			updateField: "label_hu",
			updateValue: "BOG-55 Településtípus átnevezve",
			cleanupSQL:  `DELETE FROM settlement_location_types WHERE id = $1`,
		},
		{
			path: "/api/admin/weather_translations",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{"source_text": testName("weather"), "lang": "hu", "translated_text": "BOG-55 fordítás"}
			},
			createStatus: http.StatusCreated,
			updateField:  "translated_text",
			updateValue:  "BOG-55 fordítás javítva",
			cleanupSQL:   `DELETE FROM weather_translations WHERE id = $1`,
		},
		{
			path: "/api/admin/catalog_event_types",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{"slug": testName("evtype"), "label_hu": "BOG-55 Esemény típus", "sort_order": 900}
			},
			updateField: "label_hu",
			updateValue: "BOG-55 Esemény típus átnevezve",
			cleanupSQL:  `DELETE FROM catalog_event_types WHERE id = $1`,
		},
		{
			path: "/api/admin/catalog_event_subtypes",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"event_type_id": someCatalogEventTypeID(t),
					"slug":          testName("evsubtype"),
					"label_hu":      "BOG-55 Altípus",
					"sort_order":    900,
				}
			},
			updateField: "label_hu",
			updateValue: "BOG-55 Altípus átnevezve",
			cleanupSQL:  `DELETE FROM catalog_event_subtypes WHERE id = $1`,
		},
		{
			// venue_types derives its slug from label_hu and answers 201 with no
			// body, so the created row has to be found by its label.
			path: "/api/admin/venue_types",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{"label_hu": testName("venuetype")}
			},
			createStatus:  http.StatusCreated,
			identityField: "label_hu",
			updateField:   "label_hu",
			updateValue:   testName("venuetype-renamed"),
			cleanupSQL:    `DELETE FROM venue_types WHERE id = $1`,
		},
		{
			path: "/api/admin/venues",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"settlement_id": someSettlementID(t),
					"name":          testName("venue"),
					"kind":          "egyeb",
				}
			},
			updateField: "name",
			updateValue: testName("venue-renamed"),
			cleanupSQL:  `DELETE FROM venues WHERE id = $1`,
		},
		{
			path: "/api/admin/news_feeds",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"title":    testName("feed"),
					"feed_url": "https://" + testName("feed") + ".test/rss.xml",
					"bg_color": "#ff00ff",
				}
			},
			updateField: "title",
			updateValue: testName("feed-renamed"),
			cleanupSQL:  `DELETE FROM news_feeds WHERE id = $1`,
		},
		{
			path: "/api/admin/mondasok",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{"text": testName("mondas"), "display_date": "2099-01-01"}
			},
			updateField: "text",
			updateValue: testName("mondas-edited"),
			cleanupSQL:  `DELETE FROM mondasok WHERE id = $1`,
		},
		{
			path: "/api/admin/quick_links",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"title":    testName("quicklink"),
					"url":      "https://" + testName("quicklink") + ".test",
					"bg_color": "#00ff00",
				}
			},
			updateField: "title",
			updateValue: testName("quicklink-renamed"),
			cleanupSQL:  `DELETE FROM quick_links WHERE id = $1`,
		},
		{
			path: "/api/admin/entry_categories",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"name":       testName("category"),
					"parent_id":  topLevelCategoryID(t),
					"sort_order": 900,
				}
			},
			updateField: "name",
			updateValue: testName("category-renamed"),
			cleanupSQL:  `DELETE FROM entry_categories WHERE id = $1`,
		},
		{
			// The directory entry itself - the row the public app's search and
			// profile pages read. Its POST is also what createAdminEntry uses as
			// a fixture, so this case is what makes that fixture trustworthy.
			path: "/api/admin/entries",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"name":        testName("entry"),
					"location_id": someSettlementID(t),
					"type":        "Vállalkozás",
					"category_id": leafCategoryID(t),
					"phone":       "0700-000-000",
					"address":     "BOG-55 Test Address 1",
					"notes":       "BOG-55 entry",
					"languages":   []string{"HU"},
				}
			},
			updateField: "name",
			updateValue: testName("entry-renamed"),
			cleanupSQL:  `DELETE FROM entries WHERE id = $1`,
		},
		{
			// Events are keyed on a catalog type and a settlement, and every
			// required field is validated - see TestAdminEventsRejectIncomplete.
			path: "/api/admin/events",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"title":         testName("event"),
					"location_id":   someSettlementID(t),
					"start_date":    "2099-07-01",
					"start_time":    "18:00",
					"end_date":      "2099-07-01",
					"end_time":      "20:00",
					"event_type_id": someCatalogEventTypeID(t),
					"organizer":     "BOG-55",
					"entry_price":   "ingyenes",
				}
			},
			updateField: "title",
			updateValue: testName("event-renamed"),
			cleanupSQL:  `DELETE FROM events WHERE id = $1`,
		},
		{
			// Attractions are addressed by county *slug* rather than county id,
			// which is why the create body carries one.
			path: "/api/admin/attractions",
			create: func(t *testing.T) map[string]interface{} {
				return map[string]interface{}{
					"county_slug": someCountySlug(t),
					"name":        testName("attraction"),
					"description": "BOG-55 attraction",
					"activities":  []string{"BOG-55 activity"},
				}
			},
			updateField: "name",
			updateValue: testName("attraction-renamed"),
			cleanupSQL:  `DELETE FROM attractions WHERE id = $1`,
		},
	}
}

func TestAdminCatalogCRUD(t *testing.T) {
	requireDB(t)

	for _, tc := range crudCases() {
		t.Run(tc.path, func(t *testing.T) {
			body := tc.create(t)

			wantCreate := tc.createStatus
			if wantCreate == 0 {
				wantCreate = http.StatusOK
			}
			rr := asAdmin(t, http.MethodPost, tc.path, body)
			mustStatus(t, rr, wantCreate, "POST "+tc.path)

			var id int
			if tc.identityField == "" {
				id = idOf(t, decodeObject(t, rr, "POST "+tc.path), "POST "+tc.path)
			} else {
				want := fmt.Sprint(body[tc.identityField])
				row := findInAdminListBy(t, tc.path, tc.identityField, want)
				if row == nil {
					t.Fatalf("POST %s answered %d, but GET does not list a row with %s = %q",
						tc.path, rr.Code, tc.identityField, want)
				}
				id = idOf(t, row, "POST "+tc.path)
			}

			// The row goes away even if an assertion below fails, so a red run
			// does not poison the next one. See the isolation rule in
			// testsupport_test.go.
			t.Cleanup(func() {
				if tc.cleanupSQL != "" {
					db.DB.Exec(tc.cleanupSQL, id)
				}
			})

			if findInAdminList(t, tc.path, id) == nil {
				t.Fatalf("POST %s created id %d, but GET does not list it", tc.path, id)
			}

			// PUT moves one field. The body carries every field the create did,
			// because these handlers UPDATE every column rather than patching:
			// sending the field alone would blank the rest.
			update := map[string]interface{}{"id": id}
			for k, v := range body {
				update[k] = v
			}
			update[tc.updateField] = tc.updateValue
			mustOK(t, asAdmin(t, http.MethodPut, tc.path, update), "PUT "+tc.path)

			updated := findInAdminList(t, tc.path, id)
			if updated == nil {
				t.Fatalf("row %d vanished from %s after PUT", id, tc.path)
			}
			if got := fmt.Sprint(updated[tc.updateField]); got != tc.updateValue {
				t.Errorf("PUT %s: %s = %q, want %q (the handler answered 200 and stored nothing)",
					tc.path, tc.updateField, got, tc.updateValue)
			}

			mustOK(t, asAdmin(t, http.MethodDelete, tc.path+"?id="+strconv.Itoa(id), nil), "DELETE "+tc.path)
			if findInAdminList(t, tc.path, id) != nil {
				t.Errorf("row %d is still listed by %s after DELETE returned 200", id, tc.path)
			}
		})
	}
}

// TestAdminCatalogRejectsEmptyRequiredFields is the negative half of the table:
// every one of these routes validates its own required fields, and a route that
// stopped would start writing blank rows the SPA renders as empty buttons.
//
// It is a separate test rather than a branch of the loop above so a validation
// failure reads as a validation failure, not as a broken round trip.
func TestAdminCatalogRejectsEmptyRequiredFields(t *testing.T) {
	requireDB(t)

	for _, tc := range []struct {
		path string
		body map[string]interface{}
		what string
	}{
		{"/api/admin/tags", map[string]interface{}{"name": "   "}, "a blank tag name"},
		{"/api/admin/settlement_location_types", map[string]interface{}{"slug": "", "label_hu": ""}, "a blank settlement location type"},
		{"/api/admin/catalog_event_types", map[string]interface{}{"slug": "", "label_hu": ""}, "a blank event type"},
		{"/api/admin/venue_types", map[string]interface{}{"slug": "", "label_hu": ""}, "a blank venue type"},
		{"/api/admin/venues", map[string]interface{}{"name": "", "settlement_id": 0}, "a venue with no settlement"},
		{"/api/admin/mondasok", map[string]interface{}{"text": "", "display_date": ""}, "a mondás with no text or date"},
		{"/api/admin/mondasok", map[string]interface{}{"text": "x", "display_date": "not-a-date"}, "a mondás with an unparseable date"},
	} {
		t.Run(tc.what, func(t *testing.T) {
			mustStatus(t, asAdmin(t, http.MethodPost, tc.path, tc.body), http.StatusBadRequest, "POST "+tc.path+" with "+tc.what)
		})
	}
}

// TestAdminCatalogEventTypeDeleteIsRefusedWhileInUse covers the one DELETE in
// the table's routes that is allowed to say no. Deleting a type an event uses
// would orphan the event, so the handler counts first and answers 409.
func TestAdminCatalogEventTypeDeleteIsRefusedWhileInUse(t *testing.T) {
	requireDB(t)

	// Put an event on a type ourselves rather than hoping the database already
	// has one: the scratch database starts with no events.
	eventID := createTestEvent(t, "type-in-use")
	var typeID, eventCount int
	err := db.DB.QueryRow(`
		SELECT e.event_type_id, (SELECT COUNT(*) FROM events WHERE event_type_id = e.event_type_id)
		FROM events e WHERE e.id = $1`, eventID).Scan(&typeID, &eventCount)
	if err != nil {
		t.Fatalf("read the test event's type: %v", err)
	}

	rr := asAdmin(t, http.MethodDelete, "/api/admin/catalog_event_types?id="+strconv.Itoa(typeID), nil)
	mustStatus(t, rr, http.StatusConflict,
		fmt.Sprintf("DELETE a catalog_event_type used by %d events", eventCount))

	var stillThere bool
	if err := db.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM catalog_event_types WHERE id = $1)`, typeID).Scan(&stillThere); err != nil {
		t.Fatalf("re-read catalog_event_types %d: %v", typeID, err)
	}
	if !stillThere {
		t.Fatalf("catalog_event_type %d was deleted despite the 409", typeID)
	}
}

// TestAdminEntryTypesIsAClosedList pins what /api/admin/entry_types is: a read
// of a fixed set, not a CRUD route. The entry write path validates against it,
// so a test that assumed CRUD here would be asserting a feature that must not
// exist.
func TestAdminEntryTypesIsAClosedList(t *testing.T) {
	requireDB(t)

	types := decodeArray(t, asAdmin(t, http.MethodGet, "/api/admin/entry_types", nil), "GET /api/admin/entry_types")
	if len(types) == 0 {
		t.Fatal("entry_types is empty; the entry write path validates against it and would refuse every entry")
	}
	for _, item := range types {
		if name, _ := item["name"].(string); name == "" {
			t.Errorf("entry_type %#v has no name", item)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// adminList GETs one of the array routes above.
func adminList(t *testing.T, path string) []map[string]interface{} {
	t.Helper()
	rr := asAdmin(t, http.MethodGet, path, nil)
	mustOK(t, rr, "GET "+path)
	return decodeArray(t, rr, "GET "+path)
}

// findInAdminList returns the row with the given id, or nil.
func findInAdminList(t *testing.T, path string, id int) map[string]interface{} {
	t.Helper()
	return findByID(adminList(t, path), id)
}

// findInAdminListBy returns the row whose field equals want, for the routes
// whose POST does not hand back an id.
func findInAdminListBy(t *testing.T, path, field, want string) map[string]interface{} {
	t.Helper()
	for _, row := range adminList(t, path) {
		if fmt.Sprint(row[field]) == want {
			return row
		}
	}
	return nil
}

// someCountySlug returns an existing county slug. Attractions and the county
// routes address a county by slug, not by id.
func someCountySlug(t *testing.T) string {
	t.Helper()
	var slug string
	if err := db.DB.QueryRow(`SELECT slug FROM counties ORDER BY id ASC LIMIT 1`).Scan(&slug); err != nil {
		t.Skipf("no counties row: %v", err)
	}
	return slug
}

// someCatalogEventTypeID returns an existing event type to hang a subtype off.
func someCatalogEventTypeID(t *testing.T) int {
	t.Helper()
	var id int
	if err := db.DB.QueryRow(`SELECT id FROM catalog_event_types ORDER BY id ASC LIMIT 1`).Scan(&id); err != nil {
		t.Skipf("no catalog_event_types row: %v", err)
	}
	return id
}
