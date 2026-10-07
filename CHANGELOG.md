# Changelog

All notable changes to lamsza-admin are recorded here, in English, for
developers. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This app has no public changelog page (WAYS_OF_WORKING R17): every notable change
adds a line under `[Unreleased]` in the same commit.

---

## [Unreleased]

### Added
- The network's error page (`+error.svelte` renders the shared `ErrorPage`): the lantern, "Hoppácska!", the code and a Hungarian explanation, noindex.
- A back-to-top button, as on the other apps; it follows the scroll of the admin's main column.
- Basic small-screen rules for the admin shell: a narrower icon rail, the header buttons above the title, tighter spacing, full-width edit windows.

### Changed
- Sign-in follows the network gate (lamsza UI_BASELINE "adm-gate"): a card with "Az admin felülethez lépj be." and the shared `SignInDialog`, which opens by itself; a signed-in non-admin is sent on to Lámsza. The full-page login box is gone.
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
