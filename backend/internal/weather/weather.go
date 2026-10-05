package weather

import (
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/settings"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	providerOpenMeteo      = "open_meteo"
	providerWeatherAPICom  = "weatherapi_com"
	providerOpenWeatherMap = "openweathermap"
)

// fallbackDescHU: if an API returns English, we show Hungarian (admin can extend via site_settings later)
var fallbackDescHU = map[string]string{
	"overcast": "borult", "partly cloudy": "részben felhős", "clear": "tiszta ég",
	"cloudy": "felhős", "light rain": "enyhe eső", "moderate rain": "mérsékelt eső",
	"heavy rain": "erős eső", "light snow": "enyhe hó", "snow": "hó", "fog": "köd",
	"mist": "köd", "thunderstorm": "zivatar", "drizzle": "szitálás", "patchy rain": "helyenkénti eső",
	"patchy snow": "helyenkénti hó", "freezing fog": "fagyos köd", "patchy light rain": "helyenkénti enyhe eső",
	"light rain shower": "enyhe zápor", "moderate or heavy rain shower": "mérsékelt vagy erős zápor",
	"torrential rain shower": "zúduló zápor", "light sleet": "enyhe ónos eső", "sleet": "ónos eső",
	"light snow showers": "enyhe havas zápor", "blizzard": "hóvihar", "blowing snow": "havas szél",
	"patchy light snow": "helyenkénti enyhe hó", "moderate snow": "mérsékelt hó",
	"heavy snow": "erős hó", "thundery outbreaks": "zivataros kitörések",
	"patchy freezing drizzle": "helyenkénti fagyos szitálás", "freezing drizzle": "fagyos szitálás",
	"heavy freezing drizzle": "erős fagyos szitálás", "patchy sleet": "helyenkénti ónos eső",
	"moderate or heavy sleet": "mérsékelt vagy erős ónos eső", "light drizzle": "enyhe szitálás",
	"patchy rain possible": "helyenkénti eső lehetséges", "sunny": "napos",
}

