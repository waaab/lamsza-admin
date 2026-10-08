import assert from "node:assert/strict";
import { test } from "node:test";
import { dashboardLines, dayName, waitForTask, weekDates, weekMatrix } from "../src/lib/tajszorejtvenyAdmin.js";

test("a week is the seven dates from its Monday, across a month end", () => {
    assert.deepEqual(weekDates("2026-10-26"), [
        "2026-10-26",
        "2026-10-27",
        "2026-10-28",
        "2026-10-29",
        "2026-10-30",
        "2026-10-31",
        "2026-11-01",
    ]);
    assert.deepEqual(weekDates(""), []);
});

test("day names come from the date itself, also on the clock-change Sunday", () => {
    assert.equal(dayName("2026-10-12"), "hétfő");
    assert.equal(dayName("2026-10-25"), "vasárnap");
    assert.equal(dayName("2026-03-29"), "vasárnap");
});

test("the matrix has a row per day and a cell per level, empty where nothing is made", () => {
    const m = weekMatrix({
        week_start: "2026-10-12",
        dailies: [
            { id: "a", date: "2026-10-12", level: 1 },
            { id: "b", date: "2026-10-18", level: 4 },
        ],
    });
    assert.equal(m.length, 7);
    assert.equal(m[0].cells[0]?.id, "a");
    assert.equal(m[0].cells[1], null);
    assert.equal(m[6].date, "2026-10-18");
    assert.equal(m[6].cells[3]?.id, "b");
    assert.deepEqual(weekMatrix(null), []);
});

test("waiting for a background task polls until it stops and reports its error", async () => {
    const replies = [
        { tasks: [{ key: "generate:2026-10-12", running: true }] },
        { tasks: [{ key: "generate:2026-10-12", running: true }] },
        { tasks: [{ key: "generate:2026-10-12", running: false, error: "2 rejtvény nem készült el" }] },
    ];
    let calls = 0;
    const res = await waitForTask(async () => replies[calls++], "generate:2026-10-12", { interval: 1, sleep: async () => {} });
    assert.equal(calls, 3);
    assert.deepEqual(res, { done: true, error: "2 rejtvény nem készült el" });
});

test("a task that never stops times out instead of hanging", async () => {
    const res = await waitForTask(async () => ({ tasks: [{ key: "k", running: true }] }), "k", {
        interval: 10,
        timeout: 30,
        sleep: async () => {},
    });
    assert.deepEqual(res, { done: false, error: "" });
});

test("dashboard lines: next week's status first, then what needs a look", () => {
    const lines = dashboardLines({
        enabled: false,
        next_needs_attention: 2,
        this_week_unreviewed: 0,
        open_reports: 3,
        late_periods: 1,
        message: { level: "urgent", text: "Jövő hét: 20/28 jóváhagyva. Hétfő 0:00-kor a jóvá nem hagyott rejtvények is élesek lesznek." },
    });
    assert.equal(lines[0].level, "error");
    assert.match(lines[0].text, /^Tájszórejtvény: Jövő hét: 20\/28/);
    assert.deepEqual(
        lines.slice(1).map((l) => l.level),
        ["warning", "warning", "info", "info"],
    );
    assert.deepEqual(dashboardLines(null), []);
    assert.equal(dashboardLines({ enabled: true, message: { level: "ok", text: "Jövő hét: 28/28 jóváhagyva" } })[0].level, "success");
});
