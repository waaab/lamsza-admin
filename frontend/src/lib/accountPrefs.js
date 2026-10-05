export const userAccountTabIds = [
    "fiok",
    "adatlapjaim",
    "weboldalaim",
    "kereses",
    "bongeszes",
    "kedvencek",
    "linkjeim",
];

export const userSettingsTabIds = [
    "tema",
    "linkbeallitasok",
    "location",
];

/** @param {unknown} raw */
export function normalizePreferredLocation(raw) {
    if (!raw || typeof raw !== "object") return null;
    const row = /** @type {{ id?: unknown, slug?: unknown, name?: unknown, county_slug?: unknown }} */ (raw);
    const slug = String(row.slug ?? "").trim();
    if (!slug) return null;
    const id = Number(row.id);
    return {
        id: Number.isFinite(id) && id > 0 ? id : null,
        slug,
        name: String(row.name ?? "").trim() || slug,
        county_slug: String(row.county_slug ?? "").trim(),
    };
}

/** @param {{ title?: unknown; url?: unknown; bg_color?: unknown }} link */
function validLink(link) {
    if (!link || typeof link !== "object") return null;
    const title = String(link.title ?? "").trim();
    const url = String(link.url ?? "").trim();
    if (!title || !url) return null;
    const out = { title, url };
    if (link.bg_color != null && String(link.bg_color).trim()) {
        out.bg_color = String(link.bg_color).trim();
    }
    return out;
}

/** @param {{ slug?: unknown; name?: unknown; category?: unknown; location?: unknown; photo?: unknown }} row */
function validHistoryRow(row) {
    if (!row || typeof row !== "object") return null;
    const slug = String(row.slug ?? "").trim();
    const name = String(row.name ?? "").trim();
    if (!slug || !name) return null;
    return {
        slug,
        name,
        category: String(row.category ?? "").trim(),
        location: String(row.location ?? "").trim(),
        photo: String(row.photo ?? "").trim(),
    };
}

export function buildImportPayload({ theme, slots, links, history }) {
    const safeLinks = Array.isArray(links)
        ? links.map(validLink).filter(Boolean)
        : [];
    const safeHistory = Array.isArray(history)
        ? history.map(validHistoryRow).filter(Boolean)
        : [];
    return {
        theme: theme == null ? "" : String(theme),
        quicklink_slots: slots == null ? null : slots,
        links: safeLinks,
        history: safeHistory,
    };
}

export function meToAuthState(me) {
    return {
        loggedIn: true,
        user: me.name || me.email || "",
        email: me.email || "",
        isAdmin: !!me.is_admin,
        picture: me.picture ?? "",
        givenName: me.given_name ?? "",
        familyName: me.family_name ?? "",
        displayName: String(me.display_name ?? "").trim(),
        locale: me.locale ?? "",
        googleSub: me.google_sub ?? "",
        lastLoginAt: me.last_login_at ?? null,
        createdAt: me.created_at ?? null,
        theme: me.theme ?? null,
        quicklinkSlots: me.quicklink_slots ?? null,
        prefsImportedAt: me.prefs_imported_at ?? null,
        preferredLocation: normalizePreferredLocation(me.preferred_location),
        adminQueueCount: me.admin_queue_count ?? 0,
    };
}
