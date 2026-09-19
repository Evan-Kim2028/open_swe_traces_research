#!/usr/bin/env bash
# Classify a finished agent run. Order matters: hard infra failures first,
# then evidence of delivered work, then shape-of-failure.
out="$1"; dur="${2:-0}"; rc="${3:-0}"; MIN_OK_SECS="${MIN_OK_SECS:-120}"

# Infra failures. Status codes need request context -- a bare "401" or "429"
# in a report body is a number, not an auth failure. (2026-09-18: the loose
# patterns were mislabelling runs whose output merely contained those digits.)
grep -qiE 'rate.?limit(ed)?|too many requests|status.?429|HTTP 429|quota exceeded|overloaded' "$out" 2>/dev/null && { echo RATE_LIMIT; exit; }
grep -qiE 'unauthorized|authentication failed|not logged in|invalid api key|status.?401|HTTP 401' "$out" 2>/dev/null && { echo AUTH; exit; }

# Evidence the run delivered something. Agents cite PRs three ways: a full
# URL, a bare "PR #3139", or a merge sha. Report-only briefs land a memo in
# out/. Requiring the URL form alone scored six real successes NO_PR on
# 2026-09-18 and tripped the circuit breaker.
grep -qE 'https://github\.com/[^ ]+/pull/[0-9]+' "$out" 2>/dev/null && { echo OK; exit; }
# A bare "PR #123" is a claim, not proof. Trust it only on a clean exit; on a
# non-zero exit the run was killed and may have cited a PR it never finished,
# so surface it as PARTIAL for a human check instead of scoring it OK.
if grep -qiE '(^|[^a-z])(PR|pull request) #[0-9]{3,6}' "$out" 2>/dev/null; then
  [ "$rc" -eq 0 ] && { echo OK; exit; }
  echo PARTIAL; exit
fi
grep -qiE 'NO PR NEEDED|MERGED [0-9a-f]{7}|merged as .?[0-9a-f]{7}' "$out" 2>/dev/null && { echo OK; exit; }
grep -qE '/home/evan/devin-tasks/out/[A-Za-z0-9._-]+\.md' "$out" 2>/dev/null && { echo OK; exit; }

[ "$dur" -lt "$MIN_OK_SECS" ] && { echo FAST_FAIL; exit; }
[ "$rc" -ne 0 ] && { echo ERROR; exit; }

# Ran to completion, delivered nothing, and signed off with a question. That
# is a brief defect (autonomy not granted), not an agent defect -- separate
# verdict so it is fixable rather than lost in the NO_PR bucket.
tail -5 "$out" 2>/dev/null | grep -qiE '\?[[:space:]]*$|shall I|should I|would you like|sound right|let me know|confirm before' && { echo ASKED; exit; }
echo NO_PR
