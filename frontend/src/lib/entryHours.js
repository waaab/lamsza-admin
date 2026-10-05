export const WEEKDAYS = [
    { key: "mon", label: "Hétfő", short: "H" },
    { key: "tue", label: "Kedd", short: "K" },
    { key: "wed", label: "Szerda", short: "Sze" },
    { key: "thu", label: "Csütörtök", short: "Cs" },
    { key: "fri", label: "Péntek", short: "P" },
    { key: "sat", label: "Szombat", short: "Szo" },
    { key: "sun", label: "Vasárnap", short: "V" },
];

const WEEKDAY_FROM_SHORT = {
    Mon: "mon",
    Tue: "tue",
    Wed: "wed",
    Thu: "thu",
    Fri: "fri",
    Sat: "sat",
    Sun: "sun",
};

export function emptyDayHours() {
    return { open: "", close: "", closed: false };
}

export function emptyWeekHours() {
    return Object.fromEntries(WEEKDAYS.map(({ key }) => [key, emptyDayHours()]));
}

export function defaultDayHours() {
    return { open: "08:00", close: "20:00", closed: false };
}

export function defaultWeekHours() {
    return Object.fromEntries(WEEKDAYS.map(({ key }) => [key, defaultDayHours()]));
}

export function weekHoursAreEmpty(raw) {
    const hours = normalizeHours(raw);
    return WEEKDAYS.every(({ key }) => {
        const slot = hours[key];
        return !slot.closed && slot.open === "" && slot.close === "";
    });
}

export function withDefaultWeekHours(raw) {
    if (weekHoursAreEmpty(raw)) return defaultWeekHours();
    return normalizeHours(raw);
}

export function normalizeHours(raw) {
    const out = emptyWeekHours();
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) return out;
    for (const { key } of WEEKDAYS) {
        const slot = raw[key];
        if (!slot || typeof slot !== "object") continue;
        out[key] = {
            open: String(slot.open ?? "").trim(),
            close: String(slot.close ?? "").trim(),
            closed: Boolean(slot.closed),
        };
    }
    return out;
}

export function hoursConfigured(raw) {
    const hours = normalizeHours(raw);
    return WEEKDAYS.some(({ key }) => {
        const slot = hours[key];
        return slot.closed || (slot.open !== "" && slot.close !== "");
    });
}

export function formatDayHours(slot, placeholder = "-") {
    if (!slot) return placeholder;
    if (slot.closed) return "Zárva";
    if (slot.open && slot.close) return `${slot.open} - ${slot.close}`;
    return placeholder;
}

function minutesFromHHMM(value) {
    const match = /^(\d{1,2}):(\d{2})$/.exec(String(value || "").trim());
    if (!match) return null;
    const hours = Number(match[1]);
    const minutes = Number(match[2]);
    if (hours > 23 || minutes > 59) return null;
    return hours * 60 + minutes;
}

function weekdayKeyFromDate(date, timeZone) {
    const short = new Intl.DateTimeFormat("en-US", {
        weekday: "short",
        timeZone,
    }).format(date);
    return WEEKDAY_FROM_SHORT[short] || "mon";
}

function minutesNow(date, timeZone) {
    const parts = new Intl.DateTimeFormat("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
        hourCycle: "h23",
        timeZone,
    }).formatToParts(date);
    const hour = Number(parts.find((part) => part.type === "hour")?.value ?? 0);
    const minute = Number(parts.find((part) => part.type === "minute")?.value ?? 0);
    return hour * 60 + minute;
}

/** @param {Date} [now] @param {string} [timeZone] */
export function todayWeekdayKey(now = new Date(), timeZone = "Europe/Bucharest") {
    return weekdayKeyFromDate(now, timeZone);
}

/** @param {unknown} hours */
export function openStatus(hours, now = new Date(), timeZone = "Europe/Bucharest") {
    const normalized = normalizeHours(hours);
    if (!hoursConfigured(normalized)) {
        return { state: "unknown", label: "Nyitvatartás", detail: "-" };
    }
    const key = weekdayKeyFromDate(now, timeZone);
    const slot = normalized[key];
    const today = formatDayHours(slot);
    if (slot.closed) {
        return { state: "closed", label: "Zárva", detail: "" };
    }
    const openMinutes = minutesFromHHMM(slot.open);
    const closeMinutes = minutesFromHHMM(slot.close);
    if (openMinutes == null || closeMinutes == null) {
        return { state: "unknown", label: "Nyitvatartás", detail: today };
    }
    const nowMinutes = minutesNow(now, timeZone);
    let isOpen;
    if (closeMinutes > openMinutes) {
        isOpen = nowMinutes >= openMinutes && nowMinutes < closeMinutes;
    } else if (closeMinutes < openMinutes) {
        isOpen = nowMinutes >= openMinutes || nowMinutes < closeMinutes;
    } else {
        isOpen = true;
    }
    return {
        state: isOpen ? "open" : "closed",
        label: isOpen ? "Nyitva" : "Zárva",
        detail: `${slot.open} - ${slot.close}`,
    };
}
