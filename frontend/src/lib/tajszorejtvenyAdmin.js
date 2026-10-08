/**
 * Tájszórejtvény's admin tab (lamsza-jatszoter spec §10.3, §16.6.4): labels
 * and the small pure helpers the views share. Every date comes from Játszótér
 * (its Bucharest today, WAYS_OF_WORKING R19); nothing here reads the browser's
 * clock to decide a day.
 */

export const LEVELS = [1, 2, 3, 4];

/** @type {Record<number, string>} */
export const LEVEL_LABELS = { 1: "Könnyű", 2: "Közepes", 3: "Nehéz", 4: "Mester" };

/** @type {Record<string, string>} */
export const STATUS_LABELS = {
    draft: "Piszkozat",
    approved: "Jóváhagyva",
    needs_attention: "Átnézendő",
    live: "Élő",
    retired: "Lecserélve",
};

/** What a generator or nightly-check flag means for the reviewer. @type {Record<string, string>} */
export const FLAG_LABELS = {
    decorative_over_limit: "Sok üres (díszítő) kocka: 19-26",
    proverb_out_of_range: "A mondás hossza kívül esik a szint tartományán",
    definition_changed: "A Szótárban változott egy meghatározás: frissítve, nézd át",
    definition_changed_live: "A Szótárban változott egy meghatározás egy élő rejtvényben: a meghatározás kézzel javítható",
    headword_changed: "A Szótárban változott egy címszó",
    word_removed: "Egy szót töröltek a Szótárból vagy letiltották",
    proverb_near_mondas: "A mondás 14 napon belül a Szótár napi mondása is",
};

/** @type {Record<string, string>} */
export const REASON_LABELS = {
    typo: "Elírás a meghatározásban",
    mismatch: "A meghatározás nem illik a szóhoz",
    inappropriate: "A szó nem helyénvaló",
    other: "Egyéb",
};

/** @type {Record<string, string>} */
export const REPORT_STATUS_LABELS = { open: "Nyitott", resolved: "Megoldva", dismissed: "Elvetve" };

/** @type {Record<string, string>} */
export const PERIOD_LABELS = { week: "Hét", month: "Hónap", year: "Év" };

const DAY_NAMES = ["vasárnap", "hétfő", "kedd", "szerda", "csütörtök", "péntek", "szombat"];

/** "hétfő" for "2026-10-12": the weekday of a given date, not of today. */
export function dayName(/** @type {string} */ iso) {
    const [y, m, d] = String(iso).split("-").map(Number);
    if (!y || !m || !d) return "";
    return DAY_NAMES[new Date(Date.UTC(y, m - 1, d)).getUTCDay()];
}

/** The seven dates of the week starting on Monday `weekStart`. */
export function weekDates(/** @type {string} */ weekStart) {
    const [y, m, d] = String(weekStart).split("-").map(Number);
    if (!y || !m || !d) return [];
    return Array.from({ length: 7 }, (_, i) => new Date(Date.UTC(y, m - 1, d + i)).toISOString().slice(0, 10));
}

/**
 * The week matrix: one row per day, one cell per level (null when that day
 * and level has no puzzle yet).
 *
 * @param {{ week_start: string, dailies: any[] } | null | undefined} overview
 */
export function weekMatrix(overview) {
    if (!overview) return [];
    return weekDates(overview.week_start).map((date) => ({
        date,
        /** @type {any[]} */
        cells: LEVELS.map((level) => overview.dailies.find((p) => p.date === date && p.level === level) ?? null),
    }));
}

/**
 * Waits until Játszótér's background task `key` (a 202 reply: generating a
 * week, regenerating a puzzle, blocking a word) is no longer running.
 *
 * @param {() => Promise<{ tasks?: { key: string, running: boolean, error?: string }[] }>} fetchTasks
 * @param {string} key
 * @param {{ interval?: number, timeout?: number, sleep?: (ms: number) => Promise<void> }} [opts]
 * @returns {Promise<{ done: boolean, error: string }>} done is false when the wait timed out.
 */
export async function waitForTask(fetchTasks, key, opts = {}) {
    const interval = opts.interval ?? 2000;
    const timeout = opts.timeout ?? 10 * 60 * 1000;
    const sleep = opts.sleep ?? ((ms) => new Promise((r) => setTimeout(r, ms)));
    for (let waited = 0; waited <= timeout; waited += interval) {
        const res = await fetchTasks();
        const task = (res?.tasks ?? []).find((t) => t.key === key);
        if (!task || !task.running) return { done: true, error: task?.error ?? "" };
        await sleep(interval);
    }
    return { done: false, error: "" };
}

/** The Tájszórejtvény dashboard lines from Játszótér's stats (null while it cannot answer). */
export function dashboardLines(/** @type {any} */ s) {
    if (!s) return [];
    /** @type {{ level: string, text: string }[]} */
    const out = [];
    // The info-box variants: Játszótér's "ok" is success, "urgent" is error.
    const box = /** @type {Record<string, string>} */ ({ ok: "success", info: "info", warning: "warning", urgent: "error" });
    if (s.message?.text) out.push({ level: box[s.message.level] ?? "info", text: `Tájszórejtvény: ${s.message.text}` });
    if (s.next_needs_attention > 0)
        out.push({ level: "warning", text: `Tájszórejtvény: ${s.next_needs_attention} jövő heti rejtvény átnézendő.` });
    if (s.this_week_unreviewed > 0)
        out.push({ level: "info", text: `Tájszórejtvény: a héten ${s.this_week_unreviewed} rejtvény átnézés nélkül élesedett.` });
    if (s.open_reports > 0) out.push({ level: "warning", text: `Tájszórejtvény: ${s.open_reports} nyitott hibajelzés.` });
    if (s.late_periods > 0)
        out.push({ level: "info", text: `Tájszórejtvény: lezárt időszak utólag módosult (${s.late_periods}).` });
    if (!s.enabled) out.push({ level: "info", text: "Tájszórejtvény: a játék ki van kapcsolva, a játékosok nem látják." });
    return out;
}
