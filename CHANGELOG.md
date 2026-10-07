# Changelog

All notable changes to lamsza-admin are recorded here, in English, for
developers. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This app has no public changelog page (WAYS_OF_WORKING R17): every notable change
adds a line under `[Unreleased]` in the same commit.

---

## [Unreleased]

### Added
- The `/dictionary` (Szótár) and `/games` (Játszótér) sections (lamsza WAYS_OF_WORKING R18), with everything the apps' own `/admin` pages do: Napi szó (pin until a date, or automatic), Szavak (search, letter filter, paging, create, live edit of every sense field, delete), Szófajok (16 intro texts), Mondások (one per day), Javaslatok (accept, reject), Rovásfejtő (create with rune preview, edit, soft delete) and the Szókereső editor, moved here from Játszótér. Each section has its own sidebar (the app in a new tab, Vezérlőpult, its tabs) and dashboard cards with the same entries and counts; tabs have a `#hash` and `#szavak/<id>` opens a word's editor. Every call goes through the relay; errors show the app's own message.
- `AdminShell` holds the sign-in gate, sidebar, header and back-to-top for all three sections; the main admin looks the same (before/after screenshots identical). New `tests/adminSections.test.js`.
- The relay for the coming `/dictionary` and `/games` sections (lamsza WAYS_OF_WORKING R18): `/api/admin/dictionary/…` and `/api/admin/games/…` forward to Szótár's and Játszótér's internal admin APIs on 127.0.0.1 with the service token and the acting admin's email, behind the admin check and the audit log (a `Subtree` audit resource records the real path). Settings `SZOTAR_ADMIN_URL`/`_TOKEN`, `JATSZOTER_ADMIN_URL`/`_TOKEN`. Tests with a fake app backend.
- The network's error page through the shared minimal `ErrorShell`: the Lámsza and Admin buttons, the lantern, "Hoppácska!", the code and a Hungarian explanation; noindex, nothing fetched.
- A back-to-top button, as on the other apps; it follows the scroll of the admin's main column.
- Basic small-screen rules for the admin shell: a narrower icon rail, the header buttons above the title, tighter spacing, full-width edit windows.

### Changed
- `/dictionary`: Javaslatok is "Szójavaslatok", right under Szavak, with the new shared `word-suggestions` icon and a description that says these are users' suggested changes to words (new words, new meanings, forms). The shared `words` and `word-suggestions` icons are synced from lamsza.
- The date defaults in `/dictionary` (a new mondás) and `/games` (a new Szókereső puzzle) are Szótár's and Játszótér's Bucharest `today` (lamsza WAYS_OF_WORKING R19), not the browser's date; `localISODate` is gone from `sectionApi.js`.
- The Mondások icon is the new shared one (Google's filled `format_quote`, centered, synced from lamsza); the `svg.app-icon-quote` rules for the old `<text>` glyph are gone from `admin.css`.
- The header's section and sign-out buttons use the shared toolbar buttons (global.css `.btn.nav-btn` in a `.nav` row, as on Lámsza): pill shape, 16px icons, 0.75rem apart, the current section in the shared active state; before they were admin's own 36px squares with 20px icons. The section dashboards are titled "Vezérlőpult Szótár" and "Vezérlőpult Játszótér" (owner's decision).
- The header switcher links to `/`, `/dictionary` and `/games` in this app instead of the apps' own `/admin` pages; `VITE_ADMIN_ORIGIN`, `VITE_SZOTAR_ADMIN_ORIGIN` and `VITE_JATSZOTER_ADMIN_ORIGIN` are gone, and `VITE_SZOTAR_ORIGIN` / `VITE_JATSZOTER_ORIGIN` (with `VITE_LAMSZA_ORIGIN`) give the apps' public origins. The sidebar's "open in a new tab" icon is the shared `external` icon.
- `tests/noEmdash.test.js` (synced from lamsza) keeps em dashes out of code and UI text in `npm test`; the `.github/ci-status-issue.*` comments are corrected network-wide; the Cursor rules are synced.
- Sign-in follows the network gate (lamsza UI_BASELINE "adm-gate"): a card with "Az admin felülethez lépj be." and the shared `SignInDialog`, which opens by itself. A Google account that is not on the allowlist (the API answers 403) is sent on to Lámsza, through admin's own `signIn` for the shared button; before, it only saw "Belépés sikertelen.". The full-page login box is gone.
- The confirm and notice dialogs come last in the page, so a notice raised over an edit window shows on top of it.
- Docs: the README uses the network script as the only launcher and says the DB-backed tests skip in CI; `LOCAL_DEV_CHECKS.md` lost its stale results table and lists all 15 frontend test files; `ARCHITECTURE.md` points to WoW §1 for ports. No em dash in code comments (D5). Synced from lamsza: `global.css` (dark tokens) and `component-typography.css`.
- Confirmations use the shared `ConfirmDialog` (Mégse, then Igen) and messages the shared `NoticeDialog` (closed with "Bezárás"), instead of the inline dialog with "OK"; the one native `confirm()` (deleting an attraction) goes through the same dialog.
- The edit windows sit on the network's content-dialog shell (`.link-dialog`); a wide table inside scrolls sideways.
- English labels are Hungarian: "Vezérlőpult", "Jóváhagyás", "Elutasítás", "Felhasználó tiltása", "Lámsza megnyitása új lapon", and "Lámsza admin" in the app switcher.
- Synced from lamsza: `ConfirmDialog`, `NoticeDialog`, `SignInDialog`, `ErrorPage`, `ErrorLantern`, the updated `AppIcon` (the Játszótér cross), `GoogleSignIn` ("Belépés sikertelen."), `global.css` and the drift test; the header's logout icon comes from `AppIcon`. The Cursor UI rule is synced too.

### Fixed
- Text fields in the admin forms no longer run past their box (`box-sizing: border-box`).

---

## [0.1.0] - 2026-10-07

The first release of the admin as its own app, extracted from lamsza on
2026-10-06.

### Added
- The admin UI and API, split out of lamsza: directory, listing queue, websites,
  catalog, events and venues, pages, settings, users. Its own session table
  (`admin_sessions`) and cookie.
- An append-only audit log of every admin write (`admin_audit_log`,
  `/api/admin/audit-log`); every `/api/admin/` route is registered through one
  helper that applies it.
- A news feed check (`/api/admin/news_feeds/check`, the "Ellenőrzés" button),
  replacing a refresh button that called a route this app never had.
- The BOG-55 handler suite: route guards, a happy-path test per admin route,
  CRUD, validation, listing queue, website moderation, uploads, audit log.
- The network Cursor rules (`.cursor/rules/`, synced from lamsza).

### Changed
- The route table is built by `newMux()`.
- Request body limits: 2 MiB for JSON, 10 MiB for the two image uploads.
- Go test suites run only against a `_test` scratch database built from lamsza's
  schema (`npm run test:backend`); an exported variable beats `.env`.
- CI runs on `ubuntu-24.04` with Node from `.nvmrc` (24) and Go from `go.mod`
  (1.25.7).
- `/api/auth/me` no longer reports `prefs_imported_at`, so the shared auth store
  does not try a browser import this app cannot receive.
- `app.html` has its own title and is `noindex`.

### Fixed
- Attractions without coordinates were missing from the admin list.
- Empty event catalogs answered `null` instead of `[]`.

### Security
- The media folders no longer list their files.
- CSP, admin-only OAuth origin, `noindex`.

### Removed
- Dead code left by the extraction (news fetch cache, unused account types,
  unused seeds and constants), unused images, `scripts/restart_all.sh`.
