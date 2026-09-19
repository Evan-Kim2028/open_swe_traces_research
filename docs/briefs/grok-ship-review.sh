#!/usr/bin/env bash
# grok-ship-review.sh <repo_path> [hours] -- ONE broad cursor-agent/grok-4.6 pass
# that acts as the SHIPPING gate, not another code review.
#
# Per-PR review already happened. Cluster review (grok-batch-review.sh) already
# looked for cross-PR interactions. This asks the last question nobody else
# does: is what is on main actually correct, deployed, and safe to leave running
# unattended?
set -u
REPO="${1:-/home/evan/Documents/silphcoanalytics}"
HOURS="${2:-12}"
OUT="/home/evan/devin-tasks/grok-ship-$(basename "$REPO")-$(date +%Y%m%dT%H%M%S).md"

PROMPT="You are the final shipping gate for $REPO. Everything below is ALREADY
MERGED to main. Per-PR review and a cross-PR cluster review both already ran.

Do NOT re-review diffs line by line. Answer only what those passes cannot:

1. VERIFY ON MAIN, NOT IN THE PR. Check out main and actually run the relevant
   test suites. Claims in PR bodies are not evidence. Report what you ran and
   what it returned.

2. CONTRACT SAFETY. Did any merged change alter a public response shape, tool
   signature, or documented vocabulary in a way an existing client could not
   tolerate? This surface is consumed by external agents. A field that changed
   from object to null, a renamed key, a removed echo — say so plainly.

3. DEPLOY REALITY. Is what is on main actually what is deployed? Check the
   version the server reports against the version in source. Check that any
   docs/changelog surface states the same version. Mismatch here is the failure
   mode that ships silently.

4. UNATTENDED SAFETY. This runs without a human watching. Flag anything that
   could fail silently: a guard that cannot fire, a test that passes by not
   running, an alert that reports state rather than transitions, a threshold
   tuned to today's data, or a cache/TTL that could serve stale prices.

5. THE ONE THING most likely to cause a production incident in the next week,
   with the specific file and reason.

Ground rules in this codebase: never raise memory/CPU/timeout limits as a fix;
bounded partial progress exits 0; CI is billing-blocked so local gates are
authoritative; tests here have silently read production data from \$HOME/data
and silently skipped on missing venv/import — treat a green run as suspect
until you know it executed.

Be specific, cite file:line, rank by severity, skip praise. If everything is
genuinely fine, say so briefly rather than manufacturing findings."

cd "$REPO" || exit 1
# grok CLI (not cursor-agent): it has tool use in single-turn -p mode and
# actually reads/runs against the checkout, which is the whole point of a
# shipping gate — PR bodies are not evidence.
timeout 3600 grok -m grok-4.6 --effort high -p "$PROMPT" > "$OUT" 2>&1
echo "exit $?" >> "$OUT"
echo "written: $OUT"
