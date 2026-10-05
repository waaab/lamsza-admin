package utils

import "testing"

func TestCanonicalEntryType(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{EntryTypeSzemely, EntryTypeSzemely},
		{"Személy", EntryTypeSzemely},
		{EntryTypeVallalkozas, EntryTypeVallalkozas},
		{"Vállalkozás", EntryTypeVallalkozas},
		{EntryTypeIntezmeny, EntryTypeIntezmeny},
		{"Intézmény", EntryTypeIntezmeny},
		{"service", ""},
		{"Szolgáltatás", ""},
		{"Cég", ""},
		{"Egyéb", ""},
		{"", ""},
		{"  ", ""},
	}
	for _, c := range cases {
		got := CanonicalEntryType(c.in)
		if got != c.want {
			t.Errorf("CanonicalEntryType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
