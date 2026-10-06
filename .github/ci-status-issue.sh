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

# Exact-title match over open issues rather than `gh issue list --search`: the
# search index lags by seconds to minutes, and this runs immediately after the
# push that broke the build, so a search would miss the issue it just opened and
# file a duplicate on every subsequent red push. These repos have nowhere near
# 100 open issues.
number=$(gh issue list --state open --limit 100 --json number,title \
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
    gh issue comment "$number" --body "$body"
    echo "still red: commented on #$number"
  else
    url=$(gh issue create --title "$TITLE" --body "$body")
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

gh issue close "$number" \
  --comment "Green again on \`main\` at \`$short\`. Run: $CI_RUN_URL"
echo "recovered: closed #$number"
