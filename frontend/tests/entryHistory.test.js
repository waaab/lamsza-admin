import assert from "node:assert/strict";
import { afterEach, beforeEach, test } from "node:test";
import {
    ENTRY_HISTORY_SHOW_MAX,
    ENTRY_HISTORY_STORAGE_KEY,
    ENTRY_HISTORY_STORE_MAX,
    historyForDisplay,
    normalizeHistory,
    readHistory,
    recordAccountHistory,
    recordHistoryVisit,
} from "../src/lib/entryHistory.js";

const store = new Map();

function installLocalStorage() {
    globalThis.localStorage = {
        getItem(key) {
            return store.has(key) ? store.get(key) : null;
        },
        setItem(key, value) {
            store.set(key, String(value));
        },
        removeItem(key) {
            store.delete(key);
        },
    };
}

beforeEach(() => {
    store.clear();
    installLocalStorage();
});

afterEach(() => {
    delete globalThis.localStorage;
});

test("normalizeHistory: empty and corrupt become []", () => {
    assert.deepEqual(normalizeHistory(null), []);
    assert.deepEqual(normalizeHistory("nope"), []);
    assert.deepEqual(normalizeHistory([{ name: "No slug" }]), []);
});

test("recordHistoryVisit: prepends, dedupes by slug, caps store", () => {
    for (let i = 0; i < ENTRY_HISTORY_STORE_MAX + 3; i++) {
        recordHistoryVisit({
            slug: `e-${i}`,
            name: `N${i}`,
            category: "Vendéglő",
            location: "Kézdivásárhely",
            photo: "",
        });
    }
    const all = readHistory();
    assert.equal(all.length, ENTRY_HISTORY_STORE_MAX);
    assert.equal(all[0].slug, `e-${ENTRY_HISTORY_STORE_MAX + 2}`);
    recordHistoryVisit({ slug: "e-0", name: "N0 again" });
    const again = readHistory();
    assert.equal(again[0].slug, "e-0");
    assert.equal(again.filter((x) => x.slug === "e-0").length, 1);
    assert.equal(localStorage.getItem(ENTRY_HISTORY_STORAGE_KEY)?.[0], "[");
});

test("historyForDisplay: skips current and caps show", () => {
    const items = Array.from({ length: 10 }, (_, i) => ({
        slug: `e-${i}`,
        name: `N${i}`,
        category: "",
        location: "",
        photo: "",
    }));
    const shown = historyForDisplay(items, "e-0");
    assert.equal(shown.length, ENTRY_HISTORY_SHOW_MAX);
    assert.equal(shown[0].slug, "e-1");
    assert.ok(!shown.some((x) => x.slug === "e-0"));
});

test("readHistory: corrupt JSON is []", () => {
    localStorage.setItem(ENTRY_HISTORY_STORAGE_KEY, "{not-json");
    assert.deepEqual(readHistory(), []);
});

test("recordAccountHistory posts when logged in", async () => {
    const calls = [];
    globalThis.fetch = async (url, opts) => {
        calls.push({ url: String(url), body: opts?.body });
        return { ok: true, json: async () => [] };
    };
    await recordAccountHistory(
        { slug: "kavezo", name: "Kávézó", category: "", location: "", photo: "" },
        { loggedIn: true },
    );
    assert.equal(calls.length, 1);
    assert.match(calls[0].url, /\/api\/account\/history$/);
});
