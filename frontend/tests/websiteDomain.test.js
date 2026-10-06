import assert from "node:assert/strict";
import test from "node:test";
import { canonicalDomain, plainText } from "../src/lib/websiteDomain.js";

test("www, path, and subdomain share one key", () => {
    for (const raw of ["https://www.kezdisorozo.com/menu", "kezdisorozo.com", "shop.kezdisorozo.com"]) {
        assert.equal(canonicalDomain(raw), "kezdisorozo.com");
    }
    assert.equal(canonicalDomain("not a domain"), "");
});

test("plain text strips HTML and enforces length", () => {
    assert.equal(plainText("  <b>Csíki</b>  ", 120), "Csíki");
    assert.equal(plainText("   ", 120), "");
    assert.equal(plainText("abcd", 3), "");
});
