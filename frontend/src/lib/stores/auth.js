import { writable } from "svelte/store";
import { getApiBase } from "$lib/api.js";
import { buildImportPayload, meToAuthState } from "$lib/accountPrefs.js";
import { readHistory } from "$lib/entryHistory.js";
import { readSlotCount } from "$lib/quickLinksDisplay.js";
import { applyThemeLocal } from "$lib/stores/theme.js";

const empty = {
    loggedIn: false,
    user: "",
    email: "",
    isAdmin: false,
    picture: "",
    givenName: "",
    familyName: "",
    displayName: "",
    locale: "",
    googleSub: "",
    lastLoginAt: null,
    createdAt: null,
    theme: null,
    quicklinkSlots: null,
    prefsImportedAt: null,
    preferredLocation: null,
    adminQueueCount: 0,
    offline: false,
};

const SESSION_CACHE_KEY = "lamsza_auth_session";

function readSessionCache() {
    if (typeof sessionStorage === "undefined") return null;
    try {
        const raw = sessionStorage.getItem(SESSION_CACHE_KEY);
        if (!raw) return null;
        const data = JSON.parse(raw);
        if (!data || !data.loggedIn) return null;
        return { ...empty, ...data, offline: true };
    } catch {
        return null;
    }
}

function writeSessionCache(state) {
    if (typeof sessionStorage === "undefined") return;
    try {
        const { offline: _offline, ...rest } = state;
        sessionStorage.setItem(SESSION_CACHE_KEY, JSON.stringify(rest));
    } catch {
        /* private mode / quota */
    }
}

function clearSessionCache() {
    if (typeof sessionStorage === "undefined") return;
    try {
        sessionStorage.removeItem(SESSION_CACHE_KEY);
    } catch {
        /* ignore */
    }
}

const ACCOUNT_THEMES = new Set(["light", "dark", "system"]);

function clearLegacyStorage() {
    if (typeof window === "undefined") return;
    localStorage.removeItem("admin_auth");
    localStorage.removeItem("admin_user");
    localStorage.removeItem("admin_is_admin");
}

function readBrowserImportInput() {
    let theme = "";
    let links = [];
    if (typeof localStorage === "undefined") {
        return { theme, slots: null, links, history: [] };
    }
    try {
        theme = localStorage.getItem("theme") || "";
    } catch {
        /* private mode / quota */
    }
    try {
        const raw = localStorage.getItem("user_quick_links");
        links = raw ? JSON.parse(raw) : [];
        if (!Array.isArray(links)) links = [];
    } catch {
        links = [];
    }
    return {
        theme,
        slots: readSlotCount(),
        links,
        history: readHistory(),
    };
}

function applyAccountTheme(themeValue) {
    if (themeValue && ACCOUNT_THEMES.has(themeValue)) {
        applyThemeLocal(themeValue);
    }
}

function createAuthStore() {
    const { subscribe, set } = writable(empty);
    /** @type {Promise<typeof empty> | null} */
    let refreshPromise = null;

    async function refresh() {
        if (typeof window === "undefined") {
            set(empty);
            return empty;
        }
        if (refreshPromise) {
            return refreshPromise;
        }

        refreshPromise = (async () => {
            clearLegacyStorage();
            try {
                const res = await fetch(`${getApiBase()}/api/auth/me`, {
                    credentials: "include",
                });
                if (!res.ok) {
                    clearSessionCache();
                    set(empty);
                    return empty;
                }
                const me = await res.json();
                const next = { ...meToAuthState(me), offline: false };
                writeSessionCache(next);
                set(next);
                applyAccountTheme(next.theme);

                if (next.prefsImportedAt == null && typeof localStorage !== "undefined") {
                    try {
                        const importRes = await fetch(`${getApiBase()}/api/account/import`, {
                            method: "POST",
                            credentials: "include",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify(
                                buildImportPayload(readBrowserImportInput()),
                            ),
                        });
                        if (importRes.ok) {
                            try {
                                const meRes = await fetch(`${getApiBase()}/api/auth/me`, {
                                    credentials: "include",
                                });
                                if (meRes.ok) {
                                    const updated = { ...meToAuthState(await meRes.json()), offline: false };
                                    writeSessionCache(updated);
                                    set(updated);
                                    applyAccountTheme(updated.theme);
                                    return updated;
                                }
                            } catch {
                                /* keep pre-import session */
                            }
                        }
                    } catch {
                        /* keep session; prefsImportedAt stays null */
                    }
                }

                return next;
            } catch {
                const cached = readSessionCache();
                if (cached) {
                    set(cached);
                    return cached;
                }
                set(empty);
                return empty;
            } finally {
                refreshPromise = null;
            }
        })();

        return refreshPromise;
    }

    return {
        subscribe,
        init() {
            return refresh();
        },
        refresh,
        async logout() {
            try {
                await fetch(`${getApiBase()}/api/auth/logout`, {
                    method: "POST",
                    credentials: "include",
                });
            } catch {
                /* still clear locally */
            }
            clearSessionCache();
            set(empty);
        },
    };
}

export const auth = createAuthStore();
