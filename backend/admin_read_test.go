package main

// Happy-path reads for every admin route that answers GET, plus the ledger that
// makes "every routed admin Handle* has a happy-path test" checkable instead of
// a thing somebody once asserted in a task comment (BOG-55).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"backend/internal/auth"
	"backend/internal/db"
)

// adminReads lists every admin route whose happy path is a GET, with the JSON
// shape it must answer in. The shape is part of the contract: the admin SPA
// reads `.websites` off one and iterates the other, so a handler that starts
// returning an array where an object was is a break even at 200.
var adminReads = []struct {
	path   string
	object bool // true: a JSON object; false: a JSON array
	// key, when set, must be present in the object - the field the SPA reads.
	key string
}{
	{path: "/api/admin/listing-queue", object: true, key: "unpublished"},
	{path: "/api/admin/websites", object: true, key: "websites"},
	{path: "/api/admin/entries"},
	{path: "/api/admin/entry_categories"},
	{path: "/api/admin/entry_types"},
	{path: "/api/admin/tags"},
	{path: "/api/admin/locations"},
	{path: "/api/admin/settlement_location_types"},
	{path: "/api/admin/dashboard_stats", object: true, key: "entries"},
	{path: "/api/admin/users"},
	{path: "/api/admin/attraction-suggestions"},
	{path: "/api/admin/attractions"},
	{path: "/api/admin/settings", object: true},
	{path: "/api/admin/pages"},
	{path: "/api/admin/page_faq"},
	{path: "/api/admin/weather_translations"},
	{path: "/api/admin/events"},
	{path: "/api/admin/catalog_event_types"},
	{path: "/api/admin/catalog_event_subtypes"},
	{path: "/api/admin/venues"},
	{path: "/api/admin/venue_types"},
	{path: "/api/admin/news_feeds"},
	{path: "/api/admin/quick_links"},
}

func TestAdminReadsAnswerTheirShape(t *testing.T) {
	requireDB(t)
	for _, tc := range adminReads {
		t.Run(tc.path, func(t *testing.T) {
			rr := asAdmin(t, http.MethodGet, tc.path, nil)
			mustOK(t, rr, "GET "+tc.path)

			if rr.Header().Get("Access-Control-Allow-Origin") != testOrigin {
				t.Errorf("GET %s: missing or wrong CORS header %q",
					tc.path, rr.Header().Get("Access-Control-Allow-Origin"))
			}

			if tc.object {
				obj := decodeObject(t, rr, "GET "+tc.path)
				if tc.key != "" {
					if _, ok := obj[tc.key]; !ok {
						t.Errorf("GET %s: response has no %q: keys %v", tc.path, tc.key, objectKeys(obj))
					}
				}
				return
			}
			// An array route must answer `[]` and never `null`: the SPA maps
			// over it without a guard.
			if strings.TrimSpace(rr.Body.String()) == "null" {
				t.Fatalf("GET %s: answered null; an empty list must be []", tc.path)
			}
			decodeArray(t, rr, "GET "+tc.path)
		})
	}
}

// TestAdminEventScheduleRead covers the one GET route that needs a parameter.
// Without event_id it is a documented 400, which is why the sweep above cannot
// carry it.
func TestAdminEventScheduleRead(t *testing.T) {
	requireDB(t)

	rr := asAdmin(t, http.MethodGet, "/api/admin/events/schedule", nil)
	mustStatus(t, rr, http.StatusBadRequest, "GET /api/admin/events/schedule with no event_id")

	// Read the schedule of an event this test created: the scratch database
	// starts with no events, and someone else's event is not ours to rely on.
	id := float64(createTestEvent(t, "schedule-read"))
	events := decodeArray(t, asAdmin(t, http.MethodGet, "/api/admin/events", nil), "GET /api/admin/events")
	if findByID(events, int(id)) == nil {
		t.Fatalf("GET /api/admin/events does not list the event %d this test just created", int(id))
	}

	rr = asAdmin(t, http.MethodGet, "/api/admin/events/schedule?event_id="+strconv.Itoa(int(id)), nil)
	mustOK(t, rr, "GET /api/admin/events/schedule?event_id")

	// The schedule is {"event_id":N,"days":[…]} - an object, not a bare array.
	// It echoes the id back, and the SPA maps over .days without a nil guard, so
	// both halves are part of the contract.
	schedule := decodeObject(t, rr, "GET /api/admin/events/schedule")
	if got, ok := schedule["event_id"].(float64); !ok || int(got) != int(id) {
		t.Errorf("schedule event_id = %#v, want %d", schedule["event_id"], int(id))
	}
	if _, ok := schedule["days"].([]interface{}); !ok {
		t.Errorf("schedule days = %#v, want an array (an empty schedule must be [], never null)", schedule["days"])
	}
}