// normalizeDesc for DB lookup and fallback map key
func normalizeDesc(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// translatedDesc returns the description in the requested language.
// Order: (1) admin translation from DB, (2) if lang is "hu", default Hungarian fallback map, (3) original text.
// Supports future multiple languages via lang (e.g. "hu", "ro", "de").
func translatedDesc(source, lang string) string {
	if source == "" {
		return source
	}
	key := normalizeDesc(source)
	// 1. Admin-edited translation from DB
	var translated string
	err := db.DB.QueryRow(
		"SELECT translated_text FROM weather_desc_translations WHERE source_text = $1 AND lang = $2",
		key, lang,
	).Scan(&translated)
	if err == nil && translated != "" {
		return translated
	}
	// 2. Default Hungarian when no admin translation
	if lang == "hu" {
		if hu, ok := fallbackDescHU[key]; ok {
			return hu
		}
	}
	// 3. Original (e.g. API already returned Hungarian or unknown phrase)
	return source
}

// UnifiedWeatherResponse is the single shape returned by GET /api/weather
type UnifiedWeatherResponse struct {
	Temp      int      `json:"temp"`
	TempMin   *int     `json:"temp_min,omitempty"`
	Desc      string   `json:"desc"`
	Icon      string   `json:"icon"`
	Source    string   `json:"source"`
	FetchedAt int64    `json:"fetched_at"`
	Humidity  *int     `json:"humidity,omitempty"`
	WindKph   *float64 `json:"wind_kph,omitempty"`
	PrecipMm  *float64 `json:"precip_mm,omitempty"`
}

// getProviderOrder returns provider IDs to try (default first if enabled, then other enabled)
func getProviderOrder() []string {
	defaultProv, _ := getSetting("weather_provider_default", "open_meteo")
	enabled := map[string]bool{
		providerOpenMeteo:      getSettingBool("weather_provider_open_meteo_enabled", true),
		providerWeatherAPICom:  getSettingBool("weather_provider_weatherapi_enabled", true),
		providerOpenWeatherMap: getSettingBool("weather_provider_openweathermap_enabled", true),
	}
	var order []string
	if enabled[defaultProv] {
		order = append(order, defaultProv)
	}
	for _, p := range []string{providerOpenMeteo, providerWeatherAPICom, providerOpenWeatherMap} {
		if p != defaultProv && enabled[p] {
			order = append(order, p)
		}
	}
	if len(order) == 0 {
		order = []string{providerOpenMeteo, providerWeatherAPICom, providerOpenWeatherMap}
	}
	return order
}

func getSetting(key, defaultVal string) (string, error) {
	var val string
	err := db.DB.QueryRow("SELECT value FROM site_settings WHERE key = $1", key).Scan(&val)
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

func getSettingBool(key string, defaultVal bool) bool {
	s, err := getSetting(key, "")
	if err != nil || s == "" {
		return defaultVal
	}
	return strings.ToLower(s) == "true" || s == "1"
}

func HandleWeather(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	// Direct coordinates (e.g. for attractions): same provider order as slug-based
	if latStr != "" && lonStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
				if out, prov, err := fetchWeatherByCoords(lat, lon); err == nil && out != nil {
					out.FetchedAt = time.Now().Unix()
					out.Source = providerDisplayName(prov)
					out.Desc = translatedDesc(out.Desc, "hu")
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(out)
					return
				}
			}
		}
	}

	if slug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	var name, nameRo string
	var parentID *int
	err := db.DB.QueryRow(
		"SELECT name, COALESCE(name_ro, ''), parent_id FROM settlements WHERE slug = $1", slug,
	).Scan(&name, &nameRo, &parentID)
	if err != nil {
		// Try attraction
		var lat, lon float64
		if err := db.DB.QueryRow(
			"SELECT gl.latitude, gl.longitude FROM attractions a JOIN geo_locations gl ON a.location_id = gl.id WHERE a.slug = $1", slug,
		).Scan(&lat, &lon); err == nil {
			if out, prov, err := fetchWeatherByCoords(lat, lon); err == nil && out != nil {
				out.FetchedAt = time.Now().Unix()
				out.Source = providerDisplayName(prov)
				out.Desc = translatedDesc(out.Desc, "hu")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
				return
			}
		}
		log.Println("Weather slug resolution error:", err)
		name = strings.Title(strings.ReplaceAll(slug, "-", " "))
	}

	searchName := name
	if nameRo != "" {
		searchName = nameRo
	}

	order := getProviderOrder()
	now := time.Now().Unix()
	var out *UnifiedWeatherResponse

	for _, prov := range order {
		switch prov {
		case providerOpenMeteo:
			out, err = fetchOpenMeteo(searchName)
		case providerWeatherAPICom:
			out, err = fetchWeatherAPICom(searchName, config.AppConfig.WeatherAPIComKey)
		case providerOpenWeatherMap:
			out, err = fetchOpenWeatherMap(searchName, config.AppConfig.WeatherAPIKey)
		default:
			continue
		}
		if err == nil && out != nil {
			out.FetchedAt = now
			out.Source = providerDisplayName(prov)
			out.Desc = translatedDesc(out.Desc, "hu")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(out)
			return
		}
	}

	// Parent fallback (only for same provider logic - use first provider again with parent name)
	if parentID != nil && out == nil {
		var parentName, parentNameRo string
		perr := db.DB.QueryRow(
			"SELECT name, COALESCE(name_ro, '') FROM settlements WHERE id = $1", *parentID,
		).Scan(&parentName, &parentNameRo)
		if perr == nil {
			parentSearch := parentName
			if parentNameRo != "" {
				parentSearch = parentNameRo
			}
			log.Printf("Weather: trying parent %q", parentSearch)
			for _, prov := range order {
				switch prov {
				case providerOpenMeteo:
					out, err = fetchOpenMeteo(parentSearch)
				case providerWeatherAPICom:
					out, err = fetchWeatherAPICom(parentSearch, config.AppConfig.WeatherAPIComKey)
				case providerOpenWeatherMap:
					out, err = fetchOpenWeatherMap(parentSearch, config.AppConfig.WeatherAPIKey)
				default:
					continue
				}
				if err == nil && out != nil {
					out.FetchedAt = now
					out.Source = providerDisplayName(prov)
					out.Desc = translatedDesc(out.Desc, "hu")
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(out)
					return
				}
			}
		}
	}

	http.Error(w, "Weather provider error", http.StatusInternalServerError)
}

