import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import test from "node:test";

const read = (/** @type {string} */ path) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");
/** Every file under path, subfolders included (e.g. components/games/tajszorejtveny). */
const dir = (/** @type {string} */ path) =>
    readdirSync(new URL(`../${path}`, import.meta.url), { recursive: true, withFileTypes: true })
        .filter((d) => d.isFile())
        .map((d) => `${path}${d.parentPath.split(path)[1] ?? ""}/${d.name}`);

const pages = ["src/routes/+page.svelte", "src/routes/dictionary/+page.svelte", "src/routes/games/+page.svelte"];
const sectionFiles = [
    "src/routes/dictionary/+page.svelte",
    "src/routes/games/+page.svelte",
    ...dir("src/lib/components/dictionary"),
    ...dir("src/lib/components/games"),
    "src/lib/tajszorejtvenyAdmin.js",
];

test("every sidebar and card icon is one of the shared AppIcon icons", () => {
    // Sections use the network's icon set only (lamsza WAYS_OF_WORKING R18);
    // an entry without `icon` uses its id as the icon name.
    const icons = read("src/lib/icons/AppIcon.svelte");
    for (const page of pages) {
        const src = read(page);
        const nav = src.slice(src.indexOf("NAV = ["), src.indexOf("];", src.indexOf("NAV = [")));
        assert.ok(nav.length > 0, `${page} has no NAV list`);
        for (const [, id, icon] of nav.matchAll(/\{ id: "([^"]+)", title: "[^"]+"(?:, icon: "([^"]+)")?/g)) {
            const name = icon || id;
            assert.ok(icons.includes(`name === "${name}"`), `${page}: AppIcon has no "${name}" icon`);
        }
    }
});

test("the sections reach Szótár and Játszótér only through the relay helper", () => {
    // This app never calls the other apps directly (lamsza WAYS_OF_WORKING
    // R18): the browser talks to this backend, which relays with the token.
    for (const file of sectionFiles) {
        const src = read(file);
        assert.doesNotMatch(src, /\bfetch\(|apiCall\(|apiFetch\(/, `${file} calls an API without sectionFetch`);
        assert.doesNotMatch(src, /\/api\/admin\//, `${file} builds an /api/admin/ path by hand`);
        for (const [, section] of src.matchAll(/sectionFetch\(\s*"([^"]+)"/g)) {
            assert.ok(["dictionary", "games"].includes(section), `${file}: unknown section "${section}"`);
        }
    }
});

test("the sections take today from Szótár and Játszótér, never from the browser's clock", () => {
    // lamsza WAYS_OF_WORKING R19: "today" is a Bucharest day decided by the
    // server; the date defaults come from the apps' `today` (their stats).
    for (const file of sectionFiles) {
        const src = read(file);
        assert.doesNotMatch(src, /new Date\(\)|Date\.now\(|localISODate|getFullYear\(/, `${file} computes a date from the browser's clock`);
    }
});

test("word suggestions sit right under the words list, with their own icon", () => {
    const src = read("src/routes/dictionary/+page.svelte");
    const nav = src.slice(src.indexOf("NAV = ["), src.indexOf("];", src.indexOf("NAV = [")));
    const ids = [...nav.matchAll(/\{ id: "([^"]+)"/g)].map((m) => m[1]);
    assert.equal(ids[ids.indexOf("szavak") + 1], "javaslatok");
    assert.match(nav, /id: "javaslatok", title: "Szójavaslatok", icon: "word-suggestions"/);
    assert.match(nav, /id: "szavak", title: "Szavak", icon: "list"/);
});
