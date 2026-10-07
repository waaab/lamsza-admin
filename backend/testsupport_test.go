package main

// Shared harness for the admin handler suite (BOG-55).
//
// # Where the schema comes from
//
// From a scratch database, never the shared dev one. `npm run test:backend`
// (scripts/test-backend.sh) drops and recreates `lamsza_admin_test` with the
// lamsza repo's scripts/db-bootstrap.sh from lamsza/backend/schema/, and sets
// TEST_DATABASE_URL. This package runs no DDL (docs/ARCHITECTURE.md,
// boot_ddl_test.go), so the committed lamsza schema is the whole schema here.
// Without TEST_DATABASE_URL the DB-backed tests skip, as in CI; the route-guard
// tests in route_guard_test.go need no database and run everywhere.
//
// Every admin write also lands in admin_audit_log (BOG-48), so the scratch
// database collects audit rows too; that is the point of using one.
//
// # The isolation rule
//
// The scratch database is shared by every test in the run, so asserting on
// global list contents or global counts is still not allowed: that is what
// made lamsza/backend/directory_catalog_test.go fail on a dirty database
// before BOG-42. So every test here
//
//   - names its own rows with testTag() so they cannot collide with another
//     test's rows or with a run that was interrupted,
//   - finds its own row in a list rather than asserting the list's length,
//   - and removes what it created in t.Cleanup, including on failure.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/db"

	_ "github.com/lib/pq"
)

const (
	// testOrigin is on the default allowlist, so these requests look like calls
	// from the admin SPA. The allowlist itself is covered in
	// internal/middleware/middleware_test.go.
	testOrigin = "https://admin.lamsza.com"

	// testAdminEmail is the only allowlisted account while the suite runs.
	// Overriding config.AppConfig.AdminGoogleEmails keeps the suite independent
	// of whatever backend/.env happens to hold on this machine.
	testAdminEmail = "bog55-admin@test.lamsza"

	// testNonAdminEmail holds a session row in admin_sessions without being on
	// the allowlist, which is the 403 half of RequireAdmin. It cannot be created
	// by signing in: HandleGoogleLogin refuses a non-allowlisted account on
	// purpose (BOG-45), so this one is minted in SQL.
	testNonAdminEmail = "bog55-not-admin@test.lamsza"
)

// testMux is the real route table, built by newMux() in main.go. Tests serve
// through it rather than through a hand-copied mux so the wrappers, the order
// and the feature flags under test are the ones the process actually runs.
var testMux *http.ServeMux

// dbErr is nil when the scratch test database answered and carried the
// schema. Anything else is the reason the DB-backed tests skip.
var dbErr error

func TestMain(m *testing.M) {
	config.Load()
	config.AppConfig.GoogleClientID = "test-google-client-id"
	config.AppConfig.AdminGoogleEmails = []string{testAdminEmail}
	auth.VerifyIDToken = auth.ParseTestIDToken

	// Every feature flag on, so the route table under test is the whole table.
	// A flag left off in .env would otherwise silently drop routes from the
	// guard sweep - the opposite of what that sweep is for.
	config.AppConfig.Features.Weather = true
	config.AppConfig.Features.Events = true
	config.AppConfig.Features.News = true
	config.AppConfig.Features.Mondasok = true
	config.AppConfig.Features.QuickLinks = true

	testMux = newMux()
	dbErr = openTestDB()

	code := m.Run()
	dropSuiteAccounts()
	os.Exit(code)
}

// dropSuiteAccounts removes the two accounts the suite signs in as. They are
// not per-test fixtures - they are cached for the whole run - so no t.Cleanup
// owns them, and without this every run leaves another pair of rows in the
// users table.
func dropSuiteAccounts() {
	if dbErr != nil {
		return
	}
	for _, email := range []string{testAdminEmail, testNonAdminEmail} {
		id, err := userIDByEmail(email)
		if err != nil {
			continue
		}
		db.DB.Exec(`DELETE FROM admin_sessions WHERE user_id = $1`, id)
		db.DB.Exec(`DELETE FROM users WHERE id = $1`, id)
	}
}

