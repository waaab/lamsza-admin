# Changelog

All notable changes to lamsza-admin are recorded here, in English, for
developers. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This app has no public changelog page (WAYS_OF_WORKING R17): every notable change
adds a line under `[Unreleased]` in the same commit.

---

## [Unreleased]

### Added
- Attraction facts for Lámsza's attraction page: Magasság (m), Felszín (km²), Mélység (m) in the create and edit forms (a decimal comma is fine; empty = not shown). `/api/admin/attractions` reads and writes `elevation_m`, `area_km2`, `depth_m` and refuses a negative area or depth (400). Tests: `TestAdminAttractionFacts`, `optionalNumber`.
- Bejegyzés kategóriák: "Kiemelt kategóriák a kezdőlapon" chooses the Lámsza home page's category chips (at most six, subcategories too, in order; one save replaces the list). New route `GET/PUT /api/admin/entry_categories/featured` (`{"ids": [...]}`), audited as `featured_categories`; it writes lamsza's `entry_categories.featured_order`.
- Settlement coordinates can be edited: `PUT /api/admin/locations` saves "latitude, longitude" to the settlement's `geo_locations` row (or creates and links one, or unlinks it when cleared); it used to drop the field, so coordinates could only be set on create. Garbled or out-of-range pairs are refused with 400. lamsza's weather is fetched for these coordinates. Test `TestAdminSettlementCoordinates`.
- Synced from lamsza: `AccountMenu` without the photo (unused here), `global.css` (menu item highlight inside the panel, Lámsza's home-page hero styles) and `typography.css` (`--text-hero` up to 7rem; the admin app does not use it).
- Synced from lamsza: the shared theme and auth stores (signed out, or a session the server no longer knows, clears the saved theme: the device's theme applies), the account menu and Fiók components (`AccountMenu`, `AccountPage`, `AccountDetails`, `ThemeSettings`, `accountDetails.js` and its test; unused in the admin app), and `global.css` (their styles, the launcher's hover colour, the dark `--thin-grey` from lamsza 36c3f64).
- Synced from lamsza: the shared `apps` and `plus` icons, the apps launcher (`AppsLauncher.svelte`, `networkApps.js`, `networkOrigins.js`, its `.apps-launcher` CSS and test). The admin app does not use the launcher (UI_BASELINE "tb-apps-launcher").
- `/games` Tájszórejtvény tab (lamsza-jatszoter spec §10.3, §16.6.4) on Játszótér's `/internal/admin/tajszorejtveny/`: this week and next week as a day × level matrix with status, flags, Székely words, empty cells and repeats; "Az egész hét jóváhagyása" and "Generálás most"; a puzzle with its answers (grid, proverb and source, every word and clue) to approve, regenerate, replace with a same-level spare, edit a clue (optionally kept as the word's game clue), change the proverb or block a word, the slow actions waited for through Játszótér's task list; the players' reports (resolve, dismiss, reopen with a note, the word linked to `/dictionary#szavak/<id>`); the word metadata (familiarity, game clue, example and its review, block; filters for unset familiarity and unreviewed examples); the filler words (add, edit, switch off); closed periods that got results after their winners; the game's on/off switch. The Vezérlőpult card shows next week's approved/made count and "Üzenetek" its lines (next week's approval with Játszótér's urgency, puzzles to look at, open reports, late periods, the game switched off). `tests/adminSections.test.js` now walks component subfolders.
- Synced from lamsza: the shared `rosette` icon (the Székely six-petal rosette).
- Synced from lamsza: nine shared icons (`tajszorejtveny`, `zoom_in`, `zoom_out`, `flag`, `text_size`, `keyboard`, `sound_on`, `sound_off`, `help`) and the content-dialog shell taking the screen's width on phones (up to 768px).
- The `/dictionary` (Szótár) and `/games` (Játszótér) sections (lamsza WAYS_OF_WORKING R18), with everything the apps' own `/admin` pages do: Napi szó (pin until a date, or automatic), Szavak (search, letter filter, paging, create, live edit of every sense field, delete), Szófajok (16 intro texts), Mondások (one per day), Javaslatok (accept, reject), Rovásfejtő (create with rune preview, edit, soft delete) and the Szókereső editor, moved here from Játszótér. Each section has its own sidebar (the app in a new tab, Vezérlőpult, its tabs) and dashboard cards with the same entries and counts; tabs have a `#hash` and `#szavak/<id>` opens a word's editor. Every call goes through the relay; errors show the app's own message.
- `AdminShell` holds the sign-in gate, sidebar, header and back-to-top for all three sections; the main admin looks the same (before/after screenshots identical). New `tests/adminSections.test.js`.
- The relay for the coming `/dictionary` and `/games` sections (lamsza WAYS_OF_WORKING R18): `/api/admin/dictionary/…` and `/api/admin/games/…` forward to Szótár's and Játszótér's internal admin APIs on 127.0.0.1 with the service token and the acting admin's email, behind the admin check and the audit log (a `Subtree` audit resource records the real path). Settings `SZOTAR_ADMIN_URL`/`_TOKEN`, `JATSZOTER_ADMIN_URL`/`_TOKEN`. Tests with a fake app backend.
- The network's error page through the shared minimal `ErrorShell`: the Lámsza and Admin buttons, the lantern, "Hoppácska!", the code and a Hungarian explanation; noindex, nothing fetched.
- A back-to-top button, as on the other apps; it follows the scroll of the admin's main column.
- Basic small-screen rules for the admin shell: a narrower icon rail, the header buttons above the title, tighter spacing, full-width edit windows.

