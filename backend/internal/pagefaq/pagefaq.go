package pagefaq

import (
	"backend/internal/db"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// FAQItem is one question/answer pair (public + admin JSON).
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Section is FAQ + disclaimer copy for one logical site area (e.g. hirek).
type Section struct {
	ID                 int       `json:"id"`
	SectionKey         string    `json:"section_key"`
	LabelHu            string    `json:"label_hu"`
	FAQTitle           string    `json:"faq_title"`
	FAQItems           []FAQItem `json:"faq_items"`
	DisclaimerMarkdown string    `json:"disclaimer_markdown"`
	UpdatedAt          string    `json:"updated_at"`
}

var mdFAQHeading = regexp.MustCompile(`(?m)^###\s+(.+)$`)

// HandleAdmin GET all, PUT update
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query(
			`SELECT id, section_key, label_hu, faq_title, COALESCE(faq_items, '[]'::jsonb), COALESCE(disclaimer_markdown,''), updated_at::text
			 FROM page_faq_sections ORDER BY LOWER(label_hu) ASC, id ASC`,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var list []Section
		for rows.Next() {
			var s Section
			var raw []byte
			if err := rows.Scan(&s.ID, &s.SectionKey, &s.LabelHu, &s.FAQTitle, &raw, &s.DisclaimerMarkdown, &s.UpdatedAt); err != nil {
				continue
			}
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &s.FAQItems)
			}
			if s.FAQItems == nil {
				s.FAQItems = []FAQItem{}
			}
			list = append(list, s)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []Section{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPut:
		var body struct {
			ID                 int       `json:"id"`
			LabelHu            string    `json:"label_hu"`
			FAQTitle           string    `json:"faq_title"`
			FAQItems           []FAQItem `json:"faq_items"`
			DisclaimerMarkdown string    `json:"disclaimer_markdown"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if body.ID == 0 {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if body.FAQItems == nil {
			body.FAQItems = []FAQItem{}
		}
		raw, err := json.Marshal(body.FAQItems)
		if err != nil {
			http.Error(w, "Invalid faq_items", http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec(
			`UPDATE page_faq_sections SET label_hu = $1, faq_title = $2, faq_items = $3::jsonb, disclaimer_markdown = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $5`,
			strings.TrimSpace(body.LabelHu),
			strings.TrimSpace(body.FAQTitle),
			raw,
			body.DisclaimerMarkdown,
			body.ID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
