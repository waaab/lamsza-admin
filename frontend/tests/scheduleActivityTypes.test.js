import assert from "node:assert/strict";
import { test } from "node:test";
import {
    SCHEDULE_ACTIVITY_TYPES,
    SCHEDULE_ACTIVITY_TYPE_LABELS,
} from "../src/lib/scheduleActivityTypes.js";

test("the activity type keys are the DB values, in order", () => {
    assert.deepEqual(SCHEDULE_ACTIVITY_TYPES, [
        "opening",
        "match",
        "closing",
        "other",
    ]);
});

test("every key has a Hungarian label and nothing extra has one", () => {
    for (const key of SCHEDULE_ACTIVITY_TYPES) {
        const label = SCHEDULE_ACTIVITY_TYPE_LABELS[key];
        assert.equal(typeof label, "string", `missing label for ${key}`);
        assert.ok(label.length > 0, `empty label for ${key}`);
    }
    assert.deepEqual(
        Object.keys(SCHEDULE_ACTIVITY_TYPE_LABELS).sort(),
        [...SCHEDULE_ACTIVITY_TYPES].sort(),
    );
});
