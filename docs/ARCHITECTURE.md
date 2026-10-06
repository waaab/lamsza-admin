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
- **Schema migrations are owned by the main `lamsza` backend.** This admin process does not run migrate helpers on startup.
- A future DB split is planned; not implemented here.

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

`admin.lamsza.com` / nginx / `admin.service` on `:8083` is **not** part of this delivery. Wire-up later when that environment is ready.
