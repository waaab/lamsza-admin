package account

import (
	"encoding/json"
	"strings"
)

type membershipResponse struct {
	EntryID int    `json:"entry_id"`
	Role    string `json:"role"`
	Status  string `json:"status"`
}

// jsonInt accepts a JSON number or a numeric string. Public entries expose id as a string.
type jsonInt int

type claimBody struct {
	EntryID jsonInt `json:"entry_id"`
}

type createListingBody struct {
	WebsiteID       int             `json:"website_id"`
	Name            string          `json:"name"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	Photos          json.RawMessage `json:"photos"`
	Tags            []string        `json:"tags"`
}

type updateListingBody struct {
	Name            string          `json:"name"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	SocialLinks     json.RawMessage `json:"social_links"`
	Photos          json.RawMessage `json:"photos"`
	RatingsEnabled  bool            `json:"ratings_enabled"`
	Tags            []string        `json:"tags"`
}

type listingItem struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Published  bool   `json:"published"`
	Verified   bool   `json:"verified"`
	LocationID int    `json:"location_id"`
	CategoryID int    `json:"category_id"`
	TypeID     int    `json:"type_id"`
}

type listingsResponse struct {
	Owned       []listingItem `json:"owned"`
	Member      []listingItem `json:"member"`
	Pending     []listingItem `json:"pending"`
	Unpublished []listingItem `json:"unpublished"`
}

const maxListingPhotos = 24
const maxListingPhotoTextRunes = 500
const listingImageURLPrefix = "/api/media/entry-images/"
const defaultListingPhotoWidth = 1600
const defaultListingPhotoHeight = 1200

type listingPhoto struct {
	URL         string `json:"url"`
	Alt         string `json:"alt"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Copyright   string `json:"copyright"`
	Uploader    string `json:"uploader"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type listingSocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func sanitizeSocialLinks(raw json.RawMessage) json.RawMessage {
	raw = photosArrayOrEmpty(raw)
	var in []listingSocialLink
	if err := json.Unmarshal(raw, &in); err != nil {
		return json.RawMessage(`[]`)
	}
	out := make([]listingSocialLink, 0, len(in))
	for _, link := range in {
		url := strings.TrimSpace(link.URL)
		lower := strings.ToLower(url)
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			continue
		}
		out = append(out, listingSocialLink{
			Label: strings.TrimSpace(link.Label),
			URL:   url,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

func photosArrayOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`[]`)
	}
	if !json.Valid(b) || !strings.HasPrefix(s, "[") {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}

type catalogItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id,omitempty"`
}

type listingCatalogResponse struct {
	Categories []catalogItem `json:"categories"`
	Types      []catalogItem `json:"types"`
}

type listingDetailResponse struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	LocationID      int             `json:"location_id"`
	CategoryID      int             `json:"category_id"`
	CategoryIDs     []int           `json:"category_ids"`
	TypeID          int             `json:"type_id"`
	URL             string          `json:"url"`
	Phone           string          `json:"phone"`
	Address         string          `json:"address"`
	Notes           string          `json:"notes"`
	Languages       []string        `json:"languages"`
	Hours           json.RawMessage `json:"hours"`
	HoursEnabled    bool            `json:"hours_enabled"`
	DeliveryHours   json.RawMessage `json:"delivery_hours"`
	DeliveryEnabled bool            `json:"delivery_enabled"`
	SocialLinks     json.RawMessage `json:"social_links"`
	Photos          json.RawMessage `json:"photos"`
	Published       bool            `json:"published"`
	RatingsEnabled  bool            `json:"ratings_enabled"`
	Tags            []string        `json:"tags"`
}

type listingMemberRow struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}
