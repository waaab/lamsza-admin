/** API / DB keys for event_schedule_activities.activity_type */
export const SCHEDULE_ACTIVITY_TYPES = /** @type {const} */ ([
    "opening",
    "match",
    "closing",
    "other",
]);

/** @type {Record<string, string>} */
export const SCHEDULE_ACTIVITY_TYPE_LABELS = {
    opening: "Megnyitó / rajt",
    match: "Mérkőzés / program",
    closing: "Záró",
    other: "Egyéb",
};
