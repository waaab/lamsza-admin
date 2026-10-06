/**
 * Targets for the header app switcher.
 *
 * The three admin UIs sit on three different origins — this app on its own
 * host, Szótár and Játszótér still on `/admin` inside their own apps — so the
 * links cannot be relative. They come from build-time `VITE_*` values.
 *
 * The variable names below are the contract with `.env.example`. If you rename
 * one here, rename it there; `tests/adminApps.test.js` pins the names so a
 * silent drift fails the suite instead of quietly falling back to localhost.
 */

/** Fixed local dev ports, from `docs/ARCHITECTURE.md`. */
export const ADMIN_APP_DEFAULTS = {
    directory: "http://localhost:5173",
    szotar: "http://localhost:5175/admin",
    jatszoter: "http://localhost:5176/admin",
};

/**
 * @param {Record<string, unknown>} [env] normally `import.meta.env`
 * @returns {{ id: string, icon: string, label: string, href: string, current: boolean }[]}
 */
export function adminAppLinks(env) {
    const read = (name, fallback) => {
        const raw = env && typeof env === "object" ? env[name] : undefined;
        const value = typeof raw === "string" ? raw.trim() : "";
        // A trailing slash would make `http://host//admin` out of a joined path
        // later and reads as a different origin in the OAuth console.
        return (value || fallback).replace(/\/+$/, "");
    };

    return [
        {
            id: "directory",
            icon: "dashboard",
            label: "Directory admin",
            href: read("VITE_ADMIN_ORIGIN", ADMIN_APP_DEFAULTS.directory),
            current: true,
        },
        {
            id: "szotar",
            icon: "szotar",
            label: "Szótár admin",
            href: read("VITE_SZOTAR_ADMIN_ORIGIN", ADMIN_APP_DEFAULTS.szotar),
            current: false,
        },
        {
            id: "jatszoter",
            icon: "jatszoter",
            label: "Játszótér admin",
            href: read("VITE_JATSZOTER_ADMIN_ORIGIN", ADMIN_APP_DEFAULTS.jatszoter),
            current: false,
        },
    ];
}
