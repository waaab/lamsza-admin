package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNoDirListingServesFilesButNotIndexes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kep.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := http.StripPrefix("/media/", NoDirListing(http.FileServer(http.Dir(dir))))
	for path, want := range map[string]int{
		"/media/kep.png": http.StatusOK,
		"/media/":        http.StatusNotFound,
		"/media/sub/":    http.StatusNotFound,
		"/media/nincs":   http.StatusNotFound,
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rr.Code, want)
		}
	}
}
