package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The site_settings contract (lamsza OPEN_ITEMS): every key the Beállítások
// tab writes (frontend/src/lib/settingsSections.js) is one lamsza's backend
// reads, in internal/settings or the weather worker. A key renamed on one
// side would otherwise save happily and change nothing. Needs the lamsza repo
// (LAMSZA_REPO, else next to this one); CI checks it out.
func TestSiteSettingsKeysAreReadByLamsza(t *testing.T) {
	lamsza := os.Getenv("LAMSZA_REPO")
	if lamsza == "" {
		lamsza = filepath.Join("..", "..", "lamsza")
	}
	var lamszaSrc strings.Builder
	for _, f := range []string{"backend/internal/settings/settings.go", "backend/internal/weather/worker.go"} {
		b, err := os.ReadFile(filepath.Join(lamsza, f))
		if err != nil {
			if os.Getenv("ADMIN_TEST_DB_REQUIRED") != "" {
				t.Fatalf("lamsza repo not found at %s: %v", lamsza, err)
			}
			t.Skipf("lamsza repo not found at %s (set LAMSZA_REPO): %v", lamsza, err)
		}
		lamszaSrc.Write(b)
	}

	sections, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "settingsSections.js"))
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, list := range regexp.MustCompile(`keys:\s*\[([^\]]*)\]`).FindAllStringSubmatch(string(sections), -1) {
		for _, k := range regexp.MustCompile(`"([a-z0-9_]+)"`).FindAllStringSubmatch(list[1], -1) {
			keys = append(keys, k[1])
		}
	}
	if len(keys) < 9 {
		t.Fatalf("found %d keys in settingsSections.js, want the 9 the tab edits: %v", len(keys), keys)
	}
	for _, k := range keys {
		if !strings.Contains(lamszaSrc.String(), `"`+k+`"`) {
			t.Errorf("admin writes site_settings key %q, which lamsza never reads", k)
		}
	}
}
