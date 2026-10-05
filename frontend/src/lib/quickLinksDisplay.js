export const DEFAULT_QUICKLINK_SLOTS = 7;
export const MIN_QUICKLINK_SLOTS = 7;
export const MAX_QUICKLINK_SLOTS = 14;
export const QUICKLINK_SLOTS_STORAGE_KEY = "quick_links_display_count";

/** @param {unknown} value */
export function clampSlotCount(value) {
    if (value == null || value === "") return DEFAULT_QUICKLINK_SLOTS;
    const n = typeof value === "number" ? value : Number.parseInt(String(value), 10);
    if (!Number.isFinite(n)) return DEFAULT_QUICKLINK_SLOTS;
    const truncated = Math.trunc(n);
    return Math.min(MAX_QUICKLINK_SLOTS, Math.max(MIN_QUICKLINK_SLOTS, truncated));
}

export function readSlotCount() {
    if (typeof localStorage === "undefined") return DEFAULT_QUICKLINK_SLOTS;
    try {
        return clampSlotCount(localStorage.getItem(QUICKLINK_SLOTS_STORAGE_KEY));
    } catch {
        return DEFAULT_QUICKLINK_SLOTS;
    }
}

/** @param {unknown} n */
export function writeSlotCount(n) {
    const v = clampSlotCount(n);
    if (typeof localStorage === "undefined") return v;
    try {
        localStorage.setItem(QUICKLINK_SLOTS_STORAGE_KEY, String(v));
    } catch {
        /* ignore quota / private-mode failures */
    }
    return v;
}

/** Stored slot counts above the default mean the wide homepage row. */
export function isWideQuicklinkLayout(slots) {
    return clampSlotCount(slots) > DEFAULT_QUICKLINK_SLOTS;
}

/** @param {boolean} wide */
export function slotsForQuicklinkLayout(wide) {
    return wide ? DEFAULT_QUICKLINK_SLOTS + 1 : DEFAULT_QUICKLINK_SLOTS;
}
