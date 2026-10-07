package main

// The admin routes that are not CRUD (BOG-55).
//
// Nine routes do not fit the table in admin_crud_test.go, each for its own
// reason: settings is a key/value document, pages and page_faq and the two
// geography routes are update-only (their rows are seeded, never created from
// the admin UI), county_seat is a single exclusive flag, users and
// attraction-suggestions are read-only, the two image routes take multipart,
// and clear-weather-cache is a command with no body at all.
//
// Each of them writes something the public app reads, so each is asserted on
// stored state and then put back the way it was.

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"backend/internal/db"
)

// ---------------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------------

// TestAdminSettingsUpdate writes one key and reads it back. The handler upserts
// whatever keys the body carries and leaves the rest alone, which is what makes
// it safe to test against the shared database - but only if that is true, and
// this is what checks it.
func TestAdminSettingsUpdate(t *testing.T) {
	requireDB(t)

	key := testName("setting")
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM site_settings WHERE key = $1`, key) })

	before := adminSettings(t)
	if _, exists := before[key]; exists {
		t.Fatalf("setting %q already exists; the test tag is not unique", key)
	}
	if len(before) == 0 {
		t.Fatal("GET /api/admin/settings returned nothing; the site has settings and this would hide a broken query")
	}

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/settings", map[string]string{key: "first"}), "PUT a new setting")
	if got := adminSettings(t)[key]; got != "first" {
		t.Fatalf("setting %q = %#v after PUT, want \"first\"", key, got)
	}

	// The second PUT is the upsert half: same key, new value, no duplicate row.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/settings", map[string]string{key: "second"}), "PUT the same setting again")
	after := adminSettings(t)
	if got := after[key]; got != "second" {
		t.Fatalf("setting %q = %#v after the second PUT, want \"second\"", key, got)
	}

	// Nothing else moved. A handler that replaced the document instead of
	// upserting would wipe the site's configuration, and no single-key
	// assertion above would notice.
	for k, v := range before {
		if after[k] != v {
			t.Errorf("PUT of one key changed unrelated setting %q from %q to %q", k, v, after[k])
		}
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/settings", "{not json"),
		http.StatusBadRequest, "PUT settings with a malformed body")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/settings", map[string]string{key: "x"}),
		http.StatusMethodNotAllowed, "POST to settings")
}

// TestAdminClearWeatherCache: the one route whose whole job is to bump a
// counter. The frontend keys its weather cache on that number, so a no-op here
// means the admin's "clear cache" button silently does nothing.
func TestAdminClearWeatherCache(t *testing.T) {
	requireDB(t)

	before := adminSettings(t)["weather_cache_version"]
	mustOK(t, asAdmin(t, http.MethodPost, "/api/admin/settings/clear-weather-cache", nil), "clear the weather cache")
	after := adminSettings(t)["weather_cache_version"]

	if after == before {
		t.Fatalf("weather_cache_version stayed at %q; clearing the cache must bump it", before)
	}
	n, err := strconv.Atoi(after)
	if err != nil {
		t.Fatalf("weather_cache_version = %q, which is not a number the frontend can compare", after)
	}
	if m, err := strconv.Atoi(before); err == nil && n != m+1 {
		t.Errorf("weather_cache_version went %d -> %d; it must increment by one", m, n)
	}

	mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/settings/clear-weather-cache", nil),
		http.StatusMethodNotAllowed, "GET the clear-weather-cache route")
}

func adminSettings(t *testing.T) map[string]string {
	t.Helper()
	rr := asAdmin(t, http.MethodGet, "/api/admin/settings", nil)
	mustOK(t, rr, "GET /api/admin/settings")
	obj := decodeObject(t, rr, "GET /api/admin/settings")
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		out[k] = fmt.Sprint(v)
	}
	return out
}

// ---------------------------------------------------------------------------
// pages and page_faq - update-only documents
// ---------------------------------------------------------------------------

// TestAdminPagesUpdate edits a page of its own rather than one of the site's.
// The route has no POST, so the row is inserted here; editing a real page would
// be touching policy-page content, which is deferred work and not this task's.
func TestAdminPagesUpdate(t *testing.T) {
	requireDB(t)

	slug := testName("page")
	var id int
	err := db.DB.QueryRow(`
		INSERT INTO pages (slug, title, greeting, content) VALUES ($1, $2, '', '') RETURNING id`,
		slug, testName("page-title"),
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert pages fixture: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM pages WHERE id = $1`, id) })

	page := findInAdminList(t, "/api/admin/pages", id)
	if page == nil {
		t.Fatalf("page %d is not in GET /api/admin/pages", id)
	}

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/pages", map[string]interface{}{
		"id": id, "title": "BOG-55 Page", "greeting": "BOG-55 greeting", "content": "BOG-55 body",
	}), "PUT a page")

	updated := findInAdminList(t, "/api/admin/pages", id)
	if updated == nil {
		t.Fatalf("page %d vanished after PUT", id)
	}
	for field, want := range map[string]string{
		"title": "BOG-55 Page", "greeting": "BOG-55 greeting", "content": "BOG-55 body",
	} {
		if got := fmt.Sprint(updated[field]); got != want {
			t.Errorf("page %s = %q, want %q", field, got, want)
		}
	}
	// The slug is not editable through this route - the public app routes on it.
	if got := fmt.Sprint(updated["slug"]); got != slug {
		t.Errorf("page slug changed to %q; PUT must not move it (was %q)", got, slug)
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/pages", map[string]interface{}{"title": "x"}),
		http.StatusBadRequest, "PUT a page with no id")
	mustStatus(t, asAdmin(t, http.MethodDelete, "/api/admin/pages", nil),
		http.StatusBadRequest, "DELETE a page with no id")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/pages", map[string]interface{}{"slug": "x"}),
		http.StatusMethodNotAllowed, "POST a page")

	mustOK(t, asAdmin(t, http.MethodDelete, "/api/admin/pages?id="+strconv.Itoa(id), nil), "DELETE the page")
	if findInAdminList(t, "/api/admin/pages", id) != nil {
		t.Errorf("page %d is still listed after DELETE", id)
	}
}

