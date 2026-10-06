package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
)

func setAllowed(t *testing.T, origins ...string) {
	t.Helper()
	config.AppConfig.AllowedOrigins = origins
	config.ReloadAllowedOrigins()
}

func okHandler(hit *bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*hit = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

func serve(method, origin string, hit *bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/entries", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rr := httptest.NewRecorder()
	ApplyCORS(okHandler(hit))(rr, req)
	return rr
}

func TestAllowedOriginIsEchoed(t *testing.T) {
	setAllowed(t, "https://lamsza.com", "https://szotar.lamsza.com")

	var hit bool
	rr := serve(http.MethodGet, "https://szotar.lamsza.com", &hit)

	if !hit {
		t.Fatal("handler was not reached")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://szotar.lamsza.com" {
		t.Errorf("Allow-Origin = %q, want the allowlisted origin", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want true", got)
	}
	if got := rr.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
}

func TestUnknownOriginIsNotReflected(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	for _, origin := range []string{
		"https://evil.test",
		"https://lamsza.com.evil.test",
		"http://lamsza.com",     // wrong scheme
		"https://lamsza.com:81", // wrong port
		"null",
	} {
		var hit bool
		rr := serve(http.MethodGet, origin, &hit)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin %q: Allow-Origin = %q, want no header", origin, got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("origin %q: Allow-Credentials leaked as %q", origin, got)
		}
		// A read still runs; the browser is the one that blocks the reply.
		if !hit {
			t.Errorf("origin %q: GET should still reach the handler", origin)
		}
	}
}

func TestNoOriginGetsNoWildcard(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	var hit bool
	rr := serve(http.MethodGet, "", &hit)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want no header for a request without Origin", got)
	}
	if !hit {
		t.Error("a same-origin or non-browser request must pass through")
	}
}

func TestPreflightFromUnknownOriginIsRefused(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	var hit bool
	rr := serve(http.MethodOptions, "https://evil.test", &hit)

	if rr.Code != http.StatusForbidden {
		t.Errorf("preflight status = %d, want 403", rr.Code)
	}
	if hit {
		t.Error("preflight must not reach the handler")
	}
}

func TestPreflightFromAllowedOriginSucceeds(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	var hit bool
	rr := serve(http.MethodOptions, "https://lamsza.com", &hit)

	if rr.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want 200", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight is missing Allow-Methods")
	}
	if hit {
		t.Error("preflight must not reach the handler")
	}
}

func TestWritesFromUnknownOriginAreRefused(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		var hit bool
		rr := serve(method, "https://evil.test", &hit)

		if rr.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want 403", method, rr.Code)
		}
		if hit {
			t.Errorf("%s reached the handler despite a foreign Origin", method)
		}
	}
}

func TestWritesWithoutOriginStillWork(t *testing.T) {
	setAllowed(t, "https://lamsza.com")

	var hit bool
	rr := serve(http.MethodPost, "", &hit)

	if rr.Code != http.StatusOK || !hit {
		t.Errorf("a POST without Origin got %d (handler hit: %v), want 200", rr.Code, hit)
	}
}
