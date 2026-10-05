package webdomain

import "testing"

func TestCanonicalDomain(t *testing.T) {
	got, err := CanonicalDomain("https://www.kezdisorozo.com/menu")
	if err != nil || got != "kezdisorozo.com" {
		t.Fatalf("got %q err %v", got, err)
	}
	for _, raw := range []string{"kezdisorozo.com", "HTTP://KEZDISOROZO.com", "shop.kezdisorozo.com"} {
		got, err = CanonicalDomain(raw)
		if err != nil || got != "kezdisorozo.com" {
			t.Fatalf("%s -> %q err %v", raw, got, err)
		}
	}
	if _, err := CanonicalDomain("not a domain"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPlain(t *testing.T) {
	got, err := Plain("  <b>Csíki</b>  ", 120)
	if err != nil || got != "Csíki" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := Plain("   ", 120); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := Plain("abcd", 3); err == nil {
		t.Fatal("expected length error")
	}
}