// TestAdminPageFAQUpdate covers the jsonb half: faq_items is a list of
// question/answer pairs stored as JSON, and the round trip through jsonb is the
// part that can break while the status code stays 200.
func TestAdminPageFAQUpdate(t *testing.T) {
	requireDB(t)

	sectionKey := testName("faq")
	var id int
	err := db.DB.QueryRow(`
		INSERT INTO page_faq_sections (section_key, label_hu, faq_title, faq_items)
		VALUES ($1, $2, 'BOG-55 FAQ', '[]'::jsonb) RETURNING id`,
		sectionKey, testName("faq-label"),
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert page_faq_sections fixture: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM page_faq_sections WHERE id = $1`, id) })

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/page_faq", map[string]interface{}{
		"id":        id,
		"label_hu":  "BOG-55 Szakasz",
		"faq_title": "BOG-55 Kérdések",
		"faq_items": []map[string]string{
			{"question": "BOG-55 kérdés 1", "answer": "BOG-55 válasz 1"},
			{"question": "BOG-55 kérdés 2", "answer": "BOG-55 válasz 2"},
		},
		"disclaimer_markdown": "BOG-55 **jogi**",
	}), "PUT a page_faq section")

	section := findInAdminList(t, "/api/admin/page_faq", id)
	if section == nil {
		t.Fatalf("page_faq section %d is not listed after PUT", id)
	}
	if got := fmt.Sprint(section["faq_title"]); got != "BOG-55 Kérdések" {
		t.Errorf("faq_title = %q, want %q", got, "BOG-55 Kérdések")
	}
	if got := fmt.Sprint(section["disclaimer_markdown"]); got != "BOG-55 **jogi**" {
		t.Errorf("disclaimer_markdown = %q, want %q", got, "BOG-55 **jogi**")
	}
	items, ok := section["faq_items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("faq_items = %#v, want two pairs", section["faq_items"])
	}
	first, _ := items[0].(map[string]interface{})
	if fmt.Sprint(first["question"]) != "BOG-55 kérdés 1" || fmt.Sprint(first["answer"]) != "BOG-55 válasz 1" {
		t.Errorf("first faq item round-tripped as %#v", first)
	}

	// Sending no faq_items stores [], never null: the SPA and the public page
	// both map over it.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/page_faq", map[string]interface{}{
		"id": id, "label_hu": "BOG-55 Szakasz", "faq_title": "BOG-55 Kérdések",
	}), "PUT a page_faq section with no items")
	cleared := findInAdminList(t, "/api/admin/page_faq", id)
	if got, ok := cleared["faq_items"].([]interface{}); !ok || len(got) != 0 {
		t.Errorf("faq_items = %#v after clearing, want an empty array", cleared["faq_items"])
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/page_faq", map[string]interface{}{"label_hu": "x"}),
		http.StatusBadRequest, "PUT page_faq with no id")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/page_faq", map[string]interface{}{"id": id}),
		http.StatusMethodNotAllowed, "POST page_faq")
}

// ---------------------------------------------------------------------------
// geography
// ---------------------------------------------------------------------------

// TestAdminLocationsCRUD covers /api/admin/locations, the one route that writes
// two different tables off one body: type "megye" means counties, anything else
// means settlements. Getting that branch wrong would write a settlement into
// the county list.
func TestAdminLocationsCRUD(t *testing.T) {
	requireDB(t)

	countyName := testName("county")
	countyID, countySlug := createTestCounty(t, countyName)

	county := findInAdminList(t, "/api/admin/locations", countyID)
	if county == nil {
		t.Fatalf("county %d is not in GET /api/admin/locations", countyID)
	}
	if got := fmt.Sprint(county["type"]); got != "megye" {
		t.Errorf("the created county has type %q, want megye", got)
	}

	// A settlement in that county goes down the other branch.
	rr := asAdmin(t, http.MethodPost, "/api/admin/locations", map[string]interface{}{
		"name": testName("settlement"), "county": countyName, "type": "község", "post_code": "537000",
	})
	mustOK(t, rr, "POST a settlement")
	settlementID := idOf(t, decodeObject(t, rr, "POST a settlement"), "POST a settlement")
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM settlements WHERE id = $1`, settlementID) })

	settlement := findInAdminList(t, "/api/admin/locations", settlementID)
	if settlement == nil {
		t.Fatalf("settlement %d is not in GET /api/admin/locations", settlementID)
	}
	if got := fmt.Sprint(settlement["county_slug"]); got != countySlug {
		t.Errorf("settlement county_slug = %q, want %q", got, countySlug)
	}

	// PUT renames the settlement and re-derives its slug from the new name.
	renamed := testName("settlement-renamed")
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/locations", map[string]interface{}{
		"id": settlementID, "name": renamed, "county": countyName, "type": "község", "post_code": "537001",
	}), "PUT a settlement")

	after := findInAdminList(t, "/api/admin/locations", settlementID)
	if got := fmt.Sprint(after["name"]); got != renamed {
		t.Errorf("settlement name = %q, want %q", got, renamed)
	}
	if got := fmt.Sprint(after["slug"]); got == fmt.Sprint(settlement["slug"]) {
		t.Errorf("settlement slug stayed %q after the rename; it is derived from the name", got)
	}

	mustOK(t, asAdmin(t, http.MethodDelete, "/api/admin/locations?id="+strconv.Itoa(settlementID), nil),
		"DELETE a settlement")
	if findInAdminList(t, "/api/admin/locations", settlementID) != nil {
		t.Errorf("settlement %d is still listed after DELETE", settlementID)
	}
}

