<script>
    import { onMount } from "svelte";
    import AdminNavIcon from "$lib/components/admin/AdminNavIcon.svelte";
    import AppIcon from "$lib/icons/AppIcon.svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import { ADMIN_PAGE_SIZE, adminPageSlice } from "$lib/adminPageSlice.js";
    import { auth } from "$lib/stores/auth";
    import {
        SCHEDULE_ACTIVITY_TYPES,
        SCHEDULE_ACTIVITY_TYPE_LABELS,
    } from "$lib/scheduleActivityTypes.js";
    import { absoluteMediaUrl } from "$lib/eventImage.js";
    import { getApiBase, apiCall } from "$lib/api.js";
    import FeaturedCategoriesPanel from "$lib/components/FeaturedCategoriesPanel.svelte";
    import { optionalNumber } from "$lib/optionalNumber.js";
    import { emptyWeekHours, normalizeHours, withDefaultWeekHours } from "$lib/entryHours.js";
    import { offersDelivery } from "$lib/entryPublicExtras.js";
    import { emptyPhotos, normalizePhotos } from "$lib/entryPhotos.js";
    import CategoryMultiSelect from "$lib/components/CategoryMultiSelect.svelte";
    import EntryHoursEditor from "$lib/components/EntryHoursEditor.svelte";
    import HuDateInput from "$lib/components/HuDateInput.svelte";
    import HuTimeInput from "$lib/components/HuTimeInput.svelte";
    import EntryPhotosEditor from "$lib/components/EntryPhotosEditor.svelte";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import NoticeDialog from "$lib/components/NoticeDialog.svelte";
    import AdminShell from "$lib/components/admin/AdminShell.svelte";
    import AdminWelcomeGrid from "$lib/components/admin/AdminWelcomeGrid.svelte";
    import { APP_ORIGINS } from "$lib/adminApps.js";
    import { canonicalDomain } from "$lib/websiteDomain.js";
    import { SETTINGS_SECTIONS, settingsPayload } from "$lib/settingsSections.js";

    const lamszaOrigin = APP_ORIGINS.lamsza;

    /** @type {AdminShell | null} sign-in gate, sidebar, header (lib/components/admin/AdminShell.svelte) */
    let shell = null;
    let adminOffline = false;
    let activeTab = "welcome";

    /** Sidebar, in order; the dashboard cards are the same entries below Vezérlőpult. */
    const ADMIN_NAV = [
        { id: "welcome", title: "Vezérlőpult", icon: "dashboard" },
        { id: "quicklinks", title: "Gyorslinkek" },
        { sep: true },
        { id: "websites", title: "Weboldalak" },
        { id: "entries", title: "Index" },
        { id: "entry_categories", title: "Bejegyzés Kategóriák" },
        { id: "entry_types", title: "Bejegyzés típusok" },
        { id: "tags", title: "Címkék" },
        { sep: true },
        { id: "locations", title: "Települések" },
        { id: "counties", title: "Megyék" },
        { id: "venues", title: "Helyszínek" },
        { id: "attractions", title: "Látnivalók" },
        { sep: true },
        { id: "events", title: "Események" },
        { sep: true },
        { id: "pages", title: "Oldalak" },
        { id: "page_faq", title: "GYIK" },
        { id: "weather_translations", title: "Időjárás fordítások" },
        { sep: true },
        { id: "newsfeeds", title: "Hírfolyamok" },
        { sep: true },
        { id: "users", title: "Felhasználók" },
        { id: "settings", title: "Beállítások" },
    ];
    const ADMIN_WELCOME_ITEMS = /** @type {{ id: string, title: string }[]} */ (
        ADMIN_NAV.filter((n) => n.id && n.id !== "welcome")
    );

    /** Opens a tab through the shell, so the address bar's #tab follows. */
    function goToAdminTab(/** @type {string} */ tab) {
        if (shell) shell.select(tab);
        else loadTab(tab);
    }

    /** The shell's onSelect: show the tab and load what it needs. */
    function loadTab(/** @type {string} */ tab) {
        activeTab = tab;
        if (tab === "welcome") {
            fetchDashboardStats();
            fetchSettings();
        }
        if (tab === "counties") fetchCountyRegions();
        if (tab === "venues") {
            fetchVenuesCatalog();
            fetchVenueTypes();
        }
        if (tab === "locations") fetchSettlementLocationTypes();
        if (tab === "attractions") fetchAttractions();
        if (tab === "events") {
            fetchEvents();
            fetchAttractions();
        }
        if (tab === "settings") fetchSettings();
        if (tab === "weather_translations") fetchWeatherTranslations();
        if (tab === "pages") fetchPages();
        if (tab === "page_faq") fetchPageFaq();
        if (tab === "websites") fetchAdminWebsites();
        if (tab === "users") fetchUsers();
    }

    /** @type {{ tab: string, message: string } | null} */
    let adminTabError = null;
    $: if (adminTabError && activeTab !== adminTabError.tab) {
        adminTabError = null;
    }

    function setAdminTabError(msg) {
        adminTabError = { tab: activeTab, message: msg };
    }

    function clearAdminTabError() {
        adminTabError = null;
    }

    /** Result banner above one admin table. Cleared when the tab changes. */
    /** @type {{ tab: string, table: string, ok: boolean, text: string } | null} */
    let adminActionNotice = null;
    $: if (adminActionNotice && activeTab !== adminActionNotice.tab) {
        adminActionNotice = null;
    }

    function noticeTableFor(endpoint) {
        if (endpoint === "quick_links") return "quicklinks";
        if (endpoint === "news_feeds") return "newsfeeds";
        return endpoint;
    }

    function actionSubject(data) {
        if (!data || typeof data !== "object") return "";
        const raw =
            data.name ??
            data.title ??
            data.label_hu ??
            data.text ??
            data.source_text ??
            "";
        const s = String(raw).replace(/\s+/g, " ").trim();
        if (!s) return "";
        return s.length > 60 ? `${s.slice(0, 57)}…` : s;
    }

    function noteAdminAction(table, text, ok = true) {
        const msg =
            String(text || "").trim() ||
            (ok ? "A művelet sikerült." : "A művelet nem sikerült.");
        adminActionNotice = { tab: activeTab, table, ok, text: msg };
        if (ok) clearAdminTabError();
    }

    async function noteAdminFailure(table, resOrText, prefix = "Hiba: ") {
        let detail = "";
        if (resOrText && typeof resOrText.text === "function") {
            detail = (await resOrText.text()).trim();
        } else {
            detail = String(resOrText ?? "").trim();
        }
        const msg = detail ? `${prefix}${detail}` : "A művelet nem sikerült.";
        noteAdminAction(table, msg, false);
        setAdminTabError(msg);
    }

    let quickLinks = [];
    let newsFeeds = [];
    let locations = [];
    let attractions = [];
    let countiesFromAPI = [];
    let historicalSeatsFromAPI = [];
    /** Inline table edit state for Megyék / Történelmi székek tabs */
    let editingCounty = null;
    let editingHistoricalSeat = null;
    let entries = [];
    /** @type {Array<{ id: number, name: string, parent_id?: number | null }>} */
    let entryCategories = [];
    let entryTypes = [];
    let tags = [];
    let events = [];

    // Per-feed loading state
    let loadingFeeds = new Set();
    let feedTimestamps = {};

    // Form binding objects
    let newLink = { title: "", url: "", bg_color: "#e6f0ff" };
    let newNews = { title: "", feed_url: "", bg_color: "#ffebd6" };
    let newLocation = {
        name: "",
        name_ro: "",
        name_de: "",
        county: "",
        type: "",
        slug: "",
        post_code: "",
        coordinates: "",
        population: "",
        area: "",
        parent_id: null,
    };
    let newEntry = {
        location_id: "",
        category_id: "",
        category_extra: [],
        name: "",
        slug: "",
        url: "",
        phone: "",
        address: "",
        notes: "",
        type: "",
        languages: ["HU"],
        tags: "",
        verified: false,
        hours: emptyWeekHours(),
        delivery_hours: emptyWeekHours(),
        hours_enabled: false,
        delivery_enabled: false,
        photos: emptyPhotos(),
    };
    let newEntryCategory = { name: "", parent_id: "" };
    /** @type {{ id: number, message: string, moveTo: string } | null} */
    let categoryDeleteMove = null;
    /** @type {Record<number, string>} */
    let websiteApproveCategory = {};
    /** @type {Record<number, number[]>} */
    let websiteApproveExtra = {};
    let newEntryType = { name: "" };
    let newTag = { name: "" };
    let newEvent = {
        location_id: "",
        default_venue_id: "",
        title: "",
        description: "",
        start_date: "",
        start_time: "",
        end_date: "",
        end_time: "",
        event_type_id: "",
        event_subtype_id: "",
        access_type: "public",
        organizer: "",
        featured_image: "",
        featured_image_copyright: "",
        entry_price: "",
        attraction_id: "",
    };
    /** @type {{ id: number, slug: string, label_hu: string, sort_order: number }[]} */
    let catalogEventTypes = [];
    /** @type {{ id: number, event_type_id: number, slug: string, label_hu: string, sort_order: number }[]} */
    let catalogEventSubtypes = [];
    let newCatalogEventType = { slug: "", label_hu: "", sort_order: 0 };
    /** @type {{ id: number, slug: string, label_hu: string, sort_order: number } | null} */
    let editingCatalogEventType = null;
    let newCatalogEventSubtype = {
        event_type_id: "",
        slug: "",
        label_hu: "",
        sort_order: 0,
    };
    /** @type {{ id: number, event_type_id: number, slug: string, label_hu: string, sort_order: number } | null} */
    let editingCatalogEventSubtype = null;
    let searchCatalogTypes = "";
    let searchCatalogSubtypes = "";
    /** @type {Record<string, unknown>[]} */
    let venuesCatalog = [];
    /** @type {typeof venuesCatalog} */
    let venueOptionsNew = [];
    /** @type {typeof venuesCatalog} */
    let venueOptionsEdit = [];
    let newVenue = {
        settlement_id: "",
        name: "",
        name_ro: "",
        name_de: "",
        slug: "",
        kind: "sports_arena",
        address: "",
        latitude: "",
        longitude: "",
        seating_capacity: "",
        description: "",
        notes: "",
    };
    /** @type {{ id: number, slug: string, label_hu: string }[]} */
    let venueTypesList = [];
    let newVenueType = { label_hu: "" };
    /** @type {Record<string, unknown> | null} */
    let editingVenueType = null;

    let searchQuickLinks = "";
    let searchNewsFeeds = "";
    let searchLocations = "";
    let searchVenueTypes = "";
    let searchVenues = "";
    let searchEvents = "";
    let searchEntryCategories = "";
    let searchEntries = "";
    let searchAdminWebsites = "";
    let searchEntryTypes = "";
    let searchTags = "";
    let searchAttractions = "";
    let searchCounties = "";
    let searchHistoricalSeats = "";
    let searchWeatherTrans = "";
    let searchAdminPages = "";
    let searchPageFaqRows = "";
    let searchUsers = "";
    let pageQuickLinks = 1;
    let pageNewsFeeds = 1;
    let pageLocations = 1;
    let pageVenueTypes = 1;
    let pageVenues = 1;
    let pageEvents = 1;
    let pageEntryCategories = 1;
    let pageEntries = 1;
    let pageAdminWebsites = 1;
    /** @type {{ id: number, domain: string, title: string, description: string, status: string, submitter: string, claimed: boolean, url: string }[]} */
    let adminWebsites = [];
    let pageWeatherTrans = 1;
    let pageAdminPages = 1;
    let pagePageFaqRows = 1;
    let pageUsers = 1;
    /** @type {Array<{ id: number, email: string, name: string, given_name: string, family_name: string, display_name: string, locale: string, settlement: string, created_at: string, last_login_at: string, website_banned: boolean, is_admin: boolean }>} */
    let adminUsers = [];
    let pageEntryTypes = 1;
    let pageTags = 1;
    let pageAttractions = 1;
    let pageCounties = 1;
    let pageHistoricalSeats = 1;
    let pageCatalogTypes = 1;
    let pageCatalogSubtypes = 1;
    /** Counties / historical seats: keep inline edit row visible when search would hide it */
    $: displayCounties = (countiesFromAPI || []).filter(
        (c) =>
            editingCounty?.id === c.id ||
            countyMatchesSearch(c, searchCounties),
    );
    $: displayHistoricalSeats = (historicalSeatsFromAPI || []).filter(
        (h) =>
            editingHistoricalSeat?.id === h.id ||
            historicalSeatMatchesSearch(h, searchHistoricalSeats),
    );
    let newAttraction = {
        county_slug: "hargita",
        name: "",
        name_ro: "",
        name_de: "",
        slug: "",
        description: "",
        latitude: "",
        longitude: "",
        featured_image: "",
        featured_image_copyright: "",
        content: "",
        activities: "",
        prohibitions: "",
        elevation_m: "",
        area_km2: "",
        depth_m: "",
        images: "",
        image_copyrights: "",
    };

    let newOrganizerModalVisible = false;
    let newOrganizerEntry = {
        location_id: "",
        category_id: "",
        category_extra: [],
        name: "",
        slug: "",
        url: "",
        phone: "",
        address: "",
        notes: "",
        type: "",
        languages: ["HU"],
        tags: "",
    };

    // Edit modal state
    let editingEntry = null;
    let editTagsStr = "";
    let editingLocation = null;
    let editingCategory = null;
    let editingType = null;
    let editingTag = null;
    let editingLink = null;
    let editingNews = null;
    let editingEvent = null;
    /** @type {Array<{ schedule_date: string, notes: string, activities: Array<{ activity_type: string, starts_at: string, ends_at: string, title: string, description: string }> }>} */
    let scheduleDraftDays = [];
    let editingAttraction = null;
    /** @type {Record<string, unknown> | null} */
    let editingVenue = null;

    let orgQuery = "";
    let orgEditQuery = "";
    let orgSuggestions = [];
    let orgEditSuggestions = [];
    let orgDropdownOpen = false;
    let orgEditDropdownOpen = false;

    // Site settings (weather providers, cache)
    let siteSettings = {};
    /** The Beállítások section being saved ("social" | "location" | "weather"), or "". */
    let settingsSaving = "";
    let settingsCacheClearing = false;

    // Weather description translations (multi-language)
    let weatherTranslations = [];
    let newWeatherTrans = { source_text: "", lang: "hu", translated_text: "" };
    let editingWeatherTrans = null;
    const WEATHER_TRANS_LANGS = [
        { value: "hu", label: "Magyar" },
        { value: "ro", label: "Română" },
        { value: "de", label: "Deutsch" },
    ];

    // Pages (policy pages editor)
    let adminPages = [];
    let pageFaqSections = [];
    let editingPage = null;
    let pageSaving = false;
    let editingPageFaq = null;
    let pageFaqSaving = false;

    /** Row counts from DB (GET /api/admin/dashboard_stats); keys match ADMIN_WELCOME_ITEMS id. */
    let dashboardStats = /** @type {Record<string, number>} */ ({});
    let dashboardStatsFetched = false;
    let dashboardStatsError = "";
    let dashboardStatsRequest = 0;
    let settingsLoaded = false;
    let settingsLoadError = "";
    /** @type {{ name: string, detail: string }[]} */
    let browserCacheRows = [];
    /** @type {{ id: string, level: string, text: string, tab?: string, action?: string }[]} */
    let browserCacheNotices = [];
    /** @type {{ source: string, text: string }[]} */
    let apiNotices = [];

    const ADMIN_API_LABELS = {
        quick_links: "Gyorslinkek",
        news_feeds: "Hírfolyamok",
        locations: "Települések",
        attractions: "Látnivalók",
        websites: "Weboldalak",
        entries: "Index",
        entry_categories: "Bejegyzés kategóriák",
        entry_types: "Bejegyzés típusok",
        tags: "Címkék",
        events: "Események",
        catalog_event_types: "Eseménytípusok",
        catalog_event_subtypes: "Esemény-altípusok",
        pages: "Oldalak",
        page_faq: "GYIK",
        venues: "Helyszínek",
        venue_types: "Helyszíntípusok",
        settlement_location_types: "Településtípusok",
        counties: "Megyék",
        historical_seats: "Történelmi székek",
        weather_translations: "Időjárás fordítások",
        users: "Felhasználók",
    };

    function rememberApiError(source, message) {
        const text = String(message || "Az API nem válaszolt.");
        apiNotices = [
            ...apiNotices.filter((n) => n.source !== source),
            { source, text },
        ];
    }

    function forgetApiError(source) {
        if (!apiNotices.some((n) => n.source === source)) return;
        apiNotices = apiNotices.filter((n) => n.source !== source);
    }

    /** @type {{ id: number, name: string, slug: string, owner_email: string }[]} */
    let listingQueueUnpublished = [];
    /** @type {{ entry_id: number, entry_name: string, user_id: number, email: string }[]} */
    let listingQueueMembers = [];
    /** @type {{ entry_id: number, entry_name: string, user_id: number, email: string }[]} */
    let listingQueueClaims = [];
    /** @type {{ id: number, entry_id: number, entry_name: string, email: string, changes: Record<string, unknown>, note: string }[]} */
    let listingQueueSuggestions = [];
    /** @type {{ id: number, domain: string, title: string, description: string, submitter: string }[]} */
    let listingQueueWebsites = [];
    let listingQueueError = "";
    let listingQueueFetched = false;
    /** @type {{ id: number, attraction_id: number, attraction_name: string, user_name: string, changes: Record<string, unknown>, note: string, created_at: string }[]} */
    let attractionSuggestions = [];

    function describeApiFailure(subject, status, body) {
        const raw = String(body || "").trim();
        if (/data api stopped/i.test(raw)) {
            return `${subject} nem tölthető be: a tartalom API le van állítva, csak a Google-belépés fut.`;
        }
        if (status === 401 || status === 403) {
            return `${subject} nem tölthető be: a szerver elutasította a kérést (HTTP ${status}). A munkamenet lejárt, vagy a fiók nem admin.`;
        }
        if (status === 404) {
            return `${subject} nem tölthető be: ez az útvonal nincs bekötve a szerveren (HTTP 404).`;
        }
        if (status >= 500) {
            return `${subject} nem tölthető be: a szerver hibával válaszolt (HTTP ${status}).`;
        }
        if (status) {
            return `${subject} nem tölthető be: a kérés nem sikerült (HTTP ${status}).`;
        }
        return `${subject} nem tölthető be: a kérés nem ért el a szerverig.`;
    }

    function describeTransportError(subject, error) {
        const msg = String(error?.message || error || "");
        if (/failed to fetch|networkerror|load failed|network request failed/i.test(msg)) {
            return `${subject} nem tölthető be: a kérés nem ért el a szerverig. Az API nem fut, vagy a böngésző nem éri el.`;
        }
        if (msg.startsWith(subject)) return msg;
        return msg || `${subject} nem tölthető be.`;
    }

    async function fetchDashboardStats() {
        const requestId = ++dashboardStatsRequest;
        if (!dashboardStatsFetched) dashboardStatsError = "";
        let lastError = "";
        for (let attempt = 0; attempt < 2; attempt++) {
            if (requestId !== dashboardStatsRequest) return;
            try {
                const res = await apiCall("/api/admin/dashboard_stats");
                if (requestId !== dashboardStatsRequest) return;
                if (!res.ok) {
                    throw new Error(describeApiFailure("A táblaszámlálók", res.status, await res.text()));
                }
                const raw = await res.json();
                if (requestId !== dashboardStatsRequest) return;
                if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
                    throw new Error("Az API válasza nem tartalmazza a táblaszámlálókat.");
                }
                const next = /** @type {Record<string, number>} */ ({});
                for (const item of ADMIN_WELCOME_ITEMS) {
                    const n = Number(raw[item.id]);
                    if (!Number.isFinite(n)) {
                        throw new Error("Az API válaszából hiányoznak a táblaszámlálók.");
                    }
                    next[item.id] = n;
                }
                dashboardStats = next;
                dashboardStatsFetched = true;
                dashboardStatsError = "";
                return;
            } catch (e) {
                lastError = describeTransportError("A táblaszámlálók", e);
                if (attempt === 0) {
                    await new Promise((resolve) => setTimeout(resolve, 400));
                }
            }
        }
        if (requestId !== dashboardStatsRequest) return;
        dashboardStatsError = lastError;
        console.error(lastError);
    }

    function collectBrowserCaches() {
        /** @type {{ name: string, detail: string }[]} */
        const rows = [];
        const weatherVersion =
            siteSettings.weather_cache_version != null
                ? String(siteSettings.weather_cache_version)
                : "";
        const linksVersion =
            siteSettings.quick_links_version != null
                ? String(siteSettings.quick_links_version)
                : "";
        const ttlMin = Number(siteSettings.weather_cache_ttl_minutes);
        const weatherTtlMs = Number.isFinite(ttlMin) && ttlMin > 0 ? ttlMin * 60 * 1000 : 15 * 60 * 1000;
        const hourMs = 60 * 60 * 1000;
        const newsTtlMs = 30 * 60 * 1000;
        let weatherTotal = 0;
        let weatherFresh = 0;
        let weatherStale = 0;
        let newsTotal = 0;
        let newsFresh = 0;
        let newsStale = 0;
        let promotedDetail = "nincs mentett gyorslink-válasz ebben a böngészőben";
        let promotedState = "nincs";

        const ageState = (timestamp, ttlMs, version, expectedVersion) => {
            if (!Number.isFinite(timestamp)) return "nincs időbélyeg";
            const expired = Date.now() - timestamp >= ttlMs;
            const versionOk = !expectedVersion || String(version ?? "") === expectedVersion;
            if (!versionOk) return "régi szerververzió";
            if (expired) return "lejárt";
            return "érvényes";
        };

        try {
            for (let i = 0; i < localStorage.length; i++) {
                const key = localStorage.key(i);
                if (!key) continue;
                let parsed = null;
                try {
                    parsed = JSON.parse(localStorage.getItem(key) || "");
                } catch {
                    parsed = null;
                }
                if (key.startsWith("weather_cache_")) {
                    weatherTotal += 1;
                    const state = parsed
                        ? ageState(parsed.timestamp, weatherTtlMs, parsed.cache_version, weatherVersion)
                        : "sérült";
                    if (state === "érvényes") weatherFresh += 1;
                    else weatherStale += 1;
                } else if (key === "promoted_links_cache") {
                    promotedState = parsed
                        ? ageState(parsed.timestamp, hourMs, parsed.version, linksVersion)
                        : "sérült";
                    promotedDetail = promotedState;
                } else if (key === "hirek_cache" || key.startsWith("news_cache")) {
                    newsTotal += 1;
                    const state = parsed ? ageState(parsed.timestamp, newsTtlMs, "", "") : "sérült";
                    if (state === "érvényes") newsFresh += 1;
                    else newsStale += 1;
                }
            }
        } catch {
            /* private mode */
        }

        const weatherServer = settingsLoaded
            ? `A szerver TTL ${ttlMin || "-"} perc, verzió ${weatherVersion || "-"}.`
            : "A szerver verziója most nem ismert.";
        rows.push({
            name: "Időjárás",
            detail: `${weatherServer}. Ebben a böngészőben ${weatherTotal} mentés, ${weatherFresh} érvényes.`,
        });
        rows.push({
            name: "Gyorslinkek",
            detail: `${
                settingsLoaded ? `szerververzió ${linksVersion || "-"}, böngésző TTL 60 perc` : "szerververzió még nincs betöltve"
            }. ${promotedDetail}.`,
        });
        rows.push({
            name: "Hírek",
            detail:
                newsTotal > 0
                    ? `30 perces böngésző TTL, szerververzió nélkül. ${newsTotal} mentés, ${newsFresh} érvényes, ${newsStale} lejárt vagy sérült.`
                    : "30 perces böngésző TTL, szerververzió nélkül. Ebben a böngészőben nincs hírmásolat.",
        });
        browserCacheRows = rows;

        /** @type {{ id: string, level: string, text: string, action: string }[]} */
        const notices = [];
        if (weatherTotal === 0) {
            notices.push({
                id: "cache-weather",
                level: "info",
                text: `Időjárás: ebben a böngészőben nincs mentett előrejelzés. ${weatherServer}`,
                action: "cache-refresh",
            });
        } else if (weatherStale > 0) {
            notices.push({
                id: "cache-weather",
                level: "warning",
                text: `Időjárás: ${weatherStale} mentés lejárt, régi verziójú vagy sérült. Érvényes: ${weatherFresh}. ${weatherServer}`,
                action: "cache-refresh",
            });
        } else {
            notices.push({
                id: "cache-weather",
                level: "success",
                text: `Időjárás: ${weatherFresh} mentés érvényes. ${weatherServer}`,
                action: "cache-refresh",
            });
        }

        const linksServer = settingsLoaded
            ? `Szerververzió ${linksVersion || "-"}, böngésző TTL 60 perc.`
            : "A szerververzió most nem ismert. Böngésző TTL 60 perc.";
        if (promotedState === "nincs") {
            notices.push({
                id: "cache-links",
                level: "info",
                text: `Gyorslinkek: ebben a böngészőben nincs mentett lista. ${linksServer}`,
                action: "cache-refresh",
            });
        } else if (promotedState === "érvényes") {
            notices.push({
                id: "cache-links",
                level: "success",
                text: `Gyorslinkek: a mentett lista érvényes. ${linksServer}`,
                action: "cache-refresh",
            });
        } else {
            notices.push({
                id: "cache-links",
                level: "warning",
                text: `Gyorslinkek: a mentett lista ${promotedDetail}. ${linksServer}`,
                action: "cache-refresh",
            });
        }

        if (newsTotal === 0) {
            notices.push({
                id: "cache-news",
                level: "info",
                text: "Hírek: ebben a böngészőben nincs mentett hírlista. A másolat 30 percig él, szerververzió nélkül.",
                action: "cache-refresh",
            });
        } else if (newsStale > 0) {
            notices.push({
                id: "cache-news",
                level: "warning",
                text: `Hírek: ${newsStale} mentés lejárt vagy sérült. Érvényes: ${newsFresh}. A másolat 30 percig él, szerververzió nélkül.`,
                action: "cache-refresh",
            });
        } else {
            notices.push({
                id: "cache-news",
                level: "success",
                text: `Hírek: ${newsFresh} mentés érvényes. A másolat 30 percig él, szerververzió nélkül.`,
                action: "cache-refresh",
            });
        }
        browserCacheNotices = notices;
    }

    async function fetchListingQueue() {
        listingQueueError = "";
        try {
            const res = await apiCall("/api/admin/listing-queue");
            if (!res.ok) {
                listingQueueError = describeApiFailure(
                    "A bejegyzés-jóváhagyások",
                    res.status,
                    await res.text(),
                );
                listingQueueFetched = true;
                return;
            }
            const data = await res.json();
            listingQueueUnpublished = Array.isArray(data.unpublished) ? data.unpublished : [];
            listingQueueMembers = Array.isArray(data.members) ? data.members : [];
            listingQueueClaims = Array.isArray(data.claims) ? data.claims : [];
            listingQueueSuggestions = Array.isArray(data.suggestions) ? data.suggestions : [];
            listingQueueWebsites = Array.isArray(data.websites) ? data.websites : [];
            listingQueueFetched = true;
        } catch (e) {
            listingQueueError = describeTransportError("A bejegyzés-jóváhagyások", e);
            listingQueueFetched = true;
            console.error(e);
        }
    }

    async function publishListingQueueEntry(entryId) {
        const res = await apiCall("/api/admin/listing-queue/publish", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ entry_id: entryId }),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction("welcome", `A közzététel nem sikerült. ${detail}`, false);
            return;
        }
        listingQueueError = "";
        noteAdminAction("welcome", "A bejegyzés közzétéve.");
        await fetchListingQueue();
        await auth.refresh();
    }

    async function approveListingQueueMember(entryId, userId) {
        const res = await apiCall("/api/admin/listing-queue/member", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ entry_id: entryId, user_id: userId, action: "approve" }),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction("welcome", `A jóváhagyás nem sikerült. ${detail}`, false);
            return;
        }
        listingQueueError = "";
        noteAdminAction("welcome", "A tagság jóváhagyva.");
        await fetchListingQueue();
        await auth.refresh();
    }

    const SUGGESTION_FIELD_LABELS = {
        name: "Név",
        tags: "Címkék",
        notes: "Bemutatkozás",
        location_id: "Település",
        address: "Cím",
        hours: "Nyitvatartás",
        delivery_hours: "Kiszállítás",
        url: "Weboldal",
        phone: "Telefon",
        social_links: "Közösségi oldalak",
        languages: "Nyelvek",
        description: "Rövid leírás",
        content: "Tartalom",
        name_ro: "Román név",
        name_de: "Német név",
        activities: "Tevékenységek",
        prohibitions: "Mit nem szabad",
    };

    /** @param {unknown} value */
    function suggestionValueText(value) {
        if (Array.isArray(value)) {
            if (value.every((item) => typeof item === "string")) return value.join(", ") || "-";
            return value
                .map((item) => {
                    if (item && typeof item === "object") {
                        const label = String(item.label ?? "").trim();
                        const url = String(item.url ?? "").trim();
                        return label ? `${label}: ${url}` : url;
                    }
                    return String(item ?? "");
                })
                .filter(Boolean)
                .join(", ") || "-";
        }
        if (value && typeof value === "object") return JSON.stringify(value);
        const text = String(value ?? "").trim();
        return text || "-";
    }

    async function decideListingQueueSuggestion(id, action) {
        const res = await apiCall("/api/admin/listing-queue/suggestion", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id, action }),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction(
                "welcome",
                action === "accept"
                    ? `A javaslat elfogadása nem sikerült. ${detail}`
                    : `A javaslat elutasítása nem sikerült. ${detail}`,
                false,
            );
            return;
        }
        listingQueueError = "";
        noteAdminAction(
            "welcome",
            action === "accept" ? "A javaslat elfogadva." : "A javaslat elutasítva.",
        );
        await fetchListingQueue();
        await auth.refresh();
    }

    async function decideListingQueueClaim(entryId, userId, action) {
        const res = await apiCall("/api/admin/listing-queue/claim", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ entry_id: entryId, user_id: userId, action }),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction(
                "welcome",
                action === "accept"
                    ? `Az átvétel elfogadása nem sikerült. ${detail}`
                    : `Az átvétel elutasítása nem sikerült. ${detail}`,
                false,
            );
            return;
        }
        listingQueueError = "";
        noteAdminAction(
            "welcome",
            action === "accept" ? "Az átvétel elfogadva." : "Az átvétel elutasítva.",
        );
        await fetchListingQueue();
        await auth.refresh();
    }

    async function rejectListingQueueMember(entryId, userId) {
        const res = await apiCall("/api/admin/listing-queue/member", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ entry_id: entryId, user_id: userId, action: "reject" }),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction("welcome", `Az elutasítás nem sikerült. ${detail}`, false);
            return;
        }
        listingQueueError = "";
        noteAdminAction("welcome", "A tagság elutasítva.");
        await fetchListingQueue();
        await auth.refresh();
    }

    async function reviewWebsite(websiteId, action, categoryId = 0, categoryIds = []) {
        const body = { id: websiteId, action };
        if (action === "approve" && categoryId > 0) {
            body.category_id = categoryId;
            body.category_ids = (Array.isArray(categoryIds) ? categoryIds : [])
                .map((id) => Number(id))
                .filter((id) => id > 0);
        }
        const res = await apiCall("/api/admin/websites", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
        });
        if (!res.ok) {
            const detail = (await res.text()) || `HTTP ${res.status}`;
            listingQueueError = detail;
            noteAdminAction("welcome", `A művelet nem sikerült. ${detail}`, false);
            return;
        }
        listingQueueError = "";
        const done =
            action === "approve"
                ? "A weboldal jóváhagyva."
                : action === "reject"
                  ? "A weboldal elutasítva."
                  : action === "ban"
                    ? "A felhasználó tiltva."
                    : "A művelet sikerült.";
        noteAdminAction("welcome", done);
        await fetchListingQueue();
        await fetchAdminWebsites();
        await auth.refresh();
    }

    /** Cím + rövid köszöntő / leírás - minden admin-fülön egységes fejléc. */
    const ADMIN_PAGE_COPY = {
        welcome: {
            title: "Vezérlőpult",
            greeting:
                "Üdvözöllek. A kártyákon a táblák rekordjainak száma látható, a név pedig megegyezik az oldalsáv gombjaival. Kattintva megnyílik a megfelelő kezelőfelület.",
        },
        quicklinks: {
            title: "Gyorslinkek",
            greeting: "Kezdőlap gyors hivatkozásai: cím, URL és háttérszín.",
        },
        newsfeeds: {
            title: "Hírfolyamok",
            greeting: "RSS és Atom hírcsatornák a Hírek oldalhoz.",
        },
        locations: {
            title: "Települések",
            greeting:
                "Települések, megyék és kapcsolódó metaadatok (címer, irányítószám, típus).",
        },
        counties: {
            title: "Megyék",
            greeting: "Megyei tartalom és beállítások a történelmi székekhez kapcsolódóan.",
        },
        venues: {
            title: "Helyszínek",
            greeting:
                "Rendezvényhelyszínek: típusok, településhez kötés, és eseményekhez való hozzárendelés.",
        },
        attractions: {
            title: "Látnivalók",
            greeting: "Megyei látnivalók, leírások és képgaléria.",
        },
        events: {
            title: "Események",
            greeting:
                "Közösségi és sportesemények: időpontok, helyszín, típusok és opcionális program.",
        },
        websites: {
            title: "Weboldalak",
            greeting: "Jóváhagyott és várakozó weboldalak: domain, cím, leírás és beküldő.",
        },
        entries: {
            title: "Bejegyzések",
            greeting: "Címtár-bejegyzések: kategória, típus, elérhetőségek és címkék.",
        },
        entry_categories: {
            title: "Bejegyzés kategóriák",
            greeting: "Index kategóriák felvétele, sorrendezése és törlése.",
        },
        entry_types: {
            title: "Bejegyzés típusok",
            greeting: "A címtárban használható bejegyzés-típusok kezelése.",
        },
        tags: {
            title: "Címkék",
            greeting: "Keresztszavak a bejegyzésekhez. A címke nem ismétli a kategória nevét.",
        },
        pages: {
            title: "Oldalak",
            greeting: "Statikus oldalak (szabályzatok, szöveges tartalmak) szerkesztése.",
        },
        page_faq: {
            title: "GYIK",
            greeting: "Oldalankénti gyakori kérdések és felelősségkizárások.",
        },
        weather_translations: {
            title: "Időjárás fordítások",
            greeting:
                "Az időjárás API szövegeinek fordítása magyarra, románra, németre.",
        },
        users: {
            title: "Felhasználók",
            greeting: "Regisztrált felhasználók és a regisztráció adatai.",
        },
        settings: {
            title: "Beállítások",
            greeting: "Rendszer-, időjárás- és egyéb szolgáltatás-beállítások.",
        },
    };

    $: adminPageHead =
        ADMIN_PAGE_COPY[activeTab] || {
            title: "Admin",
            greeting: "",
        };

    function filterOrganizers(query, target) {
        if (!query || query.length < 2) return [];
        const q = query.toLowerCase();
        return entries
            .filter((e) => e.name.toLowerCase().includes(q))
            .slice(0, 8);
    }

    function onOrgInput(isEdit = false) {
        const q = isEdit ? orgEditQuery : orgQuery;
        const results = filterOrganizers(q);
        if (isEdit) {
            orgEditSuggestions = results;
            orgEditDropdownOpen = results.length > 0;
        } else {
            orgSuggestions = results;
            orgDropdownOpen = results.length > 0;
        }
    }

    function selectOrganizer(name, isEdit = false) {
        if (isEdit) {
            editingEvent.organizer = name;
            orgEditQuery = name;
            orgEditDropdownOpen = false;
        } else {
            newEvent.organizer = name;
            orgQuery = name;
            orgDropdownOpen = false;
        }
    }

    function handleOrgBlur(isEdit = false) {
        setTimeout(() => {
            if (isEdit) orgEditDropdownOpen = false;
            else orgDropdownOpen = false;
        }, 200);
    }

    const LANGUAGES = ["HU", "RO", "DE", "EN"];
    const COUNTIES = ["Hargita", "Kovászna", "Maros"];

    /** @type {{ id: number, slug: string, label_hu: string, sort_order: number }[]} */
    let settlementLocationTypes = [];
    let newSettlementLocationType = { slug: "", label_hu: "", sort_order: 0 };
    /** @type {{ id: number, slug: string, label_hu: string, sort_order: number } | null} */
    let editingSettlementLocationType = null;

    // Custom dialog state
    let dialogVisible = false;
    let dialogMsg = "";
    let dialogType = "alert"; // "alert" or "confirm"
    let dialogResolve = null;

    function showAlert(msg) {
        return new Promise((resolve) => {
            dialogMsg = msg;
            dialogType = "alert";
            dialogResolve = resolve;
            dialogVisible = true;
        });
    }
    function showConfirm(msg) {
        return new Promise((resolve) => {
            dialogMsg = msg;
            dialogType = "confirm";
            dialogResolve = resolve;
            dialogVisible = true;
        });
    }
    function dialogOk() {
        dialogVisible = false;
        if (dialogResolve) dialogResolve(true);
        dialogResolve = null;
    }
    function dialogCancel() {
        dialogVisible = false;
        if (dialogResolve) dialogResolve(false);
        dialogResolve = null;
    }

    /** @param {unknown[]} fields */
    function matchesSearch(q, fields) {
        const s = String(q || "").trim().toLowerCase();
        if (!s) return true;
        return fields.some((f) => String(f ?? "").toLowerCase().includes(s));
    }
    /** @template T
     * @param {T[]} rows
     * @param {string} q
     * @param {(row: T) => unknown[]} fieldFn
     */
    function filterRows(rows, q, fieldFn) {
        if (!rows || !rows.length) return rows || [];
        const s = String(q || "").trim().toLowerCase();
        if (!s) return rows;
        return rows.filter((row) =>
            fieldFn(row).some((f) =>
                String(f ?? "")
                    .toLowerCase()
                    .includes(s),
            ),
        );
    }

    const ACCESS_TYPE_LABELS = {
        public: "Nyitott (bárki)",
        members_only: "Zártkörű (csak tagoknak)",
        invitation_only: "Meghívóval (zárt kör)",
    };

    /** @param {unknown} v */
    function accessTypeLabel(v) {
        const k = String(v || "public");
        return ACCESS_TYPE_LABELS[k] || k;
    }

    /** @param {unknown} slug */
    function settlementTypeLabel(slug) {
        const s = String(slug || "").trim();
        if (!s) return "-";
        const t = settlementLocationTypes.find((x) => x.slug === s);
        return t ? t.label_hu : s;
    }

    /** @param {string} slug */
    function eventTypeLabelFromCatalog(slug) {
        const s = String(slug || "").trim();
        if (!s) return "-";
        const t = catalogEventTypes.find((x) => x.slug === s);
        return t ? t.label_hu : s;
    }

    /** @param {number} typeId @param {string} subSlug */
    function eventSubtypeLabelFromCatalog(typeId, subSlug) {
        const ss = String(subSlug || "").trim();
        if (!ss) return "-";
        const s = catalogEventSubtypes.find(
            (x) =>
                x.slug === ss &&
                (typeId ? x.event_type_id === typeId : true),
        );
        return s ? s.label_hu : ss;
    }

    $: rfQuickLinks = filterRows(quickLinks, searchQuickLinks, (q) => [
        q.title,
        q.url,
        q.bg_color,
    ]);
    $: pgQuickLinks = adminPageSlice(rfQuickLinks, pageQuickLinks);
    $: rfNewsFeeds = filterRows(newsFeeds, searchNewsFeeds, (nf) => [
        nf.title,
        nf.feed_url,
        String(nf.id),
    ]);
    $: pgNewsFeeds = adminPageSlice(rfNewsFeeds, pageNewsFeeds);
    $: rfLocations = filterRows(locations, searchLocations, (l) => {
        const parentN =
            l.parent_id != null && l.parent_id !== ""
                ? (() => {
                      const p = locations.find((x) => x.id === l.parent_id);
                      return p
                          ? `${p.name}${p.county ? " (" + p.county + ")" : ""}`
                          : String(l.parent_id);
                  })()
                : "";
        return [
            l.id,
            l.name,
            l.name_ro,
            l.name_de,
            l.county,
            l.type,
            l.post_code,
            l.coordinates,
            l.population,
            l.area,
            parentN,
        ];
    });
    $: pgLocations = adminPageSlice(rfLocations, pageLocations);
    $: rfVenueTypes = filterRows(venueTypesList, searchVenueTypes, (t) => [
        t.id,
        t.slug,
        t.label_hu,
    ]);
    $: pgVenueTypes = adminPageSlice(rfVenueTypes, pageVenueTypes);
    $: rfVenuesCatalog = filterRows(venuesCatalog, searchVenues, (v) => [
        v.name,
        v.slug,
        v.kind,
        v.settlement_name,
    ]);
    $: pgVenuesCatalog = adminPageSlice(rfVenuesCatalog, pageVenues);
    $: rfEvents = filterRows(events, searchEvents, (e) => {
        const loc = locations.find((l) => l.id === e.location_id);
        const locN = loc
            ? `${loc.name}${loc.county ? " (" + loc.county + ")" : ""}`
            : String(e.location_id ?? "");
        return [
            e.title,
            e.description,
            e.organizer,
            locN,
            e.default_venue_name,
            e.event_type,
            e.event_subtype,
            e.access_type,
            e.start_date,
            e.end_date,
            e.featured_image,
            e.entry_price,
        ];
    });
    $: pgEvents = adminPageSlice(rfEvents, pageEvents);
    $: entryCategoryParents = entryCategories.filter(
        (cat) => cat.parent_id == null,
    );
    $: entryCategoryChildren = entryCategories.filter(
        (cat) => cat.parent_id != null,
    );
    $: rfEntryCategories = filterRows(
        entryCategories,
        searchEntryCategories,
        (cat) => [cat.id, cat.name, getEntryCategoryParentName(cat)],
    );
    $: pgEntryCategories = adminPageSlice(rfEntryCategories, pageEntryCategories);
    $: rfEntries = filterRows(entries, searchEntries, (s) => {
        const locN = locations.find((l) => l.id === s.location_id);
        const locName = locN
            ? `${locN.name}${locN.county ? " (" + locN.county + ")" : ""}`
            : String(s.location_id ?? "");
        const cat = entryCategories.find((c) => c.id === s.category_id);
        const catName = cat ? cat.name : String(s.category_id ?? "");
        return [
            s.name,
            s.type,
            s.url,
            s.phone,
            s.address,
            s.notes,
            locName,
            catName,
            (s.languages || []).join(","),
            (s.tags || []).join(","),
        ];
    });
    $: pgEntries = adminPageSlice(rfEntries, pageEntries);
    $: rfAdminWebsites = filterRows(adminWebsites, searchAdminWebsites, (site) => [
        site.title,
        site.domain,
        site.description,
        site.submitter,
        site.approver,
        site.status,
        site.claimed ? "Átvéve" : "Gazdátlan",
    ]);
    $: pgAdminWebsites = adminPageSlice(rfAdminWebsites, pageAdminWebsites);
    $: rfWeatherTrans = filterRows(
        weatherTranslations,
        searchWeatherTrans,
        (wt) => [wt.source_text, wt.lang, wt.translated_text],
    );
    $: pgWeatherTrans = adminPageSlice(rfWeatherTrans, pageWeatherTrans);
    $: rfAdminPages = filterRows(adminPages, searchAdminPages, (pg) => [
        pg.slug,
        pg.title,
        pg.greeting,
        pg.updated_at,
    ]);
    $: pgAdminPages = adminPageSlice(rfAdminPages, pageAdminPages);
    $: rfPageFaqRows = filterRows(pageFaqSections, searchPageFaqRows, (row) => [
        row.section_key,
        row.label_hu,
        row.faq_title,
        String((row.faq_items || []).length),
        row.updated_at,
    ]);
    $: pgPageFaqRows = adminPageSlice(rfPageFaqRows, pagePageFaqRows);
    $: rfUsers = filterRows(adminUsers, searchUsers, (u) => [
        u.email,
        u.name,
        u.given_name,
        u.family_name,
        u.display_name,
        u.locale,
        u.settlement,
    ]);
    $: pgUsers = adminPageSlice(rfUsers, pageUsers);
    $: rfEntryTypes = filterRows(entryTypes, searchEntryTypes, (et) => [
        et.id,
        et.name,
    ]);
    $: pgEntryTypes = adminPageSlice(rfEntryTypes, pageEntryTypes);
    $: rfTags = filterRows(tags, searchTags, (tag) => [tag.id, tag.name, tag.usage]);
    $: pgTags = adminPageSlice(rfTags, pageTags);
    $: rfAttractions = filterRows(attractions, searchAttractions, (att) => [
        att.name,
        att.slug,
        att.county_slug,
    ]);
    $: pgAttractions = adminPageSlice(rfAttractions, pageAttractions);
    $: pgCounties = adminPageSlice(
        displayCounties,
        pageCounties,
        editingCounty
            ? Math.max(ADMIN_PAGE_SIZE, displayCounties.length)
            : ADMIN_PAGE_SIZE,
    );
    $: pgHistoricalSeats = adminPageSlice(
        displayHistoricalSeats,
        pageHistoricalSeats,
        editingHistoricalSeat
            ? Math.max(ADMIN_PAGE_SIZE, displayHistoricalSeats.length)
            : ADMIN_PAGE_SIZE,
    );
    $: rfCatalogTypes = filterRows(
        catalogEventTypes,
        searchCatalogTypes,
        (t) => [t.id, t.slug, t.label_hu, String(t.sort_order)],
    );
    $: pgCatalogTypes = adminPageSlice(
        rfCatalogTypes,
        pageCatalogTypes,
        editingCatalogEventType
            ? Math.max(ADMIN_PAGE_SIZE, rfCatalogTypes.length)
            : ADMIN_PAGE_SIZE,
    );
    $: rfCatalogSubtypes = filterRows(
        catalogEventSubtypes,
        searchCatalogSubtypes,
        (s) => [s.id, s.slug, s.label_hu, String(s.sort_order), s.event_type_id],
    );
    $: pgCatalogSubtypes = adminPageSlice(
        rfCatalogSubtypes,
        pageCatalogSubtypes,
        editingCatalogEventSubtype
            ? Math.max(ADMIN_PAGE_SIZE, rfCatalogSubtypes.length)
            : ADMIN_PAGE_SIZE,
    );
    $: subtypesForNewEvent = catalogEventSubtypes.filter(
        (s) => s.event_type_id === Number(newEvent.event_type_id),
    );
    $: subtypesForEditEvent = editingEvent
        ? catalogEventSubtypes.filter(
              (s) =>
                  s.event_type_id === Number(editingEvent.event_type_id),
          )
        : [];

    onMount(() => {

        const storedTs = localStorage.getItem("news_feed_timestamps");
        if (storedTs) {
            try {
                feedTimestamps = JSON.parse(storedTs);
            } catch (e) {}
        }
    });

    /** The shell's onReady: the person is in as admin. */
    function onShellReady(/** @type {{ offline: boolean }} */ me) {
        adminOffline = me.offline;
        fetchAll();
    }

    async function fetchAll() {
        collectBrowserCaches();
        await fetchDashboardStats();
        await fetchListingQueue();
        fetchQuickLinks();
        fetchNewsFeeds();
        fetchLocations();
        fetchSettlementLocationTypes();
        fetchAttractions();
        fetchEntries();
        fetchAdminWebsites();
        fetchEntryCategories();
        fetchEntryTypes();
        fetchTags();
        fetchEvents();
        fetchVenuesCatalog();
        fetchVenueTypes();
        fetchSettings();
        fetchWeatherTranslations();
        fetchPages();
        fetchPageFaq();
        fetchCountyRegions();
        fetchUsers();
    }

    function fetchUsers() {
        loadData("users", (d) => (adminUsers = Array.isArray(d) ? d : []));
    }

    async function fetchWeatherTranslations() {
        await loadData("weather_translations", (d) => (weatherTranslations = d));
    }

    async function saveWeatherTranslation(e) {
        e?.preventDefault();
        if (editingWeatherTrans) {
            const res = await apiCall(`/api/admin/weather_translations`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(editingWeatherTrans),
            });
            if (res.ok) {
                noteAdminAction("weather_translations", "A fordítás mentve.");
                fetchWeatherTranslations();
                editingWeatherTrans = null;
            } else {
                await noteAdminFailure("weather_translations", res);
            }
        } else {
            const res = await apiCall(`/api/admin/weather_translations`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(newWeatherTrans),
            });
            if (res.ok) {
                noteAdminAction("weather_translations", "A fordítás hozzáadva.");
                fetchWeatherTranslations();
                newWeatherTrans = { source_text: "", lang: "hu", translated_text: "" };
            } else {
                await noteAdminFailure("weather_translations", res);
            }
        }
    }

    function startEditWeatherTrans(t) {
        editingWeatherTrans = { ...t };
    }

    function cancelEditWeatherTrans() {
        editingWeatherTrans = null;
    }

    async function deleteWeatherTranslation(id) {
        const ok = await showConfirm("Biztosan törölni szeretnéd ezt a fordítást?");
        if (!ok) return;
        const res = await apiCall(`/api/admin/weather_translations?id=${id}`, { method: "DELETE" });
        if (res.ok) {
            noteAdminAction("weather_translations", "A fordítás törölve.");
            fetchWeatherTranslations();
        } else {
            await noteAdminFailure("weather_translations", res);
        }
    }

    async function fetchSettings() {
        try {
            const res = await apiCall(`/api/admin/settings`);
            if (!res.ok) {
                settingsLoadError = describeApiFailure("A beállítások", res.status, await res.text());
                collectBrowserCaches();
                return;
            }
            const data = await res.json();
            siteSettings = {
                weather_cache_ttl_minutes: data.weather_cache_ttl_minutes ?? "15",
                weather_cache_version: data.weather_cache_version ?? "1",
                quick_links_version: data.quick_links_version ?? "1",
                weather_icon_style: data.weather_icon_style ?? "emoji",
                weather_provider_metno_enabled: data.weather_provider_metno_enabled ?? "true",
                weather_provider_weatherapi_enabled: data.weather_provider_weatherapi_enabled ?? "true",
                weather_provider_openweathermap_enabled: data.weather_provider_openweathermap_enabled ?? "true",
                my_location_slug: data.my_location_slug ?? "csikszereda",
                social_facebook_url: data.social_facebook_url ?? "",
                social_twitter_url: data.social_twitter_url ?? "",
                social_instagram_url: data.social_instagram_url ?? "",
                ...data,
            };
            settingsLoaded = true;
            settingsLoadError = "";
            collectBrowserCaches();
        } catch (e) {
            settingsLoadError = describeTransportError("A beállítások", e);
            collectBrowserCaches();
            console.error(e);
        }
    }

    /**
     * Saves one section of the Beállítások tab: only its own keys
     * (lib/settingsSections.js), so one section's Mentés never rewrites another.
     * @param {keyof typeof SETTINGS_SECTIONS} section
     */
    async function saveSettings(section) {
        settingsSaving = section;
        try {
            const res = await apiCall(`/api/admin/settings`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(settingsPayload(siteSettings, section)),
            });
            if (res.ok) {
                noteAdminAction("settings", `${SETTINGS_SECTIONS[section].label}: mentve.`);
            } else await noteAdminFailure("settings", res);
        } catch (e) {
            await noteAdminFailure("settings", e.message);
        } finally {
            settingsSaving = "";
        }
    }

    async function clearWeatherCache() {
        settingsCacheClearing = true;
        try {
            const res = await apiCall(`/api/admin/settings/clear-weather-cache`, { method: "POST" });
            if (res.ok) {
                noteAdminAction(
                    "settings",
                    "Időjárás cache verzió növelve – látogatók friss adatot fognak kapni.",
                );
                fetchSettings();
            } else await noteAdminFailure("settings", res);
        } catch (e) {
            await noteAdminFailure("settings", e.message);
        } finally {
            settingsCacheClearing = false;
        }
    }

    async function fetchPages() {
        await loadData("pages", (d) => (adminPages = d));
    }

    async function fetchPageFaq() {
        await loadData("page_faq", (d) => (pageFaqSections = d));
    }

    function startEditPage(page) {
        editingPageFaq = null;
        editingPage = { ...page, greeting: page.greeting ?? "" };
    }

    function cancelEditPage() {
        editingPage = null;
    }

    async function savePage() {
        if (!editingPage) return;
        pageSaving = true;
        try {
            const res = await apiCall(`/api/admin/pages`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(editingPage),
            });
            if (res.ok) {
                noteAdminAction("pages", "Oldal mentve.");
                editingPage = null;
                fetchPages();
            } else {
                await noteAdminFailure("pages", res);
            }
        } catch (e) {
            await noteAdminFailure("pages", e.message);
        } finally {
            pageSaving = false;
        }
    }

    function startEditPageFaq(row) {
        editingPage = null;
        const items = Array.isArray(row.faq_items)
            ? row.faq_items.map((x) => ({
                  question: x.question ?? "",
                  answer: x.answer ?? "",
              }))
            : [];
        editingPageFaq = { ...row, faq_items: items };
    }

    function cancelEditPageFaq() {
        editingPageFaq = null;
    }

    function addFaqItem() {
        if (!editingPageFaq) return;
        editingPageFaq.faq_items = [
            ...(editingPageFaq.faq_items || []),
            { question: "", answer: "" },
        ];
    }

    function removeFaqItem(index) {
        if (!editingPageFaq?.faq_items) return;
        editingPageFaq.faq_items = editingPageFaq.faq_items.filter(
            (_, i) => i !== index,
        );
    }

    async function savePageFaq() {
        if (!editingPageFaq) return;
        pageFaqSaving = true;
        try {
            const res = await apiCall(`/api/admin/page_faq`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id: editingPageFaq.id,
                    label_hu: editingPageFaq.label_hu ?? "",
                    faq_title: editingPageFaq.faq_title ?? "",
                    faq_items: editingPageFaq.faq_items ?? [],
                    disclaimer_markdown: editingPageFaq.disclaimer_markdown ?? "",
                }),
            });
            if (res.ok) {
                noteAdminAction("page_faq", "GYIK / disclaimer mentve.");
                editingPageFaq = null;
                fetchPageFaq();
            } else {
                await noteAdminFailure("page_faq", res);
            }
        } catch (e) {
            await noteAdminFailure("page_faq", e.message);
        } finally {
            pageFaqSaving = false;
        }
    }

    // generic fetch helper
    async function loadData(endpoint, setter) {
        const path = endpoint.startsWith("/") ? endpoint : `/api/admin/${endpoint}`;
        const source = endpoint.replace(/^\/api\/(?:admin\/)?/, "");
        try {
            const res = await apiCall(path);
            if (!res.ok) {
                const label = ADMIN_API_LABELS[source] || source;
                rememberApiError(source, describeApiFailure(label, res.status, await res.text()));
                return;
            }
            setter(await res.json());
            forgetApiError(source);
        } catch (e) {
            rememberApiError(source, describeTransportError(ADMIN_API_LABELS[source] || source, e));
            console.error(e);
        }
    }

    // --- specific fetches ---
    function fetchQuickLinks() {
        loadData("quick_links", (d) => (quickLinks = d));
    }
    function fetchNewsFeeds() {
        loadData("news_feeds", (d) => (newsFeeds = d));
    }
    function fetchLocations() {
        loadData("locations", (d) => (locations = d));
    }

    async function fetchSettlementLocationTypes() {
        await loadData("settlement_location_types", (d) => (settlementLocationTypes = d));
    }

    async function submitNewSettlementLocationType(e) {
        e.preventDefault();
        const label_hu = String(newSettlementLocationType.label_hu || "").trim();
        let slug = String(newSettlementLocationType.slug || "")
            .trim()
            .toLowerCase();
        const sort_order = Number(newSettlementLocationType.sort_order) || 0;
        if (!label_hu) {
            noteAdminAction("settlement_location_types", "A megnevezés kötelező.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/settlement_location_types`,
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ slug, label_hu, sort_order }),
                },
            );
            if (!res.ok) {
                await noteAdminFailure("settlement_location_types", res, "");
                return;
            }
            newSettlementLocationType = { slug: "", label_hu: "", sort_order: 0 };
            noteAdminAction("settlement_location_types", `„${label_hu}” hozzáadva.`);
            await fetchSettlementLocationTypes();
        } catch (err) {
            await noteAdminFailure("settlement_location_types", err.message || err, "");
        }
    }

    /** @param {Record<string, unknown>} t */
    function startEditSettlementLocationType(t) {
        editingSettlementLocationType = {
            id: Number(t.id),
            slug: String(t.slug || ""),
            label_hu: String(t.label_hu || ""),
            sort_order: Number(t.sort_order) || 0,
        };
    }
    function cancelEditSettlementLocationType() {
        editingSettlementLocationType = null;
    }
    async function saveEditSettlementLocationType() {
        if (!editingSettlementLocationType) return;
        const id = parseInt(String(editingSettlementLocationType.id || ""), 10);
        const label_hu = String(
            editingSettlementLocationType.label_hu || "",
        ).trim();
        const sort_order =
            Number(editingSettlementLocationType.sort_order) || 0;
        if (!Number.isFinite(id) || id < 1 || !label_hu) return;
        try {
            const res = await apiCall(`/api/admin/settlement_location_types`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        id,
                        label_hu,
                        sort_order,
                    }),
                },
            );
            if (!res.ok) {
                await noteAdminFailure("settlement_location_types", res, "");
                return;
            }
            editingSettlementLocationType = null;
            noteAdminAction("settlement_location_types", `„${label_hu}” mentve.`);
            await fetchSettlementLocationTypes();
        } catch (err) {
            await noteAdminFailure("settlement_location_types", err.message || err, "");
        }
    }
    async function deleteSettlementLocationTypeRow(id) {
        const ok = await showConfirm(
            "Biztosan törlöd ezt a településtípust? (Nem lehetséges, ha van ilyen típusú település.)",
        );
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/settlement_location_types?id=${encodeURIComponent(id)}`,
                { method: "DELETE" },
            );
            if (!res.ok) {
                await noteAdminFailure("settlement_location_types", res, "");
                return;
            }
            noteAdminAction("settlement_location_types", "A településtípus törölve.");
            await fetchSettlementLocationTypes();
        } catch (err) {
            await noteAdminFailure("settlement_location_types", err.message || err, "");
        }
    }

    // Entries/events use settlement_id; filter out counties (type=megye)
    $: settlementsForSelect = locations.filter((l) => l.type !== "megye");

    /** @returns {Promise<boolean>} */
    async function setCountySeat(locationId) {
        try {
            const res = await apiCall(`/api/admin/county_seat`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ location_id: locationId }),
            });
            if (res.ok) {
                fetchLocations();
                return true;
            }
            console.error("Failed to set county seat:", await res.text());
            return false;
        } catch (e) {
            console.error("Error setting county seat:", e);
            return false;
        }
    }
    function fetchEntries() {
        loadData("entries", (d) => (entries = d));
    }
    async function fetchAdminWebsites() {
        try {
            const res = await apiCall("/api/admin/websites");
            if (!res.ok) return;
            const data = await res.json();
            adminWebsites = Array.isArray(data.websites) ? data.websites : [];
        } catch (e) {
            console.error(e);
        }
    }
    function fetchEntryCategories() {
        loadData("entry_categories", (d) => (entryCategories = d));
    }
    function fetchEntryTypes() {
        loadData("entry_types", (d) => (entryTypes = d));
    }
    function fetchTags() {
        loadData("tags", (d) => (tags = d));
    }

    function submitTag(e) {
        e.preventDefault();
        createRecord(
            "tags",
            { name: String(newTag.name ?? "").trim() },
            fetchTags,
            () => (newTag = { name: "" }),
        );
    }

    async function startEditTag(tag) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        editingTag = { ...tag };
    }
    function cancelEditTag() {
        editingTag = null;
    }
    async function saveEditTag() {
        if (!editingTag) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        await updateRecord("tags", { id: editingTag.id, name: editingTag.name }, fetchTags);
        editingTag = null;
    }
    async function fetchEvents() {
        await Promise.all([
            loadData("events", (d) => (events = d)),
            loadData("catalog_event_types", (d) => (catalogEventTypes = d)),
            loadData("catalog_event_subtypes", (d) => (catalogEventSubtypes = d)),
        ]);
    }

    async function submitCatalogEventType(e) {
        e.preventDefault();
        const slug = String(newCatalogEventType.slug || "")
            .trim()
            .toLowerCase()
            .replace(/\s+/g, "-");
        const label_hu = String(newCatalogEventType.label_hu || "").trim();
        const sort_order = Number(newCatalogEventType.sort_order) || 0;
        if (!slug || !label_hu) {
            noteAdminAction("catalog_event_types", "Slug és megnevezés kötelező.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/catalog_event_types`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ slug, label_hu, sort_order }),
            });
            if (!res.ok) {
                await noteAdminFailure("catalog_event_types", res, "");
                return;
            }
            newCatalogEventType = { slug: "", label_hu: "", sort_order: 0 };
            noteAdminAction("catalog_event_types", `„${label_hu}” hozzáadva.`);
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_types", err.message || err, "");
        }
    }

    /** @param {Record<string, unknown>} t */
    function startEditCatalogEventType(t) {
        editingCatalogEventType = {
            id: Number(t.id),
            slug: String(t.slug || ""),
            label_hu: String(t.label_hu || ""),
            sort_order: Number(t.sort_order) || 0,
        };
    }
    function cancelEditCatalogEventType() {
        editingCatalogEventType = null;
    }
    async function saveEditCatalogEventType() {
        if (!editingCatalogEventType) return;
        const id = parseInt(String(editingCatalogEventType.id || ""), 10);
        const label_hu = String(editingCatalogEventType.label_hu || "").trim();
        const sort_order = Number(editingCatalogEventType.sort_order) || 0;
        if (!Number.isFinite(id) || id < 1 || !label_hu) return;
        try {
            const res = await apiCall(`/api/admin/catalog_event_types`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id,
                    label_hu,
                    sort_order,
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("catalog_event_types", res, "");
                return;
            }
            editingCatalogEventType = null;
            noteAdminAction("catalog_event_types", `„${label_hu}” mentve.`);
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_types", err.message || err, "");
        }
    }
    async function deleteCatalogEventTypeRow(id) {
        const ok = await showConfirm(
            "Biztosan törlöd ezt az eseménytípust? (Csak akkor sikerül, ha nincs hozzá esemény.)",
        );
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/catalog_event_types?id=${encodeURIComponent(id)}`,
                { method: "DELETE" },
            );
            if (!res.ok) {
                await noteAdminFailure("catalog_event_types", res, "");
                return;
            }
            noteAdminAction("catalog_event_types", "Az eseménytípus törölve.");
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_types", err.message || err, "");
        }
    }

    async function submitCatalogEventSubtype(e) {
        e.preventDefault();
        const event_type_id = parseInt(
            String(newCatalogEventSubtype.event_type_id || ""),
            10,
        );
        const slug = String(newCatalogEventSubtype.slug || "")
            .trim()
            .toLowerCase()
            .replace(/\s+/g, "-");
        const label_hu = String(newCatalogEventSubtype.label_hu || "").trim();
        const sort_order = Number(newCatalogEventSubtype.sort_order) || 0;
        if (!Number.isFinite(event_type_id) || event_type_id < 1 || !slug || !label_hu) {
            noteAdminAction("catalog_event_subtypes", "Típus, slug és megnevezés kötelező.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/catalog_event_subtypes`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    event_type_id,
                    slug,
                    label_hu,
                    sort_order,
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("catalog_event_subtypes", res, "");
                return;
            }
            newCatalogEventSubtype = {
                event_type_id: "",
                slug: "",
                label_hu: "",
                sort_order: 0,
            };
            noteAdminAction("catalog_event_subtypes", `„${label_hu}” hozzáadva.`);
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_subtypes", err.message || err, "");
        }
    }

    /** @param {Record<string, unknown>} s */
    function startEditCatalogEventSubtype(s) {
        editingCatalogEventSubtype = {
            id: Number(s.id),
            event_type_id: Number(s.event_type_id),
            slug: String(s.slug || ""),
            label_hu: String(s.label_hu || ""),
            sort_order: Number(s.sort_order) || 0,
        };
    }
    function cancelEditCatalogEventSubtype() {
        editingCatalogEventSubtype = null;
    }
    async function saveEditCatalogEventSubtype() {
        if (!editingCatalogEventSubtype) return;
        const id = parseInt(String(editingCatalogEventSubtype.id || ""), 10);
        const label_hu = String(editingCatalogEventSubtype.label_hu || "").trim();
        const sort_order = Number(editingCatalogEventSubtype.sort_order) || 0;
        if (!Number.isFinite(id) || id < 1 || !label_hu) return;
        try {
            const res = await apiCall(`/api/admin/catalog_event_subtypes`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id,
                    label_hu,
                    sort_order,
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("catalog_event_subtypes", res, "");
                return;
            }
            editingCatalogEventSubtype = null;
            noteAdminAction("catalog_event_subtypes", `„${label_hu}” mentve.`);
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_subtypes", err.message || err, "");
        }
    }
    async function deleteCatalogEventSubtypeRow(id) {
        const ok = await showConfirm(
            "Biztosan törlöd ezt az altípust? (Csak akkor sikerül, ha nincs hozzá esemény.)",
        );
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/catalog_event_subtypes?id=${encodeURIComponent(id)}`,
                { method: "DELETE" },
            );
            if (!res.ok) {
                await noteAdminFailure("catalog_event_subtypes", res, "");
                return;
            }
            noteAdminAction("catalog_event_subtypes", "Az altípus törölve.");
            await fetchEvents();
        } catch (err) {
            await noteAdminFailure("catalog_event_subtypes", err.message || err, "");
        }
    }
    async function fetchVenuesCatalog() {
        await loadData("venues", (d) => (venuesCatalog = d));
    }
    async function fetchVenueTypes() {
        try {
            const res = await apiCall(`/api/admin/venue_types`);
            if (!res.ok) {
                rememberApiError(
                    "venue_types",
                    describeApiFailure("A helyszíntípusok", res.status, await res.text()),
                );
                return;
            }
            forgetApiError("venue_types");
            venueTypesList = await res.json();
            if (venueTypesList.length) {
                    const slugs = new Set(venueTypesList.map((t) => t.slug));
                    if (!slugs.has(String(newVenue.kind))) {
                        newVenue = {
                            ...newVenue,
                            kind: venueTypesList[0].slug,
                        };
                    }
                    if (
                        editingVenue &&
                        !slugs.has(String(editingVenue.kind))
                    ) {
                        editingVenue = {
                            ...editingVenue,
                            kind: venueTypesList[0].slug,
                        };
                    }
            }
        } catch (e) {
            rememberApiError("venue_types", describeTransportError("A helyszíntípusok", e));
            console.error(e);
        }
    }
    async function submitNewVenueType(e) {
        e.preventDefault();
        const venueTypeLabel = String(newVenueType.label_hu || "").trim();
        if (!venueTypeLabel) {
            noteAdminAction("venue_types", "A megnevezés kötelező.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/venue_types`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    label_hu: venueTypeLabel,
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("venue_types", res, "");
                return;
            }
            newVenueType = { label_hu: "" };
            noteAdminAction("venue_types", `„${venueTypeLabel}” hozzáadva.`);
            await fetchVenueTypes();
        } catch (err) {
            await noteAdminFailure("venue_types", err.message || err, "");
        }
    }
    /** @param {Record<string, unknown>} t */
    function startEditVenueType(t) {
        editingVenueType = {
            id: t.id,
            slug: t.slug,
            label_hu: t.label_hu ?? "",
        };
    }
    function cancelEditVenueType() {
        editingVenueType = null;
    }
    async function saveEditVenueType() {
        if (!editingVenueType) return;
        const id = parseInt(String(editingVenueType.id || ""), 10);
        if (!Number.isFinite(id) || id < 1) return;
        try {
            const res = await apiCall(`/api/admin/venue_types`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id,
                    label_hu: String(editingVenueType.label_hu || "").trim(),
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("venue_types", res, "");
                return;
            }
            const savedLabel = String(editingVenueType.label_hu || "").trim();
            editingVenueType = null;
            noteAdminAction("venue_types", savedLabel ? `„${savedLabel}” mentve.` : "A helyszíntípus mentve.");
            await fetchVenueTypes();
        } catch (err) {
            await noteAdminFailure("venue_types", err.message || err, "");
        }
    }
    async function deleteVenueTypeRow(id) {
        const ok = await showConfirm(
            "Biztosan törlöd ezt a helyszíntípust? (Nem lehetséges, ha van hozzárendelt helyszín.)",
        );
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/venue_types?id=${encodeURIComponent(id)}`,
                { method: "DELETE" },
            );
            if (!res.ok) {
                await noteAdminFailure("venue_types", res, "");
                return;
            }
            noteAdminAction("venue_types", "A helyszíntípus törölve.");
            await fetchVenueTypes();
        } catch (err) {
            await noteAdminFailure("venue_types", err.message || err, "");
        }
    }
    async function loadVenuesForNewEvent() {
        const sid = parseInt(String(newEvent.location_id || ""), 10);
        if (!Number.isFinite(sid) || sid < 1) {
            venueOptionsNew = [];
            return;
        }
        try {
            const res = await apiCall(`/api/venues?settlement_id=${sid}`,
            );
            venueOptionsNew = res.ok ? await res.json() : [];
        } catch (e) {
            console.error(e);
            venueOptionsNew = [];
        }
    }
    async function loadVenuesForEditSettlement(sidRaw) {
        const sid = parseInt(String(sidRaw || ""), 10);
        if (!Number.isFinite(sid) || sid < 1) {
            venueOptionsEdit = [];
            return;
        }
        try {
            const res = await apiCall(`/api/venues?settlement_id=${sid}`,
            );
            venueOptionsEdit = res.ok ? await res.json() : [];
        } catch (e) {
            console.error(e);
            venueOptionsEdit = [];
        }
    }
    function parseOptFloatVenue(s) {
        const t = String(s ?? "")
            .trim()
            .replace(",", ".");
        if (!t) return null;
        const n = parseFloat(t);
        return Number.isFinite(n) ? n : null;
    }
    function parseOptIntVenue(s) {
        const t = String(s ?? "").trim();
        if (!t) return null;
        const n = parseInt(t, 10);
        return Number.isFinite(n) ? n : null;
    }

    function emptyNewVenue() {
        return {
            settlement_id: "",
            name: "",
            name_ro: "",
            name_de: "",
            slug: "",
            kind: "sports_arena",
            address: "",
            latitude: "",
            longitude: "",
            seating_capacity: "",
            description: "",
            notes: "",
        };
    }

    async function submitNewVenue(e) {
        e.preventDefault();
        const sid = parseInt(String(newVenue.settlement_id || ""), 10);
        if (!Number.isFinite(sid) || sid < 1 || !String(newVenue.name || "").trim()) {
            noteAdminAction("venues", "Válassz települést és adj meg nevet.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/venues`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    settlement_id: sid,
                    name: newVenue.name.trim(),
                    name_ro: String(newVenue.name_ro || "").trim(),
                    name_de: String(newVenue.name_de || "").trim(),
                    slug: newVenue.slug?.trim() || "",
                    kind: newVenue.kind || "other",
                    address: newVenue.address || "",
                    notes: newVenue.notes || "",
                    latitude: parseOptFloatVenue(newVenue.latitude),
                    longitude: parseOptFloatVenue(newVenue.longitude),
                    seating_capacity: parseOptIntVenue(newVenue.seating_capacity),
                    description: String(newVenue.description || "").trim(),
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("venues", res, "");
                return;
            }
            const savedName = newVenue.name.trim();
            await fetchVenuesCatalog();
            await loadVenuesForNewEvent();
            if (editingEvent)
                await loadVenuesForEditSettlement(editingEvent.location_id);
            newVenue = emptyNewVenue();
            noteAdminAction("venues", savedName ? `„${savedName}” hozzáadva.` : "Helyszín elmentve.");
        } catch (err) {
            await noteAdminFailure("venues", err.message || err, "");
        }
    }

    /** @param {Record<string, unknown>} v */
    function startEditVenue(v) {
        editingVenue = {
            id: v.id,
            settlement_id: String(v.settlement_id ?? ""),
            name: v.name ?? "",
            name_ro: v.name_ro ?? "",
            name_de: v.name_de ?? "",
            slug: v.slug ?? "",
            kind: v.kind ?? "other",
            address: v.address ?? "",
            notes: v.notes ?? "",
            latitude:
                v.latitude != null && v.latitude !== ""
                    ? String(v.latitude)
                    : "",
            longitude:
                v.longitude != null && v.longitude !== ""
                    ? String(v.longitude)
                    : "",
            seating_capacity:
                v.seating_capacity != null && v.seating_capacity !== ""
                    ? String(v.seating_capacity)
                    : "",
            description: v.description ?? "",
        };
    }
    function cancelEditVenue() {
        editingVenue = null;
    }
    async function saveEditVenue() {
        if (!editingVenue) return;
        const sid = parseInt(String(editingVenue.settlement_id || ""), 10);
        const id = parseInt(String(editingVenue.id || ""), 10);
        if (!Number.isFinite(sid) || sid < 1 || !Number.isFinite(id) || id < 1) {
            noteAdminAction("venues", "Érvénytelen azonosító.", false);
            return;
        }
        if (!String(editingVenue.name || "").trim()) {
            noteAdminAction("venues", "A magyar név kötelező.", false);
            return;
        }
        try {
            const res = await apiCall(`/api/admin/venues`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id,
                    settlement_id: sid,
                    name: String(editingVenue.name).trim(),
                    name_ro: String(editingVenue.name_ro || "").trim(),
                    name_de: String(editingVenue.name_de || "").trim(),
                    slug: String(editingVenue.slug || "").trim(),
                    kind: editingVenue.kind || "other",
                    address: String(editingVenue.address || ""),
                    notes: String(editingVenue.notes || ""),
                    latitude: parseOptFloatVenue(editingVenue.latitude),
                    longitude: parseOptFloatVenue(editingVenue.longitude),
                    seating_capacity: parseOptIntVenue(editingVenue.seating_capacity),
                    description: String(editingVenue.description || "").trim(),
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("venues", res, "");
                return;
            }
            const savedName = String(editingVenue.name || "").trim();
            editingVenue = null;
            noteAdminAction("venues", savedName ? `„${savedName}” mentve.` : "A helyszín mentve.");
            await fetchVenuesCatalog();
            await loadVenuesForNewEvent();
            if (editingEvent)
                await loadVenuesForEditSettlement(editingEvent.location_id);
        } catch (err) {
            await noteAdminFailure("venues", err.message || err, "");
        }
    }
    async function deleteVenueRow(id) {
        const ok = await showConfirm("Biztosan törlöd ezt a helyszínt?");
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/venues?id=${encodeURIComponent(id)}`,
                { method: "DELETE" },
            );
            if (!res.ok) {
                await noteAdminFailure("venues", res, "");
                return;
            }
            noteAdminAction("venues", "A helyszín törölve.");
            await fetchVenuesCatalog();
            await loadVenuesForNewEvent();
            if (editingEvent)
                await loadVenuesForEditSettlement(editingEvent.location_id);
        } catch (err) {
            await noteAdminFailure("venues", err.message || err, "");
        }
    }

    /** @param {Record<string, unknown>} ev */
    function eventDateTimeComplete(ev) {
        const sd = String(ev.start_date ?? "").trim();
        const ed = String(ev.end_date ?? "").trim();
        const st = String(ev.start_time ?? "").trim();
        const et = String(ev.end_time ?? "").trim();
        return !!(sd && ed && st && et);
    }

    /** @param {Record<string, unknown>} ev */
    function validateEventFields(ev) {
        const loc = ev.location_id;
        const locNum =
            typeof loc === "number"
                ? loc
                : parseInt(String(loc ?? ""), 10);
        if (
            loc === "" ||
            loc === null ||
            loc === undefined ||
            !Number.isFinite(locNum) ||
            locNum <= 0
        ) {
            return "Válassz települést / helyszínt.";
        }
        if (!String(ev.title ?? "").trim()) return "Az esemény címe kötelező.";
        if (!String(ev.start_date ?? "").trim()) return "A kezdő dátum kötelező.";
        if (!String(ev.end_date ?? "").trim()) return "A befejező dátum kötelező.";
        if (!String(ev.start_time ?? "").trim())
            return "A kezdő időpont (óra:perc) kötelező.";
        if (!String(ev.end_time ?? "").trim())
            return "A befejező időpont (óra:perc) kötelező.";
        const etid =
            typeof ev.event_type_id === "number"
                ? ev.event_type_id
                : parseInt(String(ev.event_type_id ?? "").trim(), 10);
        if (!Number.isFinite(etid) || etid < 1)
            return "Válassz eseménytípust.";
        return null;
    }

    $: eventsWithIncompleteDateTime = events.filter((e) => !eventDateTimeComplete(e));

    function apiNoticeTab(source) {
        const tabs = {
            quick_links: "quicklinks",
            news_feeds: "newsfeeds",
            venues: "venues",
            venue_types: "venues",
            catalog_event_types: "events",
            catalog_event_subtypes: "events",
            counties: "counties",
            historical_seats: "counties",
            settlement_location_types: "locations",
        };
        if (tabs[source]) return tabs[source];
        return ADMIN_API_LABELS[source] ? source : "";
    }

    function buildDashboardMessages(websites) {
        /** @type {{ id: string, level: string, text: string, tab?: string, action?: string, websiteId?: number }[]} */
        const messages = [];
        if (adminOffline) {
            messages.push({
                id: "api-offline",
                level: "error",
                text: "Az API nem elérhető. A felület a legutóbbi belépés alapján nyílt meg, adatok nélkül.",
            });
        }
        if (dashboardStatsError) {
            messages.push({
                id: "stats",
                level: "error",
                text: dashboardStatsError,
                action: "retry-stats",
            });
        }
        if (settingsLoadError) {
            messages.push({
                id: "settings",
                level: "error",
                text: settingsLoadError,
                tab: "settings",
                action: "open",
            });
        }
        if (listingQueueError) {
            messages.push({
                id: "queue-error",
                level: "error",
                text: listingQueueError,
                action: "retry-queue",
            });
        }
        for (const notice of apiNotices) {
            messages.push({
                id: `api-${notice.source}`,
                level: "error",
                text: notice.text,
                tab: apiNoticeTab(notice.source),
                action: "open",
            });
        }
        if (eventsWithIncompleteDateTime.length > 0) {
            messages.push({
                id: "events",
                level: "warning",
                text: `${eventsWithIncompleteDateTime.length} eseménynél hiányzik a kezdő vagy a befejező dátum és időpont.`,
                tab: "events",
                action: "open",
            });
        }
        for (const notice of browserCacheNotices) messages.push(notice);
        const siteRows = Array.isArray(websites) ? websites : [];
        const waiting =
            listingQueueUnpublished.length +
            listingQueueClaims.length +
            listingQueueSuggestions.length +
            attractionSuggestions.length +
            listingQueueMembers.length +
            siteRows.length;
        if (listingQueueFetched && !listingQueueError) {
            if (waiting > 0) {
                messages.push({
                    id: "queue",
                    level: "info",
                    text: `${listingQueueUnpublished.length} bejegyzés, ${listingQueueClaims.length} átvétel, ${listingQueueSuggestions.length} bejegyzés-javaslat, ${attractionSuggestions.length} látnivaló-javaslat, ${listingQueueMembers.length} tag és ${siteRows.length} weboldal vár jóváhagyásra.`,
                });
            } else {
                messages.push({
                    id: "queue-ok",
                    level: "success",
                    text: "Nincs jóváhagyásra váró bejegyzés, átvétel, javaslat, tag vagy weboldal.",
                });
            }
        }
        for (const site of siteRows) {
            messages.push({
                id: "website-" + site.id,
                level: "info",
                text: `${site.submitter} added ${site.domain}: ${site.title}. ${site.description}`,
                action: "website",
                websiteId: site.id,
            });
        }
        return messages;
    }

    $: dashboardMessages = buildDashboardMessages(
        listingQueueWebsites,
        adminOffline,
        dashboardStatsError,
        settingsLoadError,
        listingQueueError,
        listingQueueUnpublished,
        listingQueueClaims,
        listingQueueSuggestions,
        listingQueueMembers,
        eventsWithIncompleteDateTime,
        browserCacheNotices,
        apiNotices,
        listingQueueFetched,
        attractionSuggestions,
    );
    function fetchAttractions() {
        loadData("attractions", (d) => (attractions = d));
        loadData("/api/admin/attraction-suggestions", (d) => (attractionSuggestions = Array.isArray(d) ? d : []));
    }
    async function decideAttractionSuggestion(id, action) {
        const res = await apiCall("/api/admin/attraction-suggestions", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id, action }),
        });
        if (!res.ok) {
            await noteAdminFailure("attractions", res);
            return;
        }
        noteAdminAction(activeTab === "welcome" ? "welcome" : "attractions", action === "accept" ? "A javaslat elfogadva." : "A javaslat elutasítva.");
        fetchAttractions();
        await auth.refresh();
    }

    async function fetchCountyRegions() {
        await loadData("/api/counties", (d) => (countiesFromAPI = d));
        await loadData("/api/historical_seats", (d) => (historicalSeatsFromAPI = d));
    }

    /** One limit for every truncated label in admin tables (full value in title/tooltip). */
    const ADMIN_TABLE_PREVIEW_MAX = 20;

    function contentPreview(text, maxLen = ADMIN_TABLE_PREVIEW_MAX) {
        if (!text || !String(text).trim()) return "-";
        const t = String(text).replace(/\s+/g, " ").trim();
        return t.length > maxLen ? t.slice(0, maxLen) + "…" : t;
    }

    /** URLs and long strings in table cells use the same max length as contentPreview. */
    function formatAdminTime(value) {
        if (!value) return "-";
        const d = new Date(value);
        if (Number.isNaN(d.getTime())) return "-";
        return d.toLocaleString("hu-HU");
    }

    function urlPreview(url, maxLen = ADMIN_TABLE_PREVIEW_MAX) {
        if (!url || !String(url).trim()) return "-";
        const s = String(url).trim();
        return s.length > maxLen ? s.slice(0, maxLen) + "…" : s;
    }

    function formatLatLon(lat, lon) {
        const la = lat != null && lat !== "" ? Number(lat) : NaN;
        const lo = lon != null && lon !== "" ? Number(lon) : NaN;
        if (!Number.isFinite(la) && !Number.isFinite(lo)) return "-";
        if (la === 0 && lo === 0) return "-";
        const a = Number.isFinite(la) ? la.toFixed(4) : "-";
        const o = Number.isFinite(lo) ? lo.toFixed(4) : "-";
        return `${a}, ${o}`;
    }

    function settlementsForCountyName(countyName) {
        return locations
            .filter((l) => l.county === countyName && l.type !== "megye")
            .sort((a, b) => {
                if (a.is_county_seat && !b.is_county_seat) return -1;
                if (!a.is_county_seat && b.is_county_seat) return 1;
                const typeOrder = { municípium: 0, város: 1, község: 2, falu: 3 };
                const ta = typeOrder[a.type] ?? 9;
                const tb = typeOrder[b.type] ?? 9;
                if (ta !== tb) return ta - tb;
                return a.name.localeCompare(b.name);
            });
    }

    function countySeatDisplayName(c) {
        const seat = locations.find(
            (l) => l.county === c.name && l.type !== "megye" && l.is_county_seat,
        );
        return seat ? `${seat.name} (${seat.type})` : "-";
    }

    function countyMatchesSearch(c, q) {
        return matchesSearch(q, [
            c.name,
            c.name_ro,
            c.name_de,
            c.slug,
            countySeatDisplayName(c),
            c.content || "",
        ]);
    }

    function historicalSeatMatchesSearch(h, q) {
        return matchesSearch(q, [
            h.name,
            h.name_ro,
            h.name_de,
            h.slug,
            h.content || "",
        ]);
    }

    function startEditCounty(c) {
        editingHistoricalSeat = null;
        const seat = locations.find((l) => l.county === c.name && l.is_county_seat);
        editingCounty = {
            id: c.id,
            name: c.name ?? "",
            name_ro: c.name_ro ?? "",
            name_de: c.name_de ?? "",
            slug: c.slug ?? "",
            content: c.content ?? "",
            seat_location_id: seat ? String(seat.id) : "",
        };
    }

    function cancelEditCounty() {
        editingCounty = null;
    }

    async function saveEditingCounty() {
        const ec = editingCounty;
        if (!ec) return;
        try {
            const res = await apiCall(`/api/admin/counties`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id: ec.id,
                    name: ec.name ?? "",
                    name_ro: ec.name_ro ?? "",
                    name_de: ec.name_de ?? "",
                    slug: ec.slug ?? "",
                    content: ec.content ?? "",
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("counties", res, "Hiba (megye): ");
                return;
            }
            if (ec.seat_location_id) {
                const ok = await setCountySeat(Number(ec.seat_location_id));
                if (!ok) {
                    const msg = "A megye szövege mentve, de a megyeszékhely beállítása nem sikerült.";
                    noteAdminAction("counties", msg, false);
                    setAdminTabError(msg);
                    editingCounty = null;
                    fetchCountyRegions();
                    fetchLocations();
                    return;
                }
            }
            editingCounty = null;
            noteAdminAction("counties", "Megye mentve: " + ec.name);
            fetchCountyRegions();
            fetchLocations();
        } catch (e) {
            await noteAdminFailure("counties", e.message);
        }
    }

    function startEditHistoricalSeat(h) {
        editingCounty = null;
        editingHistoricalSeat = {
            id: h.id,
            name: h.name ?? "",
            name_ro: h.name_ro ?? "",
            name_de: h.name_de ?? "",
            slug: h.slug ?? "",
            content: h.content ?? "",
        };
    }

    function cancelEditHistoricalSeat() {
        editingHistoricalSeat = null;
    }

    async function saveEditingHistoricalSeat() {
        const h = editingHistoricalSeat;
        if (!h) return;
        try {
            const res = await apiCall(`/api/admin/historical_seats`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    id: h.id,
                    name: h.name ?? "",
                    name_ro: h.name_ro ?? "",
                    name_de: h.name_de ?? "",
                    slug: h.slug ?? "",
                    content: h.content ?? "",
                }),
            });
            if (!res.ok) {
                await noteAdminFailure("historical_seats", res, "Hiba (szék): ");
                return;
            }
            editingHistoricalSeat = null;
            noteAdminAction("historical_seats", "Szék mentve: " + h.name);
            fetchCountyRegions();
        } catch (e) {
            await noteAdminFailure("historical_seats", e.message);
        }
    }

    // generic create
    async function createRecord(endpoint, data, reloadFunc, resetFormFunc, noticeTable) {
        const table = noticeTable || noticeTableFor(endpoint);
        const subject = actionSubject(data);
        try {
            const res = await apiCall(`/api/admin/${endpoint}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(data),
            });
            if (res.ok) {
                noteAdminAction(
                    table,
                    subject ? `„${subject}” hozzáadva.` : "A hozzáadás sikerült.",
                );
                reloadFunc();
                resetFormFunc();
            } else {
                await noteAdminFailure(table, res);
            }
        } catch (e) {
            console.error(e);
            await noteAdminFailure(table, e && e.message ? e.message : String(e));
        }
    }

    // generic update (PUT)
    async function updateRecord(endpoint, data, reloadFunc, noticeTable) {
        const table = noticeTable || noticeTableFor(endpoint);
        const subject = actionSubject(data);
        try {
            const res = await apiCall(`/api/admin/${endpoint}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(data),
            });
            if (res.ok) {
                noteAdminAction(
                    table,
                    subject ? `„${subject}” mentve.` : "A mentés sikerült.",
                );
                reloadFunc();
            } else {
                await noteAdminFailure(table, res, "Mentési hiba: ");
            }
        } catch (e) {
            console.error(e);
            await noteAdminFailure(table, e && e.message ? e.message : String(e));
        }
    }

    // generic delete
    async function deleteRecord(endpoint, id, reloadFunc, noticeTable) {
        const table = noticeTable || noticeTableFor(endpoint);
        const ok = await showConfirm("Biztosan törölni szeretnéd?");
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/${endpoint}?id=${id}`,
                { method: "DELETE" },
            );
            if (res.ok) {
                noteAdminAction(table, "A törlés sikerült.");
                reloadFunc();
            } else {
                await noteAdminFailure(table, res);
            }
        } catch (e) {
            console.error(e);
            await noteAdminFailure(table, e && e.message ? e.message : String(e));
        }
    }

    /** Normalize YYYY-MM-DD from date input or API (may include time). */
    function normalizeYmdInput(v) {
        const s = String(v ?? "").trim();
        return s.length >= 10 ? s.slice(0, 10) : s;
    }

    // specific creates
    function submitLink(e) {
        e.preventDefault();
        createRecord(
            "quick_links",
            newLink,
            fetchQuickLinks,
            () =>
                (newLink = {
                    title: "",
                    url: "",
                    bg_color: "var(--card-bg)",
                }),
        );
    }
    function submitNews(e) {
        e.preventDefault();
        createRecord(
            "news_feeds",
            newNews,
            fetchNewsFeeds,
            () =>
                (newNews = {
                    title: "",
                    feed_url: "",
                    bg_color: "#ffebd6",
                }),
        );
    }

    /**
     * Ask the admin API to fetch the stored feed and report whether it is a
     * usable RSS feed. The public site refreshes its news by itself; this
     * only tells the admin that a feed works.
     */
    async function checkFeed(feed) {
        loadingFeeds.add(feed.id);
        loadingFeeds = new Set(loadingFeeds);

        try {
            const res = await apiCall(`/api/admin/news_feeds/check?id=${feed.id}`);
            if (!res.ok) {
                await noteAdminFailure("newsfeeds", res, "Ellenőrzési hiba: ");
                return;
            }
            const result = await res.json();
            if (result.ok) {
                feedTimestamps[feed.feed_url] = Date.now();
                localStorage.setItem(
                    "news_feed_timestamps",
                    JSON.stringify(feedTimestamps),
                );
                feedTimestamps = { ...feedTimestamps };
                noteAdminAction("newsfeeds", `A hírfolyam rendben: ${result.items} hír.`);
            } else {
                noteAdminAction("newsfeeds", `A hírfolyam hibás: ${result.error}`, false);
            }
        } catch (e) {
            console.error("Feed ellenőrzési hiba:", e);
            await noteAdminFailure("newsfeeds", e && e.message ? e.message : String(e), "Ellenőrzési hiba: ");
        } finally {
            loadingFeeds.delete(feed.id);
            loadingFeeds = new Set(loadingFeeds);
        }
    }
    function submitLocation(e) {
        e.preventDefault();
        createRecord(
            "locations",
            newLocation,
            fetchLocations,
            () =>
                (newLocation = {
                    name: "",
                    name_ro: "",
                    name_de: "",
                    county: "",
                    type: "",
                    post_code: "",
                    coordinates: "",
                    population: "",
                    area: "",
                    crest: "",
                    parent_id: null,
                }),
        );
    }
    /**
     * @param {File} file
     * @param {boolean} isEdit
     */
    async function uploadEventFeaturedImage(file, isEdit) {
        if (!file) return;
        const fd = new FormData();
        fd.append("file", file);
        if (isEdit && editingEvent?.id) {
            fd.append("event_id", String(editingEvent.id));
        }
        try {
            const res = await apiCall(`/api/admin/event-images`, {
                method: "POST",
                body: fd,
            });
            if (!res.ok) {
                await noteAdminFailure("events", res, "Feltöltés sikertelen. ");
                return;
            }
            const data = await res.json();
            const url = data.url || "";
            if (isEdit) {
                editingEvent = { ...editingEvent, featured_image: url };
            } else {
                newEvent = { ...newEvent, featured_image: url };
            }
            noteAdminAction("events", "A kép feltöltve.");
        } catch (err) {
            await noteAdminFailure("events", err.message, "Feltöltés hiba: ");
        }
    }

    async function submitEvent(e) {
        e.preventDefault();
        const rawLid = newEvent.location_id;
        const lid =
            typeof rawLid === "number" && Number.isFinite(rawLid)
                ? rawLid
                : parseInt(String(rawLid ?? "").trim(), 10);
        const dv = parseInt(String(newEvent.default_venue_id || ""), 10);
        const etid = parseInt(String(newEvent.event_type_id || ""), 10);
        const stRaw = newEvent.event_subtype_id;
        let event_subtype_id = null;
        if (stRaw !== "" && stRaw != null && String(stRaw).trim() !== "") {
            const s = parseInt(String(stRaw), 10);
            if (Number.isFinite(s) && s > 0) event_subtype_id = s;
        }
        const payload = {
            location_id: Number.isFinite(lid) && lid > 0 ? lid : 0,
            default_venue_id:
                Number.isFinite(dv) && dv > 0 ? dv : null,
            title: String(newEvent.title ?? "").trim(),
            description: String(newEvent.description ?? ""),
            featured_image: String(newEvent.featured_image ?? "").trim(),
            featured_image_copyright: String(newEvent.featured_image_copyright ?? "").trim(),
            attraction_id: positiveIdOrNull(newEvent.attraction_id),
            start_date: normalizeYmdInput(newEvent.start_date),
            end_date: normalizeYmdInput(newEvent.end_date),
            start_time: String(newEvent.start_time ?? "").trim(),
            end_time: String(newEvent.end_time ?? "").trim(),
            event_type_id: etid,
            event_subtype_id,
            access_type: String(newEvent.access_type || "public"),
            organizer: String(newEvent.organizer ?? "").trim(),
            entry_price: String(newEvent.entry_price ?? "").trim(),
        };
        const err = validateEventFields(payload);
        if (err) {
            noteAdminAction("events", err, false);
            setAdminTabError(err);
            return;
        }
        createRecord(
            "events",
            payload,
            fetchEvents,
            () =>
                (newEvent = {
                    location_id: "",
                    default_venue_id: "",
                    title: "",
                    description: "",
                    start_date: "",
                    start_time: "",
                    end_date: "",
                    end_time: "",
                    event_type_id:
                        catalogEventTypes.find((t) => t.slug === "cultural")
                            ?.id ?? catalogEventTypes[0]?.id ?? "",
                    event_subtype_id: "",
                    access_type: "public",
                    organizer: "",
                    featured_image: "",
                    featured_image_copyright: "",
                    entry_price: "",
                    attraction_id: "",
                }),
        );
    }

    function submitNewOrganizer(e) {
        e.preventDefault();
        const payload = {
            ...newOrganizerEntry,
            location_id: parseInt(newOrganizerEntry.location_id) || 0,
            category_id: newOrganizerEntry.category_id
                ? parseInt(newOrganizerEntry.category_id)
                : null,
            category_ids: [
                parseInt(newOrganizerEntry.category_id),
                ...(newOrganizerEntry.category_extra || []),
            ].filter((id) => id > 0),
            tags: tagsFromStr(newOrganizerEntry.tags),
        };
        createRecord("entries", payload, fetchEntries, () => {
            newEvent.organizer = newOrganizerEntry.name;
            orgQuery = newOrganizerEntry.name;
            newOrganizerModalVisible = false;
            newOrganizerEntry = {
                location_id: "",
                category_id: "",
                category_extra: [],
                name: "",
                slug: "",
                url: "",
                phone: "",
                address: "",
                notes: "",
                type: "",
                languages: ["HU"],
                tags: "",
            };
        }, "events");
    }

    // --- Location edit helpers ---
    function startEditLocation(loc) {
        editingLocation = { ...loc };
    }
    function cancelEditLocation() {
        editingLocation = null;
    }
    async function saveEditLocation() {
        if (!editingLocation) return;
        await updateRecord("locations", editingLocation, fetchLocations);
        cancelEditLocation();
    }

    function getEntryCategoryParentName(cat) {
        if (!cat?.parent_id) return "-";
        const parent = entryCategories.find((row) => row.id === cat.parent_id);
        return parent ? parent.name : "-";
    }

    function entryCategoryMoveTargets(cat) {
        if (!cat?.parent_id) {
            return entryCategoryParents.filter((row) => row.id !== cat.id);
        }
        return entryCategoryChildren.filter((row) => row.id !== cat.id);
    }

    // --- Submit entry category ---
    function submitEntryCategory(e) {
        e.preventDefault();
        const payload = { name: String(newEntryCategory.name ?? "").trim() };
        if (newEntryCategory.parent_id) {
            payload.parent_id = parseInt(newEntryCategory.parent_id, 10);
        }
        createRecord(
            "entry_categories",
            payload,
            fetchEntryCategories,
            () => (newEntryCategory = { name: "", parent_id: "" }),
        );
    }

    async function deleteEntryCategory(cat) {
        const table = "entry_categories";
        const ok = await showConfirm("Biztosan törölni szeretnéd?");
        if (!ok) return;
        try {
            const res = await apiCall(`/api/admin/entry_categories?id=${cat.id}`, {
                method: "DELETE",
            });
            if (res.ok) {
                categoryDeleteMove = null;
                noteAdminAction(table, "A törlés sikerült.");
                fetchEntryCategories();
                return;
            }
            if (res.status === 409) {
                const message = (await res.text()).trim();
                const targets = entryCategoryMoveTargets(cat);
                categoryDeleteMove = {
                    id: cat.id,
                    message,
                    moveTo: targets.length ? String(targets[0].id) : "",
                };
                noteAdminAction(table, message, false);
                return;
            }
            await noteAdminFailure(table, res);
        } catch (e) {
            console.error(e);
            await noteAdminFailure(table, e && e.message ? e.message : String(e));
        }
    }

    async function deleteEntryCategoryWithMove(cat) {
        if (!categoryDeleteMove || categoryDeleteMove.id !== cat.id) return;
        const moveTo = parseInt(categoryDeleteMove.moveTo, 10);
        if (!moveTo) return;
        const table = "entry_categories";
        try {
            const res = await apiCall(
                `/api/admin/entry_categories?id=${cat.id}&move_to=${moveTo}`,
                { method: "DELETE" },
            );
            if (res.ok) {
                categoryDeleteMove = null;
                noteAdminAction(table, "A törlés sikerült.");
                fetchEntryCategories();
                return;
            }
            if (res.status === 409) {
                const message = (await res.text()).trim();
                categoryDeleteMove = { ...categoryDeleteMove, message };
                noteAdminAction(table, message, false);
                return;
            }
            await noteAdminFailure(table, res);
        } catch (e) {
            console.error(e);
            await noteAdminFailure(table, e && e.message ? e.message : String(e));
        }
    }

    // --- Submit entry type ---
    function submitEntryType(e) {
        e.preventDefault();
        createRecord(
            "entry_types",
            newEntryType,
            fetchEntryTypes,
            () => (newEntryType = { name: "" }),
        );
    }

    // --- Inline edit helpers for categories ---
    async function startEditCategory(cat) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        editingCategory = { ...cat };
    }
    function cancelEditCategory() {
        editingCategory = null;
    }
    async function saveEditCategory() {
        if (!editingCategory) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        await updateRecord(
            "entry_categories",
            editingCategory,
            fetchEntryCategories,
        );
        editingCategory = null;
    }

    // --- Inline edit helpers for types ---
    async function startEditType(et) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        editingType = { ...et };
    }
    function cancelEditType() {
        editingType = null;
    }
    async function saveEditType() {
        if (!editingType) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        await updateRecord("entry_types", editingType, fetchEntryTypes);
        editingType = null;
    }

    // --- Inline edit helpers for quick links ---
    async function startEditLink(ql) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        editingLink = { ...ql };
    }
    function cancelEditLink() {
        editingLink = null;
    }
    async function saveEditLink() {
        if (!editingLink) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        await updateRecord("quick_links", editingLink, fetchQuickLinks);
        editingLink = null;
    }

    // --- Inline edit helpers for news feeds ---
    async function startEditNews(nf) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        editingNews = { ...nf };
    }
    function cancelEditNews() {
        editingNews = null;
    }
    async function saveEditNews() {
        if (!editingNews) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        await updateRecord("news_feeds", editingNews, fetchNewsFeeds);
        editingNews = null;
    }
    async function startEditEvent(ev) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        const st = ev.start_time ? String(ev.start_time) : "";
        const et = ev.end_time ? String(ev.end_time) : "";
        editingEvent = {
            ...ev,
            entry_price: ev.entry_price != null ? String(ev.entry_price) : "",
            featured_image: ev.featured_image || "",
            featured_image_copyright: ev.featured_image_copyright || "",
            attraction_id:
                ev.attraction_id != null && ev.attraction_id !== ""
                    ? String(ev.attraction_id)
                    : "",
            default_venue_id:
                ev.default_venue_id != null && ev.default_venue_id !== ""
                    ? String(ev.default_venue_id)
                    : "",
            event_type_id:
                ev.event_type_id != null && ev.event_type_id !== ""
                    ? String(ev.event_type_id)
                    : "",
            event_subtype_id:
                ev.event_subtype_id != null && ev.event_subtype_id !== ""
                    ? String(ev.event_subtype_id)
                    : "",
            access_type: ev.access_type || "public",
            start_time: st.length >= 5 ? st.slice(0, 5) : "",
            end_time: et.length >= 5 ? et.slice(0, 5) : "",
        };
        orgEditQuery = ev.organizer || "";
        await loadVenuesForEditSettlement(ev.location_id);
        await loadScheduleForEditing(ev.id);
    }
    function cancelEditEvent() {
        editingEvent = null;
        scheduleDraftDays = [];
    }

    async function loadScheduleForEditing(eventId) {
        try {
            const res = await apiCall(`/api/admin/events/schedule?event_id=${eventId}`,
            );
            if (!res.ok) {
                scheduleDraftDays = [];
                return;
            }
            const data = await res.json();
            scheduleDraftDays = (data.days || []).map((d) => ({
                schedule_date: d.schedule_date,
                notes: d.notes || "",
                activities: (d.activities || []).map((a) => ({
                    activity_type: a.activity_type || "other",
                    starts_at: a.starts_at
                        ? String(a.starts_at).slice(0, 5)
                        : "",
                    ends_at: a.ends_at ? String(a.ends_at).slice(0, 5) : "",
                    venue_id:
                        a.venue_id != null && a.venue_id !== ""
                            ? String(a.venue_id)
                            : "",
                    title: a.title || "",
                    description: a.description || "",
                })),
            }));
        } catch (e) {
            console.error(e);
            scheduleDraftDays = [];
        }
    }

    async function generateScheduleDaysFromEvent() {
        if (!editingEvent) return;
        const ok = await showConfirm(
            "A jelenlegi napi program törlődik, és a kezdő–befejező dátum közötti minden nap üres programmal kerül be. Folytatja?",
        );
        if (!ok) return;
        const s = editingEvent.start_date?.split("T")[0];
        const e = editingEvent.end_date?.split("T")[0];
        if (!s || !e) {
            noteAdminAction("events", "Előbb állítsa be a kezdő és befejező dátumot.", false);
            return;
        }
        const out = [];
        const cur = new Date(s + "T12:00:00");
        const end = new Date(e + "T12:00:00");
        while (cur <= end) {
            out.push({
                schedule_date: cur.toISOString().slice(0, 10),
                notes: "",
                activities: [],
            });
            cur.setDate(cur.getDate() + 1);
        }
        scheduleDraftDays = out;
    }

    function addScheduleDayRow() {
        scheduleDraftDays = [
            ...scheduleDraftDays,
            { schedule_date: "", notes: "", activities: [] },
        ];
    }

    function removeScheduleDayRow(i) {
        scheduleDraftDays = scheduleDraftDays.filter((_, j) => j !== i);
    }

    function addScheduleActivity(dayIndex) {
        const d = scheduleDraftDays[dayIndex];
        if (!d) return;
        d.activities = [
            ...d.activities,
            {
                activity_type: "match",
                starts_at: "",
                ends_at: "",
                venue_id: "",
                title: "",
                description: "",
            },
        ];
        scheduleDraftDays = [...scheduleDraftDays];
    }

    function removeScheduleActivity(dayIndex, actIndex) {
        const d = scheduleDraftDays[dayIndex];
        if (!d) return;
        d.activities = d.activities.filter((_, j) => j !== actIndex);
        scheduleDraftDays = [...scheduleDraftDays];
    }

    async function saveEventSchedule() {
        if (!editingEvent) return;
        try {
            const body = {
                event_id: editingEvent.id,
                days: scheduleDraftDays
                    .filter((d) => d.schedule_date && String(d.schedule_date).trim())
                    .map((d) => ({
                        schedule_date: d.schedule_date,
                        notes: "",
                        activities: (d.activities || [])
                            .filter((a) => a.title && String(a.title).trim())
                            .map((a, ai) => {
                                const vid = parseInt(
                                    String(a.venue_id || ""),
                                    10,
                                );
                                return {
                                    activity_type: a.activity_type || "other",
                                    starts_at: a.starts_at || "",
                                    ends_at: a.ends_at || "",
                                    venue_id:
                                        Number.isFinite(vid) && vid > 0
                                            ? vid
                                            : null,
                                    title: a.title.trim(),
                                    description: a.description || "",
                                    sort_order: ai,
                                };
                            }),
                    })),
            };
            const res = await apiCall(`/api/admin/events/schedule`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) {
                await noteAdminFailure("events", res, "Program mentése sikertelen: ");
                return;
            }
            noteAdminAction("events", "Napi program elmentve.");
            await loadScheduleForEditing(editingEvent.id);
        } catch (err) {
            await noteAdminFailure("events", err.message);
        }
    }
    async function saveEditEvent() {
        if (!editingEvent) return;
        const err = validateEventFields(editingEvent);
        if (err) {
            noteAdminAction("events", err, false);
            setAdminTabError(err);
            return;
        }
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        const dv = parseInt(
            String(editingEvent.default_venue_id || ""),
            10,
        );
        const rawLid = editingEvent.location_id;
        const lid =
            typeof rawLid === "number" && Number.isFinite(rawLid)
                ? rawLid
                : parseInt(String(rawLid ?? "").trim(), 10);
        const etidE = parseInt(String(editingEvent.event_type_id || ""), 10);
        const stRawE = editingEvent.event_subtype_id;
        let event_subtype_id_e = null;
        if (
            stRawE !== "" &&
            stRawE != null &&
            String(stRawE).trim() !== ""
        ) {
            const s = parseInt(String(stRawE), 10);
            if (Number.isFinite(s) && s > 0) event_subtype_id_e = s;
        }
        const payload = {
            id: editingEvent.id,
            location_id: Number.isFinite(lid) && lid > 0 ? lid : null,
            default_venue_id:
                Number.isFinite(dv) && dv > 0 ? dv : null,
            title: String(editingEvent.title ?? "").trim(),
            description: String(editingEvent.description ?? ""),
            featured_image: String(editingEvent.featured_image ?? "").trim(),
            featured_image_copyright: String(editingEvent.featured_image_copyright ?? "").trim(),
            attraction_id: positiveIdOrNull(editingEvent.attraction_id),
            start_date: normalizeYmdInput(editingEvent.start_date),
            end_date: normalizeYmdInput(editingEvent.end_date),
            start_time: String(editingEvent.start_time ?? "").trim(),
            end_time: String(editingEvent.end_time ?? "").trim(),
            event_type_id: etidE,
            event_subtype_id: event_subtype_id_e,
            access_type: String(editingEvent.access_type || "public"),
            organizer: String(editingEvent.organizer ?? "").trim(),
            entry_price: String(editingEvent.entry_price ?? "").trim(),
        };
        await updateRecord("events", payload, fetchEvents);
        editingEvent = null;
    }
    // --- Tag helpers ---
    function tagsFromStr(str) {
        return (str || "")
            .split(/[\s,]+/)
            .map((t) => t.replace(/^#/, "").trim())
            .filter(Boolean);
    }
    function tagsToStr(arr) {
        return (arr || []).map((t) => "#" + t).join(" ");
    }
    function getLocationName(id) {
        if (id == null || id === "") return "-";
        const l = locations.find((loc) => loc.id === id);
        return l ? `${l.name}${l.county ? " (" + l.county + ")" : ""}` : id;
    }
    function getCategoryName(id) {
        const c = entryCategories.find((cat) => cat.id === id);
        return c ? c.name : "-";
    }

    function adminOffersDelivery(entry) {
        const categoryName =
            String(entry?.category ?? "").trim() ||
            getCategoryName(entry?.category_id);
        return offersDelivery({ name: entry?.name, category: categoryName });
    }

    // --- Submit entry (Create) ---
    function submitEntry(e) {
        e.preventDefault();
        const payload = {
            ...newEntry,
            location_id: parseInt(newEntry.location_id) || 0,
            category_id: newEntry.category_id
                ? parseInt(newEntry.category_id)
                : null,
            category_ids: [
                parseInt(newEntry.category_id),
                ...(newEntry.category_extra || []),
            ].filter((id) => id > 0),
            tags: tagsFromStr(newEntry.tags),
            verified: Boolean(newEntry.verified),
            hours: normalizeHours(newEntry.hours),
            delivery_hours: normalizeHours(newEntry.delivery_hours),
            hours_enabled: Boolean(newEntry.hours_enabled),
            delivery_enabled: adminOffersDelivery(newEntry)
                ? Boolean(newEntry.delivery_enabled)
                : false,
            photos: normalizePhotos(newEntry.photos),
        };
        createRecord(
            "entries",
            payload,
            fetchEntries,
            () =>
                (newEntry = {
                    location_id: "",
                    category_id: "",
                    category_extra: [],
                    name: "",
                    url: "",
                    phone: "",
                    address: "",
                    notes: "",
                    type: "",
                    languages: ["HU"],
                    tags: "",
                    verified: false,
                    hours: emptyWeekHours(),
                    delivery_hours: emptyWeekHours(),
                    hours_enabled: false,
                    delivery_enabled: false,
                    photos: emptyPhotos(),
                }),
        );
    }

    // --- Edit modal ---
    async function openEdit(entry) {
        const ok = await showConfirm("Biztosan szerkeszteni szeretné?");
        if (!ok) return;
        const categoryIDs = (Array.isArray(entry.category_ids) ? entry.category_ids : [])
            .map((id) => Number(id))
            .filter((id) => id > 0);
        const categoryID = Number(entry.category_id) || categoryIDs[0] || "";
        editingEntry = {
            ...entry,
            category_id: categoryID,
            category_extra: categoryIDs.filter((id) => id !== Number(categoryID)),
            languages: entry.languages ? [...entry.languages] : ["HU"],
            verified: Boolean(entry.verified),
            hours: normalizeHours(entry.hours),
            delivery_hours: normalizeHours(entry.delivery_hours),
            hours_enabled: Boolean(entry.hours_enabled),
            delivery_enabled: Boolean(entry.delivery_enabled),
            photos: normalizePhotos(entry.photos),
        };
        editTagsStr = tagsToStr(entry.tags);
    }
    function closeEdit() {
        editingEntry = null;
        editTagsStr = "";
    }
    async function saveEdit() {
        if (!editingEntry) return;
        const ok = await showConfirm("Biztosan menteni szeretné a módosítást?");
        if (!ok) return;
        const payload = {
            ...editingEntry,
            location_id: parseInt(editingEntry.location_id) || 0,
            category_id: editingEntry.category_id
                ? parseInt(editingEntry.category_id)
                : null,
            category_ids: [
                parseInt(editingEntry.category_id),
                ...(editingEntry.category_extra || []),
            ].filter((id) => id > 0),
            tags: tagsFromStr(editTagsStr),
            verified: Boolean(editingEntry.verified),
            hours: normalizeHours(editingEntry.hours),
            delivery_hours: normalizeHours(editingEntry.delivery_hours),
            hours_enabled: Boolean(editingEntry.hours_enabled),
            delivery_enabled: adminOffersDelivery(editingEntry)
                ? Boolean(editingEntry.delivery_enabled)
                : false,
            photos: normalizePhotos(editingEntry.photos),
        };
        await updateRecord("entries", payload, fetchEntries);
        closeEdit();
    }

    // --- Attractions ---
    function submitNewAttraction(e) {
        e.preventDefault();
        const imgs = attractionImagePayload(newAttraction.images, newAttraction.image_copyrights);
        createRecord(
            "attractions",
            {
                county_slug: newAttraction.county_slug,
                name: newAttraction.name,
                name_ro: newAttraction.name_ro || "",
                name_de: newAttraction.name_de || "",
                slug: newAttraction.slug || "",
                description: newAttraction.description || "",
                latitude: parseFloat(newAttraction.latitude) || 0,
                longitude: parseFloat(newAttraction.longitude) || 0,
                featured_image: newAttraction.featured_image || "",
                featured_image_copyright: newAttraction.featured_image_copyright || "",
                content: newAttraction.content || "",
                activities: activityLines(newAttraction.activities),
                prohibitions: activityLines(newAttraction.prohibitions),
                elevation_m: optionalNumber(newAttraction.elevation_m),
                area_km2: optionalNumber(newAttraction.area_km2),
                depth_m: optionalNumber(newAttraction.depth_m),
                images: imgs,
            },
            fetchAttractions,
            () =>
                (newAttraction = {
                    county_slug: "hargita",
                    name: "",
                    name_ro: "",
                    name_de: "",
                    slug: "",
                    description: "",
                    latitude: "",
                    longitude: "",
                    featured_image: "",
                    featured_image_copyright: "",
                    content: "",
                    activities: "",
                    prohibitions: "",
                    elevation_m: "",
                    area_km2: "",
                    depth_m: "",
                    images: "",
                    image_copyrights: "",
                }),
        );
    }
    function activityLines(text) {
        return String(text || "")
            .split("\n")
            .map((line) => line.trim())
            .filter(Boolean);
    }
    function attractionImagePayload(urlsText, copyrightsText) {
        const urls = String(urlsText || "").split("\n");
        const credits = String(copyrightsText || "").split("\n");
        const out = [];
        const count = Math.max(urls.length, credits.length);
        for (let i = 0; i < count; i++) {
            const url = (urls[i] || "").trim();
            if (!url) continue;
            out.push({
                url,
                copyright: (credits[i] || "").trim().replace(/^©\s*/, ""),
            });
        }
        return out;
    }
    function attractionImageLines(images) {
        const rows = Array.isArray(images) ? images : [];
        return {
            images: rows
                .map((img) => (typeof img === "string" ? img : img?.url || ""))
                .join("\n"),
            image_copyrights: rows
                .map((img) => (typeof img === "string" ? "" : img?.copyright || ""))
                .join("\n"),
        };
    }
    function positiveIdOrNull(value) {
        const id = parseInt(String(value ?? "").trim(), 10);
        return Number.isFinite(id) && id > 0 ? id : null;
    }
    function openEditAttraction(att) {
        const lines = attractionImageLines(att.images);
        editingAttraction = {
            id: att.id,
            county_slug: att.county_slug,
            name: att.name,
            name_ro: att.name_ro || "",
            name_de: att.name_de || "",
            slug: att.slug,
            description: att.description || "",
            latitude: att.latitude ? String(att.latitude) : "",
            longitude: att.longitude ? String(att.longitude) : "",
            featured_image: att.featured_image || "",
            featured_image_copyright: att.featured_image_copyright || "",
            content: att.content || "",
            activities: Array.isArray(att.activities) ? att.activities.join("\n") : "",
            prohibitions: Array.isArray(att.prohibitions) ? att.prohibitions.join("\n") : "",
            elevation_m: factText(att.elevation_m),
            area_km2: factText(att.area_km2),
            depth_m: factText(att.depth_m),
            images: lines.images,
            image_copyrights: lines.image_copyrights,
        };
    }
    /** An attraction fact for its text field: empty when not set, decimal comma. */
    function factText(/** @type {number | null | undefined} */ v) {
        return v == null ? "" : String(v).replace(".", ",");
    }
    function cancelEditAttraction() {
        editingAttraction = null;
    }
    async function saveEditAttraction(e) {
        e.preventDefault();
        if (!editingAttraction) return;
        const imgs = attractionImagePayload(
            editingAttraction.images,
            editingAttraction.image_copyrights,
        );
        await updateRecord(
            "attractions",
            {
                id: editingAttraction.id,
                county_slug: editingAttraction.county_slug,
                name: editingAttraction.name,
                name_ro: editingAttraction.name_ro || "",
                name_de: editingAttraction.name_de || "",
                slug: editingAttraction.slug || "",
                description: editingAttraction.description || "",
                latitude: parseFloat(editingAttraction.latitude) || 0,
                longitude: parseFloat(editingAttraction.longitude) || 0,
                featured_image: editingAttraction.featured_image || "",
                featured_image_copyright: editingAttraction.featured_image_copyright || "",
                content: editingAttraction.content || "",
                activities: activityLines(editingAttraction.activities),
                prohibitions: activityLines(editingAttraction.prohibitions),
                elevation_m: optionalNumber(editingAttraction.elevation_m),
                area_km2: optionalNumber(editingAttraction.area_km2),
                depth_m: optionalNumber(editingAttraction.depth_m),
                images: imgs,
            },
            fetchAttractions,
        );
        cancelEditAttraction();
    }
    async function deleteAttraction(id) {
        if (!(await showConfirm("Biztosan törölni szeretnéd ezt a látnivalót?"))) return;
        try {
            const res = await apiCall(`/api/admin/attractions?id=${id}`, { method: "DELETE" });
            if (res.ok) {
                noteAdminAction("attractions", "A látnivaló törölve.");
                fetchAttractions();
            } else {
                await noteAdminFailure("attractions", res);
            }
        } catch (e) {
            console.error(e);
            await noteAdminFailure("attractions", e && e.message ? e.message : String(e));
        }
    }
</script>

<svelte:head>
    <title>Lámsza - Adminisztráció</title>
</svelte:head>

{#snippet adminNotice(table)}
    {#if adminActionNotice && adminActionNotice.table === table && adminActionNotice.tab === activeTab}
        <div
            class="info-box admin-action-notice {adminActionNotice.ok ? 'success' : 'error'}"
            role={adminActionNotice.ok ? "status" : "alert"}
        >
            <p>{adminActionNotice.text}</p>
        </div>
    {/if}
{/snippet}


<AdminShell
    bind:this={shell}
    title={adminPageHead.title}
    greeting={adminPageHead.greeting}
    nav={ADMIN_NAV}
    active={activeTab}
    external={{ href: lamszaOrigin, title: "Lámsza megnyitása új lapon" }}
    onSelect={loadTab}
    onReady={onShellReady}
>
                {#if activeTab === "welcome"}
                    <section class="admin-subsection" aria-labelledby="admin-messages-title">
                        <h3 id="admin-messages-title">Üzenetek</h3>
                        {@render adminNotice("welcome")}
                        {#each dashboardMessages as msg (msg.id)}
                            <div
                                class="info-box {msg.level}"
                                role={msg.level === "error" || msg.level === "warning" ? "alert" : "status"}
                            >
                                <p>{msg.text}</p>
                                {#if msg.action}
                                    <p>
                                        {#if msg.action === "retry-stats"}
                                            <button type="button" class="btn btn-sm" on:click={fetchDashboardStats}>Újra</button>
                                        {:else if msg.action === "retry-queue"}
                                            <button type="button" class="btn btn-sm" on:click={fetchListingQueue}>Újra</button>
                                        {:else if msg.action === "open" && msg.tab}
                                            <button type="button" class="btn btn-sm" on:click={() => goToAdminTab(msg.tab)}>Megnyitás</button>
                                        {:else if msg.action === "cache-refresh"}
                                            <button
                                                type="button"
                                                class="btn btn-sm"
                                                disabled
                                                title="A böngésző-mentés törlése és újratöltése később lesz bekötve."
                                            >Frissítés</button>
                                        {:else if msg.action === "website"}
                                            <CategoryMultiSelect
                                                parents={entryCategoryParents}
                                                children={entryCategoryChildren}
                                                bind:primary={websiteApproveCategory[msg.websiteId]}
                                                bind:extra={websiteApproveExtra[msg.websiteId]}
                                                primaryInputId={"website_cat_" + msg.websiteId}
                                            />
                                            <button
                                                type="button"
                                                class="btn btn-sm"
                                                on:click={() =>
                                                    reviewWebsite(
                                                        msg.websiteId,
                                                        "approve",
                                                        parseInt(
                                                            websiteApproveCategory[
                                                                msg.websiteId
                                                            ] || "0",
                                                            10,
                                                        ),
                                                        websiteApproveExtra[msg.websiteId] || [],
                                                    )}>Jóváhagyás</button
                                            >
                                            <button type="button" class="btn btn-sm" on:click={() => reviewWebsite(msg.websiteId, "reject")}>Elutasítás</button>
                                            <button type="button" class="btn btn-sm" on:click={() => reviewWebsite(msg.websiteId, "ban")}>Felhasználó tiltása</button>
                                        {/if}
                                    </p>
                                {/if}
                            </div>
                        {/each}
                        {#if listingQueueFetched && !listingQueueError && (listingQueueUnpublished.length > 0 || listingQueueClaims.length > 0 || listingQueueSuggestions.length > 0 || listingQueueMembers.length > 0 || attractionSuggestions.length > 0)}
                            {#if listingQueueUnpublished.length > 0}
                                <h4>Közzétételre váró bejegyzések</h4>
                                <div class="admin-table-wrapper">
                                    <table class="admin-table">
                                        <thead>
                                            <tr>
                                                <th>Név</th>
                                                <th>Tulajdonos</th>
                                                <th class="admin-table-col--action">Művelet</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each listingQueueUnpublished as row}
                                                <tr>
                                                    <td>{row.name}</td>
                                                    <td>{row.owner_email || "-"}</td>
                                                    <td class="admin-table-col--action">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={() => publishListingQueueEntry(row.id)}
                                                        >Közzététel</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}
                            {#if listingQueueClaims.length > 0}
                                <h4>Átvételi kérések</h4>
                                <div class="admin-table-wrapper">
                                    <table class="admin-table">
                                        <thead>
                                            <tr>
                                                <th>Bejegyzés</th>
                                                <th>Felhasználó</th>
                                                <th class="admin-table-col--action">Művelet</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each listingQueueClaims as row}
                                                <tr>
                                                    <td>{row.entry_name}</td>
                                                    <td>{row.email}</td>
                                                    <td class="admin-table-col--action">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={() => decideListingQueueClaim(row.entry_id, row.user_id, "accept")}
                                                        >Elfogad</button>
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm"
                                                            on:click={() => decideListingQueueClaim(row.entry_id, row.user_id, "deny")}
                                                        >Elutasít</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}
                            {#if listingQueueMembers.length > 0}
                                <h4>Tag-jelentkezések</h4>
                                <div class="admin-table-wrapper">
                                    <table class="admin-table">
                                        <thead>
                                            <tr>
                                                <th>Bejegyzés</th>
                                                <th>Felhasználó</th>
                                                <th class="admin-table-col--action">Művelet</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each listingQueueMembers as row}
                                                <tr>
                                                    <td>{row.entry_name}</td>
                                                    <td>{row.email}</td>
                                                    <td class="admin-table-col--action">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={() => approveListingQueueMember(row.entry_id, row.user_id)}
                                                        >Elfogadás</button>
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm"
                                                            on:click={() => rejectListingQueueMember(row.entry_id, row.user_id)}
                                                        >Elutasítás</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}
                            {#if listingQueueSuggestions.length > 0}
                                <h4>Javaslatok</h4>
                                <div class="admin-table-wrapper">
                                    <table class="admin-table">
                                        <thead>
                                            <tr>
                                                <th>Bejegyzés</th>
                                                <th>Felhasználó</th>
                                                <th>Változások</th>
                                                <th class="admin-table-col--action">Művelet</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each listingQueueSuggestions as row}
                                                <tr>
                                                    <td>{row.entry_name}</td>
                                                    <td>{row.email}</td>
                                                    <td>
                                                        {#each Object.entries(row.changes || {}) as [key, value] (key)}
                                                            <div>
                                                                <strong>{SUGGESTION_FIELD_LABELS[key] || key}:</strong>
                                                                {suggestionValueText(value)}
                                                            </div>
                                                        {/each}
                                                        {#if row.note}
                                                            <div><strong>Megjegyzés:</strong> {row.note}</div>
                                                        {/if}
                                                    </td>
                                                    <td class="admin-table-col--action">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={() => decideListingQueueSuggestion(row.id, "accept")}
                                                        >Elfogad</button>
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm"
                                                            on:click={() => decideListingQueueSuggestion(row.id, "deny")}
                                                        >Elutasít</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}
                            {#if attractionSuggestions.length > 0}
                                <h4>Látnivaló-javaslatok</h4>
                                <div class="admin-table-wrapper">
                                    <table class="admin-table">
                                        <thead>
                                            <tr>
                                                <th>Látnivaló</th>
                                                <th>Felhasználó</th>
                                                <th>Változások</th>
                                                <th class="admin-table-col--action">Művelet</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each attractionSuggestions as suggestion (suggestion.id)}
                                                <tr>
                                                    <td>{suggestion.attraction_name}</td>
                                                    <td>{suggestion.user_name}</td>
                                                    <td>
                                                        {#each Object.entries(suggestion.changes || {}) as [key, value] (key)}
                                                            <div>
                                                                <strong>{SUGGESTION_FIELD_LABELS[key] || key}:</strong>
                                                                {suggestionValueText(value)}
                                                            </div>
                                                        {/each}
                                                        {#if suggestion.note}
                                                            <div><strong>Megjegyzés:</strong> {suggestion.note}</div>
                                                        {/if}
                                                    </td>
                                                    <td class="admin-table-col--action">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={() => decideAttractionSuggestion(suggestion.id, "accept")}
                                                        >Elfogad</button>
                                                        <button
                                                            type="button"
                                                            class="btn btn-sm"
                                                            on:click={() => decideAttractionSuggestion(suggestion.id, "deny")}
                                                        >Elutasít</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}
                        {/if}
                    </section>
                    <AdminWelcomeGrid
                        items={ADMIN_WELCOME_ITEMS}
                        counts={dashboardStatsFetched ? dashboardStats : null}
                        onSelect={goToAdminTab}
                    />
                    <section class="admin-subsection" aria-labelledby="admin-cache-title">
                        <h3 id="admin-cache-title">Gyorsítótár</h3>
                        <p>
                            A kártyák számai nincsenek gyorsítótárazva. Minden megnyitáskor a szerver számolja a táblákat.
                            Az alábbi mentések csak ebben a böngészőben vannak, a nyilvános oldalak használják.
                        </p>
                        <ul class="admin-cache-list">
                            {#each browserCacheRows as row (row.name)}
                                <li>
                                    <span class="admin-cache-name">{row.name}</span>
                                    <span class="admin-cache-detail">{row.detail}</span>
                                </li>
                            {/each}
                        </ul>
                    </section>
                {/if}

                <!-- Quick Links Tab -->
                {#if activeTab === "quicklinks"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Gyorslinkek a kezdőlaphoz: cím, URL és opcionális háttérszín. A kártyák a főoldalon
                            jelennek meg.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"
                            ><span>Új gyorslink hozzáadása</span><AppIcon name="plus" size={18} /></summary
                        >
                        <form class="admin-form admin-create-form" on:submit={submitLink}>
                            <label for="link_title">Cím</label>
                            <input
                                id="link_title"
                                name="title"
                                type="text"
                                bind:value={newLink.title}
                                required
                            />

                            <label for="link_url">Weblap URL</label>
                            <input
                                id="link_url"
                                name="url"
                                type="url"
                                bind:value={newLink.url}
                                required
                            />

                            <label for="link_color">Háttérszín (pl. #e6f0ff)</label>
                            <input
                                id="link_color"
                                name="bg_color"
                                type="text"
                                bind:value={newLink.bg_color}
                                placeholder="#e6f0ff"
                            />

                            <button type="submit" class="admin-submit-btn"
                                >Hozzáadás</button
                            >
                        </form>
                    </details>

                    {@render adminNotice("quicklinks")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_quick_links"
                                name="search_quick_links"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchQuickLinks}
                                on:input={() => (pageQuickLinks = 1)}
                                placeholder="Cím, URL…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgQuickLinks.total}
                        page={pgQuickLinks.page}
                        totalPages={pgQuickLinks.totalPages}
                        from={pgQuickLinks.from}
                        to={pgQuickLinks.to}
                        on:prev={() =>
                            (pageQuickLinks = Math.max(1, pageQuickLinks - 1))}
                        on:next={() =>
                            (pageQuickLinks = Math.min(
                                pgQuickLinks.totalPages,
                                pageQuickLinks + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Szín</th>
                                    <th>Cím</th>
                                    <th>Weblap URL</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgQuickLinks.rows as q}
                                    <tr>
                                        <td>
                                            <span
                                                class="color-swatch"
                                                style:background={q.bg_color}
                                            ></span>
                                        </td>
                                        <td>{q.title}</td>
                                        <td class="admin-table-cell-preview" title={q.url || ""}>
                                            <a href={q.url} target="_blank" rel="nofollow noopener">{urlPreview(q.url)}</a>
                                        </td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditLink(q)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteRecord(
                                                        "quick_links",
                                                        q.id,
                                                        fetchQuickLinks,
                                                    )}>Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="5"
                                            >Nincsenek gyorslinkek.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgQuickLinks.total}
                        page={pgQuickLinks.page}
                        totalPages={pgQuickLinks.totalPages}
                        from={pgQuickLinks.from}
                        to={pgQuickLinks.to}
                        on:prev={() =>
                            (pageQuickLinks = Math.max(1, pageQuickLinks - 1))}
                        on:next={() =>
                            (pageQuickLinks = Math.min(
                                pgQuickLinks.totalPages,
                                pageQuickLinks + 1,
                            ))}
                    />
                {/if}

                <!-- News Feeds Tab -->
                {#if activeTab === "newsfeeds"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            RSS / Atom hírfolyamok: a <strong>Hírek</strong> oldal ezekből gyűjti a cikkeket.
                            Utolsó frissítés időpontja és egyedi szín is beállítható.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"
                            ><span>Új RSS hírfolyam hozzáadása</span><AppIcon name="plus" size={18} /></summary
                        >
                        <form class="admin-form admin-create-form" on:submit={submitNews}>
                            <label for="news_title">Hírportál neve</label>
                            <input
                                id="news_title"
                                name="title"
                                type="text"
                                bind:value={newNews.title}
                                required
                            />

                            <label for="news_url">RSS URL</label>
                            <input
                                id="news_url"
                                name="feed_url"
                                type="url"
                                bind:value={newNews.feed_url}
                                required
                            />

                            <label for="news_color">Háttérszín (pl. #ffebd6)</label>
                            <input
                                id="news_color"
                                name="bg_color"
                                type="text"
                                bind:value={newNews.bg_color}
                                placeholder="#ffebd6"
                            />

                            <button type="submit" class="admin-submit-btn"
                                >Hozzáadás</button
                            >
                        </form>
                    </details>

                    {@render adminNotice("newsfeeds")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_news_feeds"
                                name="search_news_feeds"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchNewsFeeds}
                                on:input={() => (pageNewsFeeds = 1)}
                                placeholder="Név, URL…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgNewsFeeds.total}
                        page={pgNewsFeeds.page}
                        totalPages={pgNewsFeeds.totalPages}
                        from={pgNewsFeeds.from}
                        to={pgNewsFeeds.to}
                        on:prev={() =>
                            (pageNewsFeeds = Math.max(1, pageNewsFeeds - 1))}
                        on:next={() =>
                            (pageNewsFeeds = Math.min(
                                pgNewsFeeds.totalPages,
                                pageNewsFeeds + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Név</th>
                                    <th>Forrás</th>
                                    <th>Utolsó sikeres ellenőrzés</th>
                                    <th>Szín</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgNewsFeeds.rows as nf}
                                    <tr>
                                        <td>{nf.title}</td>
                                        <td>{nf.feed_url}</td>
                                        <td>
                                            {#if feedTimestamps[nf.feed_url]}
                                                {new Date(
                                                    feedTimestamps[nf.feed_url],
                                                ).toLocaleString("hu-HU")}
                                            {:else}
                                                Soha
                                            {/if}
                                            <div class="mt-xs">
                                                <button
                                                    type="button"
                                                    class="btn-update"
                                                    disabled={loadingFeeds.has(
                                                        nf.id,
                                                    )}
                                                    on:click={() =>
                                                        checkFeed(nf)}
                                                    >{loadingFeeds.has(nf.id)
                                                        ? "Folyamatban..."
                                                        : "Ellenőrzés"}</button
                                                >
                                            </div>
                                        </td>
                                        <td>
                                            <span
                                                class="color-swatch"
                                                style:background={nf.bg_color}
                                            ></span>
                                        </td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditNews(nf)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteRecord(
                                                        "news_feeds",
                                                        nf.id,
                                                        fetchNewsFeeds,
                                                    )}>Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="6"
                                            >Nincsenek hírfolyamok.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgNewsFeeds.total}
                        page={pgNewsFeeds.page}
                        totalPages={pgNewsFeeds.totalPages}
                        from={pgNewsFeeds.from}
                        to={pgNewsFeeds.to}
                        on:prev={() =>
                            (pageNewsFeeds = Math.max(1, pageNewsFeeds - 1))}
                        on:next={() =>
                            (pageNewsFeeds = Math.min(
                                pgNewsFeeds.totalPages,
                                pageNewsFeeds + 1,
                            ))}
                    />
                {/if}

                <!-- Locations Tab -->
                {#if activeTab === "locations"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Települések (falu, város, község, municípium): név, megye, típus, irányítószám,
                            koordináták és kapcsolódó adatok. Az egész oldal ezekre az azonosítókra épül.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új település</span><AppIcon name="plus" size={18} /></summary>
                        <form class="admin-form admin-create-form" on:submit={submitLocation}>
                        <label for="loc_name">Település neve (HU)</label>
                        <input
                            id="loc_name"
                            type="text"
                            bind:value={newLocation.name}
                            required
                        />

                        <label for="loc_name_ro">Név (RO)</label>
                        <input
                            id="loc_name_ro"
                            type="text"
                            bind:value={newLocation.name_ro}
                            placeholder="opcionális"
                        />

                        <label for="loc_name_de">Név (DE)</label>
                        <input
                            id="loc_name_de"
                            type="text"
                            bind:value={newLocation.name_de}
                            placeholder="opcionális"
                        />

                        <label for="loc_county">Megye</label>
                        <select id="loc_county" bind:value={newLocation.county}>
                            <option value="">Válassz...</option>
                            {#each COUNTIES as c}<option value={c}>{c}</option
                                >{/each}
                        </select>

                        <label for="loc_type">Típus</label>
                        <select id="loc_type" bind:value={newLocation.type}>
                            <option value="">Válassz...</option>
                            {#each settlementLocationTypes as t}<option value={t.slug}
                                    >{t.label_hu}</option
                                >{/each}
                        </select>

                        <label for="loc_post_code">Posta kód</label>
                        <input
                            id="loc_post_code"
                            type="text"
                            bind:value={newLocation.post_code}
                        />

                        <label for="loc_coords">Koordináták (szélesség, hosszúság)</label>
                        <input
                            id="loc_coords"
                            type="text"
                            placeholder="46.3593, 25.8017"
                            bind:value={newLocation.coordinates}
                        />

                        <label for="loc_pop">Lakosság (fő)</label>
                        <input
                            id="loc_pop"
                            type="text"
                            bind:value={newLocation.population}
                        />

                        <label for="loc_area">Terület (km²)</label>
                        <input
                            id="loc_area"
                            type="text"
                            bind:value={newLocation.area}
                        />

                        <label for="loc_crest">Címer URL</label>
                        <input
                            id="loc_crest"
                            type="text"
                            bind:value={newLocation.crest}
                        />

                        <label for="loc_parent">Kapcsolt település</label>
                        <select
                            id="loc_parent"
                            bind:value={newLocation.parent_id}
                        >
<option value="">Válassz...</option>
                            <option value={null}
                                >Nincs (Önálló város/község)</option
                            >
                            {#each settlementsForSelect as loc}
                                <option value={loc.id}
                                    >{loc.name} ({loc.county})</option
                                >
                            {/each}
                        </select>

                        <button type="submit" class="admin-submit-btn"
                            >Hozzáadás</button
                        >
                    </form>
                    </details>

                    {@render adminNotice("locations")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_locations"
                                name="search_locations"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchLocations}
                                on:input={() => (pageLocations = 1)}
                                placeholder="Név, megye, típus, ir.sz…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgLocations.total}
                        page={pgLocations.page}
                        totalPages={pgLocations.totalPages}
                        from={pgLocations.from}
                        to={pgLocations.to}
                        on:prev={() =>
                            (pageLocations = Math.max(1, pageLocations - 1))}
                        on:next={() =>
                            (pageLocations = Math.min(
                                pgLocations.totalPages,
                                pageLocations + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Név (HU)</th>
                                    <th>Név (RO)</th>
                                    <th>Név (DE)</th>
                                    <th>Megye</th>
                                    <th>Típus</th>
                                    <th title="Posta kód">Irányítószám</th>
                                    <th>Koordináták</th>
                                    <th>Lakosság (fő)</th>
                                    <th>Terület (km²)</th>
                                    <th>Címer</th>
                                    <th>Szülő település</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgLocations.rows as l}
                                    <tr>
                                        <td>{l.id}</td>
                                        <td>{l.name}</td>
                                        <td>{l.name_ro || "-"}</td>
                                        <td>{l.name_de || "-"}</td>
                                        <td>{l.county || "-"}</td>
                                        <td>
                                            <span class="badge"
                                                >{settlementTypeLabel(l.type)}</span
                                            >
                                        </td>
                                        <td>{l.post_code || "-"}</td>
                                        <td>{l.coordinates || "-"}</td>
                                        <td>{l.population || "-"}</td>
                                        <td>{l.area || "-"}</td>
                                        <td class="admin-table-cell-preview" title={l.crest || ""}>{l.crest ? urlPreview(l.crest) : "-"}</td>
                                        <td>
                                            {#if l.parent_id}
                                                {getLocationName(l.parent_id)}
                                            {:else}
                                                -
                                            {/if}
                                        </td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditLocation(l)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteRecord(
                                                        "locations",
                                                        l.id,
                                                        fetchLocations,
                                                    )}>Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="14"
                                            >Nincsenek települések.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgLocations.total}
                        page={pgLocations.page}
                        totalPages={pgLocations.totalPages}
                        from={pgLocations.from}
                        to={pgLocations.to}
                        on:prev={() =>
                            (pageLocations = Math.max(1, pageLocations - 1))}
                        on:next={() =>
                            (pageLocations = Math.min(
                                pgLocations.totalPages,
                                pageLocations + 1,
                            ))}
                    />

                    <h3 class="admin-subsection-title">Településtípusok</h3>
                    <p class="admin-form-hint" style="margin: 0 0 0.75rem">
                        A típus <strong>slug</strong>ja szerepel a település rekordban; a megnevezés a listákban és űrlapokban
                        jelenik meg. Új slug: opcionálisan megadható; üresen a megnevezésből képződik.
                    </p>
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"
                            ><span>Új településtípus</span><AppIcon name="plus" size={18} /></summary
                        >
                        <form
                            class="admin-form admin-create-form"
                            on:submit|preventDefault={submitNewSettlementLocationType}
                        >
                            <label for="slt-slug">Slug (opcionális)</label>
                            <input
                                id="slt-slug"
                                type="text"
                                bind:value={newSettlementLocationType.slug}
                                placeholder="pl. varos - üresen automatikus"
                            />
                            <label for="slt-label">Megnevezés (HU) *</label>
                            <input
                                id="slt-label"
                                type="text"
                                bind:value={newSettlementLocationType.label_hu}
                                required
                            />
                            <label for="slt-order">Sorrend</label>
                            <input
                                id="slt-order"
                                type="number"
                                bind:value={newSettlementLocationType.sort_order}
                            />
                            <button type="submit" class="admin-submit-btn"
                                >Típus hozzáadása</button
                            >
                        </form>
                    </details>

                    {#if editingSettlementLocationType}
                        <form
                            class="admin-form admin-venues-type-edit"
                            on:submit|preventDefault={saveEditSettlementLocationType}
                        >
                            <p class="admin-form-hint">
                                Slug (azonosító, nem módosítható):
                                <code>{editingSettlementLocationType.slug}</code>
                            </p>
                            <label for="slt-edit-label">Megnevezés (HU)</label>
                            <input
                                id="slt-edit-label"
                                type="text"
                                bind:value={editingSettlementLocationType.label_hu}
                                required
                            />
                            <label for="slt-edit-order">Sorrend</label>
                            <input
                                id="slt-edit-order"
                                type="number"
                                bind:value={editingSettlementLocationType.sort_order}
                            />
                            <div class="flex gap-md">
                                <button type="submit" class="admin-submit-btn"
                                    >Mentés</button
                                >
                                <button
                                    type="button"
                                    class="btn-update"
                                    on:click={cancelEditSettlementLocationType}
                                    >Mégse</button
                                >
                            </div>
                        </form>
                    {/if}

                    {@render adminNotice("settlement_location_types")}
                    <div class="admin-table-wrapper">
                        <table class="admin-table admin-table--compact">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Slug</th>
                                    <th>Megnevezés (HU)</th>
                                    <th>Sorrend</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each settlementLocationTypes as t}
                                    <tr>
                                        <td>{t.id}</td>
                                        <td><code>{t.slug}</code></td>
                                        <td>{t.label_hu}</td>
                                        <td>{t.sort_order}</td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditSettlementLocationType(t)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteSettlementLocationTypeRow(t.id)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="6"
                                            >Nincs típus (futtasd a backend migrációt).</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}

                <!-- Venues Tab -->
                {#if activeTab === "venues"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Rendezvényhelyszínek (csarnokok, terek, pályák) településhez kötve. Előbb add meg az
                            <strong>új helyszínt</strong> (ha szükséges), alatta a <strong>helyszíntípusok</strong>
                            katalógusa (slug a megnevezésből, sorrend), majd az összes helyszín listája - az
                            <strong>Események</strong> napi programjában itt választhatók.
                        </p>
                    {/if}

                    <h3 class="admin-subsection-title">Helyszínek</h3>
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"
                            ><span>Új helyszín hozzáadása</span><AppIcon name="plus" size={18} /></summary
                        >
                    <form
                        class="admin-form admin-venues-form admin-create-form"
                        on:submit|preventDefault={submitNewVenue}
                    >
                        <div class="flex gap-lg flex-wrap">
                            <label class="flex-1" style="min-width:12rem"
                                >Település (város / falu)
                                <select
                                    id="new-venue-settlement"
                                    name="settlement_id"
                                    bind:value={newVenue.settlement_id}
                                    required
                                >
                                    <option value="">Válassz...</option>
                                    {#each settlementsForSelect as loc}
                                        <option value={loc.id}
                                            >{loc.name} ({loc.county})</option
                                        >
                                    {/each}
                                </select>
                            </label>
                            <label class="flex-1" style="min-width:12rem"
                                >Név (HU) *
                                <input
                                    id="new-venue-name"
                                    name="name"
                                    type="text"
                                    bind:value={newVenue.name}
                                    required
                                    placeholder="pl. Deme László Műjégpálya"
                                />
                            </label>
                        </div>
                        <div class="flex gap-lg flex-wrap">
                            <label
                                class="flex-1"
                                style="min-width:10rem"
                                for="new-venue-name-ro"
                                >Név (RO)
                                <input
                                    id="new-venue-name-ro"
                                    type="text"
                                    bind:value={newVenue.name_ro}
                                    placeholder="opcionális"
                                />
                            </label>
                            <label
                                class="flex-1"
                                style="min-width:10rem"
                                for="new-venue-name-de"
                                >Név (DE)
                                <input
                                    id="new-venue-name-de"
                                    type="text"
                                    bind:value={newVenue.name_de}
                                    placeholder="opcionális"
                                />
                            </label>
                        </div>
                        <div class="flex gap-lg flex-wrap">
                            <label class="flex-1" style="min-width:8rem"
                                >Slug (opcionális)
                                <input
                                    id="new-venue-slug"
                                    name="slug"
                                    type="text"
                                    bind:value={newVenue.slug}
                                    placeholder="auto, ha üres"
                                />
                            </label>
                            <label class="flex-1" style="min-width:10rem"
                                >Típus
                                <select id="new-venue-kind" name="kind" bind:value={newVenue.kind}>
<option value="">Válassz...</option>
                                    {#each venueTypesList as vt}
                                        <option value={vt.slug}
                                            >{vt.label_hu}</option
                                        >
                                    {/each}
                                </select>
                            </label>
                        </div>
                        <label for="new-venue-address">Cím</label>
                        <input
                            id="new-venue-address"
                            type="text"
                            bind:value={newVenue.address}
                            placeholder="Utca, házszám"
                        />
                        <div class="flex gap-lg flex-wrap">
                            <label class="flex-1" style="min-width:8rem"
                                >Szélesség (lat)
                                <input
                                    id="new-venue-latitude"
                                    name="latitude"
                                    type="text"
                                    bind:value={newVenue.latitude}
                                    placeholder="pl. 46.1234"
                                />
                            </label>
                            <label class="flex-1" style="min-width:8rem"
                                >Hosszúság (lon)
                                <input
                                    id="new-venue-longitude"
                                    name="longitude"
                                    type="text"
                                    bind:value={newVenue.longitude}
                                    placeholder="pl. 25.5678"
                                />
                            </label>
                            <label class="flex-1" style="min-width:8rem"
                                >Férőhely
                                <input
                                    id="new-venue-seating"
                                    name="seating_capacity"
                                    type="text"
                                    bind:value={newVenue.seating_capacity}
                                    placeholder="ülőhely / kapacitás"
                                />
                            </label>
                        </div>
                        <label for="new-venue-description">Leírás</label>
                        <textarea
                            id="new-venue-description"
                            bind:value={newVenue.description}
                            rows="3"
                        ></textarea>
                        <label for="new-venue-notes">Belső megjegyzés</label>
                        <textarea
                            id="new-venue-notes"
                            bind:value={newVenue.notes}
                            rows="2"
                        ></textarea>
                        <button type="submit" class="admin-submit-btn"
                            >Helyszín hozzáadása</button
                        >
                    </form>
                    </details>

                    <h3 class="admin-subsection-title">Helyszíntípusok</h3>
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"
                            ><span>Új helyszíntípus</span><AppIcon name="plus" size={18} /></summary
                        >
                        <form
                            class="admin-form admin-create-form"
                            on:submit|preventDefault={submitNewVenueType}
                        >
                            <label for="vt-label">Megnevezés (HU) *</label>
                            <input
                                id="vt-label"
                                type="text"
                                bind:value={newVenueType.label_hu}
                                required
                            />
                            <button type="submit" class="admin-submit-btn"
                                >Típus hozzáadása</button
                            >
                        </form>
                    </details>

                    {#if editingVenueType}
                        <form
                            class="admin-form admin-venues-type-edit"
                            on:submit|preventDefault={saveEditVenueType}
                        >
                            <p class="admin-form-hint">
                                Slug (automatikusan a megnevezésből; mentéskor frissül, és a hozzá tartozó
                                helyszínek <code>kind</code> mezője is ehhez igazodik):
                                <code>{editingVenueType.slug}</code>
                            </p>
                            <label for="vt-edit-label">Megnevezés (HU)</label>
                            <input
                                id="vt-edit-label"
                                type="text"
                                bind:value={editingVenueType.label_hu}
                                required
                            />
                            <div class="flex gap-md">
                                <button type="submit" class="admin-submit-btn"
                                    >Mentés</button
                                >
                                <button
                                    type="button"
                                    class="btn-update"
                                    on:click={cancelEditVenueType}>Mégse</button
                                >
                            </div>
                        </form>
                    {/if}

                    {@render adminNotice("venue_types")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (típusok)
                            <input
                                id="search_venue_types"
                                name="search_venue_types"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchVenueTypes}
                                on:input={() => (pageVenueTypes = 1)}
                                placeholder="Slug, megnevezés…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgVenueTypes.total}
                        page={pgVenueTypes.page}
                        totalPages={pgVenueTypes.totalPages}
                        from={pgVenueTypes.from}
                        to={pgVenueTypes.to}
                        on:prev={() =>
                            (pageVenueTypes = Math.max(1, pageVenueTypes - 1))}
                        on:next={() =>
                            (pageVenueTypes = Math.min(
                                pgVenueTypes.totalPages,
                                pageVenueTypes + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table admin-table--compact">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Slug</th>
                                    <th>Megnevezés (HU)</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgVenueTypes.rows as t}
                                    <tr>
                                        <td>{t.id}</td>
                                        <td><code>{t.slug}</code></td>
                                        <td>{t.label_hu}</td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditVenueType(t)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteVenueTypeRow(t.id)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="5"
                                            >Nincs típus (futtasd a migrációt).</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgVenueTypes.total}
                        page={pgVenueTypes.page}
                        totalPages={pgVenueTypes.totalPages}
                        from={pgVenueTypes.from}
                        to={pgVenueTypes.to}
                        on:prev={() =>
                            (pageVenueTypes = Math.max(1, pageVenueTypes - 1))}
                        on:next={() =>
                            (pageVenueTypes = Math.min(
                                pgVenueTypes.totalPages,
                                pageVenueTypes + 1,
                            ))}
                    />

                    {@render adminNotice("venues")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (helyszínek)
                            <input
                                id="search_venues"
                                name="search_venues"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchVenues}
                                on:input={() => (pageVenues = 1)}
                                placeholder="Név, település, típus…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgVenuesCatalog.total}
                        page={pgVenuesCatalog.page}
                        totalPages={pgVenuesCatalog.totalPages}
                        from={pgVenuesCatalog.from}
                        to={pgVenuesCatalog.to}
                        on:prev={() =>
                            (pageVenues = Math.max(1, pageVenues - 1))}
                        on:next={() =>
                            (pageVenues = Math.min(
                                pgVenuesCatalog.totalPages,
                                pageVenues + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table admin-table--compact">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Település</th>
                                    <th>Név (HU)</th>
                                    <th>Név (RO)</th>
                                    <th>Név (DE)</th>
                                    <th>Slug</th>
                                    <th>Típus</th>
                                    <th>Cím</th>
                                    <th>Koordináták</th>
                                    <th>Férőhely</th>
                                    <th>Leírás</th>
                                    <th>Belső megjegyzés</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgVenuesCatalog.rows as v}
                                    <tr>
                                        <td>{v.id}</td>
                                        <td
                                            >{v.settlement_name}, {v.county_name}</td
                                        >
                                        <td>{v.name}</td>
                                        <td>{v.name_ro || "-"}</td>
                                        <td>{v.name_de || "-"}</td>
                                        <td><code>{v.slug || "-"}</code></td>
                                        <td
                                            >{v.kind_label || v.kind}</td
                                        >
                                        <td class="admin-table-cell-preview" title={v.address || ""}>{contentPreview(v.address)}</td>
                                        <td
                                            >{#if v.latitude != null && v.longitude != null}{Number(
                                                    v.latitude,
                                                ).toFixed(4)}, {Number(
                                                    v.longitude,
                                                ).toFixed(4)}{:else}-{/if}</td
                                        >
                                        <td>{v.seating_capacity ?? "-"}</td>
                                        <td class="admin-table-cell-preview" title={v.description || ""}>{contentPreview(v.description)}</td>
                                        <td class="admin-table-cell-preview" title={v.notes || ""}>{contentPreview(v.notes)}</td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditVenue(v)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteVenueRow(v.id)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="14"
                                            >Nincs még helyszín.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgVenuesCatalog.total}
                        page={pgVenuesCatalog.page}
                        totalPages={pgVenuesCatalog.totalPages}
                        from={pgVenuesCatalog.from}
                        to={pgVenuesCatalog.to}
                        on:prev={() =>
                            (pageVenues = Math.max(1, pageVenues - 1))}
                        on:next={() =>
                            (pageVenues = Math.min(
                                pgVenuesCatalog.totalPages,
                                pageVenues + 1,
                            ))}
                    />
                {/if}

                <!-- Events Tab -->
                {#if activeTab === "events"}
                    {#if eventsWithIncompleteDateTime.length > 0}
                        <div class="info-box warning" role="alert">
                            <p>
                                <strong>Hiányos esemény-időpontok.</strong>
                                {eventsWithIncompleteDateTime.length} eseménynél nincs meg minden kötelező mező
                                (kezdő/befejező dátum és óra:perc). Szerkeszd a listában a ⚠ jelű sorokat, és töltsd
                                ki a mezőket.
                            </p>
                        </div>
                    {:else if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Közösségi és sportesemények: település, opcionális kiválasztott helyszín, típus,
                            szervező, leírás. A kezdő és befejező <strong>dátum és időpont (óra:perc)</strong> mind
                            kötelező - a mentés és a nyilvános időjelzések ettől függnek. Opcionálisan
                            <strong>napi program</strong> (több nap, helyszínenkénti tételek) adható meg a szerkesztőben.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új esemény</span><AppIcon name="plus" size={18} /></summary>
                        <p class="admin-form-hint">
                            A <strong>kezdő és befejező dátum</strong> és a hozzájuk tartozó
                            <strong>időpontok (óra:perc)</strong> mind kötelezőek - a mentés nélkülük nem lehetséges.
                        </p>
                    <form class="admin-form admin-create-form" on:submit={submitEvent}>
                        <label for="event_loc"
                            >Település / Helyszín <span class="admin-req" title="Kötelező"
                                >*</span
                            ></label
                        >
                        <select
                            id="event_loc"
                            bind:value={newEvent.location_id}
                            required
                            on:change={loadVenuesForNewEvent}
                        >
                            <option value="">Válassz...</option>
                            {#each settlementsForSelect as loc}
                                <option value={loc.id}
                                    >{loc.name} ({loc.county})</option
                                >
                            {/each}
                        </select>

                        <label for="event_default_venue"
                            >Konkrét helyszín (opcionális)</label
                        >
                        <select
                            id="event_default_venue"
                            name="default_venue_id"
                            bind:value={newEvent.default_venue_id}
                        >
<option value="">Válassz...</option>
                            <option value="">- nincs megadva -</option>
                            {#each venueOptionsNew as v}
                                <option value={String(v.id)}>{v.name}</option>
                            {/each}
                        </select>

                        <label for="event_attraction">Látnivaló (opcionális)</label>
                        <select id="event_attraction" name="attraction_id" bind:value={newEvent.attraction_id}>
<option value="">Válassz...</option>
                            <option value="">- nincs hozzárendelve -</option>
                            {#each attractions as att}
                                <option value={String(att.id)}>{att.name} ({att.county_name || att.county_slug})</option>
                            {/each}
                        </select>

                        <label for="event_title"
                            >Esemény neve <span class="admin-req" title="Kötelező">*</span
                            ></label
                        >
                        <input
                            id="event_title"
                            type="text"
                            bind:value={newEvent.title}
                            required
                        />

                        <label for="event_desc">Leírás</label>
                        <textarea
                            id="event_desc"
                            bind:value={newEvent.description}
                        ></textarea>

                        <div class="admin-event-image-block">
                            <label for="event_featured_upload">Kiemelt kép</label>
                            {#if newEvent.featured_image}
                                <img
                                    class="admin-event-image-preview"
                                    src={absoluteMediaUrl(
                                        newEvent.featured_image,
                                        getApiBase(),
                                    )}
                                    alt=""
                                />
                            {/if}
                            <input
                                id="event_featured_upload"
                                type="file"
                                accept="image/jpeg,image/png,image/webp,image/gif"
                                on:change={(e) => {
                                    const f = e.target.files?.[0];
                                    if (f) uploadEventFeaturedImage(f, false);
                                    e.target.value = "";
                                }}
                            />
                            <label for="event_featured_url" class="admin-sublabel"
                                >Vagy kép URL (külső)</label
                            >
                            <input
                                id="event_featured_url"
                                type="url"
                                bind:value={newEvent.featured_image}
                                placeholder="https://…"
                            />
                            {#if newEvent.featured_image}
                                <button
                                    type="button"
                                    class="btn-update"
                                    style="align-self: flex-start"
                                    on:click={() =>
                                        (newEvent.featured_image = "")}
                                    >Kép törlése</button
                                >
                            {/if}
                            <label for="event_featured_copyright">Szerzői jogi információ</label>
                            <input
                                id="event_featured_copyright"
                                name="featured_image_copyright"
                                type="text"
                                bind:value={newEvent.featured_image_copyright}
                                placeholder="A feltöltő által megadott szerzői jog"
                            />
                        </div>

                        <div class="flex gap-lg">
                            <div class="flex-1">
                                <label for="event_start_date"
                                    >Kezdő dátum <span class="admin-req" title="Kötelező"
                                        >*</span
                                    ></label
                                >
                                <HuDateInput
                                    id="event_start_date"
                                    bind:value={newEvent.start_date}
                                    required
                                />
                            </div>
                            <div class="flex-1">
                                <label for="event_start_time"
                                    >Kezdő időpont (óra:perc) <span
                                        class="admin-req"
                                        title="Kötelező">*</span
                                    ></label
                                >
                                <HuTimeInput
                                    id="event_start_time"
                                    bind:value={newEvent.start_time}
                                    required
                                />
                            </div>
                        </div>

                        <div class="flex gap-lg">
                            <div class="flex-1">
                                <label for="event_end_date"
                                    >Befejező dátum <span class="admin-req" title="Kötelező"
                                        >*</span
                                    ></label
                                >
                                <HuDateInput
                                    id="event_end_date"
                                    bind:value={newEvent.end_date}
                                    required
                                />
                            </div>
                            <div class="flex-1">
                                <label for="event_end_time"
                                    >Befejező időpont (óra:perc) <span
                                        class="admin-req"
                                        title="Kötelező">*</span
                                    ></label
                                >
                                <HuTimeInput
                                    id="event_end_time"
                                    bind:value={newEvent.end_time}
                                    required
                                />
                            </div>
                        </div>

                        <label for="event_type_id"
                            >Eseménytípus <span class="admin-req" title="Kötelező">*</span></label
                        >
                        <select
                            id="event_type_id"
                            bind:value={newEvent.event_type_id}
                            on:change={() => (newEvent.event_subtype_id = "")}
                            required
                        >
                            <option value="">Válassz...</option>
                            {#each [...catalogEventTypes].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || String(a.label_hu).localeCompare(String(b.label_hu), "hu")) as t}
                                <option value={String(t.id)}
                                    >{t.label_hu} ({t.slug})</option
                                >
                            {/each}
                        </select>

                        <label for="event_subtype_id">Altípus (opcionális)</label>
                        <select
                            id="event_subtype_id"
                            bind:value={newEvent.event_subtype_id}
                        >
<option value="">Válassz...</option>
                            <option value="">- nincs -</option>
                            {#each subtypesForNewEvent as s}
                                <option value={String(s.id)}
                                    >{s.label_hu} ({s.slug})</option
                                >
                            {/each}
                        </select>

                        <label for="event_access_type">Hozzáférés</label>
                        <select
                            id="event_access_type"
                            bind:value={newEvent.access_type}
                        >
<option value="">Válassz...</option>
                            <option value="public">{ACCESS_TYPE_LABELS.public}</option>
                            <option value="members_only">{ACCESS_TYPE_LABELS.members_only}</option>
                            <option value="invitation_only">{ACCESS_TYPE_LABELS.invitation_only}</option>
                        </select>

                        <label for="event_org">Szervező</label>
                        <div class="org-autosuggest-wrapper">
                            <div class="org-autosuggest-row">
                                <input
                                    id="event_org"
                                    type="text"
                                    bind:value={orgQuery}
                                    on:input={() => {
                                        newEvent.organizer = orgQuery;
                                        onOrgInput(false);
                                    }}
                                    on:focus={() => onOrgInput(false)}
                                    on:blur={() => handleOrgBlur(false)}
                                    autocomplete="off"
                                    placeholder="Keresés szervező neve..."
                                    class="flex-1"
                                />
                                <button
                                    type="button"
                                    class="btn-update"
                                    style="margin-bottom:0"
                                    on:click={() =>
                                        (newOrganizerModalVisible = true)}
                                >
                                    Új szervező
                                </button>
                            </div>
                            {#if orgDropdownOpen && orgSuggestions.length > 0}
                                <ul class="org-suggestions">
                                    {#each orgSuggestions as s}
                                        <li>
                                            <button type="button" on:click={() => selectOrganizer(s.name, false)}>
                                                <strong>{s.name}</strong>
                                                {#if s.location}<span class="org-sug-meta">{s.location}</span>{/if}
                                            </button>
                                        </li>
                                    {/each}
                                </ul>
                            {/if}
                        </div>

                        <label for="event_entry_price">Belépő / jegyár (opcionális)</label>
                        <input
                            id="event_entry_price"
                            name="entry_price"
                            type="text"
                            bind:value={newEvent.entry_price}
                            placeholder="pl. 99 RON, 15 EUR, ingyenes"
                            maxlength="128"
                            autocomplete="off"
                        />

                        <button type="submit" class="admin-submit-btn"
                            >Hozzáadás</button
                        >
                    </form>
                    </details>

                    {@render adminNotice("events")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_events"
                                name="search_events"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchEvents}
                                on:input={() => (pageEvents = 1)}
                                placeholder="Cím, szervező, helyszín…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgEvents.total}
                        page={pgEvents.page}
                        totalPages={pgEvents.totalPages}
                        from={pgEvents.from}
                        to={pgEvents.to}
                        on:prev={() =>
                            (pageEvents = Math.max(1, pageEvents - 1))}
                        on:next={() =>
                            (pageEvents = Math.min(
                                pgEvents.totalPages,
                                pageEvents + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Kép</th>
                                    <th>Cím</th>
                                    <th>Kezdés</th>
                                    <th>Befejezés</th>
                                    <th>Típus</th>
                                    <th>Altípus</th>
                                    <th>Hozzáférés</th>
                                    <th>Település</th>
                                    <th>Alapért. helyszín</th>
                                    <th>Szervező</th>
                                    <th>Belépő</th>
                                    <th>Leírás</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgEvents.rows as e}
                                    <tr
                                        class:admin-row-warn={!eventDateTimeComplete(
                                            e,
                                        )}
                                    >
                                        <td class="admin-event-thumb-cell">
                                            {#if e.featured_image}
                                                <img
                                                    src={absoluteMediaUrl(
                                                        e.featured_image,
                                                        getApiBase(),
                                                    )}
                                                    alt=""
                                                />
                                            {:else}
                                                <span class="admin-thumb-empty">-</span>
                                            {/if}
                                        </td>
                                        <td>
                                            {#if !eventDateTimeComplete(e)}
                                                <span
                                                    class="admin-req"
                                                    title="Hiányos dátum vagy időpont - szerkessze és töltse ki."
                                                    >⚠</span
                                                >
                                            {/if}
                                            {e.title}
                                        </td>
                                        <td>
                                            {new Date(
                                                e.start_date,
                                            ).toLocaleDateString("hu-HU")}
                                            {#if e.start_time}
                                                {e.start_time.slice(0, 5)}{/if}
                                        </td>
                                        <td>
                                            {#if e.end_date}
                                                {new Date(
                                                    e.end_date,
                                                ).toLocaleDateString("hu-HU")}
                                                {#if e.end_time}
                                                    {e.end_time.slice(0, 5)}{/if}
                                            {:else}
                                                -
                                            {/if}
                                        </td>
                                        <td>{eventTypeLabelFromCatalog(e.event_type)}</td>
                                        <td>{eventSubtypeLabelFromCatalog(Number(e.event_type_id), e.event_subtype || "")}</td>
                                        <td>{accessTypeLabel(e.access_type)}</td>
                                        <td>{getLocationName(e.location_id)}</td>
                                        <td class="admin-table-cell-preview" title={e.default_venue_name || ""}>{contentPreview(e.default_venue_name || "")}</td>
                                        <td>{e.organizer || "-"}</td>
                                        <td>{e.entry_price && String(e.entry_price).trim() ? e.entry_price : "-"}</td>
                                        <td class="admin-table-cell-preview" title={e.description || ""}>{contentPreview(e.description)}</td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditEvent(e)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteRecord(
                                                        "events",
                                                        e.id,
                                                        fetchEvents,
                                                    )}>Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="14"
                                            >Nincsenek események.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgEvents.total}
                        page={pgEvents.page}
                        totalPages={pgEvents.totalPages}
                        from={pgEvents.from}
                        to={pgEvents.to}
                        on:prev={() =>
                            (pageEvents = Math.max(1, pageEvents - 1))}
                        on:next={() =>
                            (pageEvents = Math.min(
                                pgEvents.totalPages,
                                pageEvents + 1,
                            ))}
                    />

                    <h3 class="admin-subsection-title">Eseménytípusok (katalógus)</h3>
                    <p class="admin-hint">
                        A típusok és altípusok itt szerkeszthetők. Az események <code>event_type_id</code> értéke ezekre
                        mutat.
                    </p>
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új eseménytípus</span><AppIcon name="plus" size={18} /></summary>
                        <form
                            class="admin-form admin-create-form"
                            on:submit|preventDefault={submitCatalogEventType}
                        >
                            <label for="cet-slug">Slug (URL, egyedi, pl. <code>sports</code>) *</label>
                            <input
                                id="cet-slug"
                                type="text"
                                bind:value={newCatalogEventType.slug}
                                required
                                placeholder="pl. workshop"
                            />
                            <label for="cet-label">Megnevezés (HU) *</label>
                            <input
                                id="cet-label"
                                type="text"
                                bind:value={newCatalogEventType.label_hu}
                                required
                            />
                            <label for="cet-sort">Sorrend</label>
                            <input
                                id="cet-sort"
                                type="number"
                                bind:value={newCatalogEventType.sort_order}
                            />
                            <button type="submit" class="admin-submit-btn"
                                >Típus hozzáadása</button
                            >
                        </form>
                    </details>

                    {#if editingCatalogEventType}
                        <form
                            class="admin-form admin-venues-type-edit"
                            on:submit|preventDefault={saveEditCatalogEventType}
                        >
                            <p class="admin-form-hint">
                                Slug: <code>{editingCatalogEventType.slug}</code> (nem változtatható)
                            </p>
                            <label for="cet-edit-label">Megnevezés (HU)</label>
                            <input
                                id="cet-edit-label"
                                type="text"
                                bind:value={editingCatalogEventType.label_hu}
                                required
                            />
                            <label for="cet-edit-sort">Sorrend</label>
                            <input
                                id="cet-edit-sort"
                                type="number"
                                bind:value={editingCatalogEventType.sort_order}
                            />
                            <div class="flex gap-md">
                                <button type="submit" class="admin-submit-btn"
                                    >Mentés</button
                                >
                                <button
                                    type="button"
                                    class="btn-update"
                                    on:click={cancelEditCatalogEventType}>Mégse</button
                                >
                            </div>
                        </form>
                    {/if}

                    {@render adminNotice("catalog_event_types")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (típusok)
                            <input
                                id="search-catalog-types"
                                name="search_catalog_types"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchCatalogTypes}
                                on:input={() => (pageCatalogTypes = 1)}
                                placeholder="Slug, név…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgCatalogTypes.total}
                        page={pgCatalogTypes.page}
                        totalPages={pgCatalogTypes.totalPages}
                        from={pgCatalogTypes.from}
                        to={pgCatalogTypes.to}
                        on:prev={() =>
                            (pageCatalogTypes = Math.max(1, pageCatalogTypes - 1))}
                        on:next={() =>
                            (pageCatalogTypes = Math.min(
                                pgCatalogTypes.totalPages,
                                pageCatalogTypes + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table admin-table--compact">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Slug</th>
                                    <th>Megnevezés (HU)</th>
                                    <th>Sorrend</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgCatalogTypes.rows as t}
                                    <tr>
                                        <td>{t.id}</td>
                                        <td><code>{t.slug}</code></td>
                                        <td>{t.label_hu}</td>
                                        <td>{t.sort_order}</td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditCatalogEventType(t)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteCatalogEventTypeRow(t.id)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="6">Nincs típus.</td></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgCatalogTypes.total}
                        page={pgCatalogTypes.page}
                        totalPages={pgCatalogTypes.totalPages}
                        from={pgCatalogTypes.from}
                        to={pgCatalogTypes.to}
                        on:prev={() =>
                            (pageCatalogTypes = Math.max(1, pageCatalogTypes - 1))}
                        on:next={() =>
                            (pageCatalogTypes = Math.min(
                                pgCatalogTypes.totalPages,
                                pageCatalogTypes + 1,
                            ))}
                    />

                    <h3 class="admin-subsection-title">Esemény altípusok</h3>
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új altípus</span><AppIcon name="plus" size={18} /></summary>
                        <form
                            class="admin-form admin-create-form"
                            on:submit|preventDefault={submitCatalogEventSubtype}
                        >
                            <label for="ces-type">Főtípus *</label>
                            <select
                                id="ces-type"
                                bind:value={newCatalogEventSubtype.event_type_id}
                                required
                            >
<option value="">Válassz...</option>
                                {#each [...catalogEventTypes].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || String(a.label_hu).localeCompare(String(b.label_hu), "hu")) as ct}
                                    <option value={String(ct.id)}
                                        >{ct.label_hu} ({ct.slug})</option
                                    >
                                {/each}
                            </select>
                            <label for="ces-slug">Slug *</label>
                            <input
                                id="ces-slug"
                                type="text"
                                bind:value={newCatalogEventSubtype.slug}
                                required
                                placeholder="pl. hockey"
                            />
                            <label for="ces-label">Megnevezés (HU) *</label>
                            <input
                                id="ces-label"
                                type="text"
                                bind:value={newCatalogEventSubtype.label_hu}
                                required
                            />
                            <label for="ces-sort">Sorrend</label>
                            <input
                                id="ces-sort"
                                type="number"
                                bind:value={newCatalogEventSubtype.sort_order}
                            />
                            <button type="submit" class="admin-submit-btn"
                                >Altípus hozzáadása</button
                            >
                        </form>
                    </details>

                    {#if editingCatalogEventSubtype}
                        <form
                            class="admin-form admin-venues-type-edit"
                            on:submit|preventDefault={saveEditCatalogEventSubtype}
                        >
                            <p class="admin-form-hint">
                                Slug: <code>{editingCatalogEventSubtype.slug}</code> · főtípus ID:
                                {editingCatalogEventSubtype.event_type_id}
                            </p>
                            <label for="ces-edit-label">Megnevezés (HU)</label>
                            <input
                                id="ces-edit-label"
                                type="text"
                                bind:value={editingCatalogEventSubtype.label_hu}
                                required
                            />
                            <label for="ces-edit-sort">Sorrend</label>
                            <input
                                id="ces-edit-sort"
                                type="number"
                                bind:value={editingCatalogEventSubtype.sort_order}
                            />
                            <div class="flex gap-md">
                                <button type="submit" class="admin-submit-btn"
                                    >Mentés</button
                                >
                                <button
                                    type="button"
                                    class="btn-update"
                                    on:click={cancelEditCatalogEventSubtype}>Mégse</button
                                >
                            </div>
                        </form>
                    {/if}

                    {@render adminNotice("catalog_event_subtypes")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (altípusok)
                            <input
                                id="search-catalog-subtypes"
                                name="search_catalog_subtypes"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchCatalogSubtypes}
                                on:input={() => (pageCatalogSubtypes = 1)}
                                placeholder="Slug, név, típus ID…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgCatalogSubtypes.total}
                        page={pgCatalogSubtypes.page}
                        totalPages={pgCatalogSubtypes.totalPages}
                        from={pgCatalogSubtypes.from}
                        to={pgCatalogSubtypes.to}
                        on:prev={() =>
                            (pageCatalogSubtypes = Math.max(
                                1,
                                pageCatalogSubtypes - 1,
                            ))}
                        on:next={() =>
                            (pageCatalogSubtypes = Math.min(
                                pgCatalogSubtypes.totalPages,
                                pageCatalogSubtypes + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table admin-table--compact">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Főtípus ID</th>
                                    <th>Slug</th>
                                    <th>Megnevezés (HU)</th>
                                    <th>Sorrend</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgCatalogSubtypes.rows as s}
                                    <tr>
                                        <td>{s.id}</td>
                                        <td>{s.event_type_id}</td>
                                        <td><code>{s.slug}</code></td>
                                        <td>{s.label_hu}</td>
                                        <td>{s.sort_order}</td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditCatalogEventSubtype(s)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteCatalogEventSubtypeRow(s.id)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="7">Nincs altípus.</td></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgCatalogSubtypes.total}
                        page={pgCatalogSubtypes.page}
                        totalPages={pgCatalogSubtypes.totalPages}
                        from={pgCatalogSubtypes.from}
                        to={pgCatalogSubtypes.to}
                        on:prev={() =>
                            (pageCatalogSubtypes = Math.max(
                                1,
                                pageCatalogSubtypes - 1,
                            ))}
                        on:next={() =>
                            (pageCatalogSubtypes = Math.min(
                                pgCatalogSubtypes.totalPages,
                                pageCatalogSubtypes + 1,
                            ))}
                    />
                {/if}

                <!-- Entry Categories Tab -->
                {#if activeTab === "entry_categories"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Bejegyzés-kategóriák (pl. szolgáltatás típusok): a településoldali és index
                            bejegyzések csoportosításához.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új kategória</span><AppIcon name="plus" size={18} /></summary>
                        <form class="admin-form admin-create-form" on:submit={submitEntryCategory}>
                            <label for="cat_name">Kategória neve</label>
                            <input
                                id="cat_name"
                                name="name"
                                type="text"
                                bind:value={newEntryCategory.name}
                                required
                            />

                            <label for="cat_parent">Főkategória</label>
                            <select id="cat_parent" bind:value={newEntryCategory.parent_id}>
<option value="">Válassz...</option>
                                <option value="">Főkategória</option>
                                {#each entryCategoryParents as parent}
                                    <option value={parent.id}>{parent.name}</option>
                                {/each}
                            </select>

                            <button type="submit" class="admin-submit-btn"
                                >Hozzáadás</button
                            >
                        </form>
                    </details>

                    <FeaturedCategoriesPanel categories={entryCategories} />

                    {@render adminNotice("entry_categories")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label">
                            <span class="admin-search-heading">
                                Keresés
                            </span>
                            <input
                                id="search_entry_categories"
                                name="search_entry_categories"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchEntryCategories}
                                on:input={() => (pageEntryCategories = 1)}
                                placeholder="Név, ID…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgEntryCategories.total}
                        page={pgEntryCategories.page}
                        totalPages={pgEntryCategories.totalPages}
                        from={pgEntryCategories.from}
                        to={pgEntryCategories.to}
                        on:prev={() =>
                            (pageEntryCategories = Math.max(
                                1,
                                pageEntryCategories - 1,
                            ))}
                        on:next={() =>
                            (pageEntryCategories = Math.min(
                                pgEntryCategories.totalPages,
                                pageEntryCategories + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Név</th>
                                    <th>Főkategória</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgEntryCategories.rows as cat}
                                    <tr>
                                        <td>{cat.id}</td>
                                        <td>{cat.name}</td>
                                        <td>{getEntryCategoryParentName(cat)}</td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() =>
                                                    startEditCategory(cat)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteEntryCategory(cat)}
                                                >Törlés</button
                                            >
                                        </td>
                                    </tr>
                                    {#if categoryDeleteMove?.id === cat.id}
                                        <tr>
                                            <td colspan="5">
                                                <p>{categoryDeleteMove.message}</p>
                                                <label for="cat_move_{cat.id}"
                                                    >Áthelyezés ide</label
                                                >
                                                <select
                                                    id="cat_move_{cat.id}"
                                                    bind:value={categoryDeleteMove.moveTo}
                                                >
<option value="">Válassz...</option>
                                                    {#each entryCategoryMoveTargets(cat) as target}
                                                        <option value={target.id}
                                                            >{target.name}</option
                                                        >
                                                    {/each}
                                                </select>
                                                <button
                                                    type="button"
                                                    class="admin-submit-btn"
                                                    on:click={() =>
                                                        deleteEntryCategoryWithMove(
                                                            cat,
                                                        )}
                                                    >Áthelyezés és törlés</button
                                                >
                                                <button
                                                    type="button"
                                                    class="btn"
                                                    on:click={() =>
                                                        (categoryDeleteMove = null)}
                                                    >Mégse</button
                                                >
                                            </td>
                                        </tr>
                                    {/if}
                                {:else}
                                    <tr
                                        ><td colspan="5"
                                            >Nincsenek kategóriák.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgEntryCategories.total}
                        page={pgEntryCategories.page}
                        totalPages={pgEntryCategories.totalPages}
                        from={pgEntryCategories.from}
                        to={pgEntryCategories.to}
                        on:prev={() =>
                            (pageEntryCategories = Math.max(
                                1,
                                pageEntryCategories - 1,
                            ))}
                        on:next={() =>
                            (pageEntryCategories = Math.min(
                                pgEntryCategories.totalPages,
                                pageEntryCategories + 1,
                            ))}
                    />
                {/if}

                <!-- Weboldalak Tab -->
                {#if activeTab === "websites"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Jóváhagyott és várakozó weboldalak. A jóváhagyás, elutasítás és tiltás a vezérlőpult üzeneteiben történik.
                        </p>
                    {/if}
                    {@render adminNotice("websites")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_admin_websites"
                                name="search_admin_websites"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchAdminWebsites}
                                on:input={() => (pageAdminWebsites = 1)}
                                placeholder="Cím, domain, leírás…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgAdminWebsites.total}
                        page={pgAdminWebsites.page}
                        totalPages={pgAdminWebsites.totalPages}
                        from={pgAdminWebsites.from}
                        to={pgAdminWebsites.to}
                        on:prev={() =>
                            (pageAdminWebsites = Math.max(1, pageAdminWebsites - 1))}
                        on:next={() =>
                            (pageAdminWebsites = Math.min(
                                pgAdminWebsites.totalPages,
                                pageAdminWebsites + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Cím</th>
                                    <th>Domain</th>
                                    <th>Állapot</th>
                                    <th>Átvéve</th>
                                    <th>Leírás</th>
                                    <th>Beküldő</th>
                                    <th>Beküldve</th>
                                    <th>Jóváhagyva</th>
                                    <th>Jóváhagyta</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgAdminWebsites.rows as site}
                                    <tr>
                                        <td>{site.title}</td>
                                        <td>{site.domain}</td>
                                        <td>{site.status === "approved" ? "Jóváhagyott" : "Jóváhagyásra vár"}</td>
                                        <td>{site.claimed ? "Átvéve" : "Gazdátlan"}</td>
                                        <td class="admin-table-cell-preview" title={site.description || ""}>{contentPreview(site.description)}</td>
                                        <td>{site.submitter || "-"}</td>
                                        <td>{formatAdminTime(site.submitted_at)}</td>
                                        <td>{formatAdminTime(site.approved_at)}</td>
                                        <td>{site.approver || "-"}</td>
                                    </tr>
                                {:else}
                                    <tr><td colspan="9">Nincsenek weboldalak.</td></tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgAdminWebsites.total}
                        page={pgAdminWebsites.page}
                        totalPages={pgAdminWebsites.totalPages}
                        from={pgAdminWebsites.from}
                        to={pgAdminWebsites.to}
                        on:prev={() =>
                            (pageAdminWebsites = Math.max(1, pageAdminWebsites - 1))}
                        on:next={() =>
                            (pageAdminWebsites = Math.min(
                                pgAdminWebsites.totalPages,
                                pageAdminWebsites + 1,
                            ))}
                    />
                {/if}

                <!-- Entries Tab -->
                {#if activeTab === "entries"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Településhez kötött bejegyzések (üzletek, szervezetek, szolgáltatások): típus,
                            kategória, elérhetőség, nyelvek és címkék. Ezek a város/falu oldalakon és indexeken
                            jelennek meg.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új bejegyzés</span><AppIcon name="plus" size={18} /></summary>
                    <form class="admin-form admin-create-form" on:submit={submitEntry}>
                        <label for="serv_type">Típus</label>
                        <select id="serv_type" bind:value={newEntry.type}>
<option value="">Válassz...</option>
                            {#each entryTypes as t}<option value={t.name}
                                    >{t.name}</option
                                >{/each}
                        </select>

                        <label for="serv_loc">Település</label>
                        <select
                            id="serv_loc"
                            bind:value={newEntry.location_id}
                            required
                        >
                            <option value="">Válassz...</option>
                            {#each settlementsForSelect as loc}
                                <option value={loc.id}
                                    >{loc.name} ({loc.county})</option
                                >
                            {/each}
                        </select>

                        <CategoryMultiSelect
                            parents={entryCategoryParents}
                            children={entryCategoryChildren}
                            bind:primary={newEntry.category_id}
                            bind:extra={newEntry.category_extra}
                            primaryInputId="serv_cat"
                        />

                        <label for="serv_name">Név</label>
                        <input
                            id="serv_name"
                            type="text"
                            bind:value={newEntry.name}
                            required
                        />

                        <label for="serv_url">Weblap URL</label>
                        <input
                            id="serv_url"
                            type="url"
                            bind:value={newEntry.url}
                        />

                        <label for="serv_phone">Telefon</label>
                        <input
                            id="serv_phone"
                            type="text"
                            bind:value={newEntry.phone}
                        />

                        <label for="serv_addr">Cím</label>
                        <input
                            id="serv_addr"
                            type="text"
                            bind:value={newEntry.address}
                        />

                        <label for="serv_notes">Bemutatkozás</label>
                        <textarea id="serv_notes" bind:value={newEntry.notes}
                        ></textarea>

                        <label for="serv_tags">Címkék (#cimke1 #cimke2)</label>
                        <input
                            id="serv_tags"
                            type="text"
                            bind:value={newEntry.tags}
                            placeholder="#cimke1 #cimke2"
                        />

                        <span class="form-group-label">Nyelvek</span>
                        <div class="flex gap-lg flex-wrap mb-lg">
                            {#each LANGUAGES as lang}
                                <label
                                    class="flex items-center gap-xs font-normal"
                                >
                                    <input
                                        type="checkbox"
                                        name={`new-entry-lang-${lang}`}
                                        checked={newEntry.languages.includes(
                                            lang,
                                        )}
                                        on:change={() =>
                                            (newEntry.languages =
                                                newEntry.languages.includes(
                                                    lang,
                                                )
                                                    ? newEntry.languages.filter(
                                                          (l) => l !== lang,
                                                      )
                                                    : [
                                                          ...newEntry.languages,
                                                          lang,
                                                      ])}
                                        class="w-auto"
                                    />
                                    {lang}
                                </label>
                            {/each}
                        </div>

                        <label class="flex items-center gap-xs font-normal">
                            <input
                                id="new-entry-verified"
                                name="verified"
                                type="checkbox"
                                bind:checked={newEntry.verified}
                                class="w-auto"
                            />
                            Ellenőrzött
                        </label>
                        <p class="admin-form-hint">Alapértelmezett: Nem ellenőrzött (szürke jelvény). A jelvény csak adminnal kapcsolható be.</p>

                        <label class="flex items-center gap-xs font-normal">
                            <input
                                id="new-entry-hours-enabled"
                                name="hours_enabled"
                                type="checkbox"
                                bind:checked={newEntry.hours_enabled}
                                class="w-auto"
                                on:change={() => {
                                    if (newEntry.hours_enabled) {
                                        newEntry.hours = withDefaultWeekHours(newEntry.hours);
                                    }
                                }}
                            />
                            Nyitvatartás / Program
                        </label>
                        {#if newEntry.hours_enabled}
                            <EntryHoursEditor bind:hours={newEntry.hours} />
                        {/if}

                        {#if adminOffersDelivery(newEntry)}
                            <label class="flex items-center gap-xs font-normal">
                                <input
                                    id="new-entry-delivery-enabled"
                                    name="delivery_enabled"
                                    type="checkbox"
                                    bind:checked={newEntry.delivery_enabled}
                                    class="w-auto"
                                    on:change={() => {
                                        if (newEntry.delivery_enabled) {
                                            newEntry.delivery_hours = withDefaultWeekHours(
                                                newEntry.delivery_hours,
                                            );
                                        }
                                    }}
                                />
                                Kiszállítási idő
                            </label>
                            {#if newEntry.delivery_enabled}
                                <EntryHoursEditor bind:hours={newEntry.delivery_hours} />
                            {/if}
                        {/if}

                        <span class="form-group-label">Fotók</span>
                        <EntryPhotosEditor
                            bind:photos={newEntry.photos}
                            alertError={(msg) => showAlert(msg)}
                        />

                        <button type="submit" class="admin-submit-btn"
                            >Hozzáadás</button
                        >
                    </form>
                    </details>

                    {@render adminNotice("entries")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_entries"
                                name="search_entries"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchEntries}
                                on:input={() => (pageEntries = 1)}
                                placeholder="Név, domain, címke…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgEntries.total}
                        page={pgEntries.page}
                        totalPages={pgEntries.totalPages}
                        from={pgEntries.from}
                        to={pgEntries.to}
                        on:prev={() =>
                            (pageEntries = Math.max(1, pageEntries - 1))}
                        on:next={() =>
                            (pageEntries = Math.min(
                                pgEntries.totalPages,
                                pageEntries + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Név</th>
                                    <th>Típus</th>
                                    <th>Ellenőrzött</th>
                                    <th>Domain</th>
                                    <th>Település</th>
                                    <th>Kategória</th>
                                    <th>Telefon</th>
                                    <th>Cím</th>
                                    <th>Bemutatkozás</th>
                                    <th>Nyelvek</th>
                                    <th>Címkék</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgEntries.rows as s}
                                    <tr>
                                        <td>{s.name}</td>
                                        <td
                                            ><span class="badge"
                                                >{s.type || "entry"}</span
                                            ></td
                                        >
                                        <td>{s.verified ? "Ellenőrzött" : "Nem ellenőrzött"}</td>
                                        <td>{canonicalDomain(String(s.url ?? "")) || "-"}</td>
                                        <td>{getLocationName(s.location_id)}</td
                                        >
                                        <td>{getCategoryName(s.category_id)}</td
                                        >
                                        <td>{s.phone || "-"}</td>
                                        <td class="admin-table-cell-preview" title={s.address || ""}>{contentPreview(s.address)}</td>
                                        <td class="admin-table-cell-preview" title={s.notes || ""}>{contentPreview(s.notes)}</td>
                                        <td>{(s.languages || []).join(", ")}</td
                                        >
                                        <td>
                                            {#if s.tags && s.tags.length > 0}
                                                <div class="admin-table-tags">
                                                    {s.tags
                                                        .map((t) => "#" + t)
                                                        .join(" ")}
                                                </div>
                                            {:else}-{/if}
                                        </td>
                                        <td>
                                            <button
                                                class="btn-update"
                                                on:click={() => openEdit(s)}
                                                >Szerk.</button
                                            >
                                        </td>
                                        <td>
                                            <button
                                                class="btn-delete"
                                                on:click={() =>
                                                    deleteRecord(
                                                        "entries",
                                                        s.id,
                                                        fetchEntries,
                                                    )}>Törlés</button
                                            >
                                        </td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="12"
                                            >Nincsenek bejegyzések.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgEntries.total}
                        page={pgEntries.page}
                        totalPages={pgEntries.totalPages}
                        from={pgEntries.from}
                        to={pgEntries.to}
                        on:prev={() =>
                            (pageEntries = Math.max(1, pageEntries - 1))}
                        on:next={() =>
                            (pageEntries = Math.min(
                                pgEntries.totalPages,
                                pageEntries + 1,
                            ))}
                    />
                {/if}

                {#if activeTab === "users"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {/if}
                    {@render adminNotice("users")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_users"
                                name="search_users"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchUsers}
                                on:input={() => (pageUsers = 1)}
                                placeholder="E-mail, név, település…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgUsers.total}
                        page={pgUsers.page}
                        totalPages={pgUsers.totalPages}
                        from={pgUsers.from}
                        to={pgUsers.to}
                        on:prev={() => (pageUsers = Math.max(1, pageUsers - 1))}
                        on:next={() =>
                            (pageUsers = Math.min(pgUsers.totalPages, pageUsers + 1))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>E-mail</th>
                                    <th>Név</th>
                                    <th>Megjelenített név</th>
                                    <th>Nyelv</th>
                                    <th>Település</th>
                                    <th>Regisztráció</th>
                                    <th>Utolsó belépés</th>
                                    <th>Tiltva</th>
                                    <th>Admin</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgUsers.rows as user (user.id)}
                                    <tr>
                                        <td>{user.email}</td>
                                        <td>{[user.given_name, user.family_name].filter(Boolean).join(" ") || user.name || "-"}</td>
                                        <td>{user.display_name || user.name || "-"}</td>
                                        <td>{user.locale || "-"}</td>
                                        <td>{user.settlement || "-"}</td>
                                        <td>{formatAdminTime(user.created_at)}</td>
                                        <td>{formatAdminTime(user.last_login_at)}</td>
                                        <td>{user.website_banned ? "Igen" : "Nem"}</td>
                                        <td>{user.is_admin ? "Igen" : "Nem"}</td>
                                    </tr>
                                {:else}
                                    <tr>
                                        <td colspan="9">Nincs regisztrált felhasználó.</td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgUsers.total}
                        page={pgUsers.page}
                        totalPages={pgUsers.totalPages}
                        from={pgUsers.from}
                        to={pgUsers.to}
                        on:prev={() => (pageUsers = Math.max(1, pageUsers - 1))}
                        on:next={() =>
                            (pageUsers = Math.min(pgUsers.totalPages, pageUsers + 1))}
                    />
                {/if}

                <!-- Beállítások (Settings) Tab -->
                {#if activeTab === "settings"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Oldalszintű beállítások: alapértelmezett település vendégeknek és azoknak, akik nem választottak saját települést (kezdőlap időjárás, eseményszűrés),
                            időjárás-szolgáltatók engedélyezése, ikon stílus és a böngésző-cache ideje. Minden szakasz Mentés gombja csak a saját szakaszát menti.
                            A <strong>cache törlése</strong> új verziószámot ad - a látogatók frissebb időjárást kapnak.
                        </p>
                    {/if}
                    {@render adminNotice("settings")}
                    <section class="admin-form-section">
                        <h3>Közösségi oldalak</h3>
                        <p class="admin-hint">Ezek a linkek a Lámsza, a Játszótér és a Szótár láblécében jelennek meg. Üres mező: az a link nem látszik. Teljes cím, https://-sel.</p>
                        <div class="admin-form">
                            <label for="social_facebook_url">Facebook</label>
                            <input id="social_facebook_url" name="social_facebook_url" type="url" bind:value={siteSettings.social_facebook_url} placeholder="https://www.facebook.com/…" />

                            <label for="social_twitter_url">Twitter</label>
                            <input id="social_twitter_url" name="social_twitter_url" type="url" bind:value={siteSettings.social_twitter_url} placeholder="https://twitter.com/…" />

                            <label for="social_instagram_url">Instagram</label>
                            <input id="social_instagram_url" name="social_instagram_url" type="url" bind:value={siteSettings.social_instagram_url} placeholder="https://www.instagram.com/…" />

                            <div class="flex gap-md mt-md">
                                <button type="button" class="admin-submit-btn" on:click={() => saveSettings("social")} disabled={settingsSaving !== ""}>
                                    {settingsSaving === "social" ? 'Mentés…' : 'Mentés'}
                                </button>
                            </div>
                        </div>
                    </section>

                    <section class="admin-form-section">
                        <h3>Alapértelmezett település (MyLocation)</h3>
                        <p class="admin-hint">Ez a vendégek, és a saját település nélküli felhasználók alaphelye a kezdőlapon és az index közelségi rendezésénél. A saját települést mindenki a felhasználói beállításokban állítja; a kereső és az index szűrője csak a találatokat szűri.</p>
                        <div class="admin-form">
                            <label for="my_location_slug">Település</label>
                            <select id="my_location_slug" name="my_location_slug" bind:value={siteSettings.my_location_slug}>
<option value="">Válassz...</option>
                                {#each settlementsForSelect as loc}
                                    <option value={loc.slug}>{loc.name}{loc.county ? ` (${loc.county})` : ''}{loc.type ? ` – ${loc.type}` : ''}</option>
                                {/each}
                            </select>
                            <div class="flex gap-md mt-md">
                                <button type="button" class="admin-submit-btn" on:click={() => saveSettings("location")} disabled={settingsSaving !== ""}>
                                    {settingsSaving === "location" ? 'Mentés…' : 'Mentés'}
                                </button>
                            </div>
                        </div>
                    </section>

                    <section class="admin-form-section">
                        <h3>Időjárás (Weather)</h3>
                        <div class="admin-form">
                            <span class="form-group-label">Szolgáltatók</span>
                            <p class="admin-form-hint">Fix sorrend: a MET Norway a fő forrás (kb. 9 nap, az archívum is ebből készül); a WeatherAPI.com és az OpenWeatherMap csak akkor kell, ha a MET órák óta nem válaszol. Csak üzleti használatra is ingyenes források; az Open-Meteo ezért került ki.</p>
                            <div class="flex gap-lg flex-wrap mb-lg">
                                <label class="flex items-center gap-xs font-normal">
                                    <input id="weather_provider_metno_enabled" name="weather_provider_metno_enabled" type="checkbox" checked={siteSettings.weather_provider_metno_enabled !== 'false'} on:change={(e) => siteSettings.weather_provider_metno_enabled = e.target.checked ? 'true' : 'false'} class="w-auto" />
                                    MET Norway
                                </label>
                                <label class="flex items-center gap-xs font-normal">
                                    <input id="weather_provider_weatherapi_enabled" name="weather_provider_weatherapi_enabled" type="checkbox" checked={siteSettings.weather_provider_weatherapi_enabled === 'true'} on:change={(e) => siteSettings.weather_provider_weatherapi_enabled = e.target.checked ? 'true' : 'false'} class="w-auto" />
                                    WeatherAPI.com
                                </label>
                                <label class="flex items-center gap-xs font-normal">
                                    <input id="weather_provider_openweathermap_enabled" name="weather_provider_openweathermap_enabled" type="checkbox" checked={siteSettings.weather_provider_openweathermap_enabled === 'true'} on:change={(e) => siteSettings.weather_provider_openweathermap_enabled = e.target.checked ? 'true' : 'false'} class="w-auto" />
                                    OpenWeatherMap
                                </label>
                            </div>

                            <label for="weather_icon_style">Időjárás ikon stílus</label>
                            <select id="weather_icon_style" name="weather_icon_style" bind:value={siteSettings.weather_icon_style}>
                                <option value="svg">Animált ikonok</option>
                                <option value="emoji">Emoji</option>
                            </select>
                            <p class="admin-form-hint">A kezdőlap, a település- és a megyeoldal időjárás-dobozára vonatkozik; az Időjárás oldalak mindig az animált ikonokat mutatják. Az összes ikon: <a href="{lamszaOrigin}/idojaras/ikonok" target="_blank" rel="noopener noreferrer">Lámsza › Időjárás-ikonok</a>.</p>

                            <label for="weather_cache_ttl">Böngésző-cache (perc)</label>
                            <input id="weather_cache_ttl" name="weather_cache_ttl_minutes" type="number" min="1" max="1440" bind:value={siteSettings.weather_cache_ttl_minutes} />

                            <div class="flex gap-md mt-md flex-wrap">
                                <button type="button" class="admin-submit-btn" on:click={() => saveSettings("weather")} disabled={settingsSaving !== ""}>
                                    {settingsSaving === "weather" ? 'Mentés…' : 'Mentés'}
                                </button>
                                <button type="button" class="btn-update" on:click={clearWeatherCache} disabled={settingsCacheClearing}>
                                    {settingsCacheClearing ? '…' : 'Időjárás cache törlése'}
                                </button>
                            </div>
                        </div>
                    </section>
                {/if}

                <!-- Időjárás fordítások (Weather translations) Tab -->
                {#if activeTab === "weather_translations"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Az időjárás API angol (vagy más) szövegeinek fordítása (hu, ro, de). Ha nincs egyedi sor,
                            a rendszer az alapértelmezett magyar megnevezést használja. Új sor: eredeti szöveg =
                            pontos egyezés a bejövő API szöveggel.
                        </p>
                    {/if}
                    {#if editingWeatherTrans}
                        <details class="admin-create-panel" open>
                            <summary class="admin-create-summary">Fordítás szerkesztése</summary>
                            <form class="admin-form admin-create-form" on:submit={saveWeatherTranslation} style="max-width: 28rem;">
                                <label for="wet_src">Eredeti szöveg (pl. API angol)</label>
                                <input id="wet_src" name="source_text" type="text" bind:value={editingWeatherTrans.source_text} required />
                                <label for="wet_lang">Nyelv</label>
                                <select id="wet_lang" name="lang" bind:value={editingWeatherTrans.lang}>
<option value="">Válassz...</option>
                                    {#each WEATHER_TRANS_LANGS as opt}
                                        <option value={opt.value}>{opt.label}</option>
                                    {/each}
                                </select>
                                <label for="wet_txt">Lefordított szöveg</label>
                                <input id="wet_txt" name="translated_text" type="text" bind:value={editingWeatherTrans.translated_text} required />
                                <div class="flex gap-md mt-md">
                                    <button type="submit" class="admin-submit-btn">Mentés</button>
                                    <button type="button" class="btn-update" on:click={cancelEditWeatherTrans}>Mégse</button>
                                </div>
                            </form>
                        </details>
                    {:else}
                        <details class="admin-create-panel">
                            <summary class="admin-create-summary"><span>Új fordítás</span><AppIcon name="plus" size={18} /></summary>
                        <form class="admin-form admin-create-form" on:submit={saveWeatherTranslation} style="max-width: 28rem;">
                            <label for="wt_src">Időjárás-kód (MET Norway, pl. partlycloudy, lightrainshowers)</label>
                            <input id="wt_src" name="source_text" type="text" bind:value={newWeatherTrans.source_text} required placeholder="pl. partlycloudy" />
                            <label for="wt_lang">Nyelv</label>
                            <select id="wt_lang" name="lang" bind:value={newWeatherTrans.lang}>
<option value="">Válassz...</option>
                                {#each WEATHER_TRANS_LANGS as opt}
                                    <option value={opt.value}>{opt.label}</option>
                                {/each}
                            </select>
                            <label for="wt_txt">Lefordított szöveg</label>
                            <input id="wt_txt" name="translated_text" type="text" bind:value={newWeatherTrans.translated_text} required placeholder="pl. borult" />
                            <button type="submit" class="admin-submit-btn">Hozzáadás</button>
                        </form>
                        </details>
                    {/if}
                    {@render adminNotice("weather_translations")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_weather_trans"
                                name="search_weather_trans"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchWeatherTrans}
                                on:input={() => (pageWeatherTrans = 1)}
                                placeholder="Szöveg, nyelv…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgWeatherTrans.total}
                        page={pgWeatherTrans.page}
                        totalPages={pgWeatherTrans.totalPages}
                        from={pgWeatherTrans.from}
                        to={pgWeatherTrans.to}
                        on:prev={() =>
                            (pageWeatherTrans = Math.max(
                                1,
                                pageWeatherTrans - 1,
                            ))}
                        on:next={() =>
                            (pageWeatherTrans = Math.min(
                                pgWeatherTrans.totalPages,
                                pageWeatherTrans + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Eredeti</th>
                                    <th>Nyelv</th>
                                    <th>Fordítás</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgWeatherTrans.rows as wt}
                                    <tr>
                                        <td>{wt.source_text}</td>
                                        <td>{wt.lang}</td>
                                        <td>{wt.translated_text}</td>
                                        <td><button type="button" class="btn-update" on:click={() => startEditWeatherTrans(wt)}>Szerk.</button></td>
                                        <td><button type="button" class="btn-delete" on:click={() => deleteWeatherTranslation(wt.id)}>Törlés</button></td>
                                    </tr>
                                {:else}
                                    <tr><td colspan="5">Nincs egyéni fordítás. Az alapértelmezett magyar szavak érvényesek.</td></tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgWeatherTrans.total}
                        page={pgWeatherTrans.page}
                        totalPages={pgWeatherTrans.totalPages}
                        from={pgWeatherTrans.from}
                        to={pgWeatherTrans.to}
                        on:prev={() =>
                            (pageWeatherTrans = Math.max(
                                1,
                                pageWeatherTrans - 1,
                            ))}
                        on:next={() =>
                            (pageWeatherTrans = Math.min(
                                pgWeatherTrans.totalPages,
                                pageWeatherTrans + 1,
                            ))}
                    />
                {/if}

                <!-- Oldalak (Pages) Tab -->
                {#if activeTab === "pages"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {/if}
                    {#if editingPage}
                        {@render adminNotice("pages")}
                        <h3>Oldal szerkesztése: {editingPage.title}</h3>
                        <form class="admin-form" on:submit|preventDefault={savePage} style="max-width: 48rem;">
                            <label for="page_title">Cím</label>
                            <input id="page_title" name="title" type="text" bind:value={editingPage.title} required />

                            <label for="page_greeting">Bevezető (a cím alatt, nyilvános oldalakon)</label>
                            <textarea id="page_greeting" name="greeting" bind:value={editingPage.greeting} rows="3" placeholder="Rövid bevezető szöveg…"></textarea>

                            <label for="page_content">Tartalom (HTML)</label>
                            <textarea id="page_content" name="content" bind:value={editingPage.content} rows="20" class="input-mono"></textarea>

                            <div class="flex gap-md mt-md">
                                <button type="submit" class="admin-submit-btn" disabled={pageSaving}>
                                    {pageSaving ? 'Mentés…' : 'Mentés'}
                                </button>
                                <button type="button" class="btn-update" on:click={cancelEditPage}>Mégse</button>
                            </div>
                        </form>
                    {:else}
                        {#if !adminTabError}
                            <p class="admin-info">
                                Statikus oldalak: <strong>cím</strong>, <strong>bevezető</strong> (a főcím alatt) és <strong>HTML tartalom</strong> (pl. irányelvek).
                            </p>
                        {/if}
                        <h3 class="admin-subtab-heading">Irányelvek és statikus oldalak</h3>
                        {@render adminNotice("pages")}
                        <div class="admin-table-toolbar">
                            <label class="admin-search-label"
                                >Keresés (oldalak)
                                <input
                                    id="search_admin_pages"
                                    name="search_admin_pages"
                                    type="search"
                                    class="admin-search-input"
                                    bind:value={searchAdminPages}
                                    on:input={() => (pageAdminPages = 1)}
                                    placeholder="Slug, cím…"
                                /></label
                            >
                        </div>
                        <AdminPaginationBar
                            total={pgAdminPages.total}
                            page={pgAdminPages.page}
                            totalPages={pgAdminPages.totalPages}
                            from={pgAdminPages.from}
                            to={pgAdminPages.to}
                            on:prev={() =>
                                (pageAdminPages = Math.max(1, pageAdminPages - 1))}
                            on:next={() =>
                                (pageAdminPages = Math.min(
                                    pgAdminPages.totalPages,
                                    pageAdminPages + 1,
                                ))}
                        />
                        <div class="admin-table-wrapper">
                            <table class="admin-table">
                                <thead>
                                    <tr>
                                        <th>Slug</th>
                                        <th>Cím</th>
                                        <th>Bevezető</th>
                                        <th>Utolsó módosítás</th>
                                        <th class="admin-table-col--action">Szerk.</th>
                                        <th class="admin-table-col--action">Törlés</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each pgAdminPages.rows as pg}
                                        <tr>
                                            <td><a href={pg.slug === 'home' ? '/' : '/' + pg.slug} target="_blank">{pg.slug === 'home' ? '/' : '/' + pg.slug}</a></td>
                                            <td>{pg.title}</td>
                                            <td class="admin-table-cell-preview" title={pg.greeting || ''}>{pg.greeting ? contentPreview(pg.greeting) : '-'}</td>
                                            <td>{pg.updated_at ? pg.updated_at.slice(0, 19) : ''}</td>
                                            <td class="admin-table-col--action"><button type="button" class="btn-update" on:click={() => startEditPage(pg)}>Szerk.</button></td>
                                            <td class="admin-table-col--action admin-table-col--action--muted">-</td>
                                        </tr>
                                    {:else}
                                        <tr
                                            ><td colspan="6"
                                                >{adminPages?.length
                                                    ? "Nincs találat a keresésre."
                                                    : "Nincsenek oldalak."}</td
                                            ></tr
                                        >
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                        <AdminPaginationBar
                            total={pgAdminPages.total}
                            page={pgAdminPages.page}
                            totalPages={pgAdminPages.totalPages}
                            from={pgAdminPages.from}
                            to={pgAdminPages.to}
                            on:prev={() =>
                                (pageAdminPages = Math.max(1, pageAdminPages - 1))}
                            on:next={() =>
                                (pageAdminPages = Math.min(
                                    pgAdminPages.totalPages,
                                    pageAdminPages + 1,
                                ))}
                        />

                    {/if}
                {/if}

                <!-- GYIK Tab -->
                {#if activeTab === "page_faq"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {/if}
                    {#if editingPageFaq}
                        {@render adminNotice("page_faq")}
                        <h3>GYIK / disclaimer: {editingPageFaq.label_hu || editingPageFaq.section_key}</h3>
                        <p class="admin-info">
                            Kulcs: <code>{editingPageFaq.section_key}</code> - a nyilvános oldalon a
                            <code>PageFaqDisclaimer</code> ugyanazt a HTML-struktúrát használja (<code>.faq</code>,
                            <code>details.faq-item</code>, <code>#disclaimer</code>, <code>.note.info</code>). Minden
                            blokk egy külön kérdés / válasz pár.
                        </p>
                        <form class="admin-form" on:submit|preventDefault={savePageFaq} style="max-width: 52rem;">
                            <label for="pfaq_label">Megjelenített név (admin)</label>
                            <input id="pfaq_label" name="label_hu" type="text" bind:value={editingPageFaq.label_hu} />

                            <label for="pfaq_title">GYIK szekció címe (H2)</label>
                            <input id="pfaq_title" name="faq_title" type="text" bind:value={editingPageFaq.faq_title} placeholder="pl. Hogyan működik ez az oldal?" />

                            <div class="admin-faq-toolbar">
                                <span class="admin-faq-toolbar-label">Kérdések és válaszok</span>
                                <button type="button" class="btn-update btn-sm" on:click={addFaqItem}
                                    >+ Új kérdés</button
                                >
                            </div>

                            {#each editingPageFaq.faq_items || [] as item, i (i)}
                                <details class="admin-faq-pair" open>
                                    <summary>Kérdés {i + 1}</summary>
                                    <div class="admin-faq-pair-fields">
                                        <label for={"pfaq_q_" + i}>Kérdés (summary)</label>
                                        <input
                                            id={"pfaq_q_" + i}
                                            name={"faq_question_" + i}
                                            type="text"
                                            bind:value={editingPageFaq.faq_items[i].question}
                                            placeholder="Rövid kérdés"
                                        />
                                        <label for={"pfaq_a_" + i}>Válasz (Markdown)</label>
                                        <textarea
                                            id={"pfaq_a_" + i}
                                            name={"faq_answer_" + i}
                                            bind:value={editingPageFaq.faq_items[i].answer}
                                            rows="5"
                                            class="input-mono"
                                            placeholder="Válasz szövege…"
                                        ></textarea>
                                        <button
                                            type="button"
                                            class="btn-delete btn-sm"
                                            on:click={() => removeFaqItem(i)}>Kérdés törlése</button
                                        >
                                    </div>
                                </details>
                            {:else}
                                <p class="admin-info">Még nincs kérdés - kattints az „Új kérdés” gombra.</p>
                            {/each}

                            <label for="pfaq_disc">Disclaimer (Markdown)</label>
                            <textarea id="pfaq_disc" bind:value={editingPageFaq.disclaimer_markdown} rows="8" class="input-mono"></textarea>

                            <div class="flex gap-md mt-md">
                                <button type="submit" class="admin-submit-btn" disabled={pageFaqSaving}>
                                    {pageFaqSaving ? 'Mentés…' : 'Mentés'}
                                </button>
                                <button type="button" class="btn-update" on:click={cancelEditPageFaq}>Mégse</button>
                            </div>
                        </form>
                    {:else}
                        <h3 class="admin-subtab-heading">GYIK és felelősségkizárások</h3>
                        <p class="admin-info">
                            Ugyanaz a kinézet, mint a <code>/hirek</code> oldalon: <code>.faq</code>,
                            <code>.faq-title</code>, <code>.faq-list</code>, <code>.faq-item</code>, <code>#disclaimer</code>,
                            <code>.note.info</code>.
                        </p>
                        {@render adminNotice("page_faq")}
                        <div class="admin-table-toolbar">
                            <label class="admin-search-label"
                                >Keresés (GYIK)
                                <input
                                    id="search_page_faq"
                                    name="search_page_faq"
                                    type="search"
                                    class="admin-search-input"
                                    bind:value={searchPageFaqRows}
                                    on:input={() => (pagePageFaqRows = 1)}
                                    placeholder="Kulcs, név, cím…"
                                /></label
                            >
                        </div>
                        <AdminPaginationBar
                            total={pgPageFaqRows.total}
                            page={pgPageFaqRows.page}
                            totalPages={pgPageFaqRows.totalPages}
                            from={pgPageFaqRows.from}
                            to={pgPageFaqRows.to}
                            on:prev={() =>
                                (pagePageFaqRows = Math.max(1, pagePageFaqRows - 1))}
                            on:next={() =>
                                (pagePageFaqRows = Math.min(
                                    pgPageFaqRows.totalPages,
                                    pagePageFaqRows + 1,
                                ))}
                        />
                        <div class="admin-table-wrapper">
                            <table class="admin-table">
                                <thead>
                                    <tr>
                                        <th>Kulcs</th>
                                        <th>Megjelenített név</th>
                                        <th>GYIK cím</th>
                                        <th>Kérdések</th>
                                        <th>Utolsó módosítás</th>
                                        <th class="admin-table-col--action">Szerk.</th>
                                        <th class="admin-table-col--action">Törlés</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each pgPageFaqRows.rows as row}
                                        <tr>
                                            <td><code>{row.section_key}</code></td>
                                            <td>{row.label_hu}</td>
                                            <td class="admin-table-cell-preview" title={row.faq_title || ''}>{contentPreview(row.faq_title || "")}</td>
                                            <td>{(row.faq_items || []).length}</td>
                                            <td>{row.updated_at ? row.updated_at.slice(0, 19) : ''}</td>
                                            <td class="admin-table-col--action"><button type="button" class="btn-update" on:click={() => startEditPageFaq(row)}>Szerk.</button></td>
                                            <td class="admin-table-col--action admin-table-col--action--muted">-</td>
                                        </tr>
                                    {:else}
                                        <tr
                                            ><td colspan="7"
                                                >{pageFaqSections?.length
                                                    ? "Nincs találat a keresésre."
                                                    : "Nincs GYIK rekord (futtasd a backend migrációt)."}</td
                                            ></tr
                                        >
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                        <AdminPaginationBar
                            total={pgPageFaqRows.total}
                            page={pgPageFaqRows.page}
                            totalPages={pgPageFaqRows.totalPages}
                            from={pgPageFaqRows.from}
                            to={pgPageFaqRows.to}
                            on:prev={() =>
                                (pagePageFaqRows = Math.max(1, pagePageFaqRows - 1))}
                            on:next={() =>
                                (pagePageFaqRows = Math.min(
                                    pgPageFaqRows.totalPages,
                                    pagePageFaqRows + 1,
                                ))}
                        />
                    {/if}
                {/if}

                <!-- Entry Types Tab -->
                {#if activeTab === "entry_types"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Bejegyzés <strong>típusok</strong> (pl. entry, business): belső címkék a bejegyzések
                            szerkezetéhez és szűréséhez - nem ugyanaz, mint a kategória.
                        </p>
                    {/if}
                    {@render adminNotice("entry_types")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_entry_types"
                                name="search_entry_types"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchEntryTypes}
                                on:input={() => (pageEntryTypes = 1)}
                                placeholder="Név, ID…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgEntryTypes.total}
                        page={pgEntryTypes.page}
                        totalPages={pgEntryTypes.totalPages}
                        from={pgEntryTypes.from}
                        to={pgEntryTypes.to}
                        on:prev={() =>
                            (pageEntryTypes = Math.max(1, pageEntryTypes - 1))}
                        on:next={() =>
                            (pageEntryTypes = Math.min(
                                pgEntryTypes.totalPages,
                                pageEntryTypes + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Név</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgEntryTypes.rows as et}
                                    <tr>
                                        <td>{et.id}</td>
                                        <td>{et.name}</td>
                                    </tr>
                                {:else}
                                    <tr
                                        ><td colspan="2">Nincsenek típusok.</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgEntryTypes.total}
                        page={pgEntryTypes.page}
                        totalPages={pgEntryTypes.totalPages}
                        from={pgEntryTypes.from}
                        to={pgEntryTypes.to}
                        on:prev={() =>
                            (pageEntryTypes = Math.max(1, pageEntryTypes - 1))}
                        on:next={() =>
                            (pageEntryTypes = Math.min(
                                pgEntryTypes.totalPages,
                                pageEntryTypes + 1,
                            ))}
                    />
                {/if}

                <!-- Tags Tab -->
                {#if activeTab === "tags"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Címkék a bejegyzéseken, a listaszűrőben és a keresésben. A törlés leveszi a címkét a bejegyzésekről. A bejegyzés megmarad.
                        </p>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új címke</span><AppIcon name="plus" size={18} /></summary>
                        <form class="admin-form admin-create-form" on:submit={submitTag}>
                            <label for="tag_name">Címke neve</label>
                            <input
                                id="tag_name"
                                name="name"
                                type="text"
                                bind:value={newTag.name}
                                required
                            />
                            <button type="submit" class="admin-submit-btn">Hozzáadás</button>
                        </form>
                    </details>
                    {@render adminNotice("tags")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label">
                            <span class="admin-search-heading">Keresés</span>
                            <input
                                id="search_tags"
                                name="search_tags"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchTags}
                                on:input={() => (pageTags = 1)}
                                placeholder="Név, ID…"
                            />
                        </label>
                    </div>
                    <AdminPaginationBar
                        total={pgTags.total}
                        page={pgTags.page}
                        totalPages={pgTags.totalPages}
                        from={pgTags.from}
                        to={pgTags.to}
                        on:prev={() => (pageTags = Math.max(1, pageTags - 1))}
                        on:next={() => (pageTags = Math.min(pgTags.totalPages, pageTags + 1))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Név</th>
                                    <th>Használat</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgTags.rows as tag}
                                    <tr>
                                        <td>{tag.id}</td>
                                        <td>{tag.name}</td>
                                        <td>{tag.usage}</td>
                                        <td>
                                            <button class="btn-update" on:click={() => startEditTag(tag)}>Szerk.</button>
                                        </td>
                                        <td>
                                            <button class="btn-delete" on:click={() => deleteRecord("tags", tag.id, fetchTags)}>Törlés</button>
                                        </td>
                                    </tr>
                                {:else}
                                    <tr><td colspan="5">Nincsenek címkék.</td></tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgTags.total}
                        page={pgTags.page}
                        totalPages={pgTags.totalPages}
                        from={pgTags.from}
                        to={pgTags.to}
                        on:prev={() => (pageTags = Math.max(1, pageTags - 1))}
                        on:next={() => (pageTags = Math.min(pgTags.totalPages, pageTags + 1))}
                    />
                {/if}

                <!-- Attractions Tab -->
                {#if activeTab === "attractions"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            Megyéhez kötött látnivalók (természet, kultúra): név, slug, rövid leírás, koordináták,
                            kiemelt kép és bővebb tartalom (Markdown). A megye és település oldalakon jelennek meg.
                            A nyilvános oldalról érkező módosítási javaslatokat itt lehet elfogadni vagy elutasítani.
                        </p>
                    {/if}
                    {#if attractionSuggestions.length}
                        <div class="admin-form mb-lg">
                            <h3>Nyitott látnivaló-javaslatok</h3>
                            {#each attractionSuggestions as suggestion (suggestion.id)}
                                <article class="admin-info">
                                    <p>
                                        <strong>{suggestion.attraction_name}</strong>
                                        · {suggestion.user_name}
                                        {#if suggestion.created_at}· {suggestion.created_at}{/if}
                                    </p>
                                    {#each Object.entries(suggestion.changes || {}) as [key, value]}
                                        <p>{key}: {suggestionValueText(value)}</p>
                                    {/each}
                                    {#if suggestion.note}
                                        <p>Megjegyzés: {suggestion.note}</p>
                                    {/if}
                                    <div class="modal-actions">
                                        <button type="button" class="admin-submit-btn" on:click={() => decideAttractionSuggestion(suggestion.id, "accept")}>Elfogadás</button>
                                        <button type="button" class="btn-delete" on:click={() => decideAttractionSuggestion(suggestion.id, "deny")}>Elutasítás</button>
                                    </div>
                                </article>
                            {/each}
                        </div>
                    {/if}
                    <details class="admin-create-panel">
                        <summary class="admin-create-summary"><span>Új látnivaló</span><AppIcon name="plus" size={18} /></summary>
                    <form class="admin-form admin-create-form mb-lg" on:submit|preventDefault={submitNewAttraction}>
                        <div class="form-row">
                            <label for="att_county">Megye</label>
                            <select id="att_county" name="county_slug" bind:value={newAttraction.county_slug}>
<option value="">Válassz...</option>
                                <option value="hargita">Hargita</option>
                                <option value="kovaszna">Kovászna</option>
                                <option value="maros">Maros</option>
                            </select>
                        </div>
                        <div class="form-row">
                            <label for="att_name">Név</label>
                            <input id="att_name" name="name" type="text" bind:value={newAttraction.name} required placeholder="pl. Szent Anna-tó" />
                        </div>
                        <div class="form-row">
                            <label for="att_desc">Rövid leírás</label>
                            <input id="att_desc" name="description" type="text" bind:value={newAttraction.description} placeholder="Közép-Európa egyetlen vulkanikus tava..." />
                        </div>
                        <div class="form-row">
                            <label for="att_coords">Koordináták (lat, lon)</label>
                            <input id="att_coords" name="latitude" type="text" bind:value={newAttraction.latitude} placeholder="46.1265" style="width:6rem" />
                            <input id="att_lon" name="longitude" type="text" bind:value={newAttraction.longitude} placeholder="25.8876" style="width:6rem" />
                        </div>
                        <div class="form-row">
                            <label for="att_elevation">Magasság (m)</label>
                            <input id="att_elevation" name="elevation_m" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={newAttraction.elevation_m} placeholder="946" style="width:6rem" />
                        </div>
                        <div class="form-row">
                            <label for="att_area">Felszín (km²)</label>
                            <input id="att_area" name="area_km2" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={newAttraction.area_km2} placeholder="0,22" style="width:6rem" />
                        </div>
                        <div class="form-row">
                            <label for="att_depth">Mélység (m)</label>
                            <input id="att_depth" name="depth_m" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={newAttraction.depth_m} placeholder="7" style="width:6rem" />
                        </div>
                        <div class="form-row">
                            <label for="att_featured">Kiemelt kép URL</label>
                            <input id="att_featured" name="featured_image" type="url" bind:value={newAttraction.featured_image} placeholder="https://..." />
                        </div>
                        <div class="form-row">
                            <label for="att_featured_copyright">Szerzői jog (kiemelt kép)</label>
                            <input id="att_featured_copyright" name="featured_image_copyright" type="text" bind:value={newAttraction.featured_image_copyright} placeholder="pl. Iliuta Goean" />
                        </div>
                        <div class="form-row">
                            <label for="att_activities">Tevékenységek / aktivitások (soronként egy)</label>
                            <textarea id="att_activities" name="activities" bind:value={newAttraction.activities} rows="4" placeholder="Túrázás&#10;Fürdés"></textarea>
                        </div>
                        <div class="form-row">
                            <label for="att_prohibitions">Mit nem szabad itt csinálni? (soronként egy)</label>
                            <textarea id="att_prohibitions" name="prohibitions" bind:value={newAttraction.prohibitions} rows="4" placeholder="Szemetelés&#10;Tűzgyújtás"></textarea>
                        </div>
                        <div class="form-row">
                            <label for="att_content">Tartalom (Markdown)</label>
                            <textarea id="att_content" name="content" bind:value={newAttraction.content} rows="6" placeholder="## Cím&#10;Szöveg..."></textarea>
                        </div>
                        <div class="form-row">
                            <label for="att_images">Galéria URL-ek (soronként egy)</label>
                            <textarea id="att_images" name="images" bind:value={newAttraction.images} rows="3" placeholder="https://kep1.jpg&#10;https://kep2.jpg"></textarea>
                        </div>
                        <div class="form-row">
                            <label for="att_image_copyrights">Galéria szerzői jog (ugyanabban a sorrendben, soronként)</label>
                            <textarea id="att_image_copyrights" name="image_copyrights" bind:value={newAttraction.image_copyrights} rows="3" placeholder="Iliuta Goean"></textarea>
                        </div>
                        <button type="submit" class="admin-submit-btn">Hozzáadás</button>
                    </form>
                    </details>
                    {@render adminNotice("attractions")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés
                            <input
                                id="search_attractions"
                                name="search_attractions"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchAttractions}
                                on:input={() => (pageAttractions = 1)}
                                placeholder="Név, slug, megye…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgAttractions.total}
                        page={pgAttractions.page}
                        totalPages={pgAttractions.totalPages}
                        from={pgAttractions.from}
                        to={pgAttractions.to}
                        on:prev={() =>
                            (pageAttractions = Math.max(1, pageAttractions - 1))}
                        on:next={() =>
                            (pageAttractions = Math.min(
                                pgAttractions.totalPages,
                                pageAttractions + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Név</th>
                                    <th>Megye</th>
                                    <th>Slug</th>
                                    <th>Rövid leírás</th>
                                    <th>Koordináták</th>
                                    <th>Kiemelt kép</th>
                                    <th>Tartalom (előnézet)</th>
                                    <th>Galéria</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgAttractions.rows as att}
                                    <tr>
                                        <td>{att.name}</td>
                                        <td>{att.county_name}</td>
                                        <td><code>{att.slug}</code></td>
                                        <td class="admin-table-cell-preview" title={att.description || ""}>{contentPreview(att.description)}</td>
                                        <td class="admin-table__mono">{formatLatLon(att.latitude, att.longitude)}</td>
                                        <td class="admin-table-cell-preview" title={att.featured_image || ""}>{urlPreview(att.featured_image)}</td>
                                        <td class="admin-table-cell-preview" title={att.content || ""}>{contentPreview(att.content)}</td>
                                        <td>{(att.images && att.images.length) || 0} kép</td>
                                        <td class="admin-table-col--action">
                                            <button type="button" class="btn-update" on:click={() => openEditAttraction(att)}>Szerk.</button>
                                        </td>
                                        <td class="admin-table-col--action">
                                            <button type="button" class="btn-delete" on:click={() => deleteAttraction(att.id)}>Törlés</button>
                                        </td>
                                    </tr>
                                {:else}
                                    <tr><td colspan="10">Nincsenek látnivalók.</td></tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgAttractions.total}
                        page={pgAttractions.page}
                        totalPages={pgAttractions.totalPages}
                        from={pgAttractions.from}
                        to={pgAttractions.to}
                        on:prev={() =>
                            (pageAttractions = Math.max(1, pageAttractions - 1))}
                        on:next={() =>
                            (pageAttractions = Math.min(
                                pgAttractions.totalPages,
                                pageAttractions + 1,
                            ))}
                    />
                {/if}

                <!-- Counties Tab -->
                {#if activeTab === "counties"}
                    {#if adminTabError && activeTab === adminTabError.tab}
                        <div class="info-box error" role="alert">
                            <p>{adminTabError.message}</p>
                        </div>
                    {:else}
                        <p class="admin-info">
                            <strong>Megyék:</strong> magyar / román / név név, URL-slug, bemutatkozó szöveg (Markdown),
                            és a <strong>megyeszékhely</strong> település kiválasztása a listából.
                            <strong>Történelmi székek</strong> (pl. Csíkszék): külön név, slug és tartalom -
                            a <code>/szekek</code> oldalakon jelennek meg.
                        </p>
                    {/if}

                    <h3 class="admin-region-heading">Megyék</h3>
                    {@render adminNotice("counties")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (megyék)
                            <input
                                id="search_counties"
                                name="search_counties"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchCounties}
                                on:input={() => (pageCounties = 1)}
                                placeholder="Név, slug, székhely…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgCounties.total}
                        page={pgCounties.page}
                        totalPages={pgCounties.totalPages}
                        from={pgCounties.from}
                        to={pgCounties.to}
                        on:prev={() =>
                            (pageCounties = Math.max(1, pageCounties - 1))}
                        on:next={() =>
                            (pageCounties = Math.min(
                                pgCounties.totalPages,
                                pageCounties + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Megye</th>
                                    <th>Név (RO)</th>
                                    <th>Név (DE)</th>
                                    <th>Slug</th>
                                    <th>Megyeszékhely</th>
                                    <th>Tartalom (előnézet)</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgCounties.rows as c (c.id)}
                                    {#if editingCounty?.id === c.id}
                                        <tr class="admin-table-edit-row">
                                            <td colspan="8">
                                                <div class="admin-region-edit-panel">
                                                    <div class="admin-region-edit-grid">
                                                        <label for="county_edit_name">
                                                            Megye (HU)
                                                            <input id="county_edit_name" name="name" type="text" bind:value={editingCounty.name} />
                                                        </label>
                                                        <label for="county_edit_name_ro">
                                                            Név (RO)
                                                            <input id="county_edit_name_ro" name="name_ro" type="text" bind:value={editingCounty.name_ro} />
                                                        </label>
                                                        <label for="county_edit_name_de">
                                                            Név (DE)
                                                            <input id="county_edit_name_de" name="name_de" type="text" bind:value={editingCounty.name_de} />
                                                        </label>
                                                        <label for="county_edit_slug">
                                                            Slug (URL)
                                                            <input id="county_edit_slug" name="slug" type="text" bind:value={editingCounty.slug} placeholder="pl. hargita" />
                                                        </label>
                                                        <label class="admin-region-edit-span2" for="county_edit_seat_location_id">
                                                            Megyeszékhely
                                                            <select id="county_edit_seat_location_id" name="seat_location_id" bind:value={editingCounty.seat_location_id}>
<option value="">Válassz...</option>
                                                                {#each settlementsForCountyName(c.name) as loc (loc.id)}
                                                                    <option value={String(loc.id)}
                                                                        >{loc.name} ({loc.type}){loc.name_ro ? " - " + loc.name_ro : ""}</option
                                                                    >
                                                                {/each}
                                                            </select>
                                                        </label>
                                                    </div>
                                                    <label class="admin-region-edit-full" for="county_edit_content">
                                                        Bemutatkozás (Markdown)
                                                        <textarea
                                                            id="county_edit_content"
                                                            name="content"
                                                            rows="10"
                                                            bind:value={editingCounty.content}
                                                            placeholder="## Bevezető&#10;..."
                                                        ></textarea>
                                                    </label>
                                                    <div class="admin-region-edit-actions">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={saveEditingCounty}>Mentés</button
                                                        >
                                                        <button
                                                            type="button"
                                                            class="btn-update"
                                                            on:click={cancelEditCounty}>Mégse</button
                                                        >
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    {:else}
                                        <tr>
                                            <td><strong>{c.name}</strong></td>
                                            <td>{c.name_ro || "-"}</td>
                                            <td>{c.name_de || "-"}</td>
                                            <td><code>{c.slug}</code></td>
                                            <td>{countySeatDisplayName(c)}</td>
                                            <td class="admin-table-cell-preview" title={c.content || ""}
                                                >{contentPreview(c.content)}</td
                                            >
                                            <td class="admin-table-col--action">
                                                <button
                                                    type="button"
                                                    class="btn-update"
                                                    on:click={() => startEditCounty(c)}>Szerk.</button
                                                >
                                            </td>
                                            <td class="admin-table-col--action admin-table-col--action--muted">-</td>
                                        </tr>
                                    {/if}
                                {:else}
                                    <tr
                                        ><td colspan="8"
                                            >{countiesFromAPI?.length
                                                ? "Nincs találat a keresésre."
                                                : "Nincs megye-adat (API / migráció)."}</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgCounties.total}
                        page={pgCounties.page}
                        totalPages={pgCounties.totalPages}
                        from={pgCounties.from}
                        to={pgCounties.to}
                        on:prev={() =>
                            (pageCounties = Math.max(1, pageCounties - 1))}
                        on:next={() =>
                            (pageCounties = Math.min(
                                pgCounties.totalPages,
                                pageCounties + 1,
                            ))}
                    />

                    <h3 class="admin-region-heading">Történelmi székek</h3>
                    <p class="admin-hint">
                        Megjelenés: <a href="/szekek" target="_blank" rel="noopener">/szekek</a> és
                        <code>/szekek/…</code> oldalak.
                    </p>
                    {@render adminNotice("historical_seats")}
                    <div class="admin-table-toolbar">
                        <label class="admin-search-label"
                            >Keresés (székek)
                            <input
                                id="search_historical_seats"
                                name="search_historical_seats"
                                type="search"
                                class="admin-search-input"
                                bind:value={searchHistoricalSeats}
                                on:input={() => (pageHistoricalSeats = 1)}
                                placeholder="Név, slug…"
                            /></label
                        >
                    </div>
                    <AdminPaginationBar
                        total={pgHistoricalSeats.total}
                        page={pgHistoricalSeats.page}
                        totalPages={pgHistoricalSeats.totalPages}
                        from={pgHistoricalSeats.from}
                        to={pgHistoricalSeats.to}
                        on:prev={() =>
                            (pageHistoricalSeats = Math.max(
                                1,
                                pageHistoricalSeats - 1,
                            ))}
                        on:next={() =>
                            (pageHistoricalSeats = Math.min(
                                pgHistoricalSeats.totalPages,
                                pageHistoricalSeats + 1,
                            ))}
                    />
                    <div class="admin-table-wrapper">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Név (HU)</th>
                                    <th>Név (RO)</th>
                                    <th>Név (DE)</th>
                                    <th>Slug</th>
                                    <th>Tartalom (előnézet)</th>
                                    <th class="admin-table-col--action">Szerk.</th>
                                    <th class="admin-table-col--action">Törlés</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pgHistoricalSeats.rows as h (h.id)}
                                    {#if editingHistoricalSeat?.id === h.id}
                                        <tr class="admin-table-edit-row">
                                            <td colspan="7">
                                                <div class="admin-region-edit-panel">
                                                    <div class="admin-region-edit-grid">
                                                        <label for="hseat_edit_name">
                                                            Név (HU)
                                                            <input id="hseat_edit_name" name="name" type="text" bind:value={editingHistoricalSeat.name} />
                                                        </label>
                                                        <label for="hseat_edit_name_ro">
                                                            Név (RO)
                                                            <input id="hseat_edit_name_ro" name="name_ro" type="text" bind:value={editingHistoricalSeat.name_ro} />
                                                        </label>
                                                        <label for="hseat_edit_name_de">
                                                            Név (DE)
                                                            <input id="hseat_edit_name_de" name="name_de" type="text" bind:value={editingHistoricalSeat.name_de} />
                                                        </label>
                                                        <label for="hseat_edit_slug">
                                                            Slug (URL)
                                                            <input id="hseat_edit_slug" name="slug" type="text" bind:value={editingHistoricalSeat.slug} placeholder="pl. csikszek" />
                                                        </label>
                                                    </div>
                                                    <label class="admin-region-edit-full" for="hseat_edit_content">
                                                        Tartalom (Markdown)
                                                        <textarea
                                                            id="hseat_edit_content"
                                                            name="content"
                                                            rows="10"
                                                            bind:value={editingHistoricalSeat.content}
                                                            placeholder="## …&#10;..."
                                                        ></textarea>
                                                    </label>
                                                    <div class="admin-region-edit-actions">
                                                        <button
                                                            type="button"
                                                            class="admin-submit-btn"
                                                            on:click={saveEditingHistoricalSeat}>Mentés</button
                                                        >
                                                        <button
                                                            type="button"
                                                            class="btn-update"
                                                            on:click={cancelEditHistoricalSeat}>Mégse</button
                                                        >
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    {:else}
                                        <tr>
                                            <td><strong>{h.name}</strong></td>
                                            <td>{h.name_ro || "-"}</td>
                                            <td>{h.name_de || "-"}</td>
                                            <td><code>{h.slug}</code></td>
                                            <td class="admin-table-cell-preview" title={h.content || ""}
                                                >{contentPreview(h.content)}</td
                                            >
                                            <td class="admin-table-col--action">
                                                <button
                                                    type="button"
                                                    class="btn-update"
                                                    on:click={() => startEditHistoricalSeat(h)}>Szerk.</button
                                                >
                                            </td>
                                            <td class="admin-table-col--action admin-table-col--action--muted">-</td>
                                        </tr>
                                    {/if}
                                {:else}
                                    <tr
                                        ><td colspan="7"
                                            >{historicalSeatsFromAPI?.length
                                                ? "Nincs találat a keresésre."
                                                : "Nincs szék-adat (API / migráció)."}</td
                                        ></tr
                                    >
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <AdminPaginationBar
                        total={pgHistoricalSeats.total}
                        page={pgHistoricalSeats.page}
                        totalPages={pgHistoricalSeats.totalPages}
                        from={pgHistoricalSeats.from}
                        to={pgHistoricalSeats.to}
                        on:prev={() =>
                            (pageHistoricalSeats = Math.max(
                                1,
                                pageHistoricalSeats - 1,
                            ))}
                        on:next={() =>
                            (pageHistoricalSeats = Math.min(
                                pgHistoricalSeats.totalPages,
                                pageHistoricalSeats + 1,
                            ))}
                    />
                {/if}
    {#snippet overlays()}

    <!-- Edit QuickLink Modal -->
    {#if editingLink}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditLink}
            on:keydown={(e) => e.key === "Escape" && cancelEditLink()}
        >
            <div class="link-dialog admin-modal">
                <h3>Gyorslink szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditLink}
                >
                    <label for="elink_title">Cím</label>
                    <input
                        id="elink_title"
                        type="text"
                        bind:value={editingLink.title}
                        required
                    />

                    <label for="elink_url">URL</label>
                    <input
                        id="elink_url"
                        type="url"
                        bind:value={editingLink.url}
                        required
                    />

                    <label for="elink_color">Háttérszín</label>
                    <input
                        id="elink_color"
                        type="text"
                        bind:value={editingLink.bg_color}
                    />

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditLink}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit News Modal -->
    {#if editingNews}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditNews}
            on:keydown={(e) => e.key === "Escape" && cancelEditNews()}
        >
            <div class="link-dialog admin-modal">
                <h3>Hírfolyam szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditNews}
                >
                    <label for="enews_title">Hírportál neve</label>
                    <input
                        id="enews_title"
                        type="text"
                        bind:value={editingNews.title}
                        required
                    />

                    <label for="enews_url">RSS URL</label>
                    <input
                        id="enews_url"
                        type="url"
                        bind:value={editingNews.feed_url}
                        required
                    />

                    <label for="enews_color">Háttérszín</label>
                    <input
                        id="enews_color"
                        type="text"
                        bind:value={editingNews.bg_color}
                    />

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditNews}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Entry Modal -->
    {#if editingEntry}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={closeEdit}
            on:keydown={(e) => e.key === "Escape" && closeEdit()}
        >
            <div class="link-dialog admin-modal">
                <h3>Bejegyzés szerkesztése</h3>
                <form class="admin-form" on:submit|preventDefault={saveEdit}>
                    <label for="edit_type">Típus</label>
                    <select id="edit_type" bind:value={editingEntry.type}>
<option value="">Válassz...</option>
                        {#each entryTypes as t}<option value={t.name}
                                >{t.name}</option
                            >{/each}
                    </select>

                    <label for="edit_loc">Település</label>
                    <select
                        id="edit_loc"
                        bind:value={editingEntry.location_id}
                        required
                    >
                        <option value="">Válassz...</option>
                        {#each settlementsForSelect as loc}
                            <option value={loc.id}
                                >{loc.name} ({loc.county})</option
                            >
                        {/each}
                    </select>

                    <CategoryMultiSelect
                        parents={entryCategoryParents}
                        children={entryCategoryChildren}
                        bind:primary={editingEntry.category_id}
                        bind:extra={editingEntry.category_extra}
                        primaryInputId="edit_cat"
                    />

                    <label for="edit_name">Név</label>
                    <input
                        id="edit_name"
                        type="text"
                        bind:value={editingEntry.name}
                        required
                    />

                    <label for="edit_slug">Slug (URL azonosító)</label>
                    <input
                        id="edit_slug"
                        type="text"
                        bind:value={editingEntry.slug}
                    />

                    <label for="edit_url">Weblap URL</label>
                    <input
                        id="edit_url"
                        type="url"
                        bind:value={editingEntry.url}
                    />

                    <label for="edit_phone">Telefon</label>
                    <input
                        id="edit_phone"
                        type="text"
                        bind:value={editingEntry.phone}
                    />

                    <label for="edit_addr">Cím</label>
                    <input
                        id="edit_addr"
                        type="text"
                        bind:value={editingEntry.address}
                    />

                    <label for="edit_notes">Bemutatkozás</label>
                    <textarea id="edit_notes" bind:value={editingEntry.notes}
                    ></textarea>

                    <label for="edit_tags">Címkék (#cimke1 #cimke2)</label>
                    <input
                        id="edit_tags"
                        type="text"
                        bind:value={editTagsStr}
                        placeholder="#cimke1 #cimke2"
                    />

                    <span class="form-group-label">Nyelvek</span>
                    <div class="flex gap-lg flex-wrap mb-lg">
                        {#each LANGUAGES as lang}
                            <label class="flex items-center gap-xs font-normal">
                                <input
                                    type="checkbox"
                                    name={`edit-entry-lang-${lang}`}
                                    checked={editingEntry.languages.includes(
                                        lang,
                                    )}
                                    on:change={() =>
                                        (editingEntry.languages =
                                            editingEntry.languages.includes(
                                                lang,
                                            )
                                                ? editingEntry.languages.filter(
                                                      (l) => l !== lang,
                                                  )
                                                : [
                                                      ...editingEntry.languages,
                                                      lang,
                                                  ])}
                                    class="w-auto"
                                />
                                {lang}
                            </label>
                            {/each}
                        </div>

                    <label class="flex items-center gap-xs font-normal">
                        <input
                            id="edit-entry-verified"
                            name="verified"
                            type="checkbox"
                            bind:checked={editingEntry.verified}
                            class="w-auto"
                        />
                        Ellenőrzött
                    </label>
                    <p class="admin-form-hint">Alapértelmezett: Nem ellenőrzött (szürke jelvény). A jelvény csak adminnal kapcsolható be.</p>

                    <label class="flex items-center gap-xs font-normal">
                        <input
                            id="edit-entry-hours-enabled"
                            name="hours_enabled"
                            type="checkbox"
                            bind:checked={editingEntry.hours_enabled}
                            class="w-auto"
                            on:change={() => {
                                if (editingEntry.hours_enabled) {
                                    editingEntry.hours = withDefaultWeekHours(editingEntry.hours);
                                }
                            }}
                        />
                        Nyitvatartás / Program
                    </label>
                    {#if editingEntry.hours_enabled}
                        <EntryHoursEditor bind:hours={editingEntry.hours} />
                    {/if}

                    {#if adminOffersDelivery(editingEntry)}
                        <label class="flex items-center gap-xs font-normal">
                            <input
                                id="edit-entry-delivery-enabled"
                                name="delivery_enabled"
                                type="checkbox"
                                bind:checked={editingEntry.delivery_enabled}
                                class="w-auto"
                                on:change={() => {
                                    if (editingEntry.delivery_enabled) {
                                        editingEntry.delivery_hours = withDefaultWeekHours(
                                            editingEntry.delivery_hours,
                                        );
                                    }
                                }}
                            />
                            Kiszállítási idő
                        </label>
                        {#if editingEntry.delivery_enabled}
                            <EntryHoursEditor bind:hours={editingEntry.delivery_hours} />
                        {/if}
                    {/if}

                    <span class="form-group-label">Fotók</span>
                    <EntryPhotosEditor
                        bind:photos={editingEntry.photos}
                        entryId={editingEntry.id}
                        alertError={(msg) => showAlert(msg)}
                    />

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={closeEdit}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Location Modal -->
    {#if editingLocation}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditLocation}
            on:keydown={(e) => e.key === "Escape" && cancelEditLocation()}
        >
            <div class="link-dialog admin-modal">
                <h3>Település szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditLocation}
                >
                    <label for="eloc_name">Név (HU)</label>
                    <input
                        id="eloc_name"
                        type="text"
                        bind:value={editingLocation.name}
                        required
                    />

                    <label for="eloc_name_ro">Név (RO)</label>
                    <input
                        id="eloc_name_ro"
                        type="text"
                        bind:value={editingLocation.name_ro}
                    />

                    <label for="eloc_name_de">Név (DE)</label>
                    <input
                        id="eloc_name_de"
                        type="text"
                        bind:value={editingLocation.name_de}
                    />

                    <label for="eloc_county">Megye</label>
                    <select
                        id="eloc_county"
                        bind:value={editingLocation.county}
                    >
<option value="">Válassz...</option>
                        <option value="">-</option>
                        {#each COUNTIES as c}<option value={c}>{c}</option
                            >{/each}
                    </select>

                    <label for="eloc_type">Típus</label>
                    <select id="eloc_type" bind:value={editingLocation.type}>
<option value="">Válassz...</option>
                        <option value="">-</option>
                        {#each settlementLocationTypes as t}<option value={t.slug}
                                >{t.label_hu}</option
                            >{/each}
                    </select>

                    <label for="eloc_post_code" title="Posta kód"
                        >Irányítószám</label
                    >
                    <input
                        id="eloc_post_code"
                        type="text"
                        bind:value={editingLocation.post_code}
                    />

                    <label for="eloc_coords">Koordináták (szélesség, hosszúság)</label>
                    <input
                        id="eloc_coords"
                        type="text"
                        placeholder="46.3593, 25.8017"
                        bind:value={editingLocation.coordinates}
                    />

                    <label for="eloc_pop">Lakosság (fő)</label>
                    <input
                        id="eloc_pop"
                        type="text"
                        bind:value={editingLocation.population}
                    />

                    <label for="eloc_area">Terület (km²)</label>
                    <input
                        id="eloc_area"
                        type="text"
                        bind:value={editingLocation.area}
                    />

                    <label for="eloc_crest">Címer URL</label>
                    <input
                        id="eloc_crest"
                        type="text"
                        bind:value={editingLocation.crest}
                    />

                    <label for="eloc_parent">Kapcsolódó település</label>
                    <select
                        id="eloc_parent"
                        bind:value={editingLocation.parent_id}
                    >
<option value="">Válassz...</option>
                        <option value={null}>Nincs (Önálló város/község)</option
                        >
                        {#each settlementsForSelect.filter((l) => l.id !== editingLocation.id) as loc}
                            <option value={loc.id}
                                >{loc.name} ({loc.county})</option
                            >
                        {/each}
                    </select>

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditLocation}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Venue Modal -->
    {#if editingVenue}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditVenue}
            on:keydown={(e) => e.key === "Escape" && cancelEditVenue()}
        >
            <div class="link-dialog admin-modal">
                <h3>Helyszín szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditVenue}
                >
                    <label for="ev-venue-settlement">Település</label>
                    <select
                        id="ev-venue-settlement"
                        bind:value={editingVenue.settlement_id}
                        required
                    >
<option value="">Válassz...</option>
                        {#each settlementsForSelect as loc}
                            <option value={String(loc.id)}
                                >{loc.name} ({loc.county})</option
                            >
                        {/each}
                    </select>

                    <label for="ev-venue-name">Név (HU)</label>
                    <input
                        id="ev-venue-name"
                        type="text"
                        bind:value={editingVenue.name}
                        required
                    />

                    <label for="ev-venue-name-ro">Név (RO)</label>
                    <input
                        id="ev-venue-name-ro"
                        type="text"
                        bind:value={editingVenue.name_ro}
                    />

                    <label for="ev-venue-name-de">Név (DE)</label>
                    <input
                        id="ev-venue-name-de"
                        type="text"
                        bind:value={editingVenue.name_de}
                    />

                    <div class="flex gap-lg flex-wrap">
                        <label class="flex-1" style="min-width:8rem"
                            >Slug
                            <input
                                id="ev-venue-slug"
                                name="slug"
                                type="text"
                                bind:value={editingVenue.slug}
                            />
                        </label>
                        <label class="flex-1" style="min-width:10rem"
                            >Típus
                            <select id="ev-venue-kind" name="kind" bind:value={editingVenue.kind}>
<option value="">Válassz...</option>
                                {#each venueTypesList as vt}
                                    <option value={vt.slug}>{vt.label_hu}</option>
                                {/each}
                            </select>
                        </label>
                    </div>

                    <label for="ev-venue-address">Cím</label>
                    <input
                        id="ev-venue-address"
                        type="text"
                        bind:value={editingVenue.address}
                    />

                    <div class="flex gap-lg flex-wrap">
                        <label class="flex-1" style="min-width:8rem"
                            >Szélesség (lat)
                            <input
                                id="ev-venue-latitude"
                                name="latitude"
                                type="text"
                                bind:value={editingVenue.latitude}
                            />
                        </label>
                        <label class="flex-1" style="min-width:8rem"
                            >Hosszúság (lon)
                            <input
                                id="ev-venue-longitude"
                                name="longitude"
                                type="text"
                                bind:value={editingVenue.longitude}
                            />
                        </label>
                        <label class="flex-1" style="min-width:8rem"
                            >Férőhely
                            <input
                                id="ev-venue-seating"
                                name="seating_capacity"
                                type="text"
                                bind:value={editingVenue.seating_capacity}
                            />
                        </label>
                    </div>

                    <label for="ev-venue-description">Leírás</label>
                    <textarea
                        id="ev-venue-description"
                        bind:value={editingVenue.description}
                        rows="4"
                    ></textarea>

                    <label for="ev-venue-notes">Belső megjegyzés</label>
                    <textarea
                        id="ev-venue-notes"
                        bind:value={editingVenue.notes}
                        rows="2"
                    ></textarea>

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditVenue}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Category Modal -->
    {#if editingCategory}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditCategory}
            on:keydown={(e) => e.key === "Escape" && cancelEditCategory()}
        >
            <div class="link-dialog admin-modal">
                <h3>Kategória szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditCategory}
                >
                    <label for="ecat_name">Kategória neve</label>
                    <input
                        id="ecat_name"
                        type="text"
                        bind:value={editingCategory.name}
                        required
                    />

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditCategory}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Type Modal -->
    {#if editingType}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditType}
            on:keydown={(e) => e.key === "Escape" && cancelEditType()}
        >
            <div class="link-dialog admin-modal">
                <h3>Típus szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditType}
                >
                    <label for="etype_name_edit">Típus neve</label>
                    <input
                        id="etype_name_edit"
                        type="text"
                        bind:value={editingType.name}
                        required
                    />

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditType}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Tag Modal -->
    {#if editingTag}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditTag}
            on:keydown={(e) => e.key === "Escape" && cancelEditTag()}
        >
            <div class="link-dialog admin-modal">
                <h3>Címke szerkesztése</h3>
                <form class="admin-form" on:submit|preventDefault={saveEditTag}>
                    <label for="etag_name">Címke neve</label>
                    <input id="etag_name" type="text" bind:value={editingTag.name} required />
                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn">Mentés</button>
                        <button type="button" class="btn-delete" on:click={cancelEditTag}>Mégse</button>
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Edit Attraction Modal -->
    {#if editingAttraction}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditAttraction}
            on:keydown={(e) => e.key === "Escape" && cancelEditAttraction()}
        >
            <div class="link-dialog admin-modal">
                <h3>Látnivaló szerkesztése</h3>
                <form
                    class="admin-form"
                    on:submit|preventDefault={saveEditAttraction}
                >
                    <label for="eatt_county">Megye</label>
                    <select id="eatt_county" name="county_slug" bind:value={editingAttraction.county_slug}>
<option value="">Válassz...</option>
                        <option value="hargita">Hargita</option>
                        <option value="kovaszna">Kovászna</option>
                        <option value="maros">Maros</option>
                    </select>
                    <label for="eatt_name">Név</label>
                    <input id="eatt_name" name="name" type="text" bind:value={editingAttraction.name} required />
                    <label for="eatt_desc">Rövid leírás</label>
                    <input id="eatt_desc" name="description" type="text" bind:value={editingAttraction.description} />
                    <label for="eatt_coords">Koordináták (lat, lon)</label>
                    <input id="eatt_coords" name="latitude" type="text" bind:value={editingAttraction.latitude} placeholder="46.1265" style="width:6rem" />
                    <input id="eatt_lon" name="longitude" type="text" bind:value={editingAttraction.longitude} placeholder="25.8876" style="width:6rem" />
                    <label for="eatt_elevation">Magasság (m)</label>
                    <input id="eatt_elevation" name="elevation_m" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={editingAttraction.elevation_m} placeholder="946" style="width:6rem" />
                    <label for="eatt_area">Felszín (km²)</label>
                    <input id="eatt_area" name="area_km2" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={editingAttraction.area_km2} placeholder="0,22" style="width:6rem" />
                    <label for="eatt_depth">Mélység (m)</label>
                    <input id="eatt_depth" name="depth_m" type="text" pattern="-?[0-9 ]*([.,][0-9]+)?" inputmode="decimal" bind:value={editingAttraction.depth_m} placeholder="7" style="width:6rem" />
                    <label for="eatt_featured">Kiemelt kép URL</label>
                    <input id="eatt_featured" name="featured_image" type="url" bind:value={editingAttraction.featured_image} />
                    <label for="eatt_featured_copyright">Szerzői jog (kiemelt kép)</label>
                    <input id="eatt_featured_copyright" name="featured_image_copyright" type="text" bind:value={editingAttraction.featured_image_copyright} />
                    <label for="eatt_activities">Tevékenységek / aktivitások (soronként egy)</label>
                    <textarea id="eatt_activities" name="activities" bind:value={editingAttraction.activities} rows="4"></textarea>
                    <label for="eatt_prohibitions">Mit nem szabad itt csinálni? (soronként egy)</label>
                    <textarea id="eatt_prohibitions" name="prohibitions" bind:value={editingAttraction.prohibitions} rows="4"></textarea>
                    <label for="eatt_content">Tartalom (Markdown)</label>
                    <textarea id="eatt_content" name="content" bind:value={editingAttraction.content} rows="6"></textarea>
                    <label for="eatt_images">Galéria URL-ek (soronként egy)</label>
                    <textarea id="eatt_images" name="images" bind:value={editingAttraction.images} rows="3"></textarea>
                    <label for="eatt_image_copyrights">Galéria szerzői jog (ugyanabban a sorrendben, soronként)</label>
                    <textarea id="eatt_image_copyrights" name="image_copyrights" bind:value={editingAttraction.image_copyrights} rows="3"></textarea>
                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn">Mentés</button>
                        <button type="button" class="btn-delete" on:click={cancelEditAttraction}>Mégse</button>
                    </div>
                </form>
            </div>
        </div>
    {/if}


    <!-- Edit Event Modal -->
    {#if editingEvent}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={cancelEditEvent}
            on:keydown={(e) => e.key === "Escape" && cancelEditEvent()}
        >
            <div class="link-dialog admin-modal">
                <h3>Esemény szerkesztése</h3>
                <p class="admin-form-hint">
                    A dátumok és időpontok (óra:perc) kötelezőek.
                </p>
                <form
                    class="admin-form"
                    autocomplete="off"
                    on:submit|preventDefault={saveEditEvent}
                >
                    <label for="edit_ev_loc"
                        >Helyszín <span class="admin-req" title="Kötelező">*</span></label
                    >
                    <select
                        id="edit_ev_loc"
                        name="location_id"
                        bind:value={editingEvent.location_id}
                        required
                        on:change={() => {
                            editingEvent.default_venue_id = "";
                            loadVenuesForEditSettlement(editingEvent.location_id);
                        }}
                    >
                        <option value="">Válassz...</option>
                        {#each settlementsForSelect as loc}
                            <option value={loc.id}
                                >{loc.name} ({loc.county})</option
                            >
                        {/each}
                    </select>

                    <label for="edit_ev_default_venue"
                        >Konkrét helyszín (opcionális)</label
                    >
                    <select
                        id="edit_ev_default_venue"
                        name="default_venue_id"
                        bind:value={editingEvent.default_venue_id}
                    >
<option value="">Válassz...</option>
                        <option value="">- nincs megadva -</option>
                        {#each venueOptionsEdit as v}
                            <option value={String(v.id)}>{v.name}</option>
                        {/each}
                    </select>

                    <label for="edit_ev_attraction">Látnivaló (opcionális)</label>
                    <select id="edit_ev_attraction" name="attraction_id" bind:value={editingEvent.attraction_id}>
<option value="">Válassz...</option>
                        <option value="">- nincs hozzárendelve -</option>
                        {#each attractions as att}
                            <option value={String(att.id)}>{att.name} ({att.county_name || att.county_slug})</option>
                        {/each}
                    </select>

                    <label for="edit_ev_title"
                        >Cím <span class="admin-req" title="Kötelező">*</span></label
                    >
                    <input
                        id="edit_ev_title"
                        name="title"
                        type="text"
                        bind:value={editingEvent.title}
                        required
                    />

                    <label for="edit_ev_desc">Leírás</label>
                    <textarea
                        id="edit_ev_desc"
                        name="description"
                        bind:value={editingEvent.description}
                    ></textarea>

                    <div class="admin-event-image-block">
                        <label for="edit_ev_featured_upload">Kiemelt kép</label>
                        {#if editingEvent.featured_image}
                            <img
                                class="admin-event-image-preview"
                                src={absoluteMediaUrl(
                                    editingEvent.featured_image,
                                    getApiBase(),
                                )}
                                alt=""
                            />
                        {/if}
                        <input
                            id="edit_ev_featured_upload"
                            type="file"
                            accept="image/jpeg,image/png,image/webp,image/gif"
                            on:change={(e) => {
                                const f = e.target.files?.[0];
                                if (f) uploadEventFeaturedImage(f, true);
                                e.target.value = "";
                            }}
                        />
                        <label for="edit_ev_featured_url" class="admin-sublabel"
                            >Vagy kép URL (külső)</label
                        >
                        <input
                            id="edit_ev_featured_url"
                            type="url"
                            bind:value={editingEvent.featured_image}
                            placeholder="https://…"
                        />
                        {#if editingEvent.featured_image}
                            <button
                                type="button"
                                class="btn-update"
                                style="align-self: flex-start"
                                on:click={() =>
                                    (editingEvent.featured_image = "")}
                                >Kép törlése</button
                            >
                        {/if}
                        <label for="edit_ev_featured_copyright">Szerzői jogi információ</label>
                        <input
                            id="edit_ev_featured_copyright"
                            name="featured_image_copyright"
                            type="text"
                            bind:value={editingEvent.featured_image_copyright}
                        />
                    </div>

                    <div class="flex gap-lg">
                        <div class="flex-1">
                            <label for="edit_ev_start_date"
                                >Kezdő dátum <span class="admin-req" title="Kötelező"
                                    >*</span
                                ></label
                            >
                            <HuDateInput
                                id="edit_ev_start_date"
                                name="start_date"
                                bind:value={editingEvent.start_date}
                                required
                            />
                        </div>
                        <div class="flex-1">
                            <label for="edit_ev_start_time"
                                >Kezdő időpont (óra:perc) <span
                                    class="admin-req"
                                    title="Kötelező">*</span
                                ></label
                            >
                            <HuTimeInput
                                id="edit_ev_start_time"
                                name="start_time"
                                bind:value={editingEvent.start_time}
                                required
                            />
                        </div>
                    </div>
                    <div class="flex gap-lg">
                        <div class="flex-1">
                            <label for="edit_ev_end_date"
                                >Befejező dátum <span class="admin-req" title="Kötelező"
                                    >*</span
                                ></label
                            >
                            <HuDateInput
                                id="edit_ev_end_date"
                                name="end_date"
                                bind:value={editingEvent.end_date}
                                required
                            />
                        </div>
                        <div class="flex-1">
                            <label for="edit_ev_end_time"
                                >Befejező időpont (óra:perc) <span
                                    class="admin-req"
                                    title="Kötelező">*</span
                                ></label
                            >
                            <HuTimeInput
                                id="edit_ev_end_time"
                                name="end_time"
                                bind:value={editingEvent.end_time}
                                required
                            />
                        </div>
                    </div>

                    <label for="edit_ev_type_id"
                        >Eseménytípus <span class="admin-req" title="Kötelező">*</span></label
                    >
                    <select
                        id="edit_ev_type_id"
                        name="event_type_id"
                        bind:value={editingEvent.event_type_id}
                        on:change={() => (editingEvent.event_subtype_id = "")}
                        required
                    >
                        <option value="">Válassz...</option>
                        {#each [...catalogEventTypes].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || String(a.label_hu).localeCompare(String(b.label_hu), "hu")) as t}
                            <option value={String(t.id)}
                                >{t.label_hu} ({t.slug})</option
                            >
                        {/each}
                    </select>

                    <label for="edit_ev_subtype_id">Altípus (opcionális)</label>
                    <select
                        id="edit_ev_subtype_id"
                        name="event_subtype_id"
                        bind:value={editingEvent.event_subtype_id}
                    >
<option value="">Válassz...</option>
                        <option value="">- nincs -</option>
                        {#each subtypesForEditEvent as s}
                            <option value={String(s.id)}
                                >{s.label_hu} ({s.slug})</option
                            >
                        {/each}
                    </select>

                    <label for="edit_ev_access">Hozzáférés</label>
                    <select
                        id="edit_ev_access"
                        name="access_type"
                        bind:value={editingEvent.access_type}
                    >
<option value="">Válassz...</option>
                        <option value="public">{ACCESS_TYPE_LABELS.public}</option>
                        <option value="members_only">{ACCESS_TYPE_LABELS.members_only}</option>
                        <option value="invitation_only">{ACCESS_TYPE_LABELS.invitation_only}</option>
                    </select>

                    <label for="edit_ev_org">Szervező</label>
                    <div class="org-autosuggest-wrapper">
                        <div class="org-autosuggest-row">
                            <input
                                id="edit_ev_org"
                                name="organizer"
                                type="text"
                                bind:value={orgEditQuery}
                                on:input={() => {
                                    editingEvent.organizer = orgEditQuery;
                                    onOrgInput(true);
                                }}
                                on:focus={() => onOrgInput(true)}
                                on:blur={() => handleOrgBlur(true)}
                                autocomplete="off"
                                placeholder="Keresés szervező neve..."
                                class="flex-1"
                            />
                            <button
                                type="button"
                                class="btn-update"
                                style="margin-bottom:0"
                                on:click={() => (newOrganizerModalVisible = true)}
                            >
                                Új szervező
                            </button>
                        </div>
                        {#if orgEditDropdownOpen && orgEditSuggestions.length > 0}
                            <ul class="org-suggestions">
                                {#each orgEditSuggestions as s}
                                    <li>
                                        <button type="button" on:click={() => selectOrganizer(s.name, true)}>
                                            <strong>{s.name}</strong>
                                            {#if s.location}<span class="org-sug-meta">{s.location}</span>{/if}
                                        </button>
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                    </div>

                    <label for="edit_ev_entry_price">Belépő / jegyár (opcionális)</label>
                    <input
                        id="edit_ev_entry_price"
                        name="entry_price"
                        type="text"
                        bind:value={editingEvent.entry_price}
                        placeholder="pl. 99 RON, 15 EUR, ingyenes"
                        maxlength="128"
                        autocomplete="off"
                    />

                    <details class="admin-schedule-details">
                        <summary>Napi program (opcionális)</summary>
                        <p class="admin-form-hint admin-schedule-hint">
                            A fenti kezdő–befejező dátum és idő továbbra is az alap; ide
                            naponkénti tételeket írhat (megnyitó, mérkőzések, záró stb.).
                            A <strong>vége</strong> idő opcionális (pl. ismeretlen mérkőzés-hossz).
                            Üresen hagyható. A helyszínt soronként a <strong>Helyszín</strong> oszlopban
                            állíthatod (üres = esemény alaphelyszíne).
                        </p>
                        <div class="schedule-toolbar">
                            <button
                                type="button"
                                class="btn-update"
                                on:click|preventDefault={generateScheduleDaysFromEvent}
                                >Napok generálása a dátumokból</button
                            >
                            <button
                                type="button"
                                class="btn-update"
                                on:click|preventDefault={addScheduleDayRow}
                                >Új nap</button
                            >
                        </div>

                        {#each scheduleDraftDays as day, di}
                            <div class="schedule-day-block">
                                <div class="schedule-day-head">
                                    <label class="schedule-inline"
                                        >Dátum
                                        <HuDateInput
                                            id={`schedule-day-${di}-date`}
                                            name={`schedule_day_${di}_date`}
                                            bind:value={day.schedule_date}
                                        /></label
                                    >
                                    <button
                                        type="button"
                                        class="btn-delete btn-xs"
                                        on:click={() =>
                                            removeScheduleDayRow(di)}
                                        >Nap törlése</button
                                    >
                                </div>
                                <table class="admin-table schedule-act-table">
                                    <thead>
                                        <tr>
                                            <th>Típus</th>
                                            <th title="Opcionális">Kezdés</th>
                                            <th title="Opcionális; mérkőzésnél gyakran üres"
                                                >Vége</th
                                            >
                                            <th title="Opcionális; üres = esemény alaphelyszíne"
                                                >Helyszín</th
                                            >
                                            <th>Cím / program</th>
                                            <th>Leírás</th>
                                            <th></th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each day.activities || [] as act, ai}
                                            <tr>
                                                <td>
                                                    <select
                                                        id={`schedule-${di}-act-${ai}-type`}
                                                        name={`schedule_${di}_act_${ai}_type`}
                                                        class="schedule-act-type"
                                                        bind:value={act.activity_type}
                                                    >
<option value="">Válassz...</option>
                                                        {#each SCHEDULE_ACTIVITY_TYPES as t}
                                                            <option value={t}
                                                                >{SCHEDULE_ACTIVITY_TYPE_LABELS[
                                                                    t
                                                                ]}</option
                                                            >
                                                        {/each}
                                                    </select>
                                                </td>
                                                <td
                                                    ><HuTimeInput
                                                        id={`schedule-${di}-act-${ai}-start`}
                                                        name={`schedule_${di}_act_${ai}_starts_at`}
                                                        bind:value={act.starts_at}
                                                /></td>
                                                <td
                                                    ><HuTimeInput
                                                        id={`schedule-${di}-act-${ai}-end`}
                                                        name={`schedule_${di}_act_${ai}_ends_at`}
                                                        bind:value={act.ends_at}
                                                /></td>
                                                <td>
                                                    <select
                                                        id={`schedule-${di}-act-${ai}-venue`}
                                                        name={`schedule_${di}_act_${ai}_venue`}
                                                        class="schedule-act-venue"
                                                        bind:value={act.venue_id}
                                                    >
<option value="">Válassz...</option>
                                                        <option value=""
                                                            >- alapértelmezett -</option
                                                        >
                                                        {#each venueOptionsEdit as v}
                                                            <option
                                                                value={String(
                                                                    v.id,
                                                                )}
                                                                >{v.name}</option
                                                            >
                                                        {/each}
                                                    </select>
                                                </td>
                                                <td
                                                    ><input
                                                        id={`schedule-${di}-act-${ai}-title`}
                                                        name={`schedule_${di}_act_${ai}_title`}
                                                        type="text"
                                                        placeholder="Kötelező cím"
                                                        bind:value={act.title}
                                                /></td>
                                                <td
                                                    ><input
                                                        id={`schedule-${di}-act-${ai}-desc`}
                                                        name={`schedule_${di}_act_${ai}_description`}
                                                        type="text"
                                                        bind:value={act.description}
                                                /></td>
                                                <td
                                                    ><button
                                                        type="button"
                                                        class="btn-delete btn-xs"
                                                        on:click={() =>
                                                            removeScheduleActivity(
                                                                di,
                                                                ai,
                                                            )}>×</button
                                                    ></td
                                                >
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                                <button
                                    type="button"
                                    class="btn-update btn-xs"
                                    on:click={() => addScheduleActivity(di)}
                                    >+ Tevékenység</button
                                >
                            </div>
                        {/each}

                        <button
                            type="button"
                            class="admin-submit-btn schedule-save-btn"
                            on:click|preventDefault={saveEventSchedule}
                            >Napi program mentése</button
                        >
                    </details>

                    <div class="modal-actions">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={cancelEditEvent}>Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- New Organizer Modal -->
    {#if newOrganizerModalVisible}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            class="link-dialog-overlay"
            role="dialog"
            tabindex="-1"
            on:click|self={() => (newOrganizerModalVisible = false)}
            on:keydown={(e) =>
                e.key === "Escape" && (newOrganizerModalVisible = false)}
        >
            <div class="link-dialog admin-modal">
                <h3>Új Szervező Hozzáadása</h3>
                <form class="admin-form" on:submit={submitNewOrganizer}>
                    <label for="org_loc">Település</label>
                    <select
                        id="org_loc"
                        bind:value={newOrganizerEntry.location_id}
                        required
                    >
                        <option value="">Válassz...</option>
                        {#each settlementsForSelect as loc}
                            <option value={loc.id}
                                >{loc.name} ({loc.county})</option
                            >
                        {/each}
                    </select>

                    <label for="org_name">Név (Szervezet Neve)</label>
                    <input
                        id="org_name"
                        type="text"
                        bind:value={newOrganizerEntry.name}
                        required
                    />

                    <CategoryMultiSelect
                        parents={entryCategoryParents}
                        children={entryCategoryChildren}
                        bind:primary={newOrganizerEntry.category_id}
                        bind:extra={newOrganizerEntry.category_extra}
                        primaryInputId="org_cat"
                    />

                    <label for="org_phone">Telefon</label>
                    <input
                        id="org_phone"
                        type="text"
                        bind:value={newOrganizerEntry.phone}
                    />

                    <div class="modal-actions mt-lg">
                        <button type="submit" class="admin-submit-btn"
                            >Mentés és Kiválasztás</button
                        >
                        <button
                            type="button"
                            class="btn-delete"
                            on:click={() => (newOrganizerModalVisible = false)}
                            >Mégse</button
                        >
                    </div>
                </form>
            </div>
        </div>
    {/if}

    <!-- Network dialogs (UI_BASELINE "dlg-confirm", "dlg-notice"): showConfirm()
         asks with Mégse next to Igen; showAlert() only informs, so it closes
         with Bezárás ("dlg-close-label"). Last in the markup: the notice shares
         the edit windows' overlay layer, so it must come after them to show on
         top of one. -->
    <ConfirmDialog
        open={dialogVisible && dialogType === "confirm"}
        message={dialogMsg}
        onYes={dialogOk}
        onNo={dialogCancel}
    />
    {#if dialogVisible && dialogType === "alert"}
        <NoticeDialog title="Üzenet" message={dialogMsg} onClose={dialogOk} />
    {/if}
    {/snippet}
</AdminShell>

<style>
    @import "../styles/admin.css";

    .badge {
        display: inline-block;
        padding: 0.15rem 0.5rem;
        border-radius: 999px;
        background: var(--accent-bg, #2a2a3e);
        color: var(--muted, #aaa);
        border: 1px solid var(--border-color, #444);
    }
    .form-group-label {
        display: block;
        font-weight: 600;
        margin-bottom: 0.35rem;
    }
    .w-full {
        width: 100%;
    }
    .gap-xs {
        gap: 0.3rem;
    }
    .gap-lg {
        gap: 1rem;
    }
    .mt-xs {
        margin-top: 5px;
    }
    .color-swatch {
        display: inline-block;
        width: 20px;
        height: 20px;
        border: 1px solid var(--border-color);
    }
    .modal-actions {
        display: flex;
        gap: 0.75rem;
    }
    .mt-lg {
        margin: 2rem auto;
    }
    .flex {
        display: flex;
    }
    .flex-wrap {
        flex-wrap: wrap;
    }
    .items-center {
        align-items: center;
    }
    .font-normal {
        font-weight: normal;
    }
    .w-auto {
        width: auto;
    }
    .mb-lg {
        margin-bottom: 1rem;
    }
    .admin-info {
        color: var(--text-faint, #666);
        margin-bottom: 1rem;
    }

    .admin-action-notice {
        margin: 0 0 0.75rem;
    }
    .admin-date-field {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex-wrap: wrap;
    }
    .admin-date-field :global(input) {
        min-width: 12rem;
    }
    .admin-date-today-badge {
        display: inline-block;
        margin-left: 0.35rem;
        padding: 0.1rem 0.45rem;
        border-radius: 999px;
        background: var(--szekely-red, #c8102e);
        color: #fff;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.03em;
    }
    tr.admin-row-today {
        background: color-mix(
            in srgb,
            var(--szekely-green, #2f4f4f) 10%,
            transparent
        );
    }
    .admin-region-heading {
        margin-top: 2.5rem;
        margin-bottom: 0.5rem;
    }
    .admin-subtab-heading {
        margin-top: 1.75rem;
        margin-bottom: 0.5rem;
        font-weight: 600;
    }
    .admin-subtab-heading:first-of-type {
        margin-top: 0;
    }

    .admin-table-edit-row td {
        vertical-align: top;
        background: var(--hover-bg, #f9fafb);
    }
    .admin-region-edit-panel {
        padding: 0.35rem 0 0.25rem;
    }
    .admin-region-edit-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 0.75rem 1rem;
        margin-bottom: 0.75rem;
    }
    .admin-region-edit-grid label {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }
    .admin-region-edit-grid input,
    .admin-region-edit-grid select {
        width: 100%;
        padding: 0.35rem 0.5rem;
    }
    .admin-region-edit-span2 {
        grid-column: span 2;
    }
    @media (max-width: 720px) {
        .admin-region-edit-span2 {
            grid-column: span 1;
        }
    }
    .admin-region-edit-full {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
        margin-bottom: 0.75rem;
    }
    .admin-region-edit-full textarea {
        width: 100%;
    }
    .admin-region-edit-actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .admin-faq-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        margin: 1rem 0 0.5rem;
        flex-wrap: wrap;
    }
    .admin-faq-toolbar-label {
        font-weight: 600;
    }
    .admin-faq-pair {
        margin-bottom: 0.75rem;
        border: 1px solid var(--border-color, #e5e7eb);
        border-radius: 8px;
        padding: 0.35rem 0.75rem 0.75rem;
        background: var(--card-bg, #fff);
    }
    .admin-faq-pair summary {
        cursor: pointer;
        font-weight: 600;
        padding: 0.35rem 0;
    }
    .admin-faq-pair-fields {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        margin-top: 0.35rem;
    }
    /* .admin-faq-pair-fields label {
    } */

    .org-autosuggest-wrapper {
        position: relative;
    }
    .org-autosuggest-row {
        display: flex;
        gap: 0.5rem;
        align-items: center;
    }
    .org-suggestions {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        z-index: 100;
        list-style: none;
        margin: 0;
        padding: 0;
        background: var(--card-bg, #1e1e2e);
        border: 1px solid var(--border-color, #444);
        border-top: none;
        border-radius: 0 0 6px 6px;
        max-height: 240px;
        overflow-y: auto;
    }
    .org-suggestions li button {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        width: 100%;
        padding: 0.5rem 0.75rem;
        border: none;
        background: none;
        color: var(--text-color, #ccc);
        cursor: pointer;
        text-align: left;
    }
    .org-suggestions li button:hover {
        background: var(--accent-bg, #2a2a3e);
    }
    .org-sug-meta {
        color: var(--text-faint, #888);
    }
</style>