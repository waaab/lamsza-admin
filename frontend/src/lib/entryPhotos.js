export const DEFAULT_PHOTO_WIDTH = 1600;
export const DEFAULT_PHOTO_HEIGHT = 1200;
export const MAX_ENTRY_PHOTOS = 24;

function clampSize(value, fallback) {
    const n = Number(value);
    if (!Number.isFinite(n) || n < 1 || n > 10000) return fallback;
    return Math.round(n);
}

function mediaUrl(stored, apiBase) {
    const s = stored != null ? String(stored).trim() : "";
    if (!s) return "";
    if (s.startsWith("http://") || s.startsWith("https://")) return s;
    const base =
        apiBase !== undefined && apiBase !== null
            ? String(apiBase).replace(/\/$/, "")
            : "";
    const path = s.startsWith("/") ? s : `/${s}`;
    return base ? `${base}${path}` : path;
}

function apiOrigin(apiBase) {
    return apiBase !== undefined && apiBase !== null
        ? String(apiBase).replace(/\/$/, "")
        : "";
}

/** Dedupe and drop empty photo URLs, keeping first-seen order. */
export function uniquePhotoUrls(urls) {
    const seen = new Set();
    /** @type {string[]} */
    const out = [];
    const list = Array.isArray(urls) ? urls : [];
    for (const raw of list) {
        const u = String(raw ?? "").trim();
        if (!u || seen.has(u)) continue;
        seen.add(u);
        out.push(u);
        if (out.length >= MAX_ENTRY_PHOTOS) break;
    }
    return out;
}

/**
 * Same-origin /api/media paths stay direct; external http(s) goes through /api/proxy.
 * @param {string} stored
 * @param {string} [apiBase]
 */
export function proxiedMediaUrl(stored, apiBase) {
    const s = stored != null ? String(stored).trim() : "";
    if (!s) return "";
    if (s.startsWith("/api/proxy?") || s.includes("/api/proxy?")) {
        return mediaUrl(s, apiBase);
    }
    if (s.startsWith("/api/media/") || s.includes("/api/media/")) {
        return mediaUrl(s, apiBase);
    }
    if (s.startsWith("http://") || s.startsWith("https://")) {
        const origin = apiOrigin(apiBase);
        const path = `/api/proxy?url=${encodeURIComponent(s)}`;
        return origin ? `${origin}${path}` : path;
    }
    return mediaUrl(s, apiBase);
}

/**
 * Prefer a larger original when the stored URL is a known thumbnail variant.
 * @param {string} url
 */
