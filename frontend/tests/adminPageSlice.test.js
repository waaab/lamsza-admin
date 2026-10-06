import assert from "node:assert/strict";
import { test } from "node:test";
import { ADMIN_PAGE_SIZE, adminPageSlice } from "../src/lib/adminPageSlice.js";

const rows = (n) => Array.from({ length: n }, (_, i) => i + 1);

test("page size is 10", () => {
    assert.equal(ADMIN_PAGE_SIZE, 10);
});

test("first page of a full table", () => {
    const out = adminPageSlice(rows(25), 1);
    assert.deepEqual(out.rows, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
    assert.equal(out.total, 25);
    assert.equal(out.totalPages, 3);
    assert.equal(out.page, 1);
    assert.equal(out.from, 1);
    assert.equal(out.to, 10);
});

test("last page can be short", () => {
    const out = adminPageSlice(rows(25), 3);
    assert.deepEqual(out.rows, [21, 22, 23, 24, 25]);
    assert.equal(out.from, 21);
    assert.equal(out.to, 25);
});

test("page number is clamped into range", () => {
    assert.equal(adminPageSlice(rows(25), 0).page, 1);
    assert.equal(adminPageSlice(rows(25), -5).page, 1);
    assert.equal(adminPageSlice(rows(25), 99).page, 3);
});

test("empty table: one page, zero counters", () => {
    const out = adminPageSlice([], 1);
    assert.deepEqual(out.rows, []);
    assert.equal(out.total, 0);
    assert.equal(out.totalPages, 1);
    assert.equal(out.from, 0);
    assert.equal(out.to, 0);
});

test("non-array input is treated as empty", () => {
    for (const bad of [null, undefined, "abc", 7, {}]) {
        const out = adminPageSlice(bad, 1);
        assert.deepEqual(out.rows, []);
        assert.equal(out.total, 0);
        assert.equal(out.totalPages, 1);
    }
});

test("a custom page size is honoured", () => {
    const out = adminPageSlice(rows(7), 2, 3);
    assert.deepEqual(out.rows, [4, 5, 6]);
    assert.equal(out.totalPages, 3);
    assert.equal(out.from, 4);
    assert.equal(out.to, 6);
});
