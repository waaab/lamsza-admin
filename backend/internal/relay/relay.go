// Package relay forwards the admin app's /dictionary and /games sections to
// Szótár's and Játszótér's internal admin APIs (lamsza WAYS_OF_WORKING R18).
//
// The browser talks only to this backend, which has already checked the person
// (RequireAdmin) and written the audit record (audit.Wrap). The relay adds the
// app's service token and the acting admin's email, and sends the request
// server-to-server to the app's backend on 127.0.0.1 (R13). Cookies never go
// along: the other app trusts the token, not a browser session.
package relay

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/auth"
)

// Target is one app's internal admin API: its backend's base URL and the
// service token it expects. Either one empty turns the relay off.
type Target struct {
	Name  string // shown in errors, e.g. "Szótár"
	URL   string
	Token string
}

// maxReply caps what the relay copies back (an admin list can be long, a
// runaway reply should not be).
const maxReply = 8 << 20

var client = &http.Client{Timeout: 10 * time.Second}

// Handler serves a section's subtree, /api/admin/<section>/<rest>, by
// forwarding it to <target URL>/internal/admin/<rest>, with the query string,
// the method and the body. target is read on each request, so tests and config
// reloads see the current values.
func Handler(target func() Target) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := target()
		if t.URL == "" || t.Token == "" {
			writeError(w, http.StatusServiceUnavailable, t.Name+" adminisztrációja nincs beállítva.")
			return
		}
		rest, ok := sectionRest(r.URL.Path)
		if !ok || strings.Contains(rest, "..") {
			writeError(w, http.StatusNotFound, "Nincs ilyen adminisztrációs útvonal.")
			return
		}
		upstream, err := url.Parse(t.URL + "/internal/admin/" + strings.TrimPrefix(rest, "/"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, t.Name+" címe hibás.")
			return
		}
		upstream.RawQuery = r.URL.RawQuery

		req, err := http.NewRequestWithContext(r.Context(), r.Method, upstream.String(), r.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "A kérés nem állítható össze.")
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+t.Token)
		if u := auth.UserFromContext(r.Context()); u != nil {
			req.Header.Set("X-Admin-Email", u.Email)
		}

		res, err := client.Do(req)
		if err != nil {
			log.Printf("relay %s %s: %v", r.Method, upstream.Path, err)
			writeError(w, http.StatusBadGateway, t.Name+" most nem érhető el.")
			return
		}
		defer res.Body.Close()

		if ct := res.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(res.StatusCode)
		if _, err := io.Copy(w, io.LimitReader(res.Body, maxReply)); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("relay %s %s: copying the reply: %v", r.Method, upstream.Path, err)
		}
	}
}

// sectionRest returns what follows /api/admin/<section>/ in path.
func sectionRest(path string) (string, bool) {
	tail, ok := strings.CutPrefix(path, "/api/admin/")
	if !ok {
		return "", false
	}
	_, rest, ok := strings.Cut(tail, "/")
	return rest, ok
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"error":"`+msg+`"}`+"\n")
}
