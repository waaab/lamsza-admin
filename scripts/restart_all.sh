#!/usr/bin/env bash
# Restart Lámsza Admin (FE :5173, BE :3000).
# Postgres is owned by the main lamsza app — start that DB first if needed.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BACKEND_DIR="backend"
FRONTEND_DIR="frontend"
BACKEND_PID_FILE="$BACKEND_DIR/server.pid"
FRONTEND_PID_FILE="$FRONTEND_DIR/server.pid"

echo "Restarting lamsza-admin..."

if [ -f "$FRONTEND_PID_FILE" ]; then
	kill "$(cat "$FRONTEND_PID_FILE")" 2>/dev/null || true
	rm -f "$FRONTEND_PID_FILE"
fi
if [ -f "$BACKEND_PID_FILE" ]; then
	kill "$(cat "$BACKEND_PID_FILE")" 2>/dev/null || true
	rm -f "$BACKEND_PID_FILE"
fi
fuser -k 3000/tcp 2>/dev/null || true
fuser -k 5173/tcp 2>/dev/null || true
sleep 1

# Prefer repo-root .env for both processes
if [ -f "$ROOT/.env" ] && [ ! -f "$BACKEND_DIR/.env" ]; then
	cp "$ROOT/.env" "$BACKEND_DIR/.env"
fi

(cd "$BACKEND_DIR" && go run . > server_backend.log 2>&1 & echo $! > server.pid)
(cd "$FRONTEND_DIR" && npm run dev > server_frontend.log 2>&1 & echo $! > server.pid)

echo "Backend http://localhost:3000  Frontend http://localhost:5173"
echo "Shared DB: use main lamsza docker compose (Postgres :5433)."