// TestAdminSetCountySeat covers the exclusivity rule: one seat per county, and
// setting a new one clears the old. It runs inside a county of its own, because
// doing it in a real county would move a seat the public app reads.
func TestAdminSetCountySeat(t *testing.T) {
	requireDB(t)

	countyName := testName("seat-county")
	_, _ = createTestCounty(t, countyName)
	first := createTestSettlement(t, countyName, "seat-a")
	second := createTestSettlement(t, countyName, "seat-b")

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/county_seat",
		map[string]int{"location_id": first}), "set the first county seat")
	if !isCountySeat(t, first) {
		t.Fatalf("settlement %d is not the county seat after the route returned 200", first)
	}

	// Moving it must clear the previous one. A handler that only set the new
	// flag would leave the county with two seats and the public app showing
	// whichever sorted first.
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/county_seat",
		map[string]int{"location_id": second}), "move the county seat")
	if !isCountySeat(t, second) {
		t.Errorf("settlement %d is not the county seat after the move", second)
	}
	if isCountySeat(t, first) {
		t.Errorf("settlement %d is still a county seat; a county has exactly one", first)
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/county_seat", map[string]int{"location_id": 0}),
		http.StatusBadRequest, "set a county seat with no location_id")
	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/county_seat", map[string]int{"location_id": 2147483646}),
		http.StatusNotFound, "set a county seat to a settlement that does not exist")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/county_seat", map[string]int{"location_id": first}),
		http.StatusMethodNotAllowed, "POST to county_seat")
}

