/**
 * The per-field diff on the Szójavaslatok review screen: the word as it is now
 * in Szótár against the suggested one. A new word has no current word, so every
 * filled field shows as added. Senses are compared by position, lists without
 * regard to order or letter case (Szótár stores them de-duplicated that way).
 */

/** @typedef {"same" | "added" | "removed" | "changed"} Change */
/** @typedef {{ key: string, label: string, before: string, after: string, change: Change }} DiffRow */
/** @typedef {{ title: string, rows: DiffRow[] }} DiffSection */

/** The word's own fields, labelled as in the admin word form. */
export const WORD_FIELDS = [
    { key: "headword", label: "Címszó" },
    { key: "pronunciation", label: "Kiejtés" },
    { key: "audio_url", label: "Hang" },
    { key: "video_url", label: "Videó" },
    { key: "img_url", label: "Kép" },
];

/** A sense's fields, in the form's order. `list` fields hold arrays. */
export const SENSE_FIELDS = [
    { key: "definition_hu", label: "Magyar jelentés" },
    { key: "speech_types", label: "Szófaj", list: true },
    { key: "description_hu", label: "Leírás" },
    { key: "example_sentences", label: "Példamondatok" },
    { key: "example_proverbs", label: "Székely mondás ezzel a szóval" },
    { key: "example_source", label: "Forrás" },
    { key: "origin", label: "Eredet" },
    { key: "history", label: "Történet" },
    { key: "conjugation", label: "Ragozás" },
    { key: "locations", label: "Helyek", list: true },
    { key: "tags", label: "Címkék", list: true },
    { key: "synonyms", label: "Szinonimák", list: true },
    { key: "antonyms", label: "Ellentétek", list: true },
    { key: "variations", label: "Változatok", list: true },
];

/** @param {unknown} v */
function text(v) {
    return typeof v === "string" ? v.trim() : v == null ? "" : String(v).trim();
}

/** @param {unknown} v */
function items(v) {
    return Array.isArray(v) ? v.map(text).filter(Boolean) : [];
}

/** @param {string[]} a @param {string[]} b */
function sameList(a, b) {
    const norm = (/** @type {string[]} */ l) => [...new Set(l.map((x) => x.toLowerCase()))].sort().join("\u0000");
    return norm(a) === norm(b);
}

/**
 * @param {string} before
 * @param {string} after
 * @param {boolean} same
 * @returns {Change}
 */
function changeOf(before, after, same) {
    if (same) return "same";
    if (!before) return "added";
    if (!after) return "removed";
    return "changed";
}

/**
 * @param {{ key: string, label: string, list?: boolean }} field
 * @param {Record<string, any> | null | undefined} cur
 * @param {Record<string, any> | null | undefined} sug
 * @returns {DiffRow}
 */
function row(field, cur, sug) {
    if (field.list) {
        const a = items(cur?.[field.key]);
        const b = items(sug?.[field.key]);
        const before = a.join(", ");
        const after = b.join(", ");
        return { key: field.key, label: field.label, before, after, change: changeOf(before, after, sameList(a, b)) };
    }
    const before = text(cur?.[field.key]);
    const after = text(sug?.[field.key]);
    return { key: field.key, label: field.label, before, after, change: changeOf(before, after, before === after) };
}

/**
 * The diff, section by section: "Szó", then "1. jelentés", "2. jelentés"...
 * Rows where both sides are empty are left out.
 *
 * @param {Record<string, any> | null | undefined} current the word now, or null for a new word
 * @param {Record<string, any> | null | undefined} suggested the suggested word
 * @returns {DiffSection[]}
 */
export function suggestionDiff(current, suggested) {
    /** @type {DiffSection[]} */
    const sections = [];
    const keep = (/** @type {DiffRow} */ r) => r.before !== "" || r.after !== "";
    sections.push({ title: "Szó", rows: WORD_FIELDS.map((f) => row(f, current, suggested)).filter(keep) });
    const a = Array.isArray(current?.definitions) ? current.definitions : [];
    const b = Array.isArray(suggested?.definitions) ? suggested.definitions : [];
    for (let i = 0; i < Math.max(a.length, b.length); i++) {
        const rows = SENSE_FIELDS.map((f) => row(f, a[i], b[i])).filter(keep);
        let title = `${i + 1}. jelentés`;
        if (current && i >= a.length) title += " (új)";
        else if (i >= b.length) title += " (törölve)";
        if (rows.length) sections.push({ title, rows });
    }
    return sections;
}

/**
 * How many fields the suggestion changes.
 * @param {DiffSection[]} sections
 */
export function changedCount(sections) {
    return sections.reduce((n, s) => n + s.rows.filter((r) => r.change !== "same").length, 0);
}

/** Hungarian label for a change, for the diff's badge and screen readers. */
/** @type {Record<Change, string>} */
export const CHANGE_LABEL = { same: "változatlan", added: "új", removed: "törölve", changed: "módosítva" };
