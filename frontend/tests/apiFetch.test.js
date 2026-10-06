import assert from "node:assert/strict";
import test from "node:test";
import { parseApiPayload } from "../src/lib/api.js";

test("empty success body is a successful response", () => {
    assert.equal(parseApiPayload(true, 200, ""), null);
    assert.equal(parseApiPayload(true, 200, "   "), null);
});

test("success json is parsed", () => {
    assert.deepEqual(parseApiPayload(true, 200, '{"ok":true}'), { ok: true });
});

test("error status throws the response text", () => {
    assert.throws(() => parseApiPayload(false, 400, "invalid link"), /invalid link/);
});