// TestAdminCountyUpdate covers /api/admin/counties, which exists alongside the
// locations route to edit the fields locations does not carry - notably the
// county's prose `content`.
func TestAdminCountyUpdate(t *testing.T) {
	requireDB(t)

	countyName := testName("county-edit")
	countyID, _ := createTestCounty(t, countyName)

	newSlug := testName("county-edit-slug")
	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/counties", map[string]interface{}{
		"id": countyID, "name": countyName + " módosítva", "name_ro": "BOG-55 RO",
		"slug": newSlug, "content": "BOG-55 county content",
	}), "PUT a county")

	var name, slug, content string
	err := db.DB.QueryRow(`SELECT name, slug, COALESCE(content,'') FROM counties WHERE id = $1`, countyID).
		Scan(&name, &slug, &content)
	if err != nil {
		t.Fatalf("re-read county %d: %v", countyID, err)
	}
	if name != countyName+" módosítva" || slug != newSlug || content != "BOG-55 county content" {
		t.Errorf("county stored as name=%q slug=%q content=%q", name, slug, content)
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/counties", map[string]interface{}{"name": "x", "slug": "y"}),
		http.StatusBadRequest, "PUT a county with no id")
	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/counties", map[string]interface{}{"id": countyID, "slug": "  "}),
		http.StatusBadRequest, "PUT a county with a blank slug")
	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/counties",
		map[string]interface{}{"id": 2147483646, "name": "x", "slug": testName("missing-county")}),
		http.StatusNotFound, "PUT a county that does not exist")
	mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/counties", nil),
		http.StatusMethodNotAllowed, "GET the admin counties route")
}

