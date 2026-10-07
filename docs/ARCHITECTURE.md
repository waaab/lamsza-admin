# Lámsza Admin — architecture notes

## Role

The network's one admin app (lamsza WAYS_OF_WORKING R18): `/` for the main Lámsza
data, `/dictionary` for Szótár, `/games` for Játszótér. Its `ADMIN_GOOGLE_EMAILS` is the
network's admin list. Szótár and Játszótér have no admin pages or admin list of their
own, and no public app links here: admins open this app by its URL.

## The relay to Szótár and Játszótér

Szótár and Játszótér own their databases; this app never connects to them. Their admin
features are reached through `/api/admin/dictionary/<x>` and `/api/admin/games/<x>`
(`internal/relay`), registered through `admin(...)` like every admin route, so they need
an allowlisted session and get an audit record (resource `szotar` / `jatszoter`, the
request path as route and id, the payload; no before/after state, since the tables live
elsewhere).

The relay forwards method, query and body to the app's internal admin API,
`<SZOTAR_ADMIN_URL | JATSZOTER_ADMIN_URL>/internal/admin/<x>`, on `127.0.0.1` (R13), with
`Authorization: Bearer <SZOTAR_ADMIN_TOKEN | JATSZOTER_ADMIN_TOKEN>` and the acting
admin's email in `X-Admin-Email`. It sends no cookie. The app trusts that request only
when it is outside `/api/`, comes from loopback without proxy headers and carries the
token (each app's `internal/adminapi`). It answers 503 when a URL or token is missing,
and 502 when the app does not answer within 10 s.

## The three sections in the frontend

`frontend/src/routes/+page.svelte` (`/`), `routes/dictionary/+page.svelte` and
`routes/games/+page.svelte` each render `lib/components/admin/AdminShell.svelte`: the
sign-in gate, the icon sidebar, the header with the section switcher (`lib/adminApps.js`)
and sign-out, and back-to-top. A page passes its tabs as `nav` (separators are
`{ sep: true }`), and the dashboard cards (`AdminWelcomeGrid.svelte`) are the same entries
below Vezérlőpult, with the section's counts. Icons come from the shared `AppIcon` only;
`tests/adminSections.test.js` checks that.

- The first sidebar button opens the section's public app in a new tab. Those origins are
  build-time values: `VITE_LAMSZA_ORIGIN`, `VITE_SZOTAR_ORIGIN`, `VITE_JATSZOTER_ORIGIN`.
- A tab has a bare hash (`/dictionary#szavak`), and `#szavak/<id>` opens that word's
  editor; Szótár's entry page links there.
- The Szótár and Játszótér tabs live in `lib/components/dictionary/` and
  `lib/components/games/` (the Szókereső editor moved here from Játszótér). They call the
  relay only through `lib/sectionApi.js`, which returns the app's own Hungarian error
  message, and ask for confirmation through `lib/confirm.svelte.js` with one
  `ConfirmHost` per page.

## Local ports

This app is backend `:3000`, frontend `:5173`. The network's port table lives in one place,
`lamsza/docs/network/WAYS_OF_WORKING.md` §1, matching `lamsza/scripts/start-lamsza-network.sh`
(rule R7: one fact, one home).

## Shared database (current)

- Admin uses the same `DATABASE_URL` / Postgres as main Lámsza.
- **Schema migrations are owned by the main `lamsza` backend. This admin process runs no DDL
  at all** — no `CREATE TABLE`, no `ALTER TABLE`, no `DROP`. Its startup path is
  `config.Load()` then `db.InitDB()`, and `db.InitDB()` only opens the connection pool.
- A future DB split is planned; not implemented here.

### Why, and how it is held

Both processes share one database, so an admin restart that touched the schema would change
a schema it does not own. Until BOG-39 it did: `db.InitDB()` ran
`DROP TABLE locations_legacy`, and `internal/auth` carried a `Migrate()` that created
`users` and `sessions` and added a foreign key to `users`. Both are deleted. The main
backend still owns those statements — it creates `users`/`sessions` on its own boot path,
and `lamsza/backend/migrations/drop_locations_legacy.sql` is the explicit form of the drop.

`backend/boot_ddl_test.go` holds the rule. It re-derives the startup path from the source on
every `go test` run — `main()` plus every package `init()`, followed across packages — and
fails if any reachable function contains a DDL statement or is a `Migrate*` helper. It needs
no database, so it runs in the normal `npm test`.

Those uncalled `Migrate*` helpers copied from the main repo are **gone** as of BOG-42, with
the rest of the code this backend never routed. **Do not wire a migration into `main()`** —
if a schema change is needed, it belongs in the main `lamsza` repo. `boot_ddl_test.go` still
holds the rule.

## The public read surface is five routes

This backend was forked from `lamsza/backend` and kept the whole public half: 31 `Handle*`
functions `main.go` never routed, and 186 functions behind them, against the same database
with no owner. BOG-42 deleted all of it — the entries read paths, the weather providers,
events, the news fetch and cache, mondasok, quick links, pages, favorites, history, links,
account prefs, the listing claim/catalog/members surface, website submit and lookup,
attractions, and `internal/webdomain` and `internal/account/prefs.go` entirely.
(`internal/search` went earlier, on BOG-40.)

**This repo is the write side.** Five public reads remain, because the admin UI needs them
for its own forms: `/api/config/public`, `/api/counties`, `/api/historical_seats`,
`/api/venues` and `/api/venue_types`. Anything else the admin UI needs from the public app,
call `lamsza` on `:3001` — do not copy the handler back. Every route lives in `newMux()` in
`main.go`, and `route_guard_test.go` reads that table on every test run; keep every handler
routed:

```bash
cd ~/projects/lamsza-network/lamsza-admin/backend && go build ./... && go test ./...
```

## Write audit log

Every mutating call on `/api/admin/*` lands one row in `admin_audit_log`: who, when, which
resource, which action, the request payload, and the before/after of the row it touched.
Added on BOG-48 — this repo is the only writer of the directory the public site reads, and
there was no way at all to answer "who deleted this entry".

Query it at `GET /api/admin/audit-log`, filtered by `resource`, `resource_id`, `actor_email`
or `action` (`limit` defaults to 100, max 500).

### Coverage is wiring, not discipline

There are thirty-odd mutating admin routes across twelve packages, so per-handler
instrumentation would be instrumentation missing from the route somebody adds next month.
Instead `main.go` registers **every** admin route through one `admin(route, handler)` helper
that applies `audit.Wrap`, and:

- `audit.Wrap` **panics at startup** on a route with no resource declared in
  `internal/audit/resources.go`. A route nobody decided the audit shape for does not boot.
- `backend/audit_routes_test.go` re-parses `main.go` on every `go test` run (the way
  `boot_ddl_test.go` does for DDL) and fails if an `/api/admin/` path is registered by a bare
  `mux.HandleFunc`, if a registered route has no declared resource, or if the registry
  declares a route `main.go` no longer registers.

So **register admin routes with `admin(path, handler)`** — never `mux.HandleFunc` directly.
Adding a route means adding its entry to `resources.go`; the build tells you if you forget.

### Append-only from the app

The only statements this app runs against the table are `INSERT` and `SELECT`. No admin route
updates or deletes a record, and the read route refuses anything but `GET`.
`internal/audit/append_only_test.go` scans the backend source and fails on any `UPDATE`,
`DELETE`, `TRUNCATE`, `DROP` or `ALTER` against `admin_audit_log`.

That is the half that lives in the code. The other half — a database role with `INSERT` and
`SELECT` but no `UPDATE`/`DELETE` on this table — belongs to the production database and is
the owner's to set, not an agent's.

### Retention and size

**No automatic retention: rows live until someone prunes them.** Nothing in the app deletes
from this table, by design — see above.

The app bounds the *row*, not the table. `payload`, `before_state`, `after_state` and `diff`
are each capped at 16 KiB; an oversized value is replaced whole with
`{"_truncated":true,…}` rather than cut, because a truncated JSON value is not valid JSONB.
Request bodies over 512 KiB are not buffered at all, and non-JSON bodies (image uploads)
never are — those records carry a marker with the content type and length instead.

A heavy admin day is a few thousand rows of a few KiB, so this grows by **megabytes a year,
not gigabytes**. Pruning is a deliberate operator action against the database when it is ever
worth it:

```sql
DELETE FROM admin_audit_log WHERE occurred_at < NOW() - INTERVAL '2 years';
```

### What it does not capture

`Tables` is empty for routes where no single row holds the resource's state — a composite
write (an event's whole schedule, an entry's category links), a key/value map
(`site_settings`), an upload, or a cache bump. **Those records carry the actor, action and
request payload, but no before/after.** Closing that gap means teaching the handlers to
report their own snapshots; it was not worth it for BOG-48.

Credential-shaped keys (`password`, `secret`, `token`, `api_key`, `client_secret`, …) are
replaced with `[redacted]` in both payloads and row snapshots — the trail is read by more
people than the secrets are shared with, and a settings write is exactly where a key would
otherwise land.

A failed audit write never fails the request: the record is written after the handler has
already committed, so refusing the response would hide a completed change behind a 500. A
dropped record is instead logged with a greppable `AUDIT WRITE FAILED:` prefix.

### The table is created by `lamsza`

`admin_audit_log` is DDL, and this process runs none (see above). The main `lamsza` backend
creates it on boot in `internal/auth.migrateAdminAuditLog()`, with the explicit form and the
same retention note in `lamsza/backend/migrations/admin_audit_log.sql`. Nothing in the
`lamsza` repo reads or writes the table — only `lamsza-admin/backend/internal/audit` does.

## Shared frontend modules

The frontend files listed in `frontend/shared-frontend-modules.json` are identical to
`lamsza`'s: helpers, the shared dialogs (`ConfirmDialog`, `NoticeDialog`, `SignInDialog`,
`GoogleSignIn`), the icons (`AppIcon`, `ErrorLantern`), the error page (`ErrorPage`,
`ErrorShell`) and the three base stylesheets. **`lamsza` owns them**; this repo carries a generated copy under
`frontend/src`. Never edit one here: edit it in `lamsza`, run
`lamsza/scripts/sync-shared-frontend.sh`, and commit lamsza and every app that changed. `frontend/tests/sharedFrontendModules.test.js` hashes the copies against
the manifest on every `npm test`, so an edit made here goes red. The rule and the rejected
alternatives are in `lamsza/docs/network/SHARED_FRONTEND_MODULES.md`.

