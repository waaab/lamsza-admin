import { test } from "node:test";
import assert from "node:assert/strict";
import { SETTINGS_SECTIONS, settingsPayload } from "../src/lib/settingsSections.js";

const page = {
    social_facebook_url: "https://www.facebook.com/szekelygugel",
    social_twitter_url: "",
    social_instagram_url: null,
    my_location_slug: "csikszereda",
    weather_provider_metno_enabled: "true",
    weather_provider_weatherapi_enabled: "true",
    weather_provider_openweathermap_enabled: "false",
    weather_icon_style: "svg",
    weather_cache_ttl_minutes: 15,
    weather_cache_version: "41",
    quick_links_version: "260",
};

test("each section saves only its own keys", () => {
    assert.deepEqual(settingsPayload(page, "location"), { my_location_slug: "csikszereda" });
    assert.deepEqual(Object.keys(settingsPayload(page, "social")).sort(), [
        "social_facebook_url",
        "social_instagram_url",
        "social_twitter_url",
    ]);
    const weather = settingsPayload(page, "weather");
    assert.equal(weather.weather_icon_style, "svg");
    assert.equal(weather.weather_cache_ttl_minutes, "15");
    assert.equal("my_location_slug" in weather, false);
});

test("no section writes the counters the server bumps", () => {
    for (const name of Object.keys(SETTINGS_SECTIONS)) {
        const body = settingsPayload(page, /** @type {any} */ (name));
        assert.equal("weather_cache_version" in body, false, name);
        assert.equal("quick_links_version" in body, false, name);
    }
});

test("no key belongs to two sections", () => {
    const all = Object.values(SETTINGS_SECTIONS).flatMap((s) => s.keys);
    assert.equal(new Set(all).size, all.length);
});

test("missing values are sent as empty strings, an unknown section throws", () => {
    assert.equal(settingsPayload(page, "social").social_instagram_url, "");
    assert.throws(() => settingsPayload(page, /** @type {any} */ ("nope")));
});
