package events

// eventSubtypeOption is a distinct (type slug, subtype slug) among upcoming events.
type eventSubtypeOption struct {
	TypeSlug string `json:"type_slug"`
	Slug     string `json:"slug"`
	LabelHu  string `json:"label_hu"`
}

// filterOptionsResponse drives the /esemenyek filters (types, locations, month list).
type filterOptionsResponse struct {
	EventTypes    []string             `json:"event_types"`
	EventSubtypes []eventSubtypeOption `json:"event_subtypes"`
	Locations     []locationOption     `json:"locations"`
	// Months are YYYY-MM values that have at least one upcoming event (by start_date).
	Months []string `json:"months"`
	// EventDays are YYYY-MM-DD start dates of upcoming events (distinct calendar days).
	EventDays []string `json:"event_days"`
	// ScheduleEventIDs lists upcoming events that have at least one napi program day (for list UI fallback).
	ScheduleEventIDs []int `json:"schedule_event_ids"`
}

type locationOption struct {
	Name         string `json:"name"`
	LocationSlug string `json:"location_slug"`
	CountySlug   string `json:"county_slug"`
	CountyName   string `json:"county_name"`
}
