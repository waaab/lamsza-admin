package utils

import "testing"

func TestCanonicalEntryCategory(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Bútor", "Bútor"},
		{"  Étterem  ", "Étterem"},
		{"egeszsegugy", "egeszsegugy"},
		{"Vendéglő", "Vendéglő"},
		{"", ""},
		{"  ", ""},
	}
	for _, c := range cases {
		got := CanonicalEntryCategory(c.in)
		if got != c.want {
			t.Errorf("CanonicalEntryCategory(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCategoryOffersDelivery(t *testing.T) {
	for _, name := range []string{"Étterem", "Kávézó", "Cukrászda", "Pékség", "Söröző"} {
		if !CategoryOffersDelivery(name) {
			t.Fatalf("CategoryOffersDelivery(%q) = false", name)
		}
	}
	if CategoryOffersDelivery("Bútor") || CategoryOffersDelivery("Étkezés") {
		t.Fatal("delivery is limited to the Étkezés subcategories that take orders out")
	}
}
