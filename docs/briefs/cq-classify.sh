#!/usr/bin/env bash
# Classify one agent run's output. Order matters: SUCCESS IS CHECKED FIRST.
#
# Two ways this has misfired, both of which halted a healthy lane by
# inflating consec_bad:
#
#   1. Success went unrecognized because the agent phrased it differently.
#      Brief 13 wrote "Merged as `5b6280b16`" (no URL) and scored ERROR.
#      Brief 25 wrote "Shipped and merged: **PR #4074** -> `main` as
#      `1eca0e139`" — no URL, and not the literal words "merged as" — so it
#      scored NO_PR and was filed under mfailed despite exiting 0 with the
#      PR merged and 1396 tests green.
#
#   2. Infra heuristics ran before success detection, so a brief whose
#      SUBJECT was rate limiting or quota could be read as having HIT one.
#      That never fired in anger, but it is one adversarial brief away.
#
# A run that demonstrably landed a PR succeeded, whatever words its report
# happens to contain. So: prove success first, diagnose failure second.
out="$1"; dur="$2"; rc="$3"

merged_evidence() {
  # A full PR URL, anywhere.
  grep -qiE 'https://github\.com/[^ ]+/pull/[0-9]+' "$out" 2>/dev/null && return 0
  # A bare "PR #1234" / "pull request #1234" anywhere. Agents do not cite a PR
  # number casually, and requiring a merge-word nearby missed brief 31, which
  # wrote the number in a heading far from any such word.
  grep -qiE '(^|[^a-z])(pr|pull request) ?#[0-9]{2,6}' "$out" 2>/dev/null && return 0
  # "...merged/landed/shipped ... #1234" with the number close by.
  grep -qiE '(merg|land|ship|open)[a-z]*[^\n]{0,60}#[0-9]{2,6}' "$out" 2>/dev/null && return 0
  # A bare commit sha on a line that mentions merging/landing/shipping.
  grep -qiE '(merg|land|ship)[a-z]*[^\n]{0,80}\b[0-9a-f]{7,40}\b' "$out" 2>/dev/null && return 0
  # Explicit no-PR-required declaration.
  grep -qiE 'NO PR NEEDED' "$out" 2>/dev/null && return 0
  return 1
}

merged_evidence && { echo OK; exit; }

# Only now consider infrastructure failures. Anchor these to lines that read
# like a reported error rather than prose discussing the concept.
grep -qiE '(^|[^a-z])(429|rate.?limit(ed|ing)? (exceeded|hit|reached)|too many requests|quota exceeded|server (is )?overloaded|service unavailable)' "$out" 2>/dev/null && { echo RATE_LIMIT; exit; }
grep -qiE '(^|[^a-z])(401|403|unauthorized|not logged in|invalid api key|authentication failed)' "$out" 2>/dev/null && { echo AUTH; exit; }

[ "$dur" -lt 120 ] && { echo FAST_FAIL; exit; }
[ "$rc" -ne 0 ] && { echo ERROR; exit; }
echo NO_PR
