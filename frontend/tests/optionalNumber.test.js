import test from "node:test";
import assert from "node:assert/strict";
import { isOptionalNumber, optionalNumber } from "../src/lib/optionalNumber.js";

test("optionalNumber reads points, commas and spaced thousands", () => {
    assert.equal(optionalNumber("946"), 946);
    assert.equal(optionalNumber("0,22"), 0.22);
    assert.equal(optionalNumber(" 1 234,5 "), 1234.5);
    assert.equal(optionalNumber("-12.5"), -12.5);
    assert.equal(optionalNumber(7), 7);
});

test("optionalNumber is null for an empty field or something else", () => {
    assert.equal(optionalNumber(""), null);
    assert.equal(optionalNumber(null), null);
    assert.equal(optionalNumber("kb. 900"), null);
    assert.equal(optionalNumber("9e3"), null);
    assert.equal(isOptionalNumber(""), true);
    assert.equal(isOptionalNumber("0,22"), true);
    assert.equal(isOptionalNumber("sok"), false);
});