func providerDisplayName(id string) string {
	switch id {
	case providerOpenMeteo:
		return "Open-Meteo"
	case providerWeatherAPICom:
		return "WeatherAPI.com"
	case providerOpenWeatherMap:
		return "OpenWeatherMap"
	}
	return id
}

// fetchWeatherByCoords tries each enabled provider in order (same policy as slug-based requests).
func fetchWeatherByCoords(lat, lon float64) (*UnifiedWeatherResponse, string, error) {
	order := getProviderOrder()
	var lastErr error
	for _, prov := range order {
		var out *UnifiedWeatherResponse
		var err error
		switch prov {
		case providerOpenMeteo:
			out, err = fetchOpenMeteoByCoords(lat, lon)
		case providerWeatherAPICom:
			out, err = fetchWeatherAPIComByCoords(lat, lon, config.AppConfig.WeatherAPIComKey)
		case providerOpenWeatherMap:
			out, err = fetchOpenWeatherMapByCoords(lat, lon, config.AppConfig.WeatherAPIKey)
		default:
			continue
		}
		if err == nil && out != nil {
			return out, prov, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("all weather providers failed for coordinates")
}

// fetchOpenMeteoByCoords fetches weather directly by coordinates (no geocoding)
func fetchOpenMeteoByCoords(lat, lon float64) (*UnifiedWeatherResponse, error) {
	forecastURL := "https://api.open-meteo.com/v1/forecast?latitude=" + strconv.FormatFloat(lat, 'f', -1, 64) +
		"&longitude=" + strconv.FormatFloat(lon, 'f', -1, 64) +
		"&current=temperature_2m,weather_code,relative_humidity_2m,wind_speed_10m,precipitation"
	resp, err := http.Get(forecastURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo forecast: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var fm struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			WeatherCode int     `json:"weather_code"`
			Humidity    float64 `json:"relative_humidity_2m"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			Precip      float64 `json:"precipitation"`
		} `json:"current"`
	}
	if err := json.Unmarshal(body, &fm); err != nil {
		return nil, err
	}
	temp := int(fm.Current.Temperature)
	icon := wmoToOwmIcon(fm.Current.WeatherCode)
	desc := wmoToDesc(fm.Current.WeatherCode)
	humidity := int(fm.Current.Humidity)
	windKph := fm.Current.WindSpeed
	precipMm := fm.Current.Precip
	return &UnifiedWeatherResponse{
		Temp: temp, TempMin: &temp, Desc: desc, Icon: icon,
		Humidity: &humidity, WindKph: &windKph, PrecipMm: &precipMm,
	}, nil
}

// geocodeNameVariants lists Open-Meteo search strings. The geocoder stores
// Miercurea Ciuc as "Miercurea-Ciuc", so a spaced Romanian name returns no hit.
func geocodeNameVariants(city string) []string {
	city = strings.TrimSpace(city)
	if city == "" {
		return nil
	}
	variants := []string{city}
	hyphenated := strings.Join(strings.Fields(city), "-")
	if hyphenated != city {
		variants = append(variants, hyphenated)
	}
	return variants
}

func isNoGeocodeResults(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no results")
}

// Open-Meteo: geocode then current weather
func fetchOpenMeteo(city string) (*UnifiedWeatherResponse, error) {
	names := geocodeNameVariants(city)
	if len(names) == 0 {
		return nil, fmt.Errorf("open-meteo geocode: no results")
	}
	var lastErr error
	for _, name := range names {
		out, err := fetchOpenMeteoByName(name)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !isNoGeocodeResults(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

func fetchOpenMeteoByName(city string) (*UnifiedWeatherResponse, error) {
	geoURL := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(city) + "&count=1"
	resp, err := http.Get(geoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo geocode: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var geo struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil, err
	}
	if len(geo.Results) == 0 {
		return nil, fmt.Errorf("open-meteo geocode: no results")
	}
	lat := geo.Results[0].Latitude
	lon := geo.Results[0].Longitude

	return fetchOpenMeteoByCoords(lat, lon)
}

func wmoToOwmIcon(code int) string {
	switch {
	case code == 0:
		return "01d"
	case code == 1:
		return "02d"
	case code == 2:
		return "03d"
	case code == 3:
		return "04d"
	case code == 45 || code == 48:
		return "50d"
	case code >= 51 && code <= 67:
		return "10d"
	case code >= 71 && code <= 77:
		return "13d"
	case code >= 80 && code <= 82:
		return "09d"
	case code >= 85 && code <= 86:
		return "13d"
	case code >= 95 && code <= 99:
		return "11d"
	default:
		return "02d"
	}
}

func wmoToDesc(code int) string {
	switch {
	case code == 0:
		return "tiszta ég"
	case code == 1:
		return "majdnem derült"
	case code == 2:
		return "részben felhős"
	case code == 3:
		return "borult"
	case code == 45 || code == 48:
		return "köd"
	case code >= 51 && code <= 67:
		return "eső"
	case code >= 71 && code <= 77:
		return "hó"
	case code >= 80 && code <= 82:
		return "zápor"
	case code >= 85 && code <= 86:
		return "havas zápor"
	case code >= 95 && code <= 99:
		return "zivatar"
	default:
		return "részben felhős"
	}
}

// WeatherAPI.com
func fetchWeatherAPICom(city, apiKey string) (*UnifiedWeatherResponse, error) {
	if apiKey == "" {
		return nil, nil
	}
	u := "https://api.weatherapi.com/v1/current.json?key=" + url.QueryEscape(apiKey) + "&q=" + url.QueryEscape(city) + "&lang=hu"
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weatherapi.com: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return parseWeatherAPIComCurrentJSON(body)
}

func fetchWeatherAPIComByCoords(lat, lon float64, apiKey string) (*UnifiedWeatherResponse, error) {
	if apiKey == "" {
		return nil, nil
	}
	q := strconv.FormatFloat(lat, 'f', -1, 64) + "," + strconv.FormatFloat(lon, 'f', -1, 64)
	u := "https://api.weatherapi.com/v1/current.json?key=" + url.QueryEscape(apiKey) + "&q=" + url.QueryEscape(q) + "&lang=hu"
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weatherapi.com: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return parseWeatherAPIComCurrentJSON(body)
}

func parseWeatherAPIComCurrentJSON(body []byte) (*UnifiedWeatherResponse, error) {
	var wa struct {
		Current struct {
			TempC     float64 `json:"temp_c"`
			Humidity  int     `json:"humidity"`
			WindKph   float64 `json:"wind_kph"`
			PrecipMm  float64 `json:"precip_mm"`
			Condition struct {
				Text string `json:"text"`
				Code int    `json:"code"`
			} `json:"condition"`
		} `json:"current"`
	}
	if err := json.Unmarshal(body, &wa); err != nil {
		return nil, err
	}
	temp := int(wa.Current.TempC)
	icon := weatherapiCodeToOwm(wa.Current.Condition.Code)
	humidity := wa.Current.Humidity
	windKph := wa.Current.WindKph
	precipMm := wa.Current.PrecipMm
	return &UnifiedWeatherResponse{
		Temp:     temp,
		TempMin:  &temp,
		Desc:     wa.Current.Condition.Text,
		Icon:     icon,
		Humidity: &humidity,
		WindKph:  &windKph,
		PrecipMm: &precipMm,
	}, nil
}

func weatherapiCodeToOwm(code int) string {
	// WeatherAPI.com condition codes -> OWM icon
	if code == 1000 {
		return "01d"
	}
	if code >= 1003 && code <= 1009 {
		return "04d"
	}
	if code >= 1063 && code <= 1183 {
		return "10d"
	}
	if code >= 1195 && code <= 1282 {
		return "11d"
	}
	if code >= 1066 && code <= 1252 {
		return "13d"
	}
	if code == 1135 || code == 1147 {
		return "50d"
	}
	return "02d"
}

// OpenWeatherMap (existing)
func fetchOpenWeatherMap(city, apiKey string) (*UnifiedWeatherResponse, error) {
	if apiKey == "" {
		return nil, nil
	}
	weatherURL := "https://api.openweathermap.org/data/2.5/weather?q=" +
		url.QueryEscape(city) + ",RO&appid=" + apiKey + "&units=metric&lang=hu"
	resp, err := http.Get(weatherURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openweathermap: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return parseOpenWeatherMapCurrentJSON(body)
}

func fetchOpenWeatherMapByCoords(lat, lon float64, apiKey string) (*UnifiedWeatherResponse, error) {
	if apiKey == "" {
		return nil, nil
	}
	weatherURL := "https://api.openweathermap.org/data/2.5/weather?lat=" +
		strconv.FormatFloat(lat, 'f', -1, 64) + "&lon=" + strconv.FormatFloat(lon, 'f', -1, 64) +
		"&appid=" + url.QueryEscape(apiKey) + "&units=metric&lang=hu"
	resp, err := http.Get(weatherURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openweathermap: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return parseOpenWeatherMapCurrentJSON(body)
}

func parseOpenWeatherMapCurrentJSON(body []byte) (*UnifiedWeatherResponse, error) {
	var owm struct {
		Main struct {
			Temp     float64 `json:"temp"`
			TempMin  float64 `json:"temp_min"`
			Humidity int     `json:"humidity"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
		Rain *struct {
			OneH float64 `json:"1h"`
		} `json:"rain"`
		Weather []struct {
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
	}
	if err := json.Unmarshal(body, &owm); err != nil || len(owm.Weather) == 0 {
		return nil, err
	}
	temp := int(owm.Main.Temp)
	tmin := int(owm.Main.TempMin)
	humidity := owm.Main.Humidity
	windKph := owm.Wind.Speed * 3.6
	var precipMm float64
	if owm.Rain != nil {
		precipMm = owm.Rain.OneH
	}
	return &UnifiedWeatherResponse{
		Temp:     temp,
		TempMin:  &tmin,
		Desc:     owm.Weather[0].Description,
		Icon:     owm.Weather[0].Icon,
		Humidity: &humidity,
		WindKph:  &windKph,
		PrecipMm: &precipMm,
	}, nil
}

type cityWeather struct {
	City         string  `json:"city"`
	Slug         string  `json:"slug"`
	Temp         float64 `json:"temp"`
	TempMin      float64 `json:"temp_min"`
	Desc         string  `json:"desc"`
	Icon         string  `json:"icon"`
	IsCountySeat bool    `json:"is_county_seat"`
	Source       string  `json:"source,omitempty"`
}

// HandleCountyWeather returns weather for all major cities in a county (multi-provider, first success per city)
func HandleCountyWeather(w http.ResponseWriter, r *http.Request) {
	countySlug := r.URL.Query().Get("slug")
	if countySlug == "" {
		http.Error(w, "Missing slug", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(
		`SELECT s.slug, s.name, COALESCE(s.name_ro, ''), COALESCE(s.is_county_seat, false) FROM settlements s
		 JOIN counties c ON s.county_id = c.id
		 WHERE c.slug = $1 AND s.type IN ('város', 'municípium')
		 ORDER BY s.name`, countySlug,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type locInfo struct {
		slug, name, nameRo string
		isCountySeat       bool
	}
	var cities []locInfo
	for rows.Next() {
		var l locInfo
		if err := rows.Scan(&l.slug, &l.name, &l.nameRo, &l.isCountySeat); err == nil {
			cities = append(cities, l)
		}
	}

	if len(cities) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	order := getProviderOrder()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []cityWeather

	for _, city := range cities {
		wg.Add(1)
		go func(c locInfo) {
			defer wg.Done()
			searchName := c.name
			if c.nameRo != "" {
				searchName = c.nameRo
			}
			var out *UnifiedWeatherResponse
			var err error
			for _, prov := range order {
				switch prov {
				case providerOpenMeteo:
					out, err = fetchOpenMeteo(searchName)
				case providerWeatherAPICom:
					out, err = fetchWeatherAPICom(searchName, config.AppConfig.WeatherAPIComKey)
				case providerOpenWeatherMap:
					out, err = fetchOpenWeatherMap(searchName, config.AppConfig.WeatherAPIKey)
				default:
					continue
				}
				if err == nil && out != nil {
					mu.Lock()
					results = append(results, cityWeather{
						City:         c.name,
						Slug:         c.slug,
						Temp:         float64(out.Temp),
						TempMin:      float64(out.Temp),
						Desc:         translatedDesc(out.Desc, "hu"),
						Icon:         out.Icon,
						IsCountySeat: c.isCountySeat,
						Source:       providerDisplayName(prov),
					})
					mu.Unlock()
					return
				}
			}
		}(city)
	}
	wg.Wait()

	if results == nil {
		results = []cityWeather{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// MigrateWeatherTranslations creates the weather_desc_translations table (multi-language support).
func MigrateWeatherTranslations() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS weather_desc_translations (
			id SERIAL PRIMARY KEY,
			source_text VARCHAR(255) NOT NULL,
			lang VARCHAR(10) NOT NULL,
			translated_text VARCHAR(255) NOT NULL,
			UNIQUE(source_text, lang)
		)
	`)
	if err != nil {
		log.Printf("weather_desc_translations create: %v", err)
		return
	}
	log.Println("Weather translations table ready")
}

// WeatherTranslation row for admin API
type WeatherTranslation struct {
	ID             int    `json:"id"`
	SourceText     string `json:"source_text"`
	Lang           string `json:"lang"`
	TranslatedText string `json:"translated_text"`
}

// HandleAdminWeatherTranslations: GET list, POST create, PUT update, DELETE
func HandleAdminWeatherTranslations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query("SELECT id, source_text, lang, translated_text FROM weather_desc_translations ORDER BY LOWER(source_text) ASC, lang ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var list []WeatherTranslation
		for rows.Next() {
			var t WeatherTranslation
			if err := rows.Scan(&t.ID, &t.SourceText, &t.Lang, &t.TranslatedText); err != nil {
				continue
			}
			list = append(list, t)
		}
		if list == nil {
			list = []WeatherTranslation{}
		}
		json.NewEncoder(w).Encode(list)
		return
	case http.MethodPost:
		var t WeatherTranslation
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		t.SourceText = normalizeDesc(t.SourceText)
		if t.SourceText == "" || t.Lang == "" || t.TranslatedText == "" {
			http.Error(w, "source_text, lang, translated_text required", http.StatusBadRequest)
			return
		}
		err := db.DB.QueryRow(
			`INSERT INTO weather_desc_translations (source_text, lang, translated_text) VALUES ($1, $2, $3) RETURNING id`,
			t.SourceText, t.Lang, t.TranslatedText,
		).Scan(&t.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		settings.IncrementWeatherCacheVersion()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)
		return
	case http.MethodPut:
		var t WeatherTranslation
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if t.ID == 0 {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		t.SourceText = normalizeDesc(t.SourceText)
		_, err := db.DB.Exec(
			`UPDATE weather_desc_translations SET source_text = $1, lang = $2, translated_text = $3 WHERE id = $4`,
			t.SourceText, t.Lang, t.TranslatedText, t.ID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		settings.IncrementWeatherCacheVersion()
		w.WriteHeader(http.StatusOK)
		return
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		_, err := db.DB.Exec("DELETE FROM weather_desc_translations WHERE id = $1", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		settings.IncrementWeatherCacheVersion()
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
