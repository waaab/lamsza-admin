package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchSzotarWordsMapsAndCaps(t *testing.T) {
	var gotQ string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/words" {
			t.Fatalf("path %s", r.URL.Path)
		}
		gotQ = r.URL.Query().Get("q")
		words := make([]map[string]any, 0, 20)
		for i := 1; i <= 20; i++ {
			words = append(words, map[string]any{
				"id":            i,
				"headword":      "szo" + string(rune('a'+i%26)),
				"definition_hu": "def",
				"speech_types":  []string{"főnév"},
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"words": words})
	}))
	defer srv.Close()

	hits := fetchSzotarWords(context.Background(), srv.URL, "gagya", srv.Client())
	if gotQ != "gagya" {
		t.Fatalf("q=%q", gotQ)
	}
	if len(hits) != 15 {
		t.Fatalf("cap got %d", len(hits))
	}
	if hits[0].ID != 1 || hits[0].DefinitionHu != "def" || len(hits[0].SpeechTypes) != 1 {
		t.Fatalf("first hit %#v", hits[0])
	}
}

func TestFetchSzotarWordsEmptyOrigin(t *testing.T) {
	hits := fetchSzotarWords(context.Background(), "", "gagya", http.DefaultClient)
	if hits == nil || len(hits) != 0 {
		t.Fatalf("want empty non-nil, got %#v", hits)
	}
}

func TestFetchSzotarWordsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	hits := fetchSzotarWords(context.Background(), srv.URL, "gagya", srv.Client())
	if hits == nil || len(hits) != 0 {
		t.Fatalf("want empty on error, got %#v", hits)
	}
}

func TestFetchSzotarWordsSkipsBadRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"words": []map[string]any{
				{"id": 0, "headword": "bad"},
				{"id": 2, "headword": "  "},
				{"id": 3, "headword": "jó", "definition_hu": "ok", "speech_types": []string{"ige"}},
			},
		})
	}))
	defer srv.Close()
	hits := fetchSzotarWords(context.Background(), srv.URL, "jó", srv.Client())
	if len(hits) != 1 || hits[0].ID != 3 || hits[0].Headword != "jó" {
		t.Fatalf("got %#v", hits)
	}
}

func TestFetchSzotarWordsTimeoutContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{"words": []any{}})
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	hits := fetchSzotarWords(ctx, srv.URL, "x", srv.Client())
	if hits == nil || len(hits) != 0 {
		t.Fatalf("want empty on cancel, got %#v", hits)
	}
}
