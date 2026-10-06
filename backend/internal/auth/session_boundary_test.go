package auth

import (
	"backend/internal/config"
	"backend/internal/db"
	"bytes"
	"database/sql"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// BOG-45: this API must not accept a session the public site minted.
//
// Both apps share one database. They used to share the cookie name
// `lamsza_session` and the `sessions` table too, so any signed-in visitor of
// lamsza.com held a token every route here accepted. Admin now has its own
// cookie name and its own `admin_sessions` table.
//
// Two of the tests below need the shared database and skip without it. The
// other two do not, so the names and the queries stay pinned in CI, where
// there is no Postgres.

// ---------------------------------------------------------------------------
// No database needed
// ---------------------------------------------------------------------------

// TestAdminSessionCookieIsNotThePublicOne pins the rename. On localhost both
// apps are host `localhost` and cookies ignore the port, so sharing this name
// means the browser itself hands the public cookie to :3000.
func TestAdminSessionCookieIsNotThePublicOne(t *testing.T) {
	if SessionCookieName == "lamsza_session" {
		t.Fatal("admin reuses the public cookie name; the browser will send the public session to this API")
	}
	if SessionCookieName != "lamsza_admin_session" {
		t.Fatalf("admin session cookie = %q, want \"lamsza_admin_session\"", SessionCookieName)
	}
}

// TestAuthPackageNeverTouchesThePublicSessionsTable is the static half of the
// boundary, in the style of boot_ddl_test.go: the cookie name is only
// presentation — anyone can put any token in any cookie — so the store is what
// actually separates the two apps. One query against `sessions` here would
// re-open the hole with every other test still green.
func TestAuthPackageNeverTouchesThePublicSessionsTable(t *testing.T) {
	// `sessions` as a table name, not `admin_sessions` and not a word that
	// merely ends in it.
	publicTable := regexp.MustCompile(`(?i)\b(FROM|INTO|UPDATE|JOIN)\s+sessions\b`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if m := publicTable.FindString(lit.Value); m != "" {
				t.Errorf("%s: SQL against the public `sessions` table (%q) — admin uses `admin_sessions`",
					filepath.Base(fset.Position(lit.Pos()).Filename), strings.TrimSpace(m))
			}
			return true
		})
	}
}

// ---------------------------------------------------------------------------
// Database needed
// ---------------------------------------------------------------------------

// withDB opens the shared database, or skips. The tests below are about which
// table a token lives in, so there is nothing to fake.
//
// It opens its own pool instead of calling db.InitDB(), which log.Fatal()s on a
// missing database and would take the whole test binary with it. CI has no
// Postgres; a skip there is honest, and the two static tests above still run.
func withDB(t *testing.T) {
	t.Helper()
	if db.DB != nil && db.DB.Ping() == nil {
		return
	}
	if config.AppConfig.DatabaseURL == "" {
		config.Load()
	}
	conn, err := sql.Open("postgres", config.AppConfig.DatabaseURL)
	if err != nil {
		t.Skipf("shared lamsza database not reachable: %v", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		t.Skipf("shared lamsza database not reachable: %v", err)
	}
	db.DB = conn
}

// testUserID upserts the shared users row. The google_sub has to be the one
// ParseTestIDToken derives, or a later sign-in as the same email hits the
// unique index on users.email instead of the ON CONFLICT (google_sub) path.
func testUserID(t *testing.T, email string) int {
	t.Helper()
	ident, err := ParseTestIDToken("test:"+email, "test-client")
	if err != nil {
		t.Fatalf("test identity for %s: %v", email, err)
	}
	var id int
	err = db.DB.QueryRow(`
		INSERT INTO users (google_sub, email, name, last_login_at)
		VALUES ($1, $2, 'Boundary Test', NOW())
		ON CONFLICT (google_sub) DO UPDATE SET last_login_at = NOW()
		RETURNING id
	`, ident.Sub, email).Scan(&id)
	if err != nil {
		t.Fatalf("test user %s: %v", email, err)
	}
	return id
}

// mintPublicSession writes a row the way the main lamsza backend does: into
// `sessions`. This is the token a visitor of the public site holds.
func mintPublicSession(t *testing.T, email string) string {
	t.Helper()
	token := "bog45-public-token-" + time.Now().Format("20060102150405.000000000")
	hash := hashToken(token)
	_, err := db.DB.Exec(
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hash, testUserID(t, email), time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert public session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.DB.Exec(`DELETE FROM sessions WHERE token_hash = $1`, hash)
	})
	return token
}

