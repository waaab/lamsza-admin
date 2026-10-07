import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
    ADMIN_SECTIONS,
    APP_ORIGIN_DEFAULTS,
    adminAppLinks,
    appOrigins,
    sectionForPath,
} from "../src/lib/adminApps.js";

test("the switcher links are this app's own section routes", () => {
    assert.deepEqual(
        adminAppLinks("/").map((a) => [a.id, a.href]),
        [
            ["directory", "/"],
            ["szotar", "/dictionary"],
            ["jatszoter", "/games"],
        ],
    );
});

test("exactly one link is current, decided by the path", () => {
    const current = (/** @type {string | undefined} */ path) => adminAppLinks(path).filter((a) => a.current).map((a) => a.id);
    assert.deepEqual(current("/"), ["directory"]);
    assert.deepEqual(current(undefined), ["directory"]);
    assert.deepEqual(current("/dictionary"), ["szotar"]);
    assert.deepEqual(current("/dictionary/"), ["szotar"]);
    assert.deepEqual(current("/games"), ["jatszoter"]);
    assert.equal(sectionForPath("/gamesx"), "directory", "a prefix alone is not the section");
});

test("app origins fall back to the fixed local dev ports", () => {
    assert.deepEqual(appOrigins(undefined), APP_ORIGIN_DEFAULTS);
});

test("build-time env overrides every app origin, cleaned up", () => {
    assert.deepEqual(
        appOrigins({
            VITE_LAMSZA_ORIGIN: "https://lamsza.com/",
            VITE_SZOTAR_ORIGIN: "  https://szotar.lamsza.com  ",
            VITE_JATSZOTER_ORIGIN: 42,
        }),
        {
            lamsza: "https://lamsza.com",
            szotar: "https://szotar.lamsza.com",
            jatszoter: APP_ORIGIN_DEFAULTS.jatszoter,
        },
    );
});

test("every icon the switcher asks for exists in AppIcon", () => {
    const icons = readFileSync(new URL("../src/lib/icons/AppIcon.svelte", import.meta.url), "utf8");
    for (const { icon } of ADMIN_SECTIONS) {
        assert.ok(icons.includes(`name === "${icon}"`), `AppIcon has no "${icon}" branch`);
    }
});

test(".env.example documents every app origin variable", () => {
    // The links silently fall back to localhost when a name drifts, so an
    // undocumented variable ships a dev URL to production with no error.
    const example = readFileSync(new URL("../../.env.example", import.meta.url), "utf8");
    for (const name of ["VITE_LAMSZA_ORIGIN", "VITE_SZOTAR_ORIGIN", "VITE_JATSZOTER_ORIGIN"]) {
        assert.match(example, new RegExp(`^${name}=`, "m"), `.env.example is missing ${name}`);
    }
});