export function upgradePhotoUrl(url) {
    const s = String(url ?? "").trim();
    if (!s) return s;

    const wm = s.match(
        /^(https?:\/\/upload\.wikimedia\.org\/wikipedia\/[^/]+\/)thumb\/(.+)\/\d+px-[^/?#]+$/i,
    );
    if (wm) return `${wm[1]}${wm[2]}`;

    if (/tripadvisor\.com\/media\//i.test(s) || /dynamic-media-cdn\.tripadvisor\.com/i.test(s)) {
        try {
            const u = new URL(s);
            if (!u.searchParams.has("w")) {
                u.searchParams.set("w", "1600");
                u.searchParams.set("h", "-1");
                u.searchParams.set("s", "1");
            }
            return u.toString();
        } catch {
            return s;
        }
    }

    return s;
}

/** @typedef {{ url: string, alt: string, title: string, description: string, copyright: string, uploader: string, width: number, height: number }} EntryPhoto */

/** @returns {EntryPhoto[]} */
export function emptyPhotos() {
    return [];
}

/** @param {unknown} raw
 *  @returns {EntryPhoto[]} */
export function normalizePhotos(raw) {
    const arr = Array.isArray(raw) ? raw : [];
    /** @type {EntryPhoto[]} */
    const out = [];
    for (const item of arr) {
        if (!item || typeof item !== "object") continue;
        const url = String(item.url ?? item.src ?? "").trim();
        if (!url) continue;
        out.push({
            url,
            alt: String(item.alt ?? "").trim(),
            title: String(item.title ?? "").trim(),
            description: String(item.description ?? "").trim(),
            copyright: String(item.copyright ?? "").trim().replace(/^©\s*/, ""),
            uploader: String(item.uploader ?? item.uploaded_by ?? "").trim(),
            width: clampSize(item.width, DEFAULT_PHOTO_WIDTH),
            height: clampSize(item.height, DEFAULT_PHOTO_HEIGHT),
        });
        if (out.length >= MAX_ENTRY_PHOTOS) break;
    }
    return out;
}

/**
 * Public gallery slides from stored photos or legacy image fields.
 * @param {Record<string, unknown> | null | undefined} entry
 * @param {{ apiBase?: string }} [opts]
 */
export function gallerySlides(entry, opts = {}) {
    const apiBase = opts.apiBase;
    const photos = normalizePhotos(entry?.photos);
    const name = String(entry?.name ?? "").trim();

    if (photos.length) {
        return photos.map((photo, i) => toSlide(photo, i, apiBase));
    }

    const legacy = String(entry?.image ?? entry?.photo ?? entry?.logo ?? "").trim();
    if (legacy) {
        return [
            toSlide(
                {
                    url: legacy,
                    alt: name || "Fotó",
                    title: name,
                    description: "",
                    copyright: "",
                    uploader: "",
                    width: DEFAULT_PHOTO_WIDTH,
                    height: DEFAULT_PHOTO_HEIGHT,
                },
                0,
                apiBase,
            ),
        ];
    }

    return [];
}

export const PHOTO_COPYRIGHT_MISSING = "A szerzői jogi információt a feltöltő nem adta meg";
export const PHOTO_UPLOADED_BY_ADMIN = "Admin töltötte fel";

/**
 * Copyright text (without the © mark) and uploader line for a gallery caption.
 * A missing copyright still gets a line; a missing uploader is the admin.
 * @param {Record<string, unknown> | null | undefined} slide
 */
export function photoCredit(slide) {
    const copyright = String(slide?.copyright ?? "").trim().replace(/^©\s*/, "");
    const uploader = String(slide?.uploader ?? slide?.uploaded_by ?? "").trim();
    return {
        copyright: copyright || PHOTO_COPYRIGHT_MISSING,
        uploaderLine: uploader ? `Feltöltötte: ${uploader}` : PHOTO_UPLOADED_BY_ADMIN,
    };
}

export function firstPhotoSrc(entry, apiBase) {
    const slides = gallerySlides(entry, { apiBase });
    return slides[0]?.src ?? "";
}

function toSlide(photo, index, apiBase) {
    const src = mediaUrl(photo.url, apiBase);
    return {
        ...photo,
        src,
        fullSrc: src,
        fallback: "",
        loading: index === 0 ? "eager" : "lazy",
        fetchpriority: index === 0 ? "high" : "low",
    };
}

/**
 * Public gallery slides for a látnivaló (featured first, then unique gallery URLs).
 * External images are loaded through /api/proxy; lightbox uses a higher-res variant when known.
 * @param {Record<string, unknown> | null | undefined} attraction
 * @param {{ apiBase?: string }} [opts]
 */
export function attractionGallerySlides(attraction, opts = {}) {
    const apiBase = opts.apiBase;
    const name = String(attraction?.name ?? "").trim();
    const description = String(attraction?.description ?? "").trim();
    const refs = attractionPhotoRefs(attraction);

    return refs.map((photo, i) => {
        const src = proxiedMediaUrl(photo.url, apiBase);
        const fullSrc = proxiedMediaUrl(upgradePhotoUrl(photo.url), apiBase);
        return {
            url: photo.url,
            src,
            fullSrc,
            fallback: src,
            alt: name ? `${name} - fotó ${i + 1}` : `Fotó ${i + 1}`,
            title: name,
            description,
            copyright: photo.copyright,
            uploader: "",
            width: DEFAULT_PHOTO_WIDTH,
            height: DEFAULT_PHOTO_HEIGHT,
            loading: i === 0 ? "eager" : "lazy",
            fetchpriority: i === 0 ? "high" : "low",
        };
    });
}

/** @param {Record<string, unknown> | null | undefined} attraction */
function attractionPhotoRefs(attraction) {
    /** @type {{ url: string, copyright: string }[]} */
    const refs = [];
    const featured = String(attraction?.featured_image ?? "").trim();
    if (featured) {
        refs.push({
            url: featured,
            copyright: String(attraction?.featured_image_copyright ?? "").trim().replace(/^©\s*/, ""),
        });
    }
    const images = Array.isArray(attraction?.images) ? attraction.images : [];
    for (const raw of images) {
        if (raw && typeof raw === "object") {
            refs.push({
                url: String(raw.url ?? "").trim(),
                copyright: String(raw.copyright ?? "").trim().replace(/^©\s*/, ""),
            });
        } else {
            refs.push({ url: String(raw ?? "").trim(), copyright: "" });
        }
    }
    const seen = new Set();
    /** @type {{ url: string, copyright: string }[]} */
    const out = [];
    for (const ref of refs) {
        if (!ref.url || seen.has(ref.url)) continue;
        seen.add(ref.url);
        out.push(ref);
        if (out.length >= MAX_ENTRY_PHOTOS) break;
    }
    return out;
}
