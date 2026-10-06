import assert from "node:assert/strict";
import { test } from "node:test";
import {
    DEFAULT_PHOTO_HEIGHT,
    DEFAULT_PHOTO_WIDTH,
    MAX_ENTRY_PHOTOS,
    attractionGallerySlides,
    emptyPhotos,
    firstPhotoSrc,
    gallerySlides,
    normalizePhotos,
    photoCredit,
    proxiedMediaUrl,
    uniquePhotoUrls,
    upgradePhotoUrl,
} from "../src/lib/entryPhotos.js";

test("normalizePhotos: empty and invalid become []", () => {
    assert.deepEqual(normalizePhotos(null), emptyPhotos());
    assert.deepEqual(normalizePhotos({}), []);
    assert.deepEqual(normalizePhotos([{ alt: "no url" }]), []);
});

test("normalizePhotos: keeps url, alt, title, description, width, height", () => {
    const photos = normalizePhotos([
        {
            url: "/api/media/entry-images/a.jpg",
            alt: "Udvar",
            title: "Udvar",
            description: "Nyári fény",
            width: 800,
            height: 600,
        },
    ]);
    assert.equal(photos.length, 1);
    assert.equal(photos[0].url, "/api/media/entry-images/a.jpg");
    assert.equal(photos[0].alt, "Udvar");
    assert.equal(photos[0].title, "Udvar");
    assert.equal(photos[0].description, "Nyári fény");
    assert.equal(photos[0].copyright, "");
    assert.equal(photos[0].uploader, "");
    assert.equal(photos[0].width, 800);
    assert.equal(photos[0].height, 600);
});

test("normalizePhotos: bad sizes fall back to defaults and cap length", () => {
    const tooMany = Array.from({ length: MAX_ENTRY_PHOTOS + 5 }, (_, i) => ({
        url: `https://cdn.example.com/${i}.jpg`,
        width: 0,
        height: -1,
    }));
    const photos = normalizePhotos(tooMany);
    assert.equal(photos.length, MAX_ENTRY_PHOTOS);
    assert.equal(photos[0].width, DEFAULT_PHOTO_WIDTH);
    assert.equal(photos[0].height, DEFAULT_PHOTO_HEIGHT);
});

test("gallerySlides: stored photos win over demo and set eager/lazy", () => {
    const slides = gallerySlides(
        {
            name: "Teszt",
            photos: [
                { url: "/api/media/entry-images/a.jpg", alt: "A", title: "A", description: "Első", width: 4, height: 3 },
                { url: "/api/media/entry-images/b.jpg", alt: "B", width: 4, height: 3 },
            ],
        },
        { apiBase: "http://127.0.0.1:3000" },
    );
    assert.equal(slides.length, 2);
    assert.equal(slides[0].src, "http://127.0.0.1:3000/api/media/entry-images/a.jpg");
    assert.equal(slides[0].loading, "eager");
    assert.equal(slides[0].fetchpriority, "high");
    assert.equal(slides[1].loading, "lazy");
    assert.equal(slides[0].alt, "A");
    assert.equal(slides[0].description, "Első");
});

test("photoCredit: always has a copyright line and an uploader line", () => {
    assert.deepEqual(photoCredit(null), {
        copyright: "A szerzői jogi információt a feltöltő nem adta meg",
        uploaderLine: "Admin töltötte fel",
    });
    assert.deepEqual(photoCredit({ copyright: "© Iliuta Goean", uploader: "Iliuta Goean" }), {
        copyright: "Iliuta Goean",
        uploaderLine: "Feltöltötte: Iliuta Goean",
    });
    assert.equal(photoCredit({ uploaded_by: "Kovács Anna" }).uploaderLine, "Feltöltötte: Kovács Anna");
});

test("firstPhotoSrc: uses first stored photo", () => {
    const src = firstPhotoSrc(
        { photos: [{ url: "/api/media/entry-images/a.jpg", width: 1, height: 1 }] },
        "http://example.test",
    );
    assert.equal(src, "http://example.test/api/media/entry-images/a.jpg");
});

test("uniquePhotoUrls: drops empty and duplicates, keeps order", () => {
    assert.deepEqual(
        uniquePhotoUrls(["a.jpg", "", "b.jpg", "a.jpg", "  ", "c.jpg"]),
        ["a.jpg", "b.jpg", "c.jpg"],
    );
});

test("proxiedMediaUrl: proxies external https, leaves media paths", () => {
    const ext = "https://cdn.example.com/lake.jpg";
    assert.equal(
        proxiedMediaUrl(ext, "http://127.0.0.1:3000"),
        `http://127.0.0.1:3000/api/proxy?url=${encodeURIComponent(ext)}`,
    );
    assert.equal(
        proxiedMediaUrl("/api/media/entry-images/a.jpg", "http://127.0.0.1:3000"),
        "http://127.0.0.1:3000/api/media/entry-images/a.jpg",
    );
});

test("upgradePhotoUrl: unwraps Wikimedia thumbs and enlarges TripAdvisor", () => {
    assert.equal(
        upgradePhotoUrl(
            "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ab/Lake.jpg/800px-Lake.jpg",
        ),
        "https://upload.wikimedia.org/wikipedia/commons/a/ab/Lake.jpg",
    );
    const ta = upgradePhotoUrl(
        "https://dynamic-media-cdn.tripadvisor.com/media/photo-o/11/67/2c/00/lacul-sfanta-ana.jpg",
    );
    assert.match(ta, /w=1600/);
    assert.match(ta, /h=-1/);
});

test("attractionGallerySlides: featured first, unique images, proxy + fullSrc", () => {
    const featured =
        "https://dynamic-media-cdn.tripadvisor.com/media/photo-o/11/67/2c/00/lacul-sfanta-ana.jpg";
    const extra = "https://dynamic-media-cdn.tripadvisor.com/media/photo-o/01/b0/be/46/st-ann-lake.jpg";
    const slides = attractionGallerySlides(
        {
            name: "Szent Anna-tó",
            description: "Vulkanikus tó.",
            featured_image: featured,
            images: [{ url: featured, copyright: "ignored duplicate" }, { url: extra, copyright: "CC BY-SA" }],
        },
        { apiBase: "http://127.0.0.1:3000" },
    );
    assert.equal(slides.length, 2);
    assert.equal(
        slides[0].src,
        `http://127.0.0.1:3000/api/proxy?url=${encodeURIComponent(featured)}`,
    );
    assert.equal(
        slides[1].src,
        `http://127.0.0.1:3000/api/proxy?url=${encodeURIComponent(extra)}`,
    );
    assert.equal(slides[1].copyright, "CC BY-SA");
    assert.notEqual(slides[0].fullSrc, slides[0].src);
    assert.match(slides[0].fullSrc, /w%3D1600|w=1600/);
    assert.equal(slides[0].title, "Szent Anna-tó");
    assert.equal(slides[0].description, "Vulkanikus tó.");
    assert.equal(slides[0].alt, "Szent Anna-tó - fotó 1");
});
