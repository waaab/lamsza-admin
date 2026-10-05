package settings

import "testing"

func TestNormalizeSocialURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"https://www.facebook.com/szekelygugel", "https://www.facebook.com/szekelygugel"},
		{"http://example.com/x", "http://example.com/x"},
		{"www.instagram.com/lamsza", "https://www.instagram.com/lamsza"},
		{"javascript:alert(1)", ""},
		{"not a url", ""},
		{"ftp://files.example.com", ""},
	}
	for _, tc := range cases {
		if got := normalizeSocialURL(tc.in); got != tc.want {
			t.Errorf("normalizeSocialURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
