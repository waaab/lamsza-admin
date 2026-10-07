package main

import (
	"net/http"
	"strings"
	"testing"

	"backend/internal/config"
)

// /api/auth/admin-status is what Szótár's and Játszótér's toolbars ask
// (lamsza WAYS_OF_WORKING R18). It must say only yes or no, admit the network's
// sites for this one GET, and leave the rest of the API's CORS untouched.
func TestAdminStatusSaysOnlyYesOrNo(t *testing.T) {
	requireDB(t)
	for _, c := range []struct {
		name   string
		cookie *http.Cookie
		want   string
	}{
		{"anonymous", nil, `{"is_admin":false}`},
		{"signed in, not on the list", nonAdminCookie(t), `{"is_admin":false}`},
		{"admin", adminCookie(t), `{"is_admin":true}`},
	} {
		rr := serve(t, http.MethodGet, "/api/auth/admin-status", nil, c.cookie)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != c.want {
			t.Errorf("%s: %d %q, want 200 %s", c.name, rr.Code, rr.Body.String(), c.want)
		}
		if rr.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control = %q", c.name, rr.Header().Get("Cache-Control"))
		}
	}
	if rr := serve(t, http.MethodPost, "/api/auth/admin-status", nil, adminCookie(t)); rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: %d, want 405", rr.Code)
	}
}

func TestAdminStatusAdmitsOnlyTheNetworkSites(t *testing.T) {
	requireDB(t)
	// Production narrows the API's own list to the admin app; this route
	// must still answer the sibling sites, and only this route.
	prev := config.AppConfig.AllowedOrigins
	config.AppConfig.AllowedOrigins = []string{"https://admin.lamsza.com"}
	config.ReloadAllowedOrigins()
	t.Cleanup(func() { config.AppConfig.AllowedOrigins = prev; config.ReloadAllowedOrigins() })

	call := func(path, origin string) *http.Response {
		req := newRequest(t, http.MethodGet, path, nil)
		req.Header.Set("Origin", origin)
		req.AddCookie(adminCookie(t))
		return record(req).Result()
	}
	for _, origin := range []string{"https://szotar.lamsza.com", "https://jatszoter.lamsza.com", "http://localhost:5175"} {
		res := call("/api/auth/admin-status", origin)
		if res.Header.Get("Access-Control-Allow-Origin") != origin || res.Header.Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("admin-status from %s: Allow-Origin %q", origin, res.Header.Get("Access-Control-Allow-Origin"))
		}
		if got := call("/api/auth/me", origin).Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("/api/auth/me from %s: Allow-Origin %q, want none in production", origin, got)
		}
	}
	for _, origin := range []string{"https://evil.example", "https://szotar.lamsza.com.evil.example"} {
		if got := call("/api/auth/admin-status", origin).Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("admin-status from %s: Allow-Origin %q, want none", origin, got)
		}
	}
}