// TestAdminHistoricalSeatUpdate is the same shape for historical_seats, whose
// rows are seeded reference data: the admin edits them and never creates one.
func TestAdminHistoricalSeatUpdate(t *testing.T) {
	requireDB(t)

	slug := testName("seat")
	var id int
	err := db.DB.QueryRow(
		`INSERT INTO historical_seats (name, slug) VALUES ($1, $2) RETURNING id`, testName("seat-name"), slug).Scan(&id)
	if err != nil {
		t.Fatalf("insert historical_seats fixture: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM historical_seats WHERE id = $1`, id) })

	// The public read is what the main app consumes, so assert through it.
	if findByID(decodeArray(t, anon(t, http.MethodGet, "/api/historical_seats", nil), "GET /api/historical_seats"), id) == nil {
		t.Fatalf("historical seat %d is not in GET /api/historical_seats", id)
	}

	mustOK(t, asAdmin(t, http.MethodPut, "/api/admin/historical_seats", map[string]interface{}{
		"id": id, "name": "BOG-55 Szék", "name_ro": "BOG-55 RO", "slug": slug, "content": "BOG-55 seat content",
	}), "PUT a historical seat")

	var name, content string
	if err := db.DB.QueryRow(
		`SELECT name, COALESCE(content,'') FROM historical_seats WHERE id = $1`, id).Scan(&name, &content); err != nil {
		t.Fatalf("re-read historical seat %d: %v", id, err)
	}
	if name != "BOG-55 Szék" || content != "BOG-55 seat content" {
		t.Errorf("historical seat stored as name=%q content=%q", name, content)
	}

	mustStatus(t, asAdmin(t, http.MethodPut, "/api/admin/historical_seats", map[string]interface{}{"name": "x", "slug": "y"}),
		http.StatusBadRequest, "PUT a historical seat with no id")
	mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/historical_seats", nil),
		http.StatusMethodNotAllowed, "GET the admin historical_seats route")
}

// ---------------------------------------------------------------------------
// read-only admin routes
// ---------------------------------------------------------------------------

// TestAdminUsersListsTheSignedInAdmin: the user list must mark who is an admin,
// because that flag is the only place the ADMIN_GOOGLE_EMAILS allowlist becomes
// visible to a person. An ordinary account must not carry it.
func TestAdminUsersListsTheSignedInAdmin(t *testing.T) {
	requireDB(t)

	ordinary := createTestUser(t, "listed-user")
	users := decodeArray(t, asAdmin(t, http.MethodGet, "/api/admin/users", nil), "GET /api/admin/users")

	adminID, err := userIDByEmail(testAdminEmail)
	if err != nil {
		t.Fatalf("admin user: %v", err)
	}
	me := findByID(users, adminID)
	if me == nil {
		t.Fatalf("the signed-in admin (id %d) is not in GET /api/admin/users", adminID)
	}
	if me["is_admin"] != true {
		t.Errorf("the signed-in admin has is_admin = %#v, want true", me["is_admin"])
	}

	them := findByID(users, ordinary)
	if them == nil {
		t.Fatalf("ordinary user %d is not in GET /api/admin/users", ordinary)
	}
	if them["is_admin"] != false {
		t.Errorf("ordinary user %d has is_admin = %#v, want false", ordinary, them["is_admin"])
	}
	if them["website_banned"] != false {
		t.Errorf("ordinary user %d has website_banned = %#v, want false", ordinary, them["website_banned"])
	}

	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/users", nil),
		http.StatusMethodNotAllowed, "POST to the users route")
}

// TestAdminAttractionSuggestionsRead is the read-only queue of visitor-proposed
// attraction edits. It is covered for shape in TestAdminReadsAnswerTheirShape;
// this pins that it is read-only, which is the part that could quietly change.
func TestAdminAttractionSuggestionsRead(t *testing.T) {
	requireDB(t)

	decodeArray(t, asAdmin(t, http.MethodGet, "/api/admin/attraction-suggestions", nil),
		"GET /api/admin/attraction-suggestions")
	mustStatus(t, asAdmin(t, http.MethodDelete, "/api/admin/attraction-suggestions", nil),
		http.StatusMethodNotAllowed, "DELETE an attraction suggestion")
}

// ---------------------------------------------------------------------------
// image uploads
// ---------------------------------------------------------------------------

