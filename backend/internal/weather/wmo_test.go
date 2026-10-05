package weather

import "testing"

func TestWmoToOwmIcon(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{0, "01d"},
		{1, "02d"},
		{2, "03d"},
		{3, "04d"},
		{45, "50d"},
		{48, "50d"},
		{61, "10d"},
		{80, "09d"},
		{95, "11d"},
		{71, "13d"},
	}
	for _, c := range cases {
		got := wmoToOwmIcon(c.code)
		if got != c.want {
			t.Errorf("wmoToOwmIcon(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

func TestWmoToDesc(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{0, "tiszta ég"},
		{1, "majdnem derült"},
		{2, "részben felhős"},
		{3, "borult"},
		{45, "köd"},
		{61, "eső"},
		{80, "zápor"},
		{95, "zivatar"},
		{71, "hó"},
	}
	for _, c := range cases {
		got := wmoToDesc(c.code)
		if got != c.want {
			t.Errorf("wmoToDesc(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}
