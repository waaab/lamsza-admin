package handlers

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const maxEntryPhotos = 24
const maxPhotoTextRunes = 500

type entryPhoto struct {
	URL         string `json:"url"`
	Alt         string `json:"alt"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Copyright   string `json:"copyright"`
	Uploader    string `json:"uploader"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func sanitizePhotos(raw json.RawMessage) json.RawMessage {
	raw = jsonArrayOrEmpty(raw)
	var in []entryPhoto
	if err := json.Unmarshal(raw, &in); err != nil {
		return json.RawMessage(`[]`)
	}
	out := make([]entryPhoto, 0, len(in))
	for _, p := range in {
		if len(out) >= maxEntryPhotos {
			break
		}
		url := strings.TrimSpace(p.URL)
		if !validPhotoURL(url) {
			continue
		}
		w, h := p.Width, p.Height
		if w < 1 || w > 10000 {
			w = defaultPhotoWidth
		}
		if h < 1 || h > 10000 {
			h = defaultPhotoHeight
		}
		out = append(out, entryPhoto{
			URL:         url,
			Alt:         clipRunes(strings.TrimSpace(p.Alt), maxPhotoTextRunes),
			Title:       clipRunes(strings.TrimSpace(p.Title), maxPhotoTextRunes),
			Description: clipRunes(strings.TrimSpace(p.Description), maxPhotoTextRunes),
			Copyright:   clipRunes(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p.Copyright), "©")), maxPhotoTextRunes),
			Uploader:    clipRunes(strings.TrimSpace(p.Uploader), maxPhotoTextRunes),
			Width:       w,
			Height:      h,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

func validPhotoURL(u string) bool {
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.Contains(u, "://") && !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	if strings.HasPrefix(u, entryImageURLPrefix) {
		base := u[len(entryImageURLPrefix):]
		return base != "" && !strings.Contains(base, "/") && !strings.Contains(base, "..")
	}
	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}

func clipRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}
