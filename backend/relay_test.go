package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"backend/internal/config"
	"backend/internal/db"
)

const relayToken = "relay-test-token"

// seen is what a fake app backend received from the relay.
type seen struct {
	method, path, query, auth, email, cookie, contentType, body string
}

// fakeApp stands in for Szótár's or Játszótér's backend: it records the
// request and answers like the internal admin API would.
func fakeApp(t *testing.T, status int, reply string) (*httptest.Server, func() seen) {
	t.Helper()
	var mu sync.Mutex
	var got seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = seen{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), r.Header.Get("X-Admin-Email"), r.Header.Get("Cookie"), r.Header.Get("Content-Type"), string(b)}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, func() seen { mu.Lock(); defer mu.Unlock(); return got }
}

// pointRelays aims both relays at url with the test token, for this test only.
func pointRelays(t *testing.T, url, token string) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig.SzotarAdminURL, config.AppConfig.SzotarAdminToken = url, token
	config.AppConfig.JatszoterAdminURL, config.AppConfig.JatszoterAdminToken = url, token
	t.Cleanup(func() {
		config.AppConfig.SzotarAdminURL, config.AppConfig.SzotarAdminToken = prev.SzotarAdminURL, prev.SzotarAdminToken
		config.AppConfig.JatszoterAdminURL, config.AppConfig.JatszoterAdminToken = prev.JatszoterAdminURL, prev.JatszoterAdminToken
	})
}

func TestRelayNeedsAnAdmin(t *testing.T) {
	requireDB(t)
	srv, got := fakeApp(t, http.StatusOK, `{}`)
	pointRelays(t, srv.URL, relayToken)
	mustStatus(t, anon(t, http.MethodGet, "/api/admin/dictionary/stats", nil), http.StatusUnauthorized, "anonymous relay")
	mustStatus(t, serve(t, http.MethodGet, "/api/admin/games/stats", nil, nonAdminCookie(t)), http.StatusForbidden, "non-admin relay")
	if got().path != "" {
		t.Fatalf("the relay reached the app for a refused caller: %+v", got())
	}
}

func TestRelayForwardsAReadWithTheServiceToken(t *testing.T) {
	requireDB(t)
	srv, got := fakeApp(t, http.StatusOK, `{"words":[{"id":1}]}`)
	pointRelays(t, srv.URL, relayToken)

	rr := asAdmin(t, http.MethodGet, "/api/admin/dictionary/words?q=b%C3%A1ng&letter=b", nil)
	mustOK(t, rr, "relayed read")
	if strings.TrimSpace(rr.Body.String()) != `{"words":[{"id":1}]}` {
		t.Fatalf("reply = %q, want the app's reply unchanged", rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	s := got()
	if s.method != http.MethodGet || s.path != "/internal/admin/words" || s.query != "q=b%C3%A1ng&letter=b" {
		t.Fatalf("forwarded %s %s?%s", s.method, s.path, s.query)
	}
	if s.auth != "Bearer "+relayToken || s.email != testAdminEmail {
		t.Fatalf("forwarded auth %q, email %q", s.auth, s.email)
	}
	if s.cookie != "" {
		t.Fatalf("a cookie went to the app: %q", s.cookie)
	}
}

func TestRelayForwardsAWriteAndAuditsIt(t *testing.T) {
	requireDB(t)
	srv, got := fakeApp(t, http.StatusOK, `{"puzzle":{"id":"p042"}}`)
	pointRelays(t, srv.URL, relayToken)

	text := testName("relay-puzzle")
	rr := asAdmin(t, http.MethodPost, "/api/admin/games/rovasfejto/puzzles", map[string]any{"text": text, "category": "kozmondas", "difficulty": 2})
	mustOK(t, rr, "relayed write")
	s := got()
	if s.method != http.MethodPost || s.path != "/internal/admin/rovasfejto/puzzles" || !strings.Contains(s.body, text) || !strings.HasPrefix(s.contentType, "application/json") {
		t.Fatalf("forwarded %+v", s)
	}

	var route, resource, resourceID, email string
	var payload []byte
	err := db.DB.QueryRow(`
		SELECT route, resource, resource_id, actor_email, payload
		FROM admin_audit_log WHERE route = $1 AND payload::text LIKE '%' || $2 || '%'
		ORDER BY id DESC LIMIT 1`, "/api/admin/games/rovasfejto/puzzles", text).Scan(&route, &resource, &resourceID, &email, &payload)
	if err != nil {
		t.Fatalf("no audit record for the relayed write: %v", err)
	}
	if resource != "jatszoter" || resourceID != "rovasfejto/puzzles" || email != testAdminEmail {
		t.Fatalf("audit record: resource %q, id %q, actor %q", resource, resourceID, email)
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil || body["text"] != text {
		t.Fatalf("audit payload = %s", payload)
	}
}

func TestRelayPassesTheAppsErrorThrough(t *testing.T) {
	requireDB(t)
	srv, _ := fakeApp(t, http.StatusConflict, `{"error":"Erre a napra már van mondás."}`)
	pointRelays(t, srv.URL, relayToken)
	rr := asAdmin(t, http.MethodPost, "/api/admin/dictionary/proverbs", map[string]any{"text": "x", "display_date": "2099-01-01"})
	mustStatus(t, rr, http.StatusConflict, "relayed conflict")
	if !strings.Contains(rr.Body.String(), "Erre a napra már van mondás.") {
		t.Fatalf("reply = %q", rr.Body.String())
	}
}

func TestRelayIsOffWithoutATokenAndSaysSo(t *testing.T) {
	requireDB(t)
	srv, got := fakeApp(t, http.StatusOK, `{}`)
	pointRelays(t, srv.URL, "")
	rr := asAdmin(t, http.MethodGet, "/api/admin/dictionary/stats", nil)
	mustStatus(t, rr, http.StatusServiceUnavailable, "relay without a token")
	if !strings.Contains(rr.Body.String(), "nincs beállítva") || got().path != "" {
		t.Fatalf("reply %q, app saw %+v", rr.Body.String(), got())
	}
}

func TestRelayReportsAnUnreachableApp(t *testing.T) {
	requireDB(t)
	srv, _ := fakeApp(t, http.StatusOK, `{}`)
	url := srv.URL
	srv.Close()
	pointRelays(t, url, relayToken)
	rr := asAdmin(t, http.MethodGet, "/api/admin/games/stats", nil)
	mustStatus(t, rr, http.StatusBadGateway, "unreachable app")
	if !strings.Contains(rr.Body.String(), "most nem érhető el") {
		t.Fatalf("reply = %q", rr.Body.String())
	}
}