// TestAdminEntryImageUpload and TestAdminEventImageUpload are the two multipart
// routes. Both write a file to disk, so each redirects its upload directory at
// a temp dir: the handler reads the env var per request, and a test that wrote
// into backend/data/ would leave litter in the repo.
func TestAdminEntryImageUpload(t *testing.T) {
	requireDB(t)
	dir := t.TempDir()
	t.Setenv("ENTRY_IMAGES_DIR", dir)

	entryID, _ := createAdminEntry(t, "image")
	url, out := uploadImage(t, "/api/admin/entry-images", "entry_id", entryID, dir)
	if !strings.Contains(url, "/api/media/entry-images/") {
		t.Errorf("entry image url = %q, want it under /api/media/entry-images/", url)
	}
	if !strings.Contains(url, fmt.Sprintf("entry-%d-", entryID)) {
		t.Errorf("entry image url = %q, want the entry id in the filename", url)
	}

	// Only this route answers with dimensions: the entry form stores them
	// alongside the photo so the public profile can reserve the space.
	if w, _ := out["width"].(float64); int(w) != 1 {
		t.Errorf("width = %#v, want 1 (the handler did not decode the image)", out["width"])
	}
	if h, _ := out["height"].(float64); int(h) != 1 {
		t.Errorf("height = %#v, want 1", out["height"])
	}

	assertImageUploadRejections(t, "/api/admin/entry-images")
}

func TestAdminEventImageUpload(t *testing.T) {
	requireDB(t)
	dir := t.TempDir()
	t.Setenv("EVENT_IMAGES_DIR", dir)

	// This route answers with a url and nothing else - no width or height. The
	// event form does not store them, so that asymmetry with entry-images is
	// deliberate rather than a gap.
	url, _ := uploadImage(t, "/api/admin/event-images", "event_id", 0, dir)
	if !strings.Contains(url, "/api/media/event-images/") {
		t.Errorf("event image url = %q, want it under /api/media/event-images/", url)
	}
	assertImageUploadRejections(t, "/api/admin/event-images")
}

// TestAdminEventImageUploadReplacesThePrevious covers the only handler in this
// package that deletes a file off disk: re-uploading for an event removes the
// image that event used to point at. A wrong path there deletes something else,
// so the test checks both that the old file is gone and that the new one is not.
func TestAdminEventImageUploadReplacesThePrevious(t *testing.T) {
	requireDB(t)
	dir := t.TempDir()
	t.Setenv("EVENT_IMAGES_DIR", dir)

	eventID := createTestEvent(t, "image-replace")

	first, _ := uploadImage(t, "/api/admin/event-images", "event_id", eventID, dir)
	// The handler only deletes what the *event row* points at, so the first
	// url has to be stored before the second upload.
	if _, err := db.DB.Exec(`UPDATE events SET featured_image = $1 WHERE id = $2`, first, eventID); err != nil {
		t.Fatalf("store featured_image on event %d: %v", eventID, err)
	}

	second, _ := uploadImage(t, "/api/admin/event-images", "event_id", eventID, dir)
	if second == first {
		t.Fatal("the second upload reused the first filename; each upload gets its own")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.Base(first))); !os.IsNotExist(err) {
		t.Errorf("the previous image %s is still on disk after a replacement upload (stat err: %v)", first, err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.Base(second))); err != nil {
		t.Errorf("the new image %s is not on disk: %v", second, err)
	}
}

// uploadImage posts a one-pixel GIF and returns the url the handler answered
// with plus the whole decoded response, having checked the file actually landed
// on disk. A handler that answered a url without writing the bytes would pass a
// status-code test and leave the admin with a broken image.
func uploadImage(t *testing.T, path, idField string, id int, dir string) (string, map[string]interface{}) {
	t.Helper()

	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	if id > 0 {
		if err := form.WriteField(idField, strconv.Itoa(id)); err != nil {
			t.Fatalf("write %s field: %v", idField, err)
		}
	}
	part, err := form.CreateFormFile("file", "bog55.gif")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(onePixelGIF); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close multipart form: %v", err)
	}

	req := newRequest(t, http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.AddCookie(adminCookie(t))
	rr := record(req)
	mustOK(t, rr, "POST "+path)

	out := decodeObject(t, rr, "POST "+path)
	url := fmt.Sprint(out["url"])
	if url == "" || url == "<nil>" {
		t.Fatalf("POST %s answered no url: %#v", path, out)
	}

	written, err := os.ReadFile(filepath.Join(dir, filepath.Base(url)))
	if err != nil {
		t.Fatalf("POST %s answered %s but wrote no file: %v", path, url, err)
	}
	if !bytes.Equal(written, onePixelGIF) {
		t.Errorf("POST %s wrote %d bytes, want the %d uploaded", path, len(written), len(onePixelGIF))
	}
	return url, out
}

