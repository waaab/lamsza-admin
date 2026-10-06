package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/db"
)

func TestHandleHealthReportsDownWithoutDB(t *testing.T) {
	prev := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = prev })

	rec := httptest.NewRecorder()
	HandleHealth(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.OK || body.DB != "down" {
		t.Fatalf("body = %+v, want ok=false db=down", body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
}
