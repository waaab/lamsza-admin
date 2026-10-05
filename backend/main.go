package main

import (
	"log"
	"net/http"

	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/events"
	"backend/internal/handlers"
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
	admin := func(h http.HandlerFunc) http.HandlerFunc {
		return middleware.ApplyCORS(auth.RequireAdmin(h))
	}

	mux.HandleFunc("/api/auth/google", middleware.ApplyCORS(auth.HandleGoogleLogin))
	mux.HandleFunc("/api/auth/me", middleware.ApplyCORS(auth.HandleMe))
	mux.HandleFunc("/api/auth/logout", middleware.ApplyCORS(auth.HandleLogout))

	mux.HandleFunc("/api/admin/listing-queue", admin(account.HandleListingQueue))
	mux.HandleFunc("/api/admin/listing-queue/publish", admin(account.HandleListingQueuePublish))
	mux.HandleFunc("/api/admin/listing-queue/member", admin(account.HandleListingQueueMember))
	mux.HandleFunc("/api/admin/listing-queue/claim", admin(account.HandleListingQueueClaim))
	mux.HandleFunc("/api/admin/listing-queue/suggestion", admin(account.HandleListingQueueSuggestion))
	mux.HandleFunc("/api/admin/websites", admin(account.HandleAdminWebsite))
	mux.HandleFunc("/api/admin/entries", admin(handlers.HandleAdminEntries))
	mux.HandleFunc("/api/admin/entry_categories", admin(handlers.HandleAdminEntryCategories))
	mux.HandleFunc("/api/admin/entry_types", admin(handlers.HandleAdminEntryTypes))
	mux.HandleFunc("/api/admin/tags", admin(handlers.HandleAdminTags))
	mux.HandleFunc("/api/admin/locations", admin(handlers.HandleAdminLocations))
	mux.HandleFunc("/api/admin/settlement_location_types", admin(handlers.HandleAdminSettlementLocationTypes))
	mux.HandleFunc("/api/admin/county_seat", admin(handlers.HandleSetCountySeat))
	mux.HandleFunc("/api/admin/dashboard_stats", admin(handlers.HandleAdminDashboardStats))
	mux.HandleFunc("/api/admin/users", admin(auth.HandleAdminUsers))
	mux.HandleFunc("/api/admin/entry-images", admin(handlers.HandleEntryImageUpload))
	mux.Handle("/api/media/entry-images/", middleware.ApplyCORS(http.StripPrefix("/api/media/entry-images/", http.FileServer(http.Dir(handlers.EntryImagesDir()))).ServeHTTP))
	mux.Handle("/api/media/event-images/", middleware.ApplyCORS(http.StripPrefix("/api/media/event-images/", http.FileServer(http.Dir(handlers.EventImagesDir()))).ServeHTTP))
	mux.HandleFunc("/api/admin/attraction-suggestions", admin(handlers.HandleAdminAttractionSuggestions))
	mux.HandleFunc("/api/admin/counties", admin(handlers.HandleAdminCounties))
	mux.HandleFunc("/api/admin/historical_seats", admin(handlers.HandleAdminHistoricalSeats))
	mux.HandleFunc("/api/admin/attractions", admin(handlers.HandleAdminAttractions))

	mux.HandleFunc("/api/config/public", middleware.ApplyCORS(settings.HandlePublicConfig))
	mux.HandleFunc("/api/admin/settings", admin(settings.HandleAdminSettings))
	mux.HandleFunc("/api/admin/settings/clear-weather-cache", admin(settings.ClearWeatherCache))

	mux.HandleFunc("/api/admin/pages", admin(pages.HandleAdminPages))
	mux.HandleFunc("/api/admin/page_faq", admin(pagefaq.HandleAdmin))

	if config.AppConfig.Features.Weather {
		mux.HandleFunc("/api/admin/weather_translations", admin(weather.HandleAdminWeatherTranslations))
		log.Println("Module [Weather] enabled")
	}

	if config.AppConfig.Features.Events {
		mux.HandleFunc("/api/admin/events", admin(events.HandleAdminEvents))
		mux.HandleFunc("/api/admin/events/schedule", admin(events.HandleAdminEventSchedule))
		mux.HandleFunc("/api/admin/catalog_event_types", admin(events.HandleAdminCatalogEventTypes))
		mux.HandleFunc("/api/admin/catalog_event_subtypes", admin(events.HandleAdminCatalogEventSubtypes))
		mux.HandleFunc("/api/admin/event-images", admin(handlers.HandleEventImageUpload))
		mux.HandleFunc("/api/admin/venues", admin(venues.HandleAdmin))
		mux.HandleFunc("/api/admin/venue_types", admin(venues.HandleAdminVenueTypes))
		log.Println("Module [Events] enabled")
	}

	if config.AppConfig.Features.News {
		mux.HandleFunc("/api/admin/news_feeds", admin(news.HandleAdminNewsFeeds))
		log.Println("Module [News] enabled")
	}

	if config.AppConfig.Features.Mondasok {
		mux.HandleFunc("/api/admin/mondasok", admin(mondasok.HandleAdminMondasok))
		log.Println("Module [Mondasok] enabled")
	}

	if config.AppConfig.Features.QuickLinks {
		mux.HandleFunc("/api/admin/quick_links", admin(links.HandleAdminQuickLinks))
		log.Println("Module [QuickLinks] enabled")
	}

	port := config.AppConfig.Port
	log.Printf("Lamsza Admin API active on port %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