## Shared media

Set absolute paths so both processes read/write the same files:

```
ENTRY_IMAGES_DIR=/home/attila/projects/lamsza-network/lamsza/backend/data/entry-images
EVENT_IMAGES_DIR=/home/attila/projects/lamsza-network/lamsza/backend/data/event-images
```

After cutover, only this admin API accepts admin upload endpoints; main Lámsza continues to **serve** public media from the same directories.

## Auth

- Google Sign-In + httpOnly session cookie (host-only on localhost via Vite proxy).
- Admin gate: `ADMIN_GOOGLE_EMAILS`.
- `/api/auth/google` refuses an account that is not on that list with `403`. Every route
  here is behind `RequireAdmin` anyway, so such a session bought nothing — but it put a
  public-grade row in the admin session table, which is the one thing a separate store
  must not hold. A row there now means an admin is signed in.

### The admin session is not the public session

| | public (`lamsza`, `:3001`) | admin (this app, `:3000`) |
|---|---|---|
| cookie | `lamsza_session` | `lamsza_admin_session` |
| table | `sessions` | `admin_sessions` |

Until BOG-45 both were the same cookie name and the same table, so a token the public site
handed to any signed-in visitor was accepted by every route on this API. Two things keep
them apart, and each covers what the other cannot:

- **A different cookie name.** Browsers scope cookies by host and *ignore the port*, so on
  localhost a cookie set by lamsza on `:3001`/`:5174` is sent to this API on `:3000` no
  matter what. A different name means the browser never offers it.
