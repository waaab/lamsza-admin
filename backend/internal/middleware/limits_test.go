package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodyBytesFor(t *testing.T) {
	for path, want := range map[string]int64{
		"/api/admin/entry-images":   UploadMaxBodyBytes,
		"/api/admin/event-images/x": UploadMaxBodyBytes,
		"/api/admin/pages":          DefaultMaxBodyBytes,
		"/api/auth/google":          DefaultMaxBodyBytes,
	} {
		if got := MaxBodyBytesFor(path); got != want {
			t.Errorf("%s limit = %d, want %d", path, got, want)
		}
	}
}

func TestLimitBodyRejectsOversizedJSON(t *testing.T) {
	h := LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	big := `{"x":"` + strings.Repeat("a", int(DefaultMaxBodyBytes)) + `"}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/auth/google", strings.NewReader(big)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("oversized body: %d, want 400", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/auth/google", strings.NewReader(`{"x":"y"}`)))
	if rr.Code != http.StatusOK {
		t.Fatalf("small body: %d, want 200", rr.Code)
	}
}

func TestLimitBodyLetsAnUploadThroughUpToItsLimit(t *testing.T) {
	var read int64
	h := LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		read = n
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		}
	}))
	body := strings.NewReader(strings.Repeat("a", int(UploadMaxBodyBytes)))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/admin/entry-images", body))
	if rr.Code != http.StatusOK || read != UploadMaxBodyBytes {
		t.Fatalf("upload at the limit: status %d, read %d", rr.Code, read)
	}
}
