# Local Dev Checks — lamsza-admin

**Audience:** anyone developing this app on this machine, agents included.
**Target:** `localhost` only. See `lamsza/docs/AGENT_ENVIRONMENT_POLICY.md` — production is
the owner's.
**Scope:** this repo. The network-wide checks (all four apps up, every API smoked, dump and
restore) live in `lamsza/docs/LOCAL_DEV_CHECKS.md`. That file stays the definition of
"verified" for the network; this one is the repo-level gate.
Commands and what to expect, not recorded results or counts (WAYS_OF_WORKING R10).

---

## The one command

```bash
cd ~/projects/lamsza-network/lamsza-admin && npm test
```

It runs both suites and stops at the first failure:

| Step | Script | What it runs |
|---|---|---|
| 1 | `npm run test:frontend` | `node --test "tests/**/*.test.js"` in `frontend/` |
| 2 | `npm run test:backend` | `scripts/test-backend.sh`: drops and recreates the scratch database `lamsza_admin_test` from `lamsza/backend/schema/`, then `go test -count=1 ./...` in `backend/` against it |

Run one side on its own with `npm run test:frontend` or `npm run test:backend`.

The backend suites never touch the shared dev `lamsza` database: they read only
`TEST_DATABASE_URL`, and a local test database name must end in `_test`
(`backend/internal/db/testguard.go`). Step 2 needs the lamsza repo next to this one and the
`lamsza-db` container up.

No `npm install` is needed for the tests. The frontend suite uses only `node:test` and
`node:assert`, and imports the modules under `frontend/src/lib/` directly.

> Keep the glob quoted — `"tests/**/*.test.js"`, the same string szotar and jatszoter use.
> `node --test tests/` (the directory form) crashes with `MODULE_NOT_FOUND` on Node 24.

---

## What the backend handler suite covers

`backend/*_test.go` (BOG-55) serves requests through the real route table,
`newMux()` in `main.go`, so the wrappers, their order and the feature flags are the
ones the process runs:

- **Route guards, no database:** every `/api/admin/` path and every `admin(...)`
  registration is behind `RequireAdmin`; anonymous calls get 401; a non-admin
  session gets 403; CORS refuses foreign origins.
- **Every admin route has a named happy-path test** (`adminHappyPath` in
  `admin_read_test.go`). Add a route and the suite goes red until it has one.
- **CRUD, validation, listing queue, website moderation, singletons, uploads and the
  audit log**, against the scratch database. Each test creates its own tagged rows
  and removes them; the catalog it needs comes from lamsza's `002_reference.sql`.

Without `TEST_DATABASE_URL` (CI) the DB-backed tests skip and the route guards run.

## What the frontend suite covers

`frontend/tests/` holds one file per module under `frontend/src/lib/`:

| Test file | Module |
|---|---|
| `accountPrefs.test.js` | `accountPrefs.js` |
| `adminApps.test.js` | `adminApps.js`: the section switcher's links and the app-origin env names |
| `adminSections.test.js` | the `/`, `/dictionary` and `/games` pages: shared icons only, every Szótár and Játszótér call through `sectionApi.js` |
| `adminPageSlice.test.js` | `adminPageSlice.js` — admin table paging |
| `apiFetch.test.js` | `api.js` — `parseApiPayload` |
| `csp.test.js` | `svelte.config.js`: the Content-Security-Policy stays strict |
| `entryHistory.test.js` | `entryHistory.js` |
| `entryHours.test.js` | `entryHours.js` |
| `entryPhotos.test.js` | `entryPhotos.js` |
| `entryPublicExtras.test.js` | `entryPublicExtras.js` |
| `entryType.test.js` | `entryType.js` — the closed three-type catalog |
| `quickLinksDisplay.test.js` | `quickLinksDisplay.js` |
| `robots.test.js` | `static/robots.txt`: admin is never crawled |
| `scheduleActivityTypes.test.js` | `scheduleActivityTypes.js` |
| `websiteDomain.test.js` | `websiteDomain.js` |
| `sharedFrontendModules.test.js` | the drift guard for the files shared with lamsza (below) |

Several of these modules are shared with the main `lamsza` app. They are generated copies:
`lamsza` owns them, `lamsza/scripts/sync-shared-frontend.sh` copies them here, and
`sharedFrontendModules.test.js` fails if a copy drifts from the hash manifest. Never edit a
shared module here; change it in `lamsza` and sync (`lamsza/docs/network/SHARED_FRONTEND_MODULES.md`).

Not covered, and why:

- `eventImage.js` imports through the `$lib/api` alias. Plain `node --test` has no SvelteKit
  resolver, so it cannot be imported without an import map. Give it tests when the alias is
  replaced with a relative import.
- `stores/auth.js`, `stores/theme.js` and the `.svelte` components need a DOM. No browser
  test runner is wired up in this repo yet.

Add a new test as `frontend/tests/<module>.test.js`. The glob picks it up — no script change.

---

## Before you call a change done

1. `npm test` green, output pasted into the task rather than summarised.
2. `cd frontend && npm run build` still passes (CI runs the build and both test suites).
3. If the change touches a module shared with `lamsza`, run that repo's suite too.
4. For anything that crosses apps, run the network checks in
   `lamsza/docs/LOCAL_DEV_CHECKS.md`.