// openTestDB opens the scratch test database the way db.InitDB() does, minus
// the log.Fatal: a missing database must skip this suite, not kill the test
// binary and take the no-database route guards down with it. It reads only
// TEST_DATABASE_URL, never DATABASE_URL or .env, which point at the shared dev
// database.
func openTestDB() error {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		return fmt.Errorf("TEST_DATABASE_URL is not set")
	}
	url, err := db.TestDatabaseURL()
	if err != nil {
		return err
	}
	conn, err := sql.Open("postgres", url)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return err
	}

	// A reachable but unmigrated database would turn every handler into a 500
	// and read as 30 product bugs. Name the missing table instead.
	for _, table := range []string{
		"users", "admin_sessions", "counties", "settlements",
		"entries", "entry_categories", "entry_types", "entry_members",
		"entry_suggestions", "websites",
	} {
		var present bool
		err := conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&present)
		if err != nil {
			conn.Close()
			return err
		}
		if !present {
			conn.Close()
			return fmt.Errorf("table %q is missing: the main lamsza backend owns this schema and has not created it", table)
		}
	}

	db.DB = conn
	return nil
}

// requireDB skips a test that needs the scratch database, saying how to get
// one so the message is actionable.
func requireDB(t *testing.T) {
	t.Helper()
	if dbErr != nil {
		t.Skipf("scratch test database unavailable: %v\n"+
			"Run `npm run test:backend` from the repo root: it builds lamsza_admin_test from lamsza/backend/schema/ and sets TEST_DATABASE_URL.", dbErr)
	}
}

// ---------------------------------------------------------------------------
// Unique names
// ---------------------------------------------------------------------------

// testTag returns a suffix unique to this process. Fixture names carry it so a
// row from this suite can never be confused with the owner's data, and a second
// run cannot trip over the first one's leftovers.
var testTag = sync.OnceValue(func() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("bog55-%d", time.Now().UnixNano())
	}
	return "bog55-" + hex.EncodeToString(buf)
})

// testName builds a fixture name that is unique per run and per caller-chosen
// label, and recognisable in the database if a cleanup ever fails.
func testName(label string) string {
	return fmt.Sprintf("zz-%s-%s", testTag(), label)
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

var (
	adminCookieOnce    sync.Once
	cachedAdminCookie  *http.Cookie
	adminCookieErr     error
	nonAdminCookieOnce sync.Once
	cachedNonAdmin     *http.Cookie
	nonAdminErr        error
)

// adminCookie signs testAdminEmail in through the real Google-login handler, so
// the session under test is one the sign-in path actually mints.
func adminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	requireDB(t)
	adminCookieOnce.Do(func() {
		body, _ := json.Marshal(map[string]string{"credential": "test:" + testAdminEmail})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", testOrigin)
		rr := httptest.NewRecorder()
		testMux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			adminCookieErr = fmt.Errorf("admin login: %d %s", rr.Code, rr.Body.String())
			return
		}
		for _, c := range rr.Result().Cookies() {
			if c.Name == auth.SessionCookieName {
				cachedAdminCookie = c
				return
			}
		}
		adminCookieErr = fmt.Errorf("admin login returned no %s cookie", auth.SessionCookieName)
	})
	if adminCookieErr != nil {
		t.Fatal(adminCookieErr)
	}
	return cachedAdminCookie
}

// nonAdminCookie returns a valid admin_sessions token for an account that is
// not on the allowlist - the 403 half of RequireAdmin. Minted in SQL because
// HandleGoogleLogin refuses to create such a session, which is itself the
// BOG-45 rule: a row in admin_sessions means an admin is signed in.
//
// This is also the realistic shape of the bug RequireAdmin's IsAdmin check
// guards: an account that *was* allowlisted, signed in, and was then removed
// from ADMIN_GOOGLE_EMAILS while its session was still live.
func nonAdminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	requireDB(t)
	nonAdminCookieOnce.Do(func() {
		userID, err := upsertTestUser(testNonAdminEmail)
		if err != nil {
			nonAdminErr = err
			return
		}
		raw, hash := mintSessionToken(t)
		_, err = db.DB.Exec(
			`INSERT INTO admin_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 hour')`,
			hash, userID,
		)
		if err != nil {
			nonAdminErr = err
			return
		}
		cachedNonAdmin = &http.Cookie{Name: auth.SessionCookieName, Value: raw}
	})
	if nonAdminErr != nil {
		t.Fatalf("mint non-admin session: %v", nonAdminErr)
	}
	return cachedNonAdmin
}

