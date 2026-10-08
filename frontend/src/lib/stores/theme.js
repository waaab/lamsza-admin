import { get, writable } from 'svelte/store';
import { getApiBase } from '$lib/api.js';
import { auth } from './auth.js';

export const theme = writable('system');

const THEMES = ["light", "dark", "system"];
export const LABELS = {
    light: "☀️ Világos mód",
    dark: "🌙 Sötét mód",
    system: "🖥️ Rendszer",
};

function persistThemeToAccount(newTheme) {
    const state = get(auth);
    if (!state.loggedIn || typeof window === "undefined") return;
    fetch(`${getApiBase()}/api/account/preferences`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ theme: newTheme }),
    }).catch(() => {});
}

/** Apply theme in the browser without persisting to the account. */
export function applyThemeLocal(newTheme) {
    if (typeof document !== "undefined") {
        if (newTheme === "system") {
            document.documentElement.removeAttribute("data-theme");
        } else {
            document.documentElement.setAttribute("data-theme", newTheme);
        }
    }
    theme.set(newTheme);
    if (typeof localStorage !== "undefined") {
        localStorage.setItem("theme", newTheme);
    }
}

/**
 * Back to the device's theme (prefers-color-scheme): a signed-out visitor
 * cannot choose one (UI_BASELINE "set-theme-store"). Called on sign-out and
 * when /api/auth/me says nobody is signed in, so theme-init.js finds nothing
 * saved on the next page load.
 */
export function clearAccountTheme() {
    if (typeof document !== "undefined") {
        document.documentElement.removeAttribute("data-theme");
    }
    theme.set("system");
    if (typeof localStorage !== "undefined") {
        try {
            localStorage.removeItem("theme");
        } catch {
            /* storage blocked: data-theme is already gone */
        }
    }
}

export function applyTheme(newTheme) {
    applyThemeLocal(newTheme);
    persistThemeToAccount(newTheme);
}

export function cycleTheme(currentTheme) {
    const idx = THEMES.indexOf(currentTheme);
    const next = THEMES[(idx + 1) % THEMES.length];
    applyTheme(next);
}
