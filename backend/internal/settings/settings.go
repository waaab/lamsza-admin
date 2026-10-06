package settings

import (
	"backend/internal/config"
	"backend/internal/db"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PublicConfig is returned by GET /api/config/public (no auth)
type PublicConfig struct {
	WeatherCacheTTLMinutes int          `json:"weather_cache_ttl_minutes"`
	WeatherCacheVersion    string       `json:"weather_cache_version"`
	QuickLinksVersion      string       `json:"quick_links_version"`
	QuickLinksCount        int          `json:"quick_links_count"`
	WeatherIconStyle       string       `json:"weather_icon_style"`
	MyLocationSlug         string       `json:"my_location_slug"`
	MyLocationName         string       `json:"my_location_name"`
	MyLocationCounty       string       `json:"my_location_county"`
	MyLocationCountySlug   string       `json:"my_location_county_slug"`
	MyLocationType         string       `json:"my_location_type"`
	GoogleClientID         string       `json:"google_client_id"`
	SocialLinks            []SocialLink `json:"social_links"`
}

// SocialLink is one footer network link. Empty admin URLs are omitted.
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// HandlePublicConfig returns weather cache config for frontend
func HandlePublicConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ttl, _ := getSetting("weather_cache_ttl_minutes", "15")
	version, _ := getSetting("weather_cache_version", "1")
	qlVersion, _ := getSetting("quick_links_version", "1")
	iconStyle, _ := getSetting("weather_icon_style", "emoji")
	myLocSlug, _ := getSetting("my_location_slug", "csikszereda")
	ttlInt, _ := strconv.Atoi(ttl)
	qlCount := 0
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM quick_links").Scan(&qlCount)

	var myLocName, myLocCounty, myLocCountySlug, myLocType string
	_ = db.DB.QueryRow(
		"SELECT COALESCE(s.name,''), COALESCE(c.name,''), COALESCE(c.slug,''), COALESCE(s.type,'') FROM settlements s JOIN counties c ON s.county_id = c.id WHERE s.slug = $1",
		myLocSlug,
	).Scan(&myLocName, &myLocCounty, &myLocCountySlug, &myLocType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PublicConfig{
		WeatherCacheTTLMinutes: ttlInt,
		WeatherCacheVersion:    version,
		QuickLinksVersion:      qlVersion,
		QuickLinksCount:        qlCount,
		WeatherIconStyle:       iconStyle,
		MyLocationSlug:         myLocSlug,
		MyLocationName:         myLocName,
		MyLocationCounty:       myLocCounty,
		MyLocationCountySlug:   myLocCountySlug,
		MyLocationType:         myLocType,
		GoogleClientID:         config.AppConfig.GoogleClientID,
		SocialLinks:            FooterSocialLinks(),
	})
}

// FooterSocialLinks returns Facebook, Twitter, and Instagram in that order.
// A network is included only when its stored value is an http(s) URL.
func FooterSocialLinks() []SocialLink {
	specs := []struct{ key, label string }{
		{"social_facebook_url", "Facebook"},
		{"social_twitter_url", "Twitter"},
		{"social_instagram_url", "Instagram"},
	}
	out := []SocialLink{}
	for _, spec := range specs {
		raw, _ := getSetting(spec.key, "")
		if link := normalizeSocialURL(raw); link != "" {
			out = append(out, SocialLink{Label: spec.label, URL: link})
		}
	}
	return out
}

// normalizeSocialURL accepts a full http(s) URL, or a host we can prefix with https://.
func normalizeSocialURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return u
}

// GetSetting returns a site_settings value or default if missing/error (e.g. table not yet migrated)
func GetSetting(key, defaultVal string) (string, error) {
	var val string
	err := db.DB.QueryRow("SELECT value FROM site_settings WHERE key = $1", key).Scan(&val)
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

func getSetting(key, defaultVal string) (string, error) {
	return GetSetting(key, defaultVal)
}

// HandleAdminSettings: GET returns all settings, PUT accepts JSON body { "key": "value", ... }
func HandleAdminSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query("SELECT key, value FROM site_settings ORDER BY LOWER(key) ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		m := make(map[string]string)
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				continue
			}
			m[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
		return
	case http.MethodPut:
		var m map[string]string
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		for k, v := range m {
			_, err := db.DB.Exec(
				`INSERT INTO site_settings (key, value, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP)
				 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = CURRENT_TIMESTAMP`,
				k, v,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// IncrementWeatherCacheVersion bumps weather_cache_version so frontend weather cache is invalidated.
func IncrementWeatherCacheVersion() {
	var val string
	_ = db.DB.QueryRow("SELECT value FROM site_settings WHERE key = 'weather_cache_version'").Scan(&val)
	next := 1
	if n, err := strconv.Atoi(val); err == nil {
		next = n + 1
	}
	_, _ = db.DB.Exec(
		`INSERT INTO site_settings (key, value, updated_at) VALUES ('weather_cache_version', $1, CURRENT_TIMESTAMP)
		 ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = CURRENT_TIMESTAMP`,
		strconv.Itoa(next),
	)
}

// ClearWeatherCache increments weather_cache_version (admin)
func ClearWeatherCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	IncrementWeatherCacheVersion()
	w.WriteHeader(http.StatusOK)
}

// IncrementQuickLinksVersion bumps quick_links_version so frontend cache is invalidated
func IncrementQuickLinksVersion() {
	var val string
	_ = db.DB.QueryRow("SELECT value FROM site_settings WHERE key = 'quick_links_version'").Scan(&val)
	next := 1
	if n, err := strconv.Atoi(val); err == nil {
		next = n + 1
	}
	_, _ = db.DB.Exec(
		`INSERT INTO site_settings (key, value, updated_at) VALUES ('quick_links_version', $1, CURRENT_TIMESTAMP)
		 ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = CURRENT_TIMESTAMP`,
		strconv.Itoa(next),
	)
}
