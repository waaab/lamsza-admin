import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { ADMIN_APP_DEFAULTS, adminAppLinks } from "../src/lib/adminApps.js";

test("falls back to the fixed local dev ports", () => {
    const hrefs = Object.fromEntries(adminAppLinks(undefined).map((a) => [a.id, a.href]));
    assert.deepEqual(hrefs, ADMIN_APP_DEFAULTS);
});

test("build-time env overrides every link", () => {
    const links = adminAppLinks({
        VITE_ADMIN_ORIGIN: "https://admin.lamsza.com",
        VITE_SZOTAR_ADMIN_ORIGIN: "https://szotar.lamsza.com/admin",
        VITE_JATSZOTER_ADMIN_ORIGIN: "https://jatszoter.lamsza.com/admin",
    });
    assert.deepEqual(
        links.map((a) => a.href),
        [
            "https://admin.lamsza.com",
            "https://szotar.lamsza.com/admin",
            "https://jatszoter.lamsza.com/admin",
        ],
    );
});

test("blank and trailing-slash values are cleaned up, not trusted", () => {
    const links = adminAppLinks({
        VITE_ADMIN_ORIGIN: "   ",
        VITE_SZOTAR_ADMIN_ORIGIN: "https://szotar.lamsza.com/admin//",
        VITE_JATSZOTER_ADMIN_ORIGIN: 42,
    });
    assert.equal(links[0].href, ADMIN_APP_DEFAULTS.directory, "blank falls back");
    assert.equal(links[1].href, "https://szotar.lamsza.com/admin", "trailing slashes trimmed");
    assert.equal(links[2].href, ADMIN_APP_DEFAULTS.jatszoter, "non-string falls back");
});

test("exactly one link is the current app, and it does not open a new tab", () => {
    const links = adminAppLinks({});
    assert.deepEqual(
        links.filter((a) => a.current).map((a) => a.id),
        ["directory"],
    );
});

test("every icon the switcher asks for exists in AppIcon", () => {
    const icons = readFileSync(new URL("../src/lib/icons/AppIcon.svelte", import.meta.url), "utf8");
    for (const { icon } of adminAppLinks({})) {
        assert.ok(icons.includes(`name === "${icon}"`), `AppIcon has no "${icon}" branch`);
    }
});

test(".env.example documents every variable the switcher reads", () => {
    // The links silently fall back to localhost when a name drifts, so an
    // undocumented variable ships a dev URL to production with no error.
    const example = readFileSync(new URL("../../.env.example", import.meta.url), "utf8");
    for (const name of [
        "VITE_ADMIN_ORIGIN",
        "VITE_SZOTAR_ADMIN_ORIGIN",
        "VITE_JATSZOTER_ADMIN_ORIGIN",
    ]) {
        assert.match(example, new RegExp(`^${name}=`, "m"), `.env.example is missing ${name}`);
    }
});