// TestPublicSessionIsRejectedByTheAdminAPI is the cross-host rejection. The
// account is on the admin allowlist, so the only thing standing between the
// token and the admin API is which table minted it.
func TestPublicSessionIsRejectedByTheAdminAPI(t *testing.T) {
	withDB(t)
	prev := config.AppConfig.AdminGoogleEmails
	t.Cleanup(func() { config.AppConfig.AdminGoogleEmails = prev })
	config.AppConfig.AdminGoogleEmails = []string{"boundary-admin@test.lamsza"}

	token := mintPublicSession(t, "boundary-admin@test.lamsza")

	// 1. Presented under the admin cookie name — a token is just a string, so
	//    nothing stops a caller doing this deliberately.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rr := httptest.NewRecorder()
	RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("public token under the admin cookie name: got %d, want 401 (body %q)", rr.Code, rr.Body.String())
	}

	// 2. Presented under the public cookie name — what the browser itself does
	//    on localhost, where cookies ignore the port.
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "lamsza_session", Value: token})
	rr = httptest.NewRecorder()
	RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("public cookie sent to the admin API: got %d, want 401 (body %q)", rr.Code, rr.Body.String())
	}

	if _, err := UserFromRequest(req); err == nil {
		t.Error("UserFromRequest resolved a user from a public session token")
	}
}

// TestAdminLoginWritesOnlyTheAdminSessionTable covers the write side end to
// end: a real sign-in must land in `admin_sessions` and leave `sessions`
// untouched, or the public API would accept the admin's token.
func TestAdminLoginWritesOnlyTheAdminSessionTable(t *testing.T) {
	withDB(t)
	prevEmails := config.AppConfig.AdminGoogleEmails
	prevClient := config.AppConfig.GoogleClientID
	prevVerify := VerifyIDToken
	t.Cleanup(func() {
		config.AppConfig.AdminGoogleEmails = prevEmails
		config.AppConfig.GoogleClientID = prevClient
		VerifyIDToken = prevVerify
	})
	config.AppConfig.AdminGoogleEmails = []string{"boundary-admin@test.lamsza"}
	config.AppConfig.GoogleClientID = "test-client"
	VerifyIDToken = ParseTestIDToken

	cookie := signIn(t, "boundary-admin@test.lamsza", http.StatusOK)
	if cookie == nil {
		t.Fatal("admin sign-in returned no session cookie")
	}
	if cookie.Name != SessionCookieName {
		t.Fatalf("session cookie name = %q, want %q", cookie.Name, SessionCookieName)
	}
	// Survives a browser restart: a persistent cookie, not a session one. A
	// MaxAge of 0 with no Expires is exactly the bug where sign-in works and
	// then silently stops on the next morning's first visit.
	if cookie.MaxAge <= 0 {
		t.Errorf("session cookie MaxAge = %d; a browser restart would drop it", cookie.MaxAge)
	}
	if !cookie.Expires.After(time.Now().Add(24 * time.Hour)) {
		t.Errorf("session cookie Expires = %v, want well in the future", cookie.Expires)
	}
	if !cookie.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}
	hash := hashToken(cookie.Value)
	t.Cleanup(func() {
		_, _ = db.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash = $1`, hash)
	})

	if got := countRows(t, `SELECT COUNT(*) FROM admin_sessions WHERE token_hash = $1`, hash); got != 1 {
		t.Errorf("admin_sessions rows for the new token: got %d, want 1", got)
	}
	if got := countRows(t, `SELECT COUNT(*) FROM sessions WHERE token_hash = $1`, hash); got != 0 {
		t.Errorf("admin sign-in wrote %d row(s) into the public sessions table", got)
	}

	// Sign-out clears it, so the token stops working after a browser restart
	// resends the cookie.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	HandleLogout(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("logout: got %d, body %q", rr.Code, rr.Body.String())
	}
	if got := countRows(t, `SELECT COUNT(*) FROM admin_sessions WHERE token_hash = $1`, hash); got != 0 {
		t.Errorf("logout left %d row(s) in admin_sessions", got)
	}

	// A non-allowlisted account never gets a row at all.
	if c := signIn(t, "stranger@test.lamsza", http.StatusForbidden); c != nil {
		t.Errorf("non-admin sign-in handed out a %s cookie", c.Name)
	}
}

func signIn(t *testing.T, email string, wantStatus int) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"credential": "test:" + email})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleGoogleLogin(rr, req)
	if rr.Code != wantStatus {
		t.Fatalf("sign-in as %s: got %d, want %d (body %q)", email, rr.Code, wantStatus, rr.Body.String())
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == SessionCookieName && c.Value != "" {
			return c
		}
	}
	return nil
}

func countRows(t *testing.T, query string, args ...interface{}) int {
	t.Helper()
	var n int
	if err := db.DB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}
