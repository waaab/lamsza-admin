import { getApiBase } from "$lib/api";

/**
 * Absolute URL for an event's featured image, or the built-in placeholder SVG path.
 * @param {Record<string, unknown>} event
 * @param {string} [apiBase] - override API origin (e.g. admin `getBase()`); default uses getApiBase().
 */
export function eventFeaturedImageUrl(event, apiBase) {
    const u = event?.featured_image;
    const s = u != null ? String(u).trim() : "";
    if (!s) return "/images/event-featured-placeholder.svg";
    return absoluteMediaUrl(s, apiBase);
}

/**
 * @param {string} stored - absolute http(s) URL or path starting with /
 * @param {string} [apiBase]
 */
export function absoluteMediaUrl(stored, apiBase) {
    const s = stored != null ? String(stored).trim() : "";
    if (!s) return "";
    if (s.startsWith("http://") || s.startsWith("https://")) return s;
    const base =
        apiBase !== undefined && apiBase !== null
            ? String(apiBase).replace(/\/$/, "")
            : getApiBase();
    return `${base}${s.startsWith("/") ? s : "/" + s}`;
}
