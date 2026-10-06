#!/usr/bin/env bash
#
# Mirrors the state of CI on `main` into one GitHub issue, so that a red CI has a
# receiver instead of only a log nobody opens.
#
# Why an issue, and why GitHub writes it instead of an agent reading it: nothing
# on the dev machine can read CI. There is no `gh` CLI there, git auth is
# SSH-key only, unauthenticated api.github.com reads from that IP are
# rate-limited to zero, and two of the four repos are private. So the signal has
# to be pushed out from inside Actions, where `GITHUB_TOKEN` already exists and
# needs no secret to be configured. An issue beats a run-failure email because it
# survives log retention, it is visible in the repo to anyone including an agent
# with a token later, and it can be closed again automatically when main goes
# green — so its mere existence is the current red/green answer.
#
# Decided on BOG-54 ("both": this, plus branch protection). See R7 in lamsza's
# docs/network/WAYS_OF_WORKING.md.
#
# Env in:
#   GH_TOKEN      credential for gh (the workflow passes github.token)
#   GH_REPO       owner/name, so gh needs no git remote
#   CI_FRONTEND   result of the `frontend` job: success | failure | …
#   CI_BACKEND    result of the `backend` job
#   CI_SHA        commit that was built
#   CI_RUN_URL    link back to the Actions run
#
# Run the unit test after editing: bash .github/ci-status-issue.test.sh
set -euo pipefail

TITLE="CI is red on main"

: "${CI_FRONTEND:?CI_FRONTEND is required}"
: "${CI_BACKEND:?CI_BACKEND is required}"
: "${CI_SHA:?CI_SHA is required}"
: "${CI_RUN_URL:?CI_RUN_URL is required}"

failed=""
[ "$CI_FRONTEND" = "failure" ] && failed="frontend"
[ "$CI_BACKEND" = "failure" ] && failed="${failed:+$failed, }backend"

# Every GitHub call below needs the `issues` scope, and the workflow asking for it
# is not sufficient. Settings -> Actions -> General -> Workflow permissions is a
# per-repo ceiling on GITHUB_TOKEN: while it is set to "Read repository contents
# and packages permissions", the issue scope is dropped however the job's
# `permissions:` block is written, and every write here returns 403.
#
# That is the worst failure this script has, because it is silent in the only
# channel that matters. The receiver stops reporting a red `main`, and the one
# symptom is the `ci-status` job going red — which is a thing only this script
# reports. Nothing on the dev machine can read Actions to notice (no gh, SSH-only
# git auth, no API budget, two private repos), so the diagnosis has to be written
# where a person landing on the run will see it without digging: the step log and
# the run summary page.
explain_failure() {
  local cmd="$1" out="$2" diag

  if printf '%s' "$out" | grep -qiE 'HTTP 403|Resource not accessible|not accessible by integration'; then
    diag="### The CI receiver is switched off by a repo setting

A red \`main\` currently reaches nobody in \`${GH_REPO:-this repo}\`.

\`gh $cmd\` was refused. This job does ask for \`issues: write\`, but
*Settings → Actions → General → Workflow permissions* caps what
\`GITHUB_TOKEN\` can be granted. While that is set to *Read repository contents
and packages permissions*, the issue scope is dropped and this script cannot
open, comment on or close the status issue.

**Fix:** set it to *Read and write permissions*. No repo content needs to
change. Decided on BOG-54; see R7 in lamsza's \`docs/network/WAYS_OF_WORKING.md\`."
  else
    diag="### The CI receiver failed

A red \`main\` may currently reach nobody in \`${GH_REPO:-this repo}\`.

\`gh $cmd\` failed for a reason this script does not recognise:

\`\`\`
$out
\`\`\`

Until it works, the local gate in \`docs/LOCAL_DEV_CHECKS.md\` is the only gate.
Run \`bash .github/ci-status-issue.test.sh\` after any fix."
  fi

  printf '%s\n' "$diag" >&2
  # Best effort: on a real runner this renders on the run's summary page, which is
  # the first thing anyone opens. Absent locally and in the unit test.
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    printf '%s\n' "$diag" >>"$GITHUB_STEP_SUMMARY" || true
  fi
}

# Runs gh, passing its stdout through on success. On failure it explains itself
# and returns gh's status, which `set -e` turns into a red job — deliberately, so
# a broken receiver is never mistaken for a quiet one.
gh_or_explain() {
  local out rc=0
  out=$(gh "$@" 2>&1) || rc=$?
  if [ "$rc" -ne 0 ]; then
    # Just "issue list", not the whole jq filter — the subcommand is what tells
    # you which call was refused, and the filter buries it.
    explain_failure "${1:-} ${2:-}" "$out"
    return "$rc"
  fi
  # Trailing newline keeps the uncaptured calls' log output on its own line; `$( )`
  # strips it again for the two callers that capture.
  printf '%s\n' "$out"
}

# Exact-title match over open issues rather than `gh issue list --search`: the
# search index lags by seconds to minutes, and this runs immediately after the
# push that broke the build, so a search would miss the issue it just opened and
# file a duplicate on every subsequent red push. These repos have nowhere near
# 100 open issues.
number=$(gh_or_explain issue list --state open --limit 100 --json number,title \
  --jq "[.[] | select(.title == \"$TITLE\")] | first | .number // empty")

short="${CI_SHA:0:8}"

if [ -n "$failed" ]; then
  body="CI failed on \`main\`.

| | |
| --- | --- |
| Failed jobs | **$failed** |
| Commit | \`$short\` |
| Run | $CI_RUN_URL |

Per R7 in \`docs/network/WAYS_OF_WORKING.md\`: do not merge more work onto
\`main\` until this is green, and run the \`docs/LOCAL_DEV_CHECKS.md\` checks
locally — CI is the weaker of the two gates.

This issue closes itself on the next green run of \`main\`."

  if [ -n "$number" ]; then
    gh_or_explain issue comment "$number" --body "$body"
    echo "still red: commented on #$number"
  else
    url=$(gh_or_explain issue create --title "$TITLE" --body "$body")
    echo "went red: opened $url"
  fi
  exit 0
fi

# Green. Only touch the issue if one is actually open, so a normal green push
# costs one API read and nothing else.
if [ -z "$number" ]; then
  echo "green, no open \"$TITLE\" issue: nothing to do"
  exit 0
fi

gh_or_explain issue close "$number" \
  --comment "Green again on \`main\` at \`$short\`. Run: $CI_RUN_URL"
echo "recovered: closed #$number"
