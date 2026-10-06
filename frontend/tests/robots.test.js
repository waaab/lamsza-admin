import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

// The admin app's robots.txt started life as a copy of the public site's file,
// which says "crawl everything" (BOG-40). admin.lamsza.com has no public page,
// so this is the guard against that copy coming back.
const robots = readFileSync(new URL("../static/robots.txt", import.meta.url), "utf8");

/** Directive lines only, comments and blanks dropped. */
const directives = robots
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"));

test("robots.txt disallows every crawler from the whole site", () => {
    assert.deepEqual(directives, ["User-agent: *", "Disallow: /"]);
});

test("robots.txt has no bare Disallow, which means allow-all", () => {
    assert.ok(
        !directives.some((line) => /^Disallow:\s*$/i.test(line)),
        "an empty Disallow value grants full access",
    );
});
