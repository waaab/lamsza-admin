package search

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const szotarWordCap = 15

type WordSearchHit struct {
	ID           int      `json:"id"`
	Headword     string   `json:"headword"`
	DefinitionHu string   `json:"definition_hu"`
	SpeechTypes  []string `json:"speech_types"`
}

func fetchSzotarWords(ctx context.Context, origin, q string, client *http.Client) []WordSearchHit {
	out := []WordSearchHit{}
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	q = strings.TrimSpace(q)
	if origin == "" || q == "" {
		return out
	}
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	u, err := url.Parse(origin + "/api/words")
	if err != nil {
		log.Printf("UnifiedSearch szotar origin: %v", err)
		return out
	}
	query := u.Query()
	query.Set("q", q)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		log.Printf("UnifiedSearch szotar request: %v", err)
		return out
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("UnifiedSearch szotar fetch: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		log.Printf("UnifiedSearch szotar status: %d", resp.StatusCode)
		return out
	}

	var payload struct {
		Words []struct {
			ID           int      `json:"id"`
			Headword     string   `json:"headword"`
			DefinitionHu string   `json:"definition_hu"`
			SpeechTypes  []string `json:"speech_types"`
		} `json:"words"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		log.Printf("UnifiedSearch szotar decode: %v", err)
		return out
	}
	for _, w := range payload.Words {
		head := strings.TrimSpace(w.Headword)
		if w.ID <= 0 || head == "" {
			continue
		}
		types := w.SpeechTypes
		if types == nil {
			types = []string{}
		}
		out = append(out, WordSearchHit{
			ID:           w.ID,
			Headword:     head,
			DefinitionHu: strings.TrimSpace(w.DefinitionHu),
			SpeechTypes:  types,
		})
		if len(out) >= szotarWordCap {
			break
		}
	}
	return out
}
