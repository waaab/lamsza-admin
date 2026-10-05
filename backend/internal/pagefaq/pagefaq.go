package pagefaq

import (
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"log"
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

// Migrate creates/updates table, migrates legacy faq_markdown → faq_items, seeds rows.
func Migrate() {
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS page_faq_sections (
			id SERIAL PRIMARY KEY,
			section_key VARCHAR(64) NOT NULL UNIQUE,
			label_hu VARCHAR(255) NOT NULL DEFAULT '',
			faq_title VARCHAR(500) NOT NULL DEFAULT '',
			disclaimer_markdown TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("page_faq_sections create: %v", err)
		return
	}
	_, _ = db.DB.Exec(`ALTER TABLE page_faq_sections ADD COLUMN IF NOT EXISTS faq_items JSONB NOT NULL DEFAULT '[]'::jsonb`)

	migrateLegacyMarkdownToItems()
	seedAllSections()
	log.Println("page_faq_sections ready")
}

func migrateLegacyMarkdownToItems() {
	var n int
	_ = db.DB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'page_faq_sections' AND column_name = 'faq_markdown'
	`).Scan(&n)
	if n == 0 {
		return
	}
	rows, err := db.DB.Query(`SELECT id, COALESCE(faq_markdown, '') FROM page_faq_sections`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var md string
		if rows.Scan(&id, &md) != nil {
			continue
		}
		items := parseMarkdownToFAQItems(md)
		raw, err := json.Marshal(items)
		if err != nil {
			continue
		}
		_, _ = db.DB.Exec(`UPDATE page_faq_sections SET faq_items = $1::jsonb WHERE id = $2`, raw, id)
	}
	if rows.Err() != nil {
		return
	}
	_, _ = db.DB.Exec(`ALTER TABLE page_faq_sections DROP COLUMN IF EXISTS faq_markdown`)
}

func parseMarkdownToFAQItems(md string) []FAQItem {
	md = strings.TrimSpace(md)
	if md == "" {
		return nil
	}
	idx := mdFAQHeading.FindAllStringSubmatchIndex(md, -1)
	if len(idx) == 0 {
		return nil
	}
	var out []FAQItem
	for i, loc := range idx {
		title := strings.TrimSpace(md[loc[2]:loc[3]])
		bodyStart := loc[1]
		var bodyEnd int
		if i+1 < len(idx) {
			bodyEnd = idx[i+1][0]
		} else {
			bodyEnd = len(md)
		}
		body := strings.TrimSpace(md[bodyStart:bodyEnd])
		out = append(out, FAQItem{Question: title, Answer: body})
	}
	return out
}

func seedAllSections() {
	genericDisc := "A Lámsza tartalma tájékoztató jellegű; a pontosságért és a harmadik féltől származó adatokért nem vállalunk felelősséget."
	genericOne := []FAQItem{
		{
			Question: "Mire való ez az oldal?",
			Answer:   "A lamsza.com erdélyi magyar közösségi oldalakat gyűjt össze és közösségi média csatornákat biztosít az erdélyi magyar közösség számára.",
		},
	}

	hirekItems := []FAQItem{
		{Question: "Honnan származnak a hírek?", Answer: "Az oldal erdélyi hírforrások RSS-feedjeit olvassa be és jeleníti meg időrendi sorrendben. A források pontos listája a jobb oldali „Erdélyi hírforrások” panelen látható."},
		{Question: "Milyen sűrűn frissülnek a hírek?", Answer: "A hírek 30 percenként töltődnek be újra a források szerveréről. Az első látogatás után a böngésző gyorsítótár (localStorage) tárolja az adatokat, így a következő 30 percen belül történő látogatás azonnali."},
		{Question: "Hogyan szűrjük a híreket forrás szerint?", Answer: "A jobb oldali „Erdélyi hírforrások” panelen kattints bármelyik forrásra, hogy csak az adott oldal híreit lásd. A „Minden forrás” gombra kattintva visszatérhetsz a teljes listához. Mobilon a panel a lap tetején jelenik meg."},
		{Question: "Mit mutat a „Leggyakoribb témák” panel?", Answer: "Ez a panel automatikusan megszámolja a címekben legtöbbször előforduló szavakat, kizárva a közönséges ragokat és névelőket. Ha egy szó legalább kétszer szerepel, megjelenik a listában. Kattintva az adott szavat tartalmazó cikkekre szűr. A számlálás megváltozik, ha forrás-szűrőt használsz."},
		{Question: "Elérhetők-e a hírek internet nélkül is?", Answer: "Ha a szerver nem érhető el, az oldal az utolsó gyorsítótárban tárolt cikkeket mutatja - akár ha régebbiek is. Az utolsó frissítés ideje látható a forrás panel alatt."},
		{Question: "Hogyan szűrjük a híreket kulcsszó szerint?", Answer: "A „Leggyakoribb témák” panelen kattints bármelyik szóra, hogy csak az adott szót tartalmazó cikkeket lásd. A szűrő törlésével visszatérhetsz a teljes listához. Mobilon a panel a lap tetején jelenik meg."},
	}
	hirekDisc := "A lamsza.com hírolvasó egy ingyenes hírgyűjtő és szűrő szolgáltatás. A hírek tartalma és a hozzájuk tartozó képek az eredeti hírforrások szerzői jogi védelme alatt állnak. A lamsza.com nem vállal felelősséget ezen források tartalmáért. Az oldalon megjelenő időpont vagy dátum azt az időpillanatot jelöli, amikor a hírt a rendszerünk indexelte."

	indexItems := []FAQItem{
		{Question: "Honnan származnak az adatok?", Answer: "Az adatok a helyi szakemberektől és intézményektől származnak, akiket a rendszerünk folyamatosan indexel, hogy a legfrissebb elérhetőségeket biztosítsa."},
		{Question: "Hogyan kerülhet be valaki a címtárba?", Answer: "A beküldési folyamat hamarosan elérhető lesz az oldalon. Addig is, ha ismersz olyan szolgáltatót, aki még nem szerepel nálunk, keress minket bizalommal."},
		{Question: "Ingyenes-e a megjelenés?", Answer: "Igen, az alapvető megjelenés és az adatok listázása teljesen ingyenes minden helyi szolgáltató, mesterember és intézmény számára."},
		{Question: "Hogyan működik a keresés?", Answer: "A keresőnk kulcsszavak, kategóriák és települések alapján szűri a találatokat. A kereső prioritást ad a közvetlen név- és kategória-egyezéseknek, de a leírásokban is keres."},
	}
	indexDisc := "A lamsza.com indexe egy ingyenes információs szolgáltatás. Az adatok pontosságáért és a szolgáltatások minőségéért a lamsza.com nem vállal felelősséget. Kérjük, minden esetben ellenőrizze az adatokat a szolgáltatóval való kapcsolatfelvétel előtt."

	seeds := []struct {
		key, label, title string
		items             []FAQItem
		disc              string
	}{
		{"home", "Főoldal (/)", "Gyakori kérdések", genericOne, genericDisc},
		{"index", "Index (/index)", "Gyakori kérdések", indexItems, indexDisc},
		{"hirek", "Erdélyi hírek (/hirek)", "Hogyan működik ez az oldal?", hirekItems, hirekDisc},
		{"megyek", "Megyék listája (/megyek)", "Gyakori kérdések", genericOne, genericDisc},
		{"megye", "Megye oldal (…-megye)", "Gyakori kérdések", genericOne, genericDisc},
		{"telepules", "Település oldal", "Gyakori kérdések", genericOne, genericDisc},
		{"szekek", "Történelmi székek (/szekek)", "Gyakori kérdések", genericOne, genericDisc},
		{"szek", "Egy szék (/szekek/…)", "Gyakori kérdések", genericOne, genericDisc},
		{"varosok", "Városok (/varosok)", "Gyakori kérdések", genericOne, genericDisc},
		{"falvak", "Falvak (/falvak)", "Gyakori kérdések", genericOne, genericDisc},
		{"esemenyek", "Események lista", "Gyakori kérdések", genericOne, genericDisc},
		{"esemeny", "Esemény részletek", "Gyakori kérdések", genericOne, genericDisc},
		{"terkep", "Térkép (/terkep)", "Gyakori kérdések", genericOne, genericDisc},
		{"valtozasnaplo", "Változásnapló", "Gyakori kérdések", genericOne, genericDisc},
		{"bejegyzes", "Bejegyzés", "Gyakori kérdések", genericOne, genericDisc},
		{"iranyelvek", "Irányelvek", "Gyakori kérdések", genericOne, genericDisc},
	}

	for _, s := range seeds {
		raw, err := json.Marshal(s.items)
		if err != nil {
			continue
		}
		_, _ = db.DB.Exec(`
			INSERT INTO page_faq_sections (section_key, label_hu, faq_title, faq_items, disclaimer_markdown)
			VALUES ($1, $2, $3, $4::jsonb, $5)
			ON CONFLICT (section_key) DO NOTHING`,
			s.key, s.label, s.title, raw, s.disc,
		)
	}
}

func scanSection(row *sql.Row) (Section, error) {
	var s Section
	var raw []byte
	err := row.Scan(&s.ID, &s.SectionKey, &s.LabelHu, &s.FAQTitle, &raw, &s.DisclaimerMarkdown, &s.UpdatedAt)
	if err != nil {
		return s, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.FAQItems)
	}
	if s.FAQItems == nil {
		s.FAQItems = []FAQItem{}
	}
	return s, nil
}

// HandlePublic GET /api/page_faq?section=hirek
func HandlePublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("section")))
	if key == "" {
		http.Error(w, "Missing section", http.StatusBadRequest)
		return
	}
	row := db.DB.QueryRow(
		`SELECT id, section_key, label_hu, faq_title, COALESCE(faq_items, '[]'::jsonb), COALESCE(disclaimer_markdown,''), updated_at::text
		 FROM page_faq_sections WHERE LOWER(section_key) = $1`,
		key,
	)
	s, err := scanSection(row)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

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
