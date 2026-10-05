package weather

import "testing"

func TestGeocodeNameVariantsHyphenatesSpacedName(t *testing.T) {
	got := geocodeNameVariants("Miercurea Ciuc")
	if len(got) != 2 || got[0] != "Miercurea Ciuc" || got[1] != "Miercurea-Ciuc" {
		t.Fatalf("got %#v", got)
	}
}

func TestGeocodeNameVariantsKeepsSingleToken(t *testing.T) {
	got := geocodeNameVariants("  Kolozsvár ")
	if len(got) != 1 || got[0] != "Kolozsvár" {
		t.Fatalf("got %#v", got)
	}
}
