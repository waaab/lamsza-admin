import assert from "node:assert/strict";
import test from "node:test";
import {
    buildImportPayload,
    meToAuthState,
    userAccountTabIds,
    userSettingsTabIds,
} from "../src/lib/accountPrefs.js";

test("buildImportPayload copies browser prefs", () => {
    const payload = buildImportPayload({
        theme: "dark",
        slots: 9,
        links: [{ title: "RMDSZ", url: "https://rmdsz.ro", bg_color: "#fff" }],
        history: [{ slug: "kavezo", name: "Kávézó", category: "", location: "", photo: "" }],
    });
    assert.equal(payload.theme, "dark");
    assert.equal(payload.quicklink_slots, 9);
    assert.equal(payload.links.length, 1);
    assert.equal(payload.history[0].slug, "kavezo");
});

test("buildImportPayload drops links missing title or url", () => {
    const payload = buildImportPayload({
        theme: "",
        slots: null,
        links: [
            { title: "OK", url: "https://example.test" },
            { title: "", url: "https://example.test" },
            { title: "No URL", url: "" },
            { url: "https://example.test" },
        ],
        history: [],
    });
    assert.equal(payload.links.length, 1);
    assert.equal(payload.links[0].title, "OK");
});

test("buildImportPayload drops history rows missing slug or name", () => {
    const payload = buildImportPayload({
        theme: "",
        slots: null,
        links: [],
        history: [
            { slug: "kavezo", name: "Kávézó" },
            { slug: "", name: "Névtelen" },
            { slug: "ures", name: "" },
            { name: "Nincs slug" },
        ],
    });
    assert.equal(payload.history.length, 1);
    assert.equal(payload.history[0].slug, "kavezo");
});

test("userAccountTabIds are the account tabs", () => {
    assert.deepEqual(userAccountTabIds, [
        "fiok",
        "adatlapjaim",
        "weboldalaim",
        "kereses",
        "bongeszes",
        "kedvencek",
        "linkjeim",
    ]);
});

test("userSettingsTabIds are the settings tabs", () => {
    assert.deepEqual(userSettingsTabIds, [
        "tema",
        "linkbeallitasok",
        "location",
    ]);
});

test("meToAuthState maps google fields", () => {
    const state = meToAuthState({
        name: "Anna",
        email: "anna@example.test",
        is_admin: false,
        picture: "https://example.test/a.jpg",
        given_name: "Anna",
        family_name: "Kiss",
        display_name: "Panna",
        locale: "hu",
        google_sub: "sub-1",
        theme: null,
        quicklink_slots: null,
        prefs_imported_at: null,
    });
    assert.equal(state.loggedIn, true);
    assert.equal(state.picture, "https://example.test/a.jpg");
    assert.equal(state.displayName, "Panna");
    assert.equal(state.theme, null);
});
