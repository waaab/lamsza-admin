# Lámsza Admin

Independent admin app for the main [Lámsza](https://github.com/waaab/lamsza) portal.

**Local URL:** http://localhost:5173/  
**API:** http://localhost:3000  

## Stack

- Frontend: SvelteKit (static SPA) + Svelte 5
- Backend: Go + shared PostgreSQL (same DB as main Lámsza for now)
- Auth: Google Sign-In + `ADMIN_GOOGLE_EMAILS`

## Layout

```
lamsza-admin/
├── frontend/   # SvelteKit UI (port 5173)
├── backend/    # Go API (port 3000)
├── docs/
└── scripts/
```

## Prerequisites

- Main Lámsza Postgres running (`cd ~/projects/lamsza-network/lamsza && docker compose up -d`)
- Schema migrations are applied by the **main** Lámsza backend — run that at least once
- Google OAuth authorized JS origins include `http://localhost:5173`

## Quick start

```bash
cp .env.example .env   # fill GOOGLE_CLIENT_ID / ADMIN_GOOGLE_EMAILS
cp .env backend/.env

cd frontend && npm install && cd ..
npm run restart        # or: cd backend && go run .  &  cd frontend && npm run dev
```

Open http://localhost:5173/

## Tests

```bash
npm test               # frontend (node:test) + backend (go test)
npm run test:frontend
npm run test:backend
```

See [`docs/LOCAL_DEV_CHECKS.md`](docs/LOCAL_DEV_CHECKS.md) for what is covered and what a
complete local pass looks like.

## Shared resources

| Resource | Owner now |
|----------|-----------|
| PostgreSQL | Shared with main Lámsza (`DATABASE_URL`) |
| Schema migrations | Main `lamsza` backend only |
| Entry/event image dirs | Shared via `ENTRY_IMAGES_DIR` / `EVENT_IMAGES_DIR` |

A later DB split is planned; not part of this app’s current scope.

## Production

`admin.lamsza.com` deploy is **Phase 2** (not configured in this delivery).

## Out of scope

Szótár admin and Játszótér admin remain in their own apps for now.
