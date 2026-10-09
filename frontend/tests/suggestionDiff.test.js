import assert from "node:assert/strict";
import { test } from "node:test";
import { changedCount, suggestionDiff } from "../src/lib/suggestionDiff.js";

/** @type {any} */
const word = {
    headword: "pityóka",
    pronunciation: "",
    audio_url: "https://storage.googleapis.com/lamsza_public_bucket/bg/hangok/pityoka.mp3",
    video_url: "",
    img_url: "",
    definitions: [
        {
            definition_hu: "Burgonya",
            speech_types: ["főnév"],
            description_hu: "",
            locations: ["Csíkszereda", "Gyergyó"],
            synonyms: [],
        },
    ],
};

test("a new word shows every filled field as added and leaves empty ones out", () => {
    const sections = suggestionDiff(null, word);
    assert.deepEqual(
        sections.map((s) => s.title),
        ["Szó", "1. jelentés"],
    );
    assert.deepEqual(
        sections[0].rows.map((r) => [r.label, r.change]),
        [
            ["Címszó", "added"],
            ["Hang", "added"],
        ],
    );
    assert.deepEqual(
        sections[1].rows.map((r) => [r.label, r.after, r.change]),
        [
            ["Magyar jelentés", "Burgonya", "added"],
            ["Szófaj", "főnév", "added"],
            ["Helyek", "Csíkszereda, Gyergyó", "added"],
        ],
    );
    assert.equal(changedCount(sections), 5);
});

test("an edit marks changed, added and removed fields and keeps the unchanged ones", () => {
    const suggested = structuredClone(word);
    suggested.definitions[0].definition_hu = "Burgonya, krumpli";
    suggested.definitions[0].description_hu = "A székely konyha alapja.";
    suggested.audio_url = "";
    const rows = suggestionDiff(word, suggested).flatMap((s) => s.rows);
    const byLabel = Object.fromEntries(rows.map((r) => [r.label, r]));
    assert.equal(byLabel["Címszó"].change, "same");
    assert.equal(byLabel["Hang"].change, "removed");
    assert.equal(byLabel["Magyar jelentés"].change, "changed");
    assert.equal(byLabel["Magyar jelentés"].before, "Burgonya");
    assert.equal(byLabel["Leírás"].change, "added");
    assert.equal(changedCount(suggestionDiff(word, suggested)), 3);
});

test("lists ignore order and letter case, so a reorder is not a change", () => {
    const suggested = structuredClone(word);
    suggested.definitions[0].locations = ["gyergyó", "Csíkszereda"];
    assert.equal(changedCount(suggestionDiff(word, suggested)), 0);
    suggested.definitions[0].locations.push("Udvarhely");
    const helyek = suggestionDiff(word, suggested)[1].rows.find((r) => r.label === "Helyek");
    assert.equal(helyek?.change, "changed");
});

test("senses are compared by position; an extra or missing sense is titled", () => {
    const more = structuredClone(word);
    more.definitions.push({ definition_hu: "Krumpli" });
    assert.deepEqual(
        suggestionDiff(word, more).map((s) => s.title),
        ["Szó", "1. jelentés", "2. jelentés (új)"],
    );
    const fewer = structuredClone(word);
    fewer.definitions = [];
    const sections = suggestionDiff(word, fewer);
    assert.equal(sections[1].title, "1. jelentés (törölve)");
    assert.ok(sections[1].rows.every((r) => r.change === "removed"));
});

test("missing or odd input does not throw", () => {
    assert.deepEqual(suggestionDiff(null, null), [{ title: "Szó", rows: [] }]);
    assert.equal(changedCount(suggestionDiff({ headword: " a " }, { headword: "a", definitions: "x" })), 0);
});
