package auth

import (
	"backend/internal/config"
	"testing"
)

func TestIsAdminAllowlist(t *testing.T) {
	prev := config.AppConfig.AdminGoogleEmails
	t.Cleanup(func() { config.AppConfig.AdminGoogleEmails = prev })
	config.AppConfig.AdminGoogleEmails = []string{"attila.bogozi@gmail.com"}

	if !IsAdmin("attila.bogozi@gmail.com") {
		t.Fatal("expected allowlisted email to be admin")
	}
	if !IsAdmin("Attila.Bogozi@gmail.com") {
		t.Fatal("expected allowlist match to be case-insensitive")
	}
	if IsAdmin("other@gmail.com") {
		t.Fatal("expected non-allowlisted email to be rejected")
	}
	if IsAdmin("") {
		t.Fatal("empty email must not be admin")
	}
}

func TestApplyGoogleProfileClearsOmittedFields(t *testing.T) {
	got := ApplyGoogleProfile(UserProfile{
		Name:      "Old Name",
		GivenName: "Anna",
		Picture:   "https://example.test/old.jpg",
		Locale:    "hu",
	}, GoogleIdentity{
		Sub:   "sub-1",
		Email: "anna@example.test",
		Name:  "",
	})
	if got.Name != "" || got.GivenName != "" || got.Picture != "" || got.Locale != "" {
		t.Fatalf("omitted fields must clear, got %+v", got)
	}
	if got.Email != "anna@example.test" || got.Sub != "sub-1" {
		t.Fatalf("present fields must overwrite, got %+v", got)
	}
}

func TestParseTestIDToken(t *testing.T) {
	ident, err := ParseTestIDToken("test:admin@test.lamsza", "test-client")
	if err != nil {
		t.Fatalf("ParseTestIDToken: %v", err)
	}
	if ident.Email != "admin@test.lamsza" {
		t.Fatalf("email: got %q", ident.Email)
	}
	if ident.Sub == "" {
		t.Fatal("expected sub")
	}
	if _, err := ParseTestIDToken("not-a-test-token", "test-client"); err == nil {
		t.Fatal("expected error for non-test token")
	}
}