// TestPublicReadsUsedByTheAdminSPA covers the open routes the admin frontend
// calls. They are not behind RequireAdmin by design, so the guard sweep skips
// them and they would otherwise have no test at all.
func TestPublicReadsUsedByTheAdminSPA(t *testing.T) {
	requireDB(t)
	for _, path := range []string{"/api/counties", "/api/historical_seats", "/api/venues", "/api/venue_types"} {
		t.Run(path, func(t *testing.T) {
			rr := anon(t, http.MethodGet, path, nil)
			mustOK(t, rr, "GET "+path)
			if strings.TrimSpace(rr.Body.String()) == "null" {
				t.Fatalf("GET %s: answered null; an empty list must be []", path)
			}
			decodeArray(t, rr, "GET "+path)
		})
	}

	rr := anon(t, http.MethodGet, "/api/config/public", nil)
	mustOK(t, rr, "GET /api/config/public")
	decodeObject(t, rr, "GET /api/config/public")

	rr = anon(t, http.MethodGet, "/api/health", nil)
	mustOK(t, rr, "GET /api/health")
	health := decodeObject(t, rr, "GET /api/health")
	if health["ok"] != true {
		t.Errorf("GET /api/health: ok = %#v, want true", health["ok"])
	}
}

// TestAdminAuthRoutes covers the three /api/auth routes. They are open so the
// SPA can sign in, and HandleMe/HandleLogout are the rest of that path.
func TestAdminAuthRoutes(t *testing.T) {
	requireDB(t)
	cookie := adminCookie(t)

	rr := serve(t, http.MethodGet, "/api/auth/me", nil, cookie)
	mustOK(t, rr, "GET /api/auth/me as admin")
	me := decodeObject(t, rr, "GET /api/auth/me")
	if !strings.EqualFold(fmt.Sprint(me["email"]), testAdminEmail) {
		t.Errorf("GET /api/auth/me: email = %#v, want %s", me["email"], testAdminEmail)
	}

	// Logging out must invalidate the session row, not only clear the cookie:
	// a token the client still holds has to stop working server-side.
	raw, hash := mintSessionToken(t)
	userID, err := userIDByEmail(testAdminEmail)
	if err != nil {
		t.Fatalf("admin user: %v", err)
	}
	if _, err := db.DB.Exec(
		`INSERT INTO admin_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 hour')`,
		hash, userID,
	); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash = $1`, hash) })
	throwaway := &http.Cookie{Name: auth.SessionCookieName, Value: raw}

	mustOK(t, serve(t, http.MethodGet, "/api/auth/me", nil, throwaway), "GET /api/auth/me with the throwaway session")
	mustOK(t, serve(t, http.MethodPost, "/api/auth/logout", nil, throwaway), "POST /api/auth/logout")
	mustStatus(t, serve(t, http.MethodGet, "/api/auth/me", nil, throwaway),
		http.StatusUnauthorized, "GET /api/auth/me after logout")

	// A bad credential is refused rather than minting a session.
	mustStatus(t, anon(t, http.MethodPost, "/api/auth/google", map[string]string{"credential": "not-a-test-token"}),
		http.StatusUnauthorized, "POST /api/auth/google with a bad credential")

	// And an account that is not on the allowlist is refused at sign-in - the
	// BOG-45 rule that keeps admin_sessions admin-only.
	mustStatus(t, anon(t, http.MethodPost, "/api/auth/google", map[string]string{"credential": "test:" + testNonAdminEmail}),
		http.StatusForbidden, "POST /api/auth/google as a non-allowlisted account")
}

// ---------------------------------------------------------------------------
// The coverage ledger
// ---------------------------------------------------------------------------

// adminHappyPath names, for every admin route, the tests that cover its happy
// path. TestEveryAdminRouteHasAHappyPath turns the acceptance criterion into a
// failing build: add a route and the suite goes red until it has a test and an
// entry here.
//
// It is a map of route -> test names rather than a count, so the failure
// message tells the next person what is missing instead of that a number moved.
// And the names are checked against the functions that actually exist in this
// package - a ledger nobody verifies is a list of claims, which is what the old
// "every handler is covered" note in the task history turned out to be.
var adminHappyPath = map[string][]string{
	"/api/admin/listing-queue":                {"TestAdminReadsAnswerTheirShape", "TestListingQueueShowsEveryPendingKind"},
	"/api/admin/listing-queue/publish":        {"TestListingQueuePublish"},
	"/api/admin/listing-queue/member":         {"TestListingQueueMemberApproveAndReject"},
	"/api/admin/listing-queue/claim":          {"TestListingQueueClaimAcceptAndDeny"},
	"/api/admin/listing-queue/suggestion":     {"TestListingQueueSuggestionAcceptAndDeny"},
	"/api/admin/entry_types":                  {"TestAdminReadsAnswerTheirShape", "TestAdminEntryTypesIsAClosedList"},
	"/api/admin/county_seat":                  {"TestAdminSetCountySeat"},
	"/api/admin/dashboard_stats":              {"TestAdminReadsAnswerTheirShape"},
	"/api/admin/users":                        {"TestAdminReadsAnswerTheirShape", "TestAdminUsersListsTheSignedInAdmin"},
	"/api/admin/audit-log":                    {"TestAdminAuditLogRecordsAnAdminWrite"},
	"/api/admin/news_feeds/check":             {"TestAdminNewsFeedCheck"},
	"/api/admin/entry-images":                 {"TestAdminEntryImageUpload"},
	"/api/admin/attraction-suggestions":       {"TestAdminReadsAnswerTheirShape", "TestAdminAttractionSuggestionsRead"},
	"/api/admin/counties":                     {"TestAdminCountyUpdate"},
	"/api/admin/historical_seats":             {"TestAdminHistoricalSeatUpdate"},
	"/api/admin/locations":                    {"TestAdminReadsAnswerTheirShape", "TestAdminLocationsCRUD"},
	"/api/admin/event-images":                 {"TestAdminEventImageUpload", "TestAdminEventImageUploadReplacesThePrevious"},
	"/api/admin/events/schedule":              {"TestAdminEventScheduleRead", "TestAdminEventScheduleWrite"},
	"/api/admin/settings":                     {"TestAdminReadsAnswerTheirShape", "TestAdminSettingsUpdate"},
	"/api/admin/settings/clear-weather-cache": {"TestAdminClearWeatherCache"},
	"/api/admin/pages":                        {"TestAdminReadsAnswerTheirShape", "TestAdminPagesUpdate"},
	"/api/admin/page_faq":                     {"TestAdminReadsAnswerTheirShape", "TestAdminPageFAQUpdate"},
	"/api/admin/websites": {
		"TestAdminReadsAnswerTheirShape",
		"TestWebsiteModerationApprove",
		"TestWebsiteModerationApproveWithCategory",
		"TestWebsiteModerationReject",
		"TestWebsiteModerationBan",
	},
	"/api/admin/entries": {
		"TestAdminReadsAnswerTheirShape",
		"TestAdminCatalogCRUD",
		"TestAdminEntriesRejectNonLeafCategory",
		"TestAdminEntriesCarryTheirCategorySet",
	},
	"/api/admin/events": {
		"TestAdminReadsAnswerTheirShape",
		"TestAdminCatalogCRUD",
		"TestAdminEventsRejectIncomplete",
	},
	// The rest share the round trip in TestAdminCatalogCRUD's table, one case
	// each, plus the read sweep above.
	"/api/admin/entry_categories":          {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/entry_categories/featured": {"TestAdminFeaturedCategories"},
	"/api/admin/tags":                      {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/settlement_location_types": {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/attractions":               {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/weather_translations":      {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/catalog_event_types":       {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/catalog_event_subtypes":    {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/venues":                    {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/venue_types":               {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/news_feeds":                {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/quick_links":               {"TestAdminReadsAnswerTheirShape", "TestAdminCatalogCRUD"},
	"/api/admin/dictionary/":               {"TestRelayForwardsAReadWithTheServiceToken", "TestRelayPassesTheAppsErrorThrough"},
	"/api/admin/games/":                    {"TestRelayForwardsAWriteAndAuditsIt"},
}

func TestEveryAdminRouteHasAHappyPath(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range parseRoutes(t) {
		if !mustBeGated(r) {
			continue
		}
		seen[r.pattern] = true
		if len(adminHappyPath[r.pattern]) == 0 {
			t.Errorf("%s has no happy-path test. Write one, then name it in adminHappyPath in admin_read_test.go.", r.pattern)
		}
	}
	for pattern := range adminHappyPath {
		if !seen[pattern] {
			t.Errorf("adminHappyPath names %s, which is no longer a routed admin path; drop the entry.", pattern)
		}
	}

	// Every name in the ledger has to be a test function in this package.
	tests := testFuncsInPackage(t)
	for pattern, names := range adminHappyPath {
		for _, name := range names {
			if !tests[name] {
				t.Errorf("adminHappyPath says %s is covered by %s, and there is no such test function. Either write it or correct the entry.",
					pattern, name)
			}
		}
	}
}

// testFuncsInPackage returns the names of every `func TestX(*testing.T)` in the
// package's _test.go files, read from source. go/ast rather than reflection,
// because a test binary cannot enumerate its own tests.
func testFuncsInPackage(t *testing.T) map[string]bool {
	t.Helper()

	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no *_test.go files found; this check is looking in the wrong directory")
	}

	out := map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
				out[fn.Name.Name] = true
			}
		}
	}
	return out
}

func objectKeys(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys
}

// TestAdminAuditLogRecordsAnAdminWrite covers /api/admin/audit-log (BOG-48):
// an admin write through the real route table lands in the log, attributed to
// the signed-in admin, and the read side finds it by resource and id.
func TestAdminAuditLogRecordsAnAdminWrite(t *testing.T) {
	requireDB(t)

	rr := asAdmin(t, http.MethodPost, "/api/admin/tags", map[string]interface{}{"name": testName("audit-tag")})
	mustStatus(t, rr, http.StatusCreated, "POST a tag")
	tagID := idOf(t, decodeObject(t, rr, "POST a tag"), "POST a tag")
	// The audit row stays: the log is append-only even here (append_only_test.go
	// fails on any DELETE of it in this repo), and the scratch database is
	// recreated on every run.
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM tags WHERE id = $1`, tagID) })

	rr = asAdmin(t, http.MethodGet, "/api/admin/audit-log?resource=tag&resource_id="+strconv.Itoa(tagID), nil)
	mustOK(t, rr, "GET /api/admin/audit-log for the new tag")
	rows := decodeArray(t, rr, "GET /api/admin/audit-log")
	if len(rows) != 1 {
		t.Fatalf("audit log has %d rows for tag %d, want 1: %v", len(rows), tagID, rows)
	}
	got := rows[0]
	if got["action"] != "create" || got["method"] != http.MethodPost || got["actor_email"] != testAdminEmail {
		t.Errorf("audit row = action %v, method %v, actor %v; want create, POST, %s",
			got["action"], got["method"], got["actor_email"], testAdminEmail)
	}
}
