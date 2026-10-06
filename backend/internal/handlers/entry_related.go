package handlers

import (
	"encoding/json"
)

const relatedEntryLimit = 8

type relatedEntry struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Category     string          `json:"category"`
	Location     string          `json:"location"`
	LocationSlug string          `json:"location_slug"`
	CountySlug   string          `json:"county_slug"`
	Photos       json.RawMessage `json:"photos"`
	Claimed      bool            `json:"claimed"`
	Verified     bool            `json:"-"`
}
