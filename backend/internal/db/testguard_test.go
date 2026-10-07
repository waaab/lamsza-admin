package db

import "testing"

func TestCheckTestDatabaseURL(t *testing.T) {
	t.Setenv("CI", "")
	cases := []struct {
		url string
		ok  bool
	}{
		{"postgres://u:p@localhost:5433/lamsza_admin_test?sslmode=disable", true},
		{"postgres://u:p@localhost:5433/lamsza?sslmode=disable", false},
		{"postgres://u:p@localhost:5433/lamsza_scratch?sslmode=disable", false},
		{"postgres://u:p@localhost:5433/?sslmode=disable", false},
	}
	for _, c := range cases {
		err := checkTestDatabaseURL(c.url)
		if (err == nil) != c.ok {
			t.Errorf("checkTestDatabaseURL(%q) err = %v, want ok=%v", c.url, err, c.ok)
		}
	}
}

func TestCheckTestDatabaseURLAllowsAnyNameInCI(t *testing.T) {
	t.Setenv("CI", "true")
	if err := checkTestDatabaseURL("postgres://u:p@localhost:5433/lamsza?sslmode=disable"); err != nil {
		t.Fatalf("CI run refused: %v", err)
	}
}

func TestTestDatabaseURLRequiresTheVariable(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "")
	if _, err := TestDatabaseURL(); err == nil {
		t.Fatal("TestDatabaseURL() with no TEST_DATABASE_URL returned no error")
	}
}

func TestGuardRefusesDevDatabaseInTests(t *testing.T) {
	t.Setenv("CI", "")
	if err := guardTestConnection("postgres://u:p@localhost:5433/lamsza?sslmode=disable"); err == nil {
		t.Fatal("guardTestConnection let a test binary open the dev database")
	}
}
