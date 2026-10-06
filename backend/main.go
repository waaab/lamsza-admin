package main

import (
	"log"
	"net/http"

	"backend/internal/account"
	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/events"
	"backend/internal/handlers"
	"backend/internal/health"
	"backend/internal/links"
	"backend/internal/middleware"
	"backend/internal/mondasok"
	"backend/internal/news"
	"backend/internal/pagefaq"
	"backend/internal/pages"
	"backend/internal/settings"
	"backend/internal/venues"
	"backend/internal/weather"
)

// Admin API only. Schema migrations are owned by the main lamsza backend.
func main() {
	config.Load()
	db.InitDB()

	mux := http.DefaultServeMux

	// Every admin route is registered through this one function, and nothing
	// else may register an /api/admin/ path (BOG-48). It is the single place
	// the audit trail is applied, so a route added later is audited whether or
	// not its author thought about it. `audit.Wrap` refuses a route that has no
	// declared resource in internal/audit/resources.go, and
	// `audit_routes_test.go` re-derives both rules from this source on every
	// `go test` run.
	//
	// Order matters: the audit layer sits inside RequireAdmin so the actor is
	// already on the context, and inside ApplyCORS so a refused cross-origin
	// call never reaches a handler or the log.
	admin := func(route string, h http.HandlerFunc) {
		mux.HandleFunc(route, middleware.ApplyCORS(auth.RequireAdmin(audit.Wrap(route, h))))
	}

	mux.HandleFunc("/api/health", middleware.ApplyCORS(health.HandleHealth))

	mux.HandleFunc("/api/auth/google", middleware.ApplyCORS(auth.HandleGoogleLogin))
	mux.HandleFunc("/api/auth/me", middleware.ApplyCORS(auth.HandleMe))
	mux.HandleFunc("/api/auth/logout", middleware.ApplyCORS(auth.HandleLogout))

	admin("/api/admin/listing-queue", account.HandleListingQueue)
	admin("/api/admin/listing-queue/publish", account.HandleListingQueuePublish)
	admin("/api/admin/listing-queue/member", account.HandleListingQueueMember)
	admin("/api/admin/listing-queue/claim", account.HandleListingQueueClaim)
	admin("/api/admin/listing-queue/suggestion", account.HandleListingQueueSuggestion)
	admin("/api/admin/websites", account.HandleAdminWebsite)
	admin("/api/admin/entries", handlers.HandleAdminEntries)
	admin("/api/admin/entry_categories", handlers.HandleAdminEntryCategories)
	admin("/api/admin/entry_types", handlers.HandleAdminEntryTypes)
	admin("/api/admin/tags", handlers.HandleAdminTags)
	admin("/api/admin/locations", handlers.HandleAdminLocations)
	admin("/api/admin/settlement_location_types", handlers.HandleAdminSettlementLocationTypes)
	admin("/api/admin/county_seat", handlers.HandleSetCountySeat)
	admin("/api/admin/dashboard_stats", handlers.HandleAdminDashboardStats)
	admin("/api/admin/users", auth.HandleAdminUsers)
	admin("/api/admin/audit-log", audit.HandleAdminAuditLog)
	admin("/api/admin/entry-images", handlers.HandleEntryImageUpload)
	mux.Handle("/api/media/entry-images/", middleware.ApplyCORS(http.StripPrefix("/api/media/entry-images/", http.FileServer(http.Dir(handlers.EntryImagesDir()))).ServeHTTP))
	mux.Handle("/api/media/event-images/", middleware.ApplyCORS(http.StripPrefix("/api/media/event-images/", http.FileServer(http.Dir(handlers.EventImagesDir()))).ServeHTTP))
	admin("/api/admin/attraction-suggestions", handlers.HandleAdminAttractionSuggestions)
	admin("/api/admin/counties", handlers.HandleAdminCounties)
	admin("/api/admin/historical_seats", handlers.HandleAdminHistoricalSeats)
	admin("/api/admin/attractions", handlers.HandleAdminAttractions)

	// Public reads used by the admin SPA (same paths as main Lámsza)
	mux.HandleFunc("/api/counties", middleware.ApplyCORS(handlers.HandleCounties))
	mux.HandleFunc("/api/historical_seats", middleware.ApplyCORS(handlers.HandleHistoricalSeats))

	mux.HandleFunc("/api/config/public", middleware.ApplyCORS(settings.HandlePublicConfig))
	admin("/api/admin/settings", settings.HandleAdminSettings)
	admin("/api/admin/settings/clear-weather-cache", settings.ClearWeatherCache)

	admin("/api/admin/pages", pages.HandleAdminPages)
	admin("/api/admin/page_faq", pagefaq.HandleAdmin)

	if config.AppConfig.Features.Weather {
		admin("/api/admin/weather_translations", weather.HandleAdminWeatherTranslations)
		log.Println("Module [Weather] enabled")
	}

	if config.AppConfig.Features.Events {
		mux.HandleFunc("/api/venues", middleware.ApplyCORS(venues.HandlePublic))
		mux.HandleFunc("/api/venue_types", middleware.ApplyCORS(venues.HandlePublicVenueTypes))
		admin("/api/admin/events", events.HandleAdminEvents)
		admin("/api/admin/events/schedule", events.HandleAdminEventSchedule)
		admin("/api/admin/catalog_event_types", events.HandleAdminCatalogEventTypes)
		admin("/api/admin/catalog_event_subtypes", events.HandleAdminCatalogEventSubtypes)
		admin("/api/admin/event-images", handlers.HandleEventImageUpload)
		admin("/api/admin/venues", venues.HandleAdmin)
		admin("/api/admin/venue_types", venues.HandleAdminVenueTypes)
		log.Println("Module [Events] enabled")
	}

	if config.AppConfig.Features.News {
		admin("/api/admin/news_feeds", news.HandleAdminNewsFeeds)
		log.Println("Module [News] enabled")
	}

	if config.AppConfig.Features.Mondasok {
		admin("/api/admin/mondasok", mondasok.HandleAdminMondasok)
		log.Println("Module [Mondasok] enabled")
	}

	if config.AppConfig.Features.QuickLinks {
		admin("/api/admin/quick_links", links.HandleAdminQuickLinks)
		log.Println("Module [QuickLinks] enabled")
	}

	port := config.AppConfig.Port
	log.Printf("Lamsza Admin API active on port %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
