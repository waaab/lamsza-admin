package weather

import (
	"backend/internal/db"
	"backend/internal/settings"
	"encoding/json"
	"net/http"
	"strings"
)

// normalizeDesc is how lamsza looks a translation up: since its move to MET
// Norway, source_text is a base symbol code such as "partlycloudy".
func normalizeDesc(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
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
