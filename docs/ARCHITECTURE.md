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

## Production (Phase 2)

`admin.lamsza.com` / nginx / `admin.service` on `:8083` is **not** part of this delivery. Wire-up later when that environment is ready.
