package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"backend/internal/db"
)

// TestAdminNewsFeedCheck covers /api/admin/news_feeds/check: it fetches a
// stored feed server-side and reports whether it is a usable RSS feed.
func TestAdminNewsFeedCheck(t *testing.T) {
	requireDB(t)

	rss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rss":
			w.Write([]byte(`<rss><channel><title>T</title>` +
				`<item><title>Egy</title><link>https://example.test/1</link></item>` +
				`<item><title>Kettő</title><link>https://example.test/2</link></item>` +
				`</channel></rss>`))
		default:
			w.Write([]byte(`<html>nem rss</html>`))
		}
	}))
	defer rss.Close()

	feed := func(label, url string) int {
		var id int
		if err := db.DB.QueryRow(`INSERT INTO news_feeds (title, feed_url) VALUES ($1, $2) RETURNING id`,
			testName(label), url).Scan(&id); err != nil {
			t.Fatalf("insert feed: %v", err)
		}
		t.Cleanup(func() { db.DB.Exec(`DELETE FROM news_feeds WHERE id = $1`, id) })
		return id
	}
	good := feed("feed-ok", rss.URL+"/rss")
	bad := feed("feed-not-rss", rss.URL+"/page")

	got := decodeObject(t, asAdmin(t, http.MethodGet, "/api/admin/news_feeds/check?id="+strconv.Itoa(good), nil), "check a good feed")
	if got["ok"] != true || got["items"] != float64(2) {
		t.Errorf("good feed check = %v, want ok with 2 items", got)
	}
	got = decodeObject(t, asAdmin(t, http.MethodGet, "/api/admin/news_feeds/check?id="+strconv.Itoa(bad), nil), "check a non-RSS feed")
	if got["ok"] != false || got["error"] == "" {
		t.Errorf("non-RSS feed check = %v, want ok=false with an error", got)
	}
	mustStatus(t, asAdmin(t, http.MethodGet, "/api/admin/news_feeds/check?id=2147483646", nil),
		http.StatusNotFound, "check a feed that does not exist")
	mustStatus(t, asAdmin(t, http.MethodPost, "/api/admin/news_feeds/check?id="+strconv.Itoa(good), nil),
		http.StatusMethodNotAllowed, "POST to the check route")
}
