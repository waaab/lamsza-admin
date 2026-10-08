/**
 * The Beállítások tab's sections and the site_settings keys each one owns.
 * Each section's Mentés button saves only its own keys: saving one section
 * must not rewrite another's, and never the counters the server bumps
 * (weather_cache_version, quick_links_version), which a stale copy in the
 * page would roll back.
 */
export const SETTINGS_SECTIONS = {
    social: {
        label: "Közösségi oldalak",
        keys: ["social_facebook_url", "social_twitter_url", "social_instagram_url"],
    },
    location: {
        label: "Alapértelmezett település",
        keys: ["my_location_slug"],
    },
    weather: {
        label: "Időjárás",
        keys: [
            "weather_provider_metno_enabled",
            "weather_provider_weatherapi_enabled",
            "weather_provider_openweathermap_enabled",
            "weather_icon_style",
            "weather_cache_ttl_minutes",
        ],
    },
};

/**
 * The PUT body for one section: its keys only, as strings.
 * @param {Record<string, unknown>} settings
 * @param {keyof typeof SETTINGS_SECTIONS} section
 * @returns {Record<string, string>}
 */
export function settingsPayload(settings, section) {
    const def = SETTINGS_SECTIONS[section];
    if (!def) throw new Error(`unknown settings section: ${section}`);
    return Object.fromEntries(def.keys.map((k) => [k, settings?.[k] != null ? String(settings[k]) : ""]));
}
