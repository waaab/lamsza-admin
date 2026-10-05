package handlers

import (
	"encoding/json"
	"testing"
)

func TestSanitizePhotosKeepsHttpAndMedia(t *testing.T) {
	raw := []byte(`[
		{"url":"javascript:alert(1)","alt":"x"},
		{"url":"/api/media/entry-images/ok.jpg","alt":"Udvar","title":"Udvar","description":"Nyár","width":800,"height":600},
		{"url":"https://cdn.example.com/p.jpg","width":0,"height":0}
	]`)
	got := sanitizePhotos(raw)
	var photos []entryPhoto
	if err := json.Unmarshal(got, &photos); err != nil {
		t.Fatal(err)
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got %d (%s)", len(photos), string(got))
	}
	if photos[0].URL != "/api/media/entry-images/ok.jpg" || photos[0].Alt != "Udvar" || photos[0].Width != 800 {
		t.Fatalf("unexpected first photo: %+v", photos[0])
	}
	if photos[1].Width != defaultPhotoWidth || photos[1].Height != defaultPhotoHeight {
		t.Fatalf("https photo should get default size, got %dx%d", photos[1].Width, photos[1].Height)
	}
}

func TestSanitizePhotosKeepsCredit(t *testing.T) {
	raw := []byte(`[{"url":"https://cdn.example.com/p.jpg","copyright":"© Iliuta Goean","uploader":"Iliuta Goean"}]`)
	got := sanitizePhotos(raw)
	var photos []entryPhoto
	if err := json.Unmarshal(got, &photos); err != nil {
		t.Fatal(err)
	}
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo, got %d (%s)", len(photos), string(got))
	}
	if photos[0].Copyright != "Iliuta Goean" || photos[0].Uploader != "Iliuta Goean" {
		t.Fatalf("credit was not kept: %+v", photos[0])
	}
}

func TestJsonArrayOrEmpty(t *testing.T) {
	if string(jsonArrayOrEmpty(nil)) != "[]" {
		t.Fatal("nil should be []")
	}
	if string(jsonArrayOrEmpty([]byte(`{}`))) != "[]" {
		t.Fatal("object should be []")
	}
	if string(jsonArrayOrEmpty([]byte(`[{"url":"https://x.test/a.jpg"}]`))) != `[{"url":"https://x.test/a.jpg"}]` {
		t.Fatal("valid array should pass through")
	}
}
