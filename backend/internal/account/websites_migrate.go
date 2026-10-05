package account

import (
	"backend/internal/db"
	"backend/internal/webdomain"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

var tagRE = regexp.MustCompile(`<[^>]*>`)

func MigrateWebsites() {
	sqlBytes, err := os.ReadFile("migrations/websites.sql")
	if err != nil {
		log.Printf("MigrateWebsites (read sql): %v", err)
		return
	}
	if _, err := db.DB.Exec(string(sqlBytes)); err != nil {
		log.Printf("MigrateWebsites (schema): %v", err)
		return
	}

	rows, err := db.DB.Query(`SELECT id, name, notes, url FROM entries WHERE url IS NOT NULL AND url <> ''`)
	if err != nil {
		log.Printf("MigrateWebsites (select entries): %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, notes, url string
		if err := rows.Scan(&id, &name, &notes, &url); err != nil {
			continue
		}
		domainKey, err := webdomain.CanonicalDomain(url)
		if err != nil {
			continue
		}
		title := websiteTitle(name)
		description := websiteDescription(notes)
		_, _ = db.DB.Exec(`
			INSERT INTO websites (domain_key, submitted_host, title, description, status, entry_id)
			VALUES ($1, $1, $2, $3, 'approved', $4)
			ON CONFLICT (domain_key) DO NOTHING
		`, domainKey, title, description, id)
	}
}

func websiteTitle(name string) string {
	if title, err := webdomain.Plain(name, 120); err == nil {
		return title
	}
	return truncateRunes(stripTags(name), 120)
}

func websiteDescription(notes string) string {
	if strings.TrimSpace(notes) == "" {
		return ""
	}
	if description, err := webdomain.Plain(notes, 300); err == nil {
		return description
	}
	return ""
}

func stripTags(s string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(s, ""))
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
