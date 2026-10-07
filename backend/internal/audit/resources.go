package audit

// The route registry. One entry per route `main.go` registers, and `Wrap`
// panics at wiring time on a route that has none - so adding an admin route
// without deciding what it audits does not start.
//
// `Tables` is the set of tables whose row carries the state of this resource.
// The snapshot is taken from every table in the list and keyed by table name,
// which is what makes a polymorphic route like `/api/admin/locations` (counties
// *or* settlements, chosen by a `type` field in the body) work without the
// audit layer having to understand the handler.
//
// An empty `Tables` means "no single row holds this" - a composite write (an
// event's whole schedule, an entry's category links), a key/value map
// (site_settings), an upload, or a cache bump. Those records carry the actor,
// the action and the request payload, but no before/after. That limit is
// deliberate and recorded in docs/ARCHITECTURE.md; closing it means teaching
// the handlers to report their own snapshots.

// Resource describes what one admin route writes.
type Resource struct {
	// Name is the stable resource label on the record. It outlives route
	// renames, so queries about "who deleted this entry" keep working.
	Name string

	// Tables are the tables whose row is the resource's state, snapshotted
	// before and after the handler runs. Empty means payload-only.
	Tables []string

	// IDParam is the query parameter holding the resource id. Defaults to
	// "id" when empty.
	IDParam string

	// IDField is the top-level JSON body field holding the resource id, tried
	// when the query parameter is absent. Defaults to "id" when empty.
	IDField string
}

func (r Resource) idParam() string {
	if r.IDParam != "" {
		return r.IDParam
	}
	return "id"
}

func (r Resource) idField() string {
	if r.IDField != "" {
		return r.IDField
	}
	return "id"
}

// resources maps route path -> what it writes. Keep it in the same order as
// main.go so the two are easy to read against each other.
var resources = map[string]Resource{
	// Listing queue - the human review surface over user-submitted entries.
	"/api/admin/listing-queue":            {Name: "listing_queue"},
	"/api/admin/listing-queue/publish":    {Name: "entry", Tables: []string{"entries"}, IDField: "entry_id"},
	"/api/admin/listing-queue/member":     {Name: "entry_member", IDField: "entry_id"},
	"/api/admin/listing-queue/claim":      {Name: "entry_claim", IDField: "entry_id"},
	"/api/admin/listing-queue/suggestion": {Name: "entry_suggestion", Tables: []string{"entry_suggestions"}},

	"/api/admin/websites": {Name: "website", Tables: []string{"websites"}},

	// The directory itself.
	"/api/admin/entries":          {Name: "entry", Tables: []string{"entries"}},
	"/api/admin/entry_categories": {Name: "entry_category", Tables: []string{"entry_categories"}},
	"/api/admin/entry_types":      {Name: "entry_type", Tables: []string{"entry_types"}},
	"/api/admin/tags":             {Name: "tag", Tables: []string{"tags"}},

	// Geography. `locations` writes counties or settlements depending on the
	// body's `type`, so both are snapshotted and the record shows which one
	// actually had a row.
	"/api/admin/locations":                 {Name: "location", Tables: []string{"counties", "settlements"}},
	"/api/admin/settlement_location_types": {Name: "settlement_location_type", Tables: []string{"settlement_location_types"}},
	"/api/admin/county_seat":               {Name: "county_seat", Tables: []string{"settlements"}, IDField: "location_id"},
	"/api/admin/counties":                  {Name: "county", Tables: []string{"counties"}},
	"/api/admin/historical_seats":          {Name: "historical_seat", Tables: []string{"historical_seats"}},
	"/api/admin/attractions":               {Name: "attraction", Tables: []string{"attractions"}},
	"/api/admin/attraction-suggestions":    {Name: "attraction_suggestion", Tables: []string{"attraction_suggestions"}},

	// Read-only routes. Wrapped anyway: the wrapper decides by method, not by
	// route, so a write method added here later is audited without anyone
	// remembering to come back.
	"/api/admin/dashboard_stats": {Name: "dashboard_stats"},
	"/api/admin/users":           {Name: "user", Tables: []string{"users"}},
	"/api/admin/audit-log":       {Name: "audit_log"},

	// Media. The bytes are not the record; who uploaded against which entry is.
	"/api/admin/entry-images": {Name: "entry_image", IDField: "entry_id"},
	"/api/admin/event-images": {Name: "event_image", IDField: "event_id"},

	// Settings are a key/value map, so the payload *is* the change set.
	"/api/admin/settings":                     {Name: "site_setting"},
	"/api/admin/settings/clear-weather-cache": {Name: "weather_cache"},

	"/api/admin/pages":                {Name: "page", Tables: []string{"pages"}},
	"/api/admin/page_faq":             {Name: "page_faq_section", Tables: []string{"page_faq_sections"}},
	"/api/admin/weather_translations": {Name: "weather_translation", Tables: []string{"weather_desc_translations"}},

	// Events.
	"/api/admin/events":                 {Name: "event", Tables: []string{"events"}},
	"/api/admin/events/schedule":        {Name: "event_schedule", IDParam: "event_id", IDField: "event_id"},
	"/api/admin/catalog_event_types":    {Name: "catalog_event_type", Tables: []string{"catalog_event_types"}},
	"/api/admin/catalog_event_subtypes": {Name: "catalog_event_subtype", Tables: []string{"catalog_event_subtypes"}},
	"/api/admin/venues":                 {Name: "venue", Tables: []string{"venues"}},
	"/api/admin/venue_types":            {Name: "venue_type", Tables: []string{"venue_types"}},

	"/api/admin/news_feeds":       {Name: "news_feed", Tables: []string{"news_feeds"}},
	"/api/admin/news_feeds/check": {Name: "news_feed_check"},
	"/api/admin/mondasok":         {Name: "mondas", Tables: []string{"mondasok"}},
	"/api/admin/quick_links":      {Name: "quick_link", Tables: []string{"quick_links"}},
}

// Routes returns every registered route path. The route-coverage test uses it
// to check the registry against main.go in both directions.
func Routes() []string {
	out := make([]string, 0, len(resources))
	for route := range resources {
		out = append(out, route)
	}
	return out
}

// Lookup reports the resource declared for a route.
func Lookup(route string) (Resource, bool) {
	res, ok := resources[route]
	return res, ok
}
