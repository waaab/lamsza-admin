/**
 * API origin for fetches.
 * - In the browser, use same-origin paths (`/api/...`) so Vite’s dev proxy (see vite.config.js)
 *   and production reverse proxies can forward to the Go backend. A hard-coded
 *   `http://localhost:3000` bypasses the proxy and breaks `npm run dev` when the
 *   backend is only reachable via the proxy or another port.
 * - During SSR/prerender (no `window`), fall back to a direct backend URL unless
 *   `VITE_API_BASE_URL` is set.
 */
export function getApiBase() {
    const env = import.meta.env.VITE_API_BASE_URL;
    if (env) return String(env).replace(/\/$/, "");
    /** Same tab as the Svelte app - absolute origin so fetches always resolve (Vite proxy / reverse proxy). */
    if (typeof window !== "undefined" && window.location?.origin) {
        return window.location.origin;
    }
    return "http://127.0.0.1:3000";
}

/**
 * Interpret an API body. A 2xx response with an empty body is success (`null`).
 * Several account mutations reply with `200` and no JSON; parsing that as JSON
 * made a successful save or delete look like "A mentés nem sikerült".
 * @param {boolean} ok
 * @param {number} status
 * @param {string} text
 */
export function parseApiPayload(ok, status, text) {
    if (!ok) {
        throw new Error(String(text || "").trim() || `API Error: ${status}`);
    }
    const body = String(text ?? "");
    if (!body.trim()) return null;
    return JSON.parse(body);
}

/**
 * Enhanced fetch wrapper for the Lamsza API
 * @param {string} endpoint - The relative endpoint (e.g. '/api/directory')
 * @param {RequestInit} options - Standard fetch options
 * @returns {Promise<any>}
 */
export async function apiFetch(endpoint, options = {}) {
    const base = getApiBase();
    const url = endpoint.startsWith("http")
        ? endpoint
        : `${base}${endpoint}`;

    try {
        const response = await fetch(url, { credentials: "include", ...options });
        const text = await response.text();
        return parseApiPayload(response.ok, response.status, text);
    } catch (error) {
        console.error(`Fetch error for ${url}:`, error);
        throw error;
    }
}

/**
 * Same-origin API fetch that returns the raw Response (for admin calls that
 * inspect status / text themselves). Always sends the session cookie.
 * @param {string} path
 * @param {RequestInit} [options]
 */
export function apiCall(path, options = {}) {
    const url = path.startsWith("http") ? path : `${getApiBase()}${path}`;
    return fetch(url, { credentials: "include", ...options });
}