- **A different table.** The name alone is only presentation — anyone can put any token in
  any cookie. The store is the real boundary: an admin token hash exists only in
  `admin_sessions`, a public one only in `sessions`, so neither API can accept the other's
  token even when it is handed over deliberately.

`users` stays shared. Identity is shared across the network; only the *session* is not.

`admin_sessions` is created by the main `lamsza` backend — this process runs no DDL (see
above). The statements live in `lamsza/backend/internal/auth/auth.go` and, in explicit
form, `lamsza/backend/migrations/admin_sessions.sql`. On a fresh database, run the main
backend or that migration; do **not** add a `CREATE TABLE` here.

Held by `backend/internal/auth/session_boundary_test.go` on this side and
`lamsza/backend/admin_session_boundary_test.go` on the other. The two DB-backed tests here
run only against the scratch database `npm run test:backend` builds (`TEST_DATABASE_URL`,
name ending in `_test`), never the shared dev database, and skip without it; the
cookie-name check and the static check that this package never queries `sessions` need no
database and run in CI.

## Cross-origin calls (CORS)

`backend/internal/middleware/middleware.go` answers a cross-origin call only when the `Origin`
header matches the allowlist exactly, and it sends `Access-Control-Allow-Credentials` only then.

Never echo the request `Origin` back. All four sites sit under one registrable domain, so the
session cookie is same-site for every sibling and `SameSite=Lax` does not hold it back: a
reflected `Origin` plus `Allow-Credentials` let *any* page read the signed-in reply of an admin
who happened to visit it. On this API that is every route, so it is enough to take the account
over.

