/**
 * The admin app's three sections and the public apps they edit (lamsza
 * WAYS_OF_WORKING R18).
 *
 * The sections are routes of this app (`/`, `/dictionary`, `/games`), so the
 * header switcher links are same-origin paths. Each section's first sidebar
 * button opens its public app in a new tab; those origins come from build-time
 * `VITE_*` values.
 *
 * The variable names below are the contract with `.env.example`. If you rename
 * one here, rename it there; `tests/adminApps.test.js` pins the names so a
 * silent drift fails the suite instead of quietly falling back to localhost.
 */

/** Header switcher targets, in header order. */
export const ADMIN_SECTIONS = [
    { id: "directory", icon: "dashboard", label: "Lámsza admin", href: "/" },
    { id: "szotar", icon: "szotar", label: "Szótár admin", href: "/dictionary" },
    { id: "jatszoter", icon: "jatszoter", label: "Játszótér admin", href: "/games" },
];

/**
 * The section a path belongs to: `/dictionary...` and `/games...` are theirs,
 * anything else is the main admin.
 * @param {string} [pathname]
 */
export function sectionForPath(pathname) {
    const path = String(pathname || "/");
    for (const s of ADMIN_SECTIONS) {
        if (s.href !== "/" && (path === s.href || path.startsWith(`${s.href}/`))) return s.id;
    }
    return "directory";
}

/**
 * @param {string} [pathname] the current `location.pathname`
 * @returns {{ id: string, icon: string, label: string, href: string, current: boolean }[]}
 */
export function adminAppLinks(pathname) {
    const current = sectionForPath(pathname);
    return ADMIN_SECTIONS.map((s) => ({ ...s, current: s.id === current }));
}

/** Fixed local dev ports, from lamsza's `docs/network/WAYS_OF_WORKING.md` §1. */
export const APP_ORIGIN_DEFAULTS = {
    lamsza: "http://localhost:5174",
    szotar: "http://localhost:5175",
    jatszoter: "http://localhost:5176",
};

/**
 * Public app origins, for the sidebar's "open in a new tab" button and the
 * links into the apps (a word's page, the games).
 * @param {Record<string, unknown>} [env] normally `import.meta.env`
 * @returns {{ lamsza: string, szotar: string, jatszoter: string }}
 */
export function appOrigins(env) {
    const read = (/** @type {string} */ name, /** @type {string} */ fallback) => {
        const raw = env && typeof env === "object" ? env[name] : undefined;
        const value = typeof raw === "string" ? raw.trim() : "";
        // A trailing slash would make `https://host//szo/1` out of a joined path.
        return (value || fallback).replace(/\/+$/, "");
    };
    return {
        lamsza: read("VITE_LAMSZA_ORIGIN", APP_ORIGIN_DEFAULTS.lamsza),
        szotar: read("VITE_SZOTAR_ORIGIN", APP_ORIGIN_DEFAULTS.szotar),
        jatszoter: read("VITE_JATSZOTER_ORIGIN", APP_ORIGIN_DEFAULTS.jatszoter),
    };
}

/** The origins for this build. */
export const APP_ORIGINS = appOrigins(typeof import.meta !== "undefined" ? import.meta.env : undefined);
