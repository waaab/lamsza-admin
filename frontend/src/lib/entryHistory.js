import { getApiBase } from "./api.js";

export const ENTRY_HISTORY_STORAGE_KEY = "lamsza_entry_history";
export const ENTRY_HISTORY_STORE_MAX = 12;
export const ENTRY_HISTORY_SHOW_MAX = 8;

function asItem(raw) {
    if (!raw || typeof raw !== "object") return null;
    const slug = String(raw.slug ?? "").trim();
    const name = String(raw.name ?? "").trim();
    if (!slug || !name) return null;
    return {
        slug,
        name,
        category: String(raw.category ?? "").trim(),
        location: String(raw.location ?? "").trim(),
        photo: String(raw.photo ?? "").trim(),
    };
}

export function normalizeHistory(raw) {
    let arr = raw;
    if (typeof raw === "string") {
        try {
            arr = JSON.parse(raw);
        } catch {
            return [];
        }
    }
    if (!Array.isArray(arr)) return [];
    const out = [];
    const seen = new Set();
    for (const row of arr) {
        const item = asItem(row);
        if (!item || seen.has(item.slug)) continue;
        seen.add(item.slug);
        out.push(item);
        if (out.length >= ENTRY_HISTORY_STORE_MAX) break;
    }
    return out;
}

export function readHistory() {
    if (typeof localStorage === "undefined") return [];
    try {
        return normalizeHistory(localStorage.getItem(ENTRY_HISTORY_STORAGE_KEY));
    } catch {
        return [];
    }
}

export function recordHistoryVisit(raw) {
    const item = asItem(raw);
    const prev = readHistory().filter((row) => !item || row.slug !== item.slug);
    const next = item ? [item, ...prev].slice(0, ENTRY_HISTORY_STORE_MAX) : prev;
    if (typeof localStorage !== "undefined") {
        try {
            localStorage.setItem(ENTRY_HISTORY_STORAGE_KEY, JSON.stringify(next));
        } catch {
            /* quota / private mode */
        }
    }
    return next;
}

export function historyForDisplay(items, currentSlug) {
    const skip = String(currentSlug ?? "").trim();
    return normalizeHistory(items)
        .filter((row) => row.slug !== skip)
        .slice(0, ENTRY_HISTORY_SHOW_MAX);
}

function accountHistoryUrl() {
    try {
        return `${getApiBase()}/api/account/history`;
    } catch {
        return "/api/account/history";
    }
}

export async function recordAccountHistory(raw, authState = {}) {
    const item = asItem(raw);
    if (authState?.loggedIn && item) {
        try {
            await fetch(accountHistoryUrl(), {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(item),
            });
        } catch {
            /* network / server - still keep browser copy */
        }
    }
    return recordHistoryVisit(raw);
}
