# Lámsza Admin — architecture notes

## Role

Admin UI + API for the **main Lámsza** portal only. Szótár and Játszótér keep their own `/admin` until a later migration.

## Local ports

| App | Frontend | Backend |
|-----|----------|---------|
| Lámsza Admin | 5173 | 3000 |
| Main Lámsza | 5174 | 3001 |
| Szótár | 5175 | 3002 |
| Játszótér | 5176 | 3003 |

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

The admin repo still carries other uncalled `Migrate*` helpers copied from the main repo.
They are dead code and the guard keeps them off the boot path; deleting them is tracked
separately. **Do not wire one into `main()`** — if a schema change is needed, it belongs in
the main `lamsza` repo.

## Shared media

Set absolute paths so both processes read/write the same files:

```
ENTRY_IMAGES_DIR=/home/attila/projects/lamsza/backend/data/entry-images
EVENT_IMAGES_DIR=/home/attila/projects/lamsza/backend/data/event-images
```

After cutover, only this admin API accepts admin upload endpoints; main Lámsza continues to **serve** public media from the same directories.

## Auth

- Google Sign-In + httpOnly session cookie (host-only on localhost via Vite proxy).
- Admin gate: `ADMIN_GOOGLE_EMAILS`.

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
