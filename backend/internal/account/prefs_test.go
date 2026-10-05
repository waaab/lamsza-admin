package account

import "testing"

func TestNormalizeTheme(t *testing.T) {
	if _, err := NormalizeTheme("blue"); err == nil {
		t.Fatal("expected invalid theme")
	}
	got, err := NormalizeTheme("dark")
	if err != nil || got != "dark" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	got, err := NormalizeDisplayName("  Attila   Bögözi  ")
	if err != nil || got != "Attila Bögözi" {
		t.Fatalf("got %q %v", got, err)
	}
	if got, err := NormalizeDisplayName("   "); err != nil || got != "" {
		t.Fatalf("empty got %q %v", got, err)
	}
	if _, err := NormalizeDisplayName("ez a név már túl hosszú ide"); err == nil {
		t.Fatal("expected long display name to fail")
	}
}

func TestClampSlots(t *testing.T) {
	if ClampSlots(3) != 7 || ClampSlots(20) != 14 || ClampSlots(9) != 9 {
		t.Fatal("slot clamp mismatch")
	}
}

func TestNormalizeHistoryItem(t *testing.T) {
	item, err := NormalizeHistoryItem("kavezo", "Kávézó", "Vendéglő", "Csíkszereda", "")
	if err != nil || item.Slug != "kavezo" || item.Name != "Kávézó" {
		t.Fatalf("got %+v %v", item, err)
	}
	if _, err := NormalizeHistoryItem("", "Kávézó", "", "", ""); err == nil {
		t.Fatal("expected missing slug to fail")
	}
}

func TestNormalizeLink(t *testing.T) {
	title, url, color, err := NormalizeLink("  RMDSZ ", "https://rmdsz.ro", "")
	if err != nil || title != "RMDSZ" || url != "https://rmdsz.ro" || color != "#e6f0ff" {
		t.Fatalf("got %q %q %q %v", title, url, color, err)
	}
	if _, _, _, err := NormalizeLink("", "https://rmdsz.ro", ""); err == nil {
		t.Fatal("expected empty title to fail")
	}
}

func TestNormalizeFavorite(t *testing.T) {
	typ, id, err := NormalizeFavorite("attraction", 4)
	if err != nil || typ != "attraction" || id != 4 {
		t.Fatalf("got %s %d %v", typ, id, err)
	}
	if _, _, err := NormalizeFavorite("county", 1); err == nil {
		t.Fatal("expected unknown type to fail")
	}
	if _, _, err := NormalizeFavorite("entry", 0); err == nil {
		t.Fatal("expected id 0 to fail")
	}
}
