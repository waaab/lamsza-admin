import assert from "node:assert/strict";
import { test } from "node:test";
import {
    ENTRY_TYPE_INTEZMENY,
    ENTRY_TYPE_SZEMELY,
    ENTRY_TYPE_VALLALKOZAS,
    canonicalEntryType,
    canonicalEntryTypeKey,
    entryTypeChoices,
    entryTypeLabelFromKey,
} from "../src/lib/entryType.js";

test("canonical labels match the admin catalog", () => {
    assert.equal(ENTRY_TYPE_SZEMELY, "Személy");
    assert.equal(ENTRY_TYPE_VALLALKOZAS, "Vállalkozás");
    assert.equal(ENTRY_TYPE_INTEZMENY, "Intézmény");
});

test("canonicalEntryType accepts only the three catalog names", () => {
    assert.equal(canonicalEntryType("Személy"), ENTRY_TYPE_SZEMELY);
    assert.equal(canonicalEntryType("Vállalkozás"), ENTRY_TYPE_VALLALKOZAS);
    assert.equal(canonicalEntryType("Intézmény"), ENTRY_TYPE_INTEZMENY);
    assert.equal(canonicalEntryType("service"), "");
    assert.equal(canonicalEntryType("Szolgáltatás"), "");
    assert.equal(canonicalEntryType("Cég"), "");
    assert.equal(canonicalEntryType("Egyéb"), "");
});

test("canonicalEntryType: empty stays empty", () => {
    assert.equal(canonicalEntryType(""), "");
    assert.equal(canonicalEntryType(null), "");
    assert.equal(canonicalEntryType(undefined), "");
});

test("canonicalEntryTypeKey folds accents and case", () => {
    assert.equal(canonicalEntryTypeKey(ENTRY_TYPE_SZEMELY), "szemely");
    assert.equal(canonicalEntryTypeKey(ENTRY_TYPE_VALLALKOZAS), "vallalkozas");
    assert.equal(canonicalEntryTypeKey(ENTRY_TYPE_INTEZMENY), "intezmeny");
    assert.equal(canonicalEntryTypeKey("service"), "");
});

test("entryTypeChoices is the closed list, in catalog order", () => {
    assert.deepEqual(entryTypeChoices(), [
        { key: "szemely", label: ENTRY_TYPE_SZEMELY },
        { key: "vallalkozas", label: ENTRY_TYPE_VALLALKOZAS },
        { key: "intezmeny", label: ENTRY_TYPE_INTEZMENY },
    ]);
});

test("entryTypeLabelFromKey maps back, unknown keys give empty", () => {
    assert.equal(entryTypeLabelFromKey("vallalkozas"), ENTRY_TYPE_VALLALKOZAS);
    assert.equal(entryTypeLabelFromKey("szemely"), ENTRY_TYPE_SZEMELY);
    assert.equal(entryTypeLabelFromKey("Vállalkozás"), "");
    assert.equal(entryTypeLabelFromKey(""), "");
    assert.equal(entryTypeLabelFromKey(null), "");
});
