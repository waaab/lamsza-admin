package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL       string
	Port              string
	WeatherAPIKey     string
	WeatherAPIComKey  string
	GoogleClientID    string
	AdminGoogleEmails []string
	SzotarOrigin      string
	// AllowedOrigins lists the exact browser origins that may read this API
	// cross-origin. Never reflect an unknown Origin back: with
	// Access-Control-Allow-Credentials that hands any site the signed-in reply.
	AllowedOrigins []string
	// DataAPI serves mondások, events, news, and the other content routes.
	// Auth and /api/config/public stay up when this is false.
	DataAPI  bool
	Features struct {
		Weather    bool
		Events     bool
		News       bool
		Mondasok   bool
		QuickLinks bool
		Search     bool
	}
}

var AppConfig Config

func Load() {
	// Load optional parent .env first, then cwd .env so the project always wins
	// (Previously ../.env alone could shadow ./.env when the server was started from repo root.)
	if _, err := os.Stat("../.env"); err == nil {
		loadEnvFile("../.env")
	}
	if _, err := os.Stat(".env"); err == nil {
		loadEnvFile(".env")
	}

	AppConfig.DatabaseURL = getEnv("DATABASE_URL", "postgres://lamsza_user:lamsza_password@localhost:5433/lamsza?sslmode=disable")
	AppConfig.Port = getEnv("PORT", "3000")
	AppConfig.WeatherAPIKey = getEnv("WEATHER_API_KEY", "")
	AppConfig.WeatherAPIComKey = getEnv("WEATHER_API_COM_KEY", "")

	AppConfig.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
	AppConfig.AdminGoogleEmails = parseEmailList(getEnv("ADMIN_GOOGLE_EMAILS", "attila.bogozi@gmail.com"))
	AppConfig.SzotarOrigin = strings.TrimRight(getEnv("SZOTAR_ORIGIN", ""), "/")
	AppConfig.AllowedOrigins = parseOriginList(getEnv("CORS_ALLOWED_ORIGINS", ""))
	ReloadAllowedOrigins()

	AppConfig.DataAPI = getBoolEnv("DATA_API", true)

	AppConfig.Features.Weather = getBoolEnv("FEATURE_WEATHER", true)
	AppConfig.Features.Events = getBoolEnv("FEATURE_EVENTS", true)
	AppConfig.Features.News = getBoolEnv("FEATURE_NEWS", true)
	AppConfig.Features.Mondasok = getBoolEnv("FEATURE_MONDASOK", true)
	AppConfig.Features.QuickLinks = getBoolEnv("FEATURE_QUICKLINKS", true)
	AppConfig.Features.Search = getBoolEnv("FEATURE_SEARCH", true)
}

// defaultAllowedOrigins covers the four production sites plus the local
// development hosts. The admin frontend reaches this API same-origin, so the
// sibling sites are here only to keep one list across the repos. Set
// CORS_ALLOWED_ORIGINS to replace the whole list, which is what production
// should do so the development hosts drop out.
var defaultAllowedOrigins = []string{
	"https://admin.lamsza.com",
	"https://lamsza.com",
	"https://www.lamsza.com",
	"https://szotar.lamsza.com",
	"https://jatszoter.lamsza.com",
	"https://admin.lamsza.test",
	"https://lamsza.test",
	"https://szotar.lamsza.test",
	"https://jatszoter.lamsza.test",
	"http://localhost:5173",
	"http://localhost:5174",
	"http://localhost:5175",
	"http://localhost:5176",
	"http://127.0.0.1:5173",
	"http://127.0.0.1:5174",
}

var allowedOriginSet map[string]bool

// OriginAllowed reports whether an Origin header value is on the allowlist.
// The match is exact: scheme, host and port must all agree, so a lookalike
// host such as "https://admin.lamsza.com.evil.test" never passes.
func OriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if allowedOriginSet == nil {
		ReloadAllowedOrigins()
	}
	return allowedOriginSet[origin]
}

// ReloadAllowedOrigins rebuilds the lookup set after AllowedOrigins changes.
// Load calls it; tests call it when they set their own list.
func ReloadAllowedOrigins() {
	origins := AppConfig.AllowedOrigins
	if len(origins) == 0 {
		origins = defaultAllowedOrigins
		AppConfig.AllowedOrigins = origins
	}
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		set[o] = true
	}
	allowedOriginSet = set
}

func parseOriginList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		origin := strings.TrimRight(strings.TrimSpace(p), "/")
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		out = append(out, origin)
	}
	return out
}

func parseEmailList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		email := strings.ToLower(strings.TrimSpace(p))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	if len(out) == 0 {
		return []string{"attila.bogozi@gmail.com"}
	}
	return out
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return strings.ToLower(value) == "true"
	}
	return fallback
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}
