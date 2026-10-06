package config

import "testing"

// The admin frontend reaches this API same-origin, but a browser still sends an
// Origin header on every write. If one of these hosts fell off the built-in
// list, every POST, PUT and DELETE from the admin UI would get a 403.
func TestDefaultListCoversTheAdminHosts(t *testing.T) {
	AppConfig.AllowedOrigins = nil
	ReloadAllowedOrigins()

	for _, origin := range []string{
		"https://admin.lamsza.com",
		"https://admin.lamsza.test",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	} {
		if !OriginAllowed(origin) {
			t.Errorf("%s is not on the built-in allowlist", origin)
		}
	}
}

func TestEnvListReplacesTheDefaults(t *testing.T) {
	AppConfig.AllowedOrigins = parseOriginList("https://admin.lamsza.com/ , https://admin.lamsza.com, ")
	ReloadAllowedOrigins()

	if got := len(AppConfig.AllowedOrigins); got != 1 {
		t.Fatalf("parsed %d origins, want 1 after the trailing slash and the duplicate drop out", got)
	}
	if !OriginAllowed("https://admin.lamsza.com") {
		t.Error("the configured origin was not allowed")
	}
	if OriginAllowed("http://localhost:5173") {
		t.Error("a development host survived CORS_ALLOWED_ORIGINS; the env list must replace the defaults")
	}
}
