/** Canonical `entry_types.name` values (admin catalog). */
export const ENTRY_TYPE_SZEMELY = "Személy";
export const ENTRY_TYPE_VALLALKOZAS = "Vállalkozás";
export const ENTRY_TYPE_INTEZMENY = "Intézmény";

/** @param {unknown} raw */
function foldType(raw) {
    return String(raw ?? "")
        .trim()
        .toLowerCase()
        .normalize("NFD")
        .replace(/\p{M}/gu, "");
}

/**
 * Map stored type strings onto catalog labels.
 * Unknown values return an empty string.
 *
 * @param {unknown} raw
 * @returns {string}
 */
export function canonicalEntryType(raw) {
    const trimmed = String(raw ?? "").trim();
    if (!trimmed) return "";
    if (
        trimmed === ENTRY_TYPE_SZEMELY ||
        trimmed === ENTRY_TYPE_VALLALKOZAS ||
        trimmed === ENTRY_TYPE_INTEZMENY
    ) {
        return trimmed;
    }
    return "";
}

/** @param {unknown} raw */
export function canonicalEntryTypeKey(raw) {
    const label = canonicalEntryType(raw);
    return label ? foldType(label) : "";
}

const ENTRY_TYPE_ORDER = [
    ENTRY_TYPE_SZEMELY,
    ENTRY_TYPE_VALLALKOZAS,
    ENTRY_TYPE_INTEZMENY,
];

/** Closed type list, in catalog order, for the services sidebar. */
export function entryTypeChoices() {
    return ENTRY_TYPE_ORDER.map((label) => ({
        key: canonicalEntryTypeKey(label),
        label,
    }));
}

/** @param {unknown} key */
export function entryTypeLabelFromKey(key) {
    const wanted = String(key ?? "");
    return entryTypeChoices().find((row) => row.key === wanted)?.label || "";
}