func assertImageUploadRejections(t *testing.T, path string) {
	t.Helper()

	mustStatus(t, asAdmin(t, http.MethodGet, path, nil), http.StatusMethodNotAllowed, "GET "+path)
	mustStatus(t, asAdmin(t, http.MethodPost, path, nil), http.StatusBadRequest,
		"POST "+path+" with no multipart body")

	// A multipart body with no "file" part, and one whose file is not an image
	// type the admin accepts. Both are 400s a form can produce by accident.
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	form.WriteField("entry_id", "1")
	form.Close()
	req := newRequest(t, http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.AddCookie(adminCookie(t))
	mustStatus(t, record(req), http.StatusBadRequest, "POST "+path+" with no file part")

	buf.Reset()
	form = multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "bog55.svg")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	part.Write([]byte("<svg/>"))
	form.Close()
	req = newRequest(t, http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.AddCookie(adminCookie(t))
	mustStatus(t, record(req), http.StatusBadRequest, "POST "+path+" with an SVG")
}

// onePixelGIF is a valid 1x1 GIF, so image.DecodeConfig reads real dimensions
// off it. A blob of random bytes would exercise the handler's fallback instead
// of its decode path.
var onePixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}

// ---------------------------------------------------------------------------
// fixtures used only here
// ---------------------------------------------------------------------------

// createTestCounty makes a county through POST /api/admin/locations with type
// "megye" - the only route that creates one - and removes it afterwards.
func createTestCounty(t *testing.T, name string) (id int, slug string) {
	t.Helper()
	rr := asAdmin(t, http.MethodPost, "/api/admin/locations", map[string]interface{}{
		"name": name, "type": "megye",
	})
	mustOK(t, rr, "POST a county")
	created := decodeObject(t, rr, "POST a county")
	id = idOf(t, created, "POST a county")
	slug = fmt.Sprint(created["slug"])
	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM settlements WHERE county_id = $1`, id)
		db.DB.Exec(`DELETE FROM counties WHERE id = $1`, id)
	})
	return id, slug
}

func createTestSettlement(t *testing.T, countyName, label string) int {
	t.Helper()
	rr := asAdmin(t, http.MethodPost, "/api/admin/locations", map[string]interface{}{
		"name": testName(label), "county": countyName, "type": "község",
	})
	mustOK(t, rr, "POST a settlement")
	id := idOf(t, decodeObject(t, rr, "POST a settlement"), "POST a settlement")
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM settlements WHERE id = $1`, id) })
	return id
}

// createTestEvent makes one event through POST /api/admin/events - the same
// body TestAdminCatalogCRUD round-trips - so the image test has an event row to
// hang a featured image off.
func createTestEvent(t *testing.T, label string) int {
	t.Helper()
	rr := asAdmin(t, http.MethodPost, "/api/admin/events", map[string]interface{}{
		"title":         testName(label),
		"location_id":   someSettlementID(t),
		"start_date":    "2099-07-01",
		"start_time":    "18:00",
		"end_date":      "2099-07-01",
		"end_time":      "20:00",
		"event_type_id": someCatalogEventTypeID(t),
	})
	mustOK(t, rr, "POST an event")
	id := idOf(t, decodeObject(t, rr, "POST an event"), "POST an event")
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM events WHERE id = $1`, id) })
	return id
}

func isCountySeat(t *testing.T, settlementID int) bool {
	t.Helper()
	var seat bool
	if err := db.DB.QueryRow(
		`SELECT is_county_seat FROM settlements WHERE id = $1`, settlementID).Scan(&seat); err != nil {
		t.Fatalf("read is_county_seat for settlement %d: %v", settlementID, err)
	}
	return seat
}
