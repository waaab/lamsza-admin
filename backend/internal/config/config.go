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

	AppConfig.DataAPI = getBoolEnv("DATA_API", true)

	AppConfig.Features.Weather = getBoolEnv("FEATURE_WEATHER", true)
	AppConfig.Features.Events = getBoolEnv("FEATURE_EVENTS", true)
	AppConfig.Features.News = getBoolEnv("FEATURE_NEWS", true)
	AppConfig.Features.Mondasok = getBoolEnv("FEATURE_MONDASOK", true)
	AppConfig.Features.QuickLinks = getBoolEnv("FEATURE_QUICKLINKS", true)
	AppConfig.Features.Search = getBoolEnv("FEATURE_SEARCH", true)
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
