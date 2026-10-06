import assert from "node:assert/strict";
import { afterEach, beforeEach, test } from "node:test";
import {
    DEFAULT_QUICKLINK_SLOTS,
    MAX_QUICKLINK_SLOTS,
    MIN_QUICKLINK_SLOTS,
    QUICKLINK_SLOTS_STORAGE_KEY,
    clampSlotCount,
    isWideQuicklinkLayout,
    readSlotCount,
    slotsForQuicklinkLayout,
    writeSlotCount,
} from "../src/lib/quickLinksDisplay.js";

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

test("constants", () => {
    assert.equal(DEFAULT_QUICKLINK_SLOTS, 7);
    assert.equal(MIN_QUICKLINK_SLOTS, 7);
    assert.equal(MAX_QUICKLINK_SLOTS, 14);
    assert.equal(QUICKLINK_SLOTS_STORAGE_KEY, "quick_links_display_count");
});

test("clampSlotCount: missing and non-numeric → default 7", () => {
    assert.equal(clampSlotCount(null), 7);
    assert.equal(clampSlotCount(undefined), 7);
    assert.equal(clampSlotCount(""), 7);
    assert.equal(clampSlotCount("abc"), 7);
    assert.equal(clampSlotCount(NaN), 7);
});

test("clampSlotCount: out of range", () => {
    assert.equal(clampSlotCount(3), 7);
    assert.equal(clampSlotCount(20), 14);
    assert.equal(clampSlotCount("6"), 7);
    assert.equal(clampSlotCount("15"), 14);
});

test("clampSlotCount: in range", () => {
    assert.equal(clampSlotCount(7), 7);
    assert.equal(clampSlotCount(10), 10);
    assert.equal(clampSlotCount(14), 14);
    assert.equal(clampSlotCount("8"), 8);
});

test("clampSlotCount: non-integer handling", () => {
    assert.equal(clampSlotCount(8.5), 8);
    assert.equal(clampSlotCount("10.9"), 10);
});

test("readSlotCount uses storage and clamps", () => {
    assert.equal(readSlotCount(), 7);
    localStorage.setItem(QUICKLINK_SLOTS_STORAGE_KEY, "11");
    assert.equal(readSlotCount(), 11);
    localStorage.setItem(QUICKLINK_SLOTS_STORAGE_KEY, "2");
    assert.equal(readSlotCount(), 7);
});

test("writeSlotCount persists clamped value", () => {
    assert.equal(writeSlotCount(9), 9);
    assert.equal(localStorage.getItem(QUICKLINK_SLOTS_STORAGE_KEY), "9");
    assert.equal(writeSlotCount(99), 14);
    assert.equal(localStorage.getItem(QUICKLINK_SLOTS_STORAGE_KEY), "14");
});

test("layout: default and 7 are narrow, anything wider is wide", () => {
    assert.equal(isWideQuicklinkLayout(null), false);
    assert.equal(isWideQuicklinkLayout(7), false);
    assert.equal(isWideQuicklinkLayout(8), true);
    assert.equal(slotsForQuicklinkLayout(false), 7);
    assert.equal(slotsForQuicklinkLayout(true), 8);
});

test("readSlotCount without localStorage returns default", () => {
    delete globalThis.localStorage;
    assert.equal(readSlotCount(), 7);
});