The allowlist doubles as the CSRF guard. A page cannot forge the `Origin` header, so a POST, PUT,
PATCH or DELETE that carries an `Origin` we do not know is refused with `403`. Clients outside a
browser (curl, the deployment scripts) send no `Origin` and are unaffected.

`CORS_ALLOWED_ORIGINS` is a comma-separated list of exact origins, for example
`https://admin.lamsza.com`. It **replaces** the built-in list, so set it in production to drop
the development hosts. The built-in list covers the four `*.lamsza.com` sites plus the
`*.lamsza.test` and `localhost` development hosts. Behaviour is covered in
`backend/internal/middleware/middleware_test.go`.

The admin frontend reaches this API same-origin — through the Vite dev proxy locally, through
Nginx in production — so there is no expected cross-origin caller today.

## Production (Phase 2)

`admin.lamsza.com` / nginx / `admin.service` on `:8083` is **not** part of this
delivery. Wire-up later when that environment is ready.

### Keep it out of the search engines

Admin is a private back office with no public page. Two independent controls,
because each one covers what the other cannot:

1. `frontend/static/robots.txt` is `Disallow: /`. That asks a well-behaved
   crawler not to fetch. It does **not** remove a URL someone else has already
   linked, and a crawler that ignores it is not breaking any rule.
2. `X-Robots-Tag: noindex, nofollow` on every response. This one is an
   instruction about *indexing*, so it keeps a URL out of the results even when
   the crawler got there by a link rather than through `robots.txt`.

The vhost must send the header on the whole host — not just on HTML — because
`dist/` is static files and admin has no application layer to add it:

```nginx
server {
    listen 443 ssl http2;
    server_name admin.lamsza.com;

    # Private back office: nothing here belongs in a search index.
    # `always` matters — without it nginx drops the header on 4xx/5xx, and an
    # error page is exactly the kind of URL that gets indexed by accident.
    add_header X-Robots-Tag "noindex, nofollow, noarchive" always;

    root /var/www/admin/public;      # the built dist/
    index app.html;

    location / {
        try_files $uri $uri/ /app.html;    # SPA fallback
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8083;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

> Any `add_header` inside a `location` block **replaces** the whole inherited
> set for that location rather than adding to it. If a `location` needs its own
> `add_header`, repeat `X-Robots-Tag` there too.

Also set, at cutover: `CORS_ALLOWED_ORIGINS=https://admin.lamsza.com` (see
above), and add `https://admin.lamsza.com` to the Google OAuth client's
authorized JavaScript origins — it is listed in
`lamsza/docs/PRODUCTION_SERVER_SETUP.md` §3.3, but the Google Console change
itself is part of the cutover task.
