import { getApiBase } from "$lib/api.js";

/**
 * Calls a section's app through this backend's relay (lamsza WAYS_OF_WORKING
 * R18): `/api/admin/dictionary/<path>` reaches Szótár's internal admin API,
 * `/api/admin/games/<path>` Játszótér's.
 *
 * Returns the parsed JSON (null for an empty reply). A non-2xx reply throws an
 * Error with the app's own Hungarian message, plus `status` and the app's
 * `code` when it sends one (e.g. Játszótér's DATE_TAKEN, FROZEN).
 *
 * @param {"dictionary" | "games"} section
 * @param {string} path e.g. "words?letter=B" or "rovasfejto/puzzles"
 * @param {{ method?: string, body?: unknown }} [opts]
 */
export async function sectionFetch(section, path, opts = {}) {
    /** @type {Record<string, string>} */
    const headers = {};
    /** @type {RequestInit} */
    const init = { method: opts.method || "GET", credentials: "include", headers };
    if (opts.body !== undefined) {
        headers["Content-Type"] = "application/json";
        init.body = JSON.stringify(opts.body);
    }
    const res = await fetch(`${getApiBase()}/api/admin/${section}/${path.replace(/^\/+/, "")}`, init);
    const text = await res.text();
    /** @type {any} */
    let data = null;
    try {
        data = text.trim() ? JSON.parse(text) : null;
    } catch {
        data = null;
    }
    if (!res.ok) {
        const message =
            (data && typeof data.error === "string" && data.error) ||
            (res.status === 401
                ? "A munkamenet lejárt, lépj be újra."
                : res.status === 403
                  ? "Ehhez nincs jogosultság."
                  : `Hiba (${res.status}).`);
        const err = /** @type {Error & { status?: number, code?: string }} */ (new Error(message));
        err.status = res.status;
        if (data && typeof data.code === "string") err.code = data.code;
        throw err;
    }
    return data;
}

/** The thrown error's message, or a fallback. */
export function errorText(/** @type {unknown} */ err, /** @type {string} */ fallback) {
    return err instanceof Error && err.message ? err.message : fallback;
}

/** "2026. október 7." from "2026-10-07" (Szótár's date wording). */
/**
 * A timestamp as a Hungarian date and time in Bucharest (lamsza WAYS_OF_WORKING
 * R19), e.g. "2026. október 9. 14:05". Empty for a missing or bad value.
 */
export function formatHuDateTime(/** @type {string | null | undefined} */ iso) {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleString("hu-HU", {
        timeZone: "Europe/Bucharest",
        year: "numeric",
        month: "long",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

export function formatHuDate(/** @type {string | null | undefined} */ iso) {
    const [year, month, day] = String(iso ?? "").slice(0, 10).split("-").map(Number);
    if (!year || !month || !day) return "";
    return new Date(year, month - 1, day).toLocaleDateString("hu-HU", {
        year: "numeric",
        month: "long",
        day: "numeric",
    });
}
