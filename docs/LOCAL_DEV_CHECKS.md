# Local Dev Checks — lamsza-admin

**Audience:** anyone developing this app on this machine, agents included.
**Target:** `localhost` only. See `lamsza/docs/AGENT_ENVIRONMENT_POLICY.md` — production is
the owner's.
**Scope:** this repo. The network-wide checks (all four apps up, every API smoked, dump and
restore) live in `lamsza/docs/LOCAL_DEV_CHECKS.md`. That file stays the definition of
"verified" for the network; this one is the repo-level gate.
**Last updated:** 2026-10-06 — every command below was run and the output recorded.

---

## The one command

```bash
cd ~/projects/lamsza-admin && npm test
```

It runs both suites and stops at the first failure:

| Step | Script | What it runs |
|---|---|---|
| 1 | `npm run test:frontend` | `node --test 'tests/*.test.js'` in `frontend/` |
| 2 | `npm run test:backend` | `go test ./...` in `backend/` |

Run one side on its own with `npm run test:frontend` or `npm run test:backend`.

No `npm install` is needed for the tests. The frontend suite uses only `node:test` and
`node:assert`, and imports the modules under `frontend/src/lib/` directly.

> Use the glob `'tests/*.test.js'`, quoted. `node --test tests/` (directory form) crashes
> with `MODULE_NOT_FOUND` on Node 24.

**Status 2026-10-06** — Node v24.11.1, Go 1.25.7:

| Suite | Result |
|---|---|
| frontend (`frontend/tests/`) | **64 / 64 pass**, 11 test files |
| backend (`backend/...`) | **pass**, 11 packages with tests |

---

## What the frontend suite covers

`frontend/tests/` holds one file per module under `frontend/src/lib/`:

| Test file | Module |
|---|---|
| `accountPrefs.test.js` | `accountPrefs.js` |
| `adminPageSlice.test.js` | `adminPageSlice.js` — admin table paging |
| `apiFetch.test.js` | `api.js` — `parseApiPayload` |
| `entryHistory.test.js` | `entryHistory.js` |
| `entryHours.test.js` | `entryHours.js` |
| `entryPhotos.test.js` | `entryPhotos.js` |
| `entryPublicExtras.test.js` | `entryPublicExtras.js` |
| `entryType.test.js` | `entryType.js` — the closed three-type catalog |
| `quickLinksDisplay.test.js` | `quickLinksDisplay.js` |
| `scheduleActivityTypes.test.js` | `scheduleActivityTypes.js` |
| `websiteDomain.test.js` | `websiteDomain.js` |

Eight of these modules are byte-identical to the main `lamsza` copies — admin was extracted
from that repo. Their tests were taken from `lamsza/tests/` on purpose: **if a shared module
changes in one repo, the same test has to pass in both.** Keep them in step.

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
2. `npm run build` still passes (CI runs the frontend build and the Go build, not the tests).
3. If the change touches a module shared with `lamsza`, run that repo's suite too.
4. For anything that crosses apps, run the network checks in
   `lamsza/docs/LOCAL_DEV_CHECKS.md`.
