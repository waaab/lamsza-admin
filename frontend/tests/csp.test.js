import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const read = (relative) =>
    readFileSync(new URL(relative, import.meta.url), "utf-8");

const svelteConfig = read("../svelte.config.js");
const appHtml = read("../src/app.html");

// Admin is the highest-value target in the family: every route behind it writes
// the directory the public site reads. script-src 'self' is what stops an
// injected tag from running, and connect-src is what stops a stolen session from
// being posted out. These tests fail loudly before either one loosens.

test("svelte.config.js declares a Content-Security-Policy", () => {
    assert.match(svelteConfig, /csp:\s*{/);
    assert.match(svelteConfig, /mode:\s*'hash'/);
});

test("script-src allows this origin and Google Sign-In only, never unsafe-inline or unsafe-eval", () => {
    const scriptSrc = svelteConfig.match(/'script-src':\s*\[([^\]]*)\]/);
    assert.ok(scriptSrc, "script-src is missing from the policy");
    assert.match(scriptSrc[1], /'self'/);
    assert.match(scriptSrc[1], /'https:\/\/accounts\.google\.com\/gsi\/client'/);
    assert.doesNotMatch(scriptSrc[1], /unsafe-inline/);
    assert.doesNotMatch(scriptSrc[1], /unsafe-eval/);
    assert.doesNotMatch(scriptSrc[1], /\bhttps:'/, "script-src must not allow any https host");
});

test("connect-src is an allowlist, so a stolen session cannot be posted out", () => {
    const connectSrc = svelteConfig.match(/'connect-src':\s*\[([^\]]*)\]/);
    assert.ok(connectSrc, "connect-src is missing from the policy");
    assert.match(connectSrc[1], /'self'/);
    assert.doesNotMatch(connectSrc[1], /\*/);
    assert.doesNotMatch(connectSrc[1], /'https:'/);
});

test("object-src and base-uri are locked down", () => {
    assert.match(svelteConfig, /'object-src':\s*\['none'\]/);
    assert.match(svelteConfig, /'base-uri':\s*\['self'\]/);
});

test("app.html carries no inline script, which script-src 'self' would block", () => {
    for (const match of appHtml.matchAll(/<script([^>]*)>([\s\S]*?)<\/script>/g)) {
        const [, attrs, body] = match;
        assert.ok(
            / src=/.test(attrs),
            `app.html has an inline <script> block; move it to static/ instead:\n${body.trim().slice(0, 120)}`,
        );
    }
});

test("the theme is set from an external file before the first paint", () => {
    assert.match(appHtml, /<script src="%sveltekit\.assets%\/theme-init\.js"><\/script>/);
    // Must stay ahead of the body so no wrong-colour frame is shown.
    assert.ok(appHtml.indexOf("theme-init.js") < appHtml.indexOf("%sveltekit.body%"));
    assert.match(read("../static/theme-init.js"), /localStorage\.getItem\('theme'\)/);
});

// `marked` was a dependency with no importer. It is the library that made the
// lamsza Markdown XSS possible (BOG-17), so if it comes back here it should come
// back with a sanitizer and a deliberate decision, not by accident.
test("no HTML-rendering Markdown dependency sneaks back in unused", () => {
    const pkg = JSON.parse(read("../package.json"));
    const deps = { ...pkg.dependencies, ...pkg.devDependencies };
    assert.ok(
        !("marked" in deps),
        "marked is back in package.json; if a page really needs Markdown, sanitize it the way lamsza src/lib/markdown.js does",
    );
});