// expiredAdminCookie mints a session for the allowlisted admin that expired an
// hour ago. The account is right; only expires_at is in the past.
func expiredAdminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	// Sign in first so the users row is the one the login path creates, then
	// mint a second, already-expired token for it. Creating the row here
	// instead would leave it with a google_sub the login path cannot match,
	// and the next sign-in would collide on users.email.
	adminCookie(t)
	userID, err := userIDByEmail(testAdminEmail)
	if err != nil {
		t.Fatalf("admin user: %v", err)
	}
	raw, hash := mintSessionToken(t)
	if _, err := db.DB.Exec(
		`INSERT INTO admin_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, NOW() - INTERVAL '1 hour')`,
		hash, userID,
	); err != nil {
		t.Fatalf("insert expired session: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash = $1`, hash) })
	return &http.Cookie{Name: auth.SessionCookieName, Value: raw}
}

// mintSessionToken returns a random token and its SHA-256 hex, which is exactly
// what the two backends store. A session is nothing more than that, so a row
// built this way is the real thing and not a stand-in.
func mintSessionToken(t *testing.T) (raw, hash string) {
	t.Helper()
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("random token: %v", err)
	}
	raw = hex.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:])
}

func userIDByEmail(email string) (int, error) {
	var id int
	err := db.DB.QueryRow(`SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&id)
	return id, err
}

// upsertTestUser returns the id of a users row for email, creating it if needed.
//
// google_sub must be the value auth.ParseTestIDToken derives - the login path
// upserts ON CONFLICT (google_sub), so a row inserted here under any other sub
// is invisible to it and the next sign-in collides on users.email instead.
func upsertTestUser(email string) (int, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id int
	err := db.DB.QueryRow(`
		INSERT INTO users (google_sub, email, name, last_login_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (google_sub) DO UPDATE SET last_login_at = NOW()
		RETURNING id`,
		"test-sub-"+email, email, "BOG-55 Test User",
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Losing the insert race, or an older row under a different sub, still
	// leaves a usable account.
	if id, lookupErr := userIDByEmail(email); lookupErr == nil {
		return id, nil
	}
	return 0, err
}

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

func requestBody(t *testing.T, body interface{}) io.Reader {
	t.Helper()
	switch v := body.(type) {
	case nil:
		return nil
	case io.Reader:
		return v
	case string:
		return strings.NewReader(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		return bytes.NewReader(raw)
	}
}

// newRequest builds a request the admin SPA could have sent. Tests that need to
// change a header before it goes out use this and record() directly.
func newRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, requestBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	return req
}

// record serves one request through the real mux.
func record(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	testMux.ServeHTTP(rr, req)
	return rr
}

// serve sends one request through the real mux. A nil cookie means anonymous.
func serve(t *testing.T, method, path string, body interface{}, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := newRequest(t, method, path, body)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return record(req)
}

// asAdmin is the happy-path caller: signed in and allowlisted.
func asAdmin(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, method, path, body, adminCookie(t))
}

// anon is the unauthenticated caller.
func anon(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, method, path, body, nil)
}

// mustOK fails with the response body, which is where the handlers put their
// reason. A bare status code is not enough to debug from.
func mustOK(t *testing.T, rr *httptest.ResponseRecorder, what string) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d; body: %s", what, rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func mustStatus(t *testing.T, rr *httptest.ResponseRecorder, want int, what string) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("%s: expected %d, got %d; body: %s", what, want, rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

// decodeObject and decodeArray keep the JSON shape assertion in one place: a
// handler that answers 200 with HTML or a bare id is a bug these would catch.
func decodeObject(t *testing.T, rr *httptest.ResponseRecorder, what string) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: response is not a JSON object: %v; body: %s", what, err, rr.Body.String())
	}
	return out
}

func decodeArray(t *testing.T, rr *httptest.ResponseRecorder, what string) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: response is not a JSON array: %v; body: %s", what, err, rr.Body.String())
	}
	return out
}

// idOf reads an id out of a decoded JSON object. Postgres ids arrive as
// float64, and formatting one back with %v yields "1e+06" past a million.
func idOf(t *testing.T, obj map[string]interface{}, what string) int {
	t.Helper()
	raw, ok := obj["id"]
	if !ok {
		t.Fatalf("%s: response has no id: %#v", what, obj)
	}
	f, ok := raw.(float64)
	if !ok || f <= 0 {
		t.Fatalf("%s: id is not a positive number: %#v", what, raw)
	}
	return int(f)
}

// findByID returns the member of list whose id is want, or nil. Tests look for
// their own row instead of asserting on the length of a shared table - the
// dirty-database trap from the isolation rule at the top of this file.
func findByID(list []map[string]interface{}, want int) map[string]interface{} {
	for _, item := range list {
		if f, ok := item["id"].(float64); ok && int(f) == want {
			return item
		}
	}
	return nil
}
