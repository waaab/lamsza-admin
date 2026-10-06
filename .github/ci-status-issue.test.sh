#!/usr/bin/env bash
#
# Unit test for .github/ci-status-issue.sh. Runs locally with no network and no
# GitHub credential: it puts a fake `gh` on PATH that records how it was called
# and answers `issue list` from a fixture.
#
# The `--jq` filter is evaluated by the real jq rather than faked, because the
# exact-title match is the part most likely to break and silently file duplicate
# issues on every red push.
#
# Usage: bash .github/ci-status-issue.test.sh
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="$here/ci-status-issue.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

command -v jq >/dev/null || { echo "FAIL: this test needs jq"; exit 1; }

# --- the fake gh ------------------------------------------------------------
mkdir -p "$tmp/bin"
cat >"$tmp/bin/gh" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FAKE_GH_LOG"
if [ "${1:-}" = "issue" ] && [ "${2:-}" = "list" ]; then
  filter='.'
  while [ $# -gt 0 ]; do
    if [ "$1" = "--jq" ]; then filter="$2"; fi
    shift
  done
  jq -r "$filter" <"$FAKE_GH_ISSUES"
  exit 0
fi
if [ "${1:-}" = "issue" ] && [ "${2:-}" = "create" ]; then
  echo "https://github.com/waaab/fake/issues/99"
fi
exit 0
FAKE
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH"

pass=0
fail=0

# $1 name, $2 fixture json, $3 frontend result, $4 backend result,
# $5 expected substring in the gh call log, $6 expected substring in stdout
run_case() {
  local name="$1" fixture="$2" fe="$3" be="$4" want_call="$5" want_out="$6"
  export FAKE_GH_LOG="$tmp/log" FAKE_GH_ISSUES="$tmp/issues.json"
  : >"$FAKE_GH_LOG"
  printf '%s' "$fixture" >"$FAKE_GH_ISSUES"

  local out
  if ! out=$(CI_FRONTEND="$fe" CI_BACKEND="$be" \
      CI_SHA="0123456789abcdef" \
      CI_RUN_URL="https://example.invalid/run/1" \
      GH_TOKEN=fake GH_REPO=waaab/fake bash "$script" 2>&1); then
    echo "FAIL $name: script exited non-zero"
    printf '%s\n' "$out" | sed 's/^/      /'
    fail=$((fail + 1))
    return
  fi

  if ! grep -qF "$want_call" "$FAKE_GH_LOG"; then
    echo "FAIL $name: expected a gh call matching '$want_call'; calls were:"
    sed 's/^/      /' "$FAKE_GH_LOG"
    fail=$((fail + 1))
    return
  fi
  if ! printf '%s' "$out" | grep -qF "$want_out"; then
    echo "FAIL $name: expected output '$want_out', got: $out"
    fail=$((fail + 1))
    return
  fi
  echo "ok   $name"
  pass=$((pass + 1))
}

# $1 name, $2 fixture, $3 frontend, $4 backend, $5 substring that must NOT appear
refute_call() {
  local name="$1" fixture="$2" fe="$3" be="$4" unwanted="$5"
  export FAKE_GH_LOG="$tmp/log" FAKE_GH_ISSUES="$tmp/issues.json"
  : >"$FAKE_GH_LOG"
  printf '%s' "$fixture" >"$FAKE_GH_ISSUES"

  CI_FRONTEND="$fe" CI_BACKEND="$be" CI_SHA="0123456789abcdef" \
    CI_RUN_URL="https://example.invalid/run/1" \
    GH_TOKEN=fake GH_REPO=waaab/fake bash "$script" >/dev/null 2>&1

  if grep -qF "$unwanted" "$FAKE_GH_LOG"; then
    echo "FAIL $name: did not expect a gh call matching '$unwanted'; calls were:"
    sed 's/^/      /' "$FAKE_GH_LOG"
    fail=$((fail + 1))
    return
  fi
  echo "ok   $name"
  pass=$((pass + 1))
}

NONE='[]'
OPEN='[{"number":7,"title":"CI is red on main"}]'
# Two decoys that a substring or search-index match would wrongly latch onto.
DECOYS='[{"number":3,"title":"CI is red on main (old, closed by hand)"},
         {"number":4,"title":"Flaky: CI is red on main sometimes"}]'
OPEN_AMONG_DECOYS='[{"number":4,"title":"Flaky: CI is red on main sometimes"},
                    {"number":7,"title":"CI is red on main"}]'

echo "— goes red —"
run_case "red with no open issue opens one" \
  "$NONE" failure success "issue create --title CI is red on main" "went red: opened"
run_case "red names the failing job" \
  "$NONE" success failure "Failed jobs | **backend**" "went red"
run_case "red names both failing jobs" \
  "$NONE" failure failure "Failed jobs | **frontend, backend**" "went red"

echo "— stays red —"
run_case "red with the issue already open comments instead of duplicating" \
  "$OPEN" failure success "issue comment 7" "still red: commented on #7"
run_case "finds the issue among similarly titled ones" \
  "$OPEN_AMONG_DECOYS" failure success "issue comment 7" "commented on #7"
refute_call "a near-miss title is not treated as the issue" \
  "$DECOYS" failure success "issue comment"

echo "— goes green —"
run_case "green with an open issue closes it" \
  "$OPEN" success success "issue close 7" "recovered: closed #7"
refute_call "green with no open issue touches nothing" \
  "$NONE" success success "issue c"

echo "— neither —"
# `cancelled` and `skipped` must not be read as failure; the workflow also gates
# on this, but the script must not depend on that gate being right.
refute_call "a cancelled run does not open an issue" \
  "$NONE" cancelled success "issue create"

echo
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
