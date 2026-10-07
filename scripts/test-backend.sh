#!/usr/bin/env bash
# Run the Go backend suites against a fresh scratch database, never the shared
# dev lamsza database.
#
# Admin owns no schema: the lamsza repo does. So the scratch database is built
# with lamsza's own scripts/db-bootstrap.sh from lamsza/backend/schema/, the
# same files CI and a fresh machine use. The lamsza repo is expected next to
# this one (~/projects/lamsza-network/lamsza); set LAMSZA_REPO to override.
#
#   scripts/test-backend.sh                     # every package
#   scripts/test-backend.sh ./internal/auth/... # just these packages
#
# Each run drops and recreates the scratch database. Override the target with
# TEST_DATABASE_URL; its name must end in _test (backend/internal/db/testguard.go).

set -euo pipefail

here=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(cd -- "$here/.." && pwd)
lamsza=${LAMSZA_REPO:-"$repo/../lamsza"}

if [[ ! -x $lamsza/scripts/db-bootstrap.sh ]]; then
	echo "cannot find lamsza/scripts/db-bootstrap.sh under '$lamsza'; set LAMSZA_REPO" >&2
	exit 2
fi
# shellcheck source=/dev/null
source "$lamsza/scripts/lib/pg-url.sh"

url=${TEST_DATABASE_URL:-"postgres://lamsza_user:lamsza_password@localhost:5433/lamsza_admin_test?sslmode=disable"}

pg_url_parse "$url"
target=$PGDATABASE
if [[ $target != *_test ]]; then
	echo "refusing: test database name must end in _test, got '$target'" >&2
	exit 2
fi

runner=$(pg_pick_runner "" psql)
PGDATABASE=postgres
pg_run "$runner" psql -qc "DROP DATABASE IF EXISTS \"$target\" WITH (FORCE)"
PGDATABASE=$target

"$lamsza/scripts/db-bootstrap.sh" --create --url "$url"

cd "$repo/backend"
if [[ $# -eq 0 ]]; then
	set -- ./...
fi
TEST_DATABASE_URL=$url go test -count=1 "$@"
