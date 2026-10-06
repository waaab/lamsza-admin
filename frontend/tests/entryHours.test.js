import assert from "node:assert/strict";
import { test } from "node:test";
import {
    defaultWeekHours,
    emptyWeekHours,
    formatDayHours,
    hoursConfigured,
    normalizeHours,
    openStatus,
    weekHoursAreEmpty,
    withDefaultWeekHours,
} from "../src/lib/entryHours.js";

test("normalizeHours: empty object is the default week", () => {
    const hours = normalizeHours(null);
    assert.deepEqual(hours, emptyWeekHours());
    assert.equal(hoursConfigured(hours), false);
});

test("withDefaultWeekHours fills an empty week and keeps saved hours", () => {
    const filled = withDefaultWeekHours(null);
    assert.deepEqual(filled, defaultWeekHours());
    assert.equal(filled.mon.open, "08:00");
    assert.equal(filled.mon.close, "20:00");
    assert.equal(weekHoursAreEmpty(filled), false);
    const saved = { mon: { open: "09:00", close: "17:00", closed: false } };
    assert.equal(withDefaultWeekHours(saved).mon.open, "09:00");
});

test("hoursConfigured: open+close or closed counts", () => {
    assert.equal(hoursConfigured({ mon: { open: "09:00", close: "17:00" } }), true);
    assert.equal(hoursConfigured({ sun: { closed: true } }), true);
    assert.equal(hoursConfigured({ mon: { open: "09:00" } }), false);
});

test("formatDayHours: closed, range, placeholder", () => {
    assert.equal(formatDayHours({ closed: true }), "Zárva");
    assert.equal(formatDayHours({ open: "09:00", close: "17:00", closed: false }), "09:00 - 17:00");
    assert.equal(formatDayHours({ open: "", close: "", closed: false }), "-");
});

test("openStatus: unknown when hours are not set", () => {
    const status = openStatus({}, new Date("2026-09-21T07:00:00.000Z"));
    assert.equal(status.state, "unknown");
    assert.equal(status.label, "Nyitvatartás");
});

test("openStatus: open and closed from weekly hours in Bucharest", () => {
    const hours = { mon: { open: "09:00", close: "17:00", closed: false } };
    const open = openStatus(hours, new Date("2026-09-21T07:00:00.000Z"));
    const closed = openStatus(hours, new Date("2026-09-21T15:00:00.000Z"));
    assert.equal(open.state, "open");
    assert.equal(open.label, "Nyitva");
    assert.equal(open.detail, "09:00 - 17:00");
    assert.equal(closed.state, "closed");
    assert.equal(closed.label, "Zárva");
});