### Changed
- "Új jelentés", "Új feladvány", "Új nap" and the featured categories' "Hozzáadás" show the shared plus icon before their text (lamsza UI_BASELINE `btn-create-plus`); the create panels already had it.
- The `site_settings` contract is tested: `TestSiteSettingsKeysAreReadByLamsza` checks every key the Beállítások tab writes is read by lamsza's backend (needs the lamsza repo; CI checks it out).
- CI runs the backend's database tests: the backend job gets a `postgres:16` service, checks out the public lamsza repo for its schema and runs `scripts/test-backend.sh`, the same path as `npm run test:backend`. With `ADMIN_TEST_DB_REQUIRED=1` (set in CI) a missing database fails a DB test instead of skipping it.
- The API server has read, write and idle timeouts (read 60s for 8 MB uploads, write 90s) instead of a bare `http.ListenAndServe`.
- `@types/node` (24.13.6, a devDependency) for the test files' `node:` imports: svelte-check 787 to 731 errors.
- Synced from lamsza: `global.css` (the hero's joined accent; unused in admin).
- Synced from lamsza: `global.css` (hero accent, search slot and descender rule; unused in admin).
- Beállítások: each section's Mentés saves only that section's keys (`lib/settingsSections.js`, tested). It used to send every setting on the page, so saving the social links also rewrote the weather settings and the default settlement, and put back a stale `weather_cache_version` and `quick_links_version`, undoing a cache clear made since the page loaded. The icon style reads "Animált ikonok" / "Emoji", says where it applies and links to Lámsza's icon page; the unused "Aktív felhasználók becslése" field is gone; the TTL is labelled as the browser cache it is.
- Weather settings follow lamsza's move to MET Norway: switches for MET Norway, WeatherAPI.com and OpenWeatherMap in a fixed order (`weather_provider_metno_enabled`); the Open-Meteo switch and the default-provider dropdown are gone, because lamsza no longer reads them. Weather translations are keyed by MET symbol code (e.g. `partlycloudy`). Unused copies of the old provider code removed from `internal/weather`.
- Receives only the shared modules it imports: the account menu and Fiók components, the launcher, `networkApps.js`, `networkOrigins.js` and their tests are removed. Synced from lamsza: a `global.css` without Lámsza's own page styles (now in Lámsza's `lamsza.css`) and `typography.css` without `--text-hero`. The admin app looks the same.
- The 18 "add new" panels draw their plus with the shared `AppIcon` "plus" instead of the admin's own `AdminPlusIcon`, which is removed (UI_BASELINE "ic-system"). Same drawing: before and after screenshots of every panel are pixel-identical in light, dark and phone width. The unused `.admin-plus-icon` rule in `admin.css` is gone too (its `:global()` never applied in a plain stylesheet).
- Removed the main admin's Mondások (OPEN_ITEMS, Mondások: Szótár's proverbs are the only store, managed in `/dictionary`): the tab, its sidebar entry and dashboard card, the "no mondás today" message, `/api/admin/mondasok`, `internal/mondasok`, `FEATURE_MONDASOK`, the audit resource, the dashboard count, `models.Mondas`, the page's local-date helper and the tests. The admin no longer reads lamsza's `mondasok` table.
- `/dictionary` Mondások has everything the main admin's Mondások page has: search by text, meaning, ID or date, 10 rows a page, "Mai nap" in the new and edit forms, the "ma" badge and highlighted row, the explanatory text, and the warning when today has no mondás, in the tab and on Vezérlőpult Szótár (with Megnyitás). "Today" is Szótár's Bucharest day (R19). Szótár's proverbs are the network's only mondás store now (owner's decision).
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
