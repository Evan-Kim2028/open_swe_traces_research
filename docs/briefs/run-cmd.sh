#!/usr/bin/env bash
# run-cmd.sh <brief.md> <outfile> -- launch a cmd/deepseek agent without nested-quoting hazards.
set -u
B="$1"; O="$2"
[ -s "$B" ] || { echo "brief missing or empty: $B" > "$O"; exit 1; }
cd /home/evan/Documents
Q="$(cat "$B")"
# --max-turns 200 was too low: an architecture brief hit the cap MID-COMMIT and
# exited 8 with ~2h of finished-but-unlanded work sitting in the tree. The cap
# is a runaway guard, not a budget — 600 still bounds a loop while letting real
# multi-file work finish. Timeout raised to match (200 turns of tool use does
# not fit in 90m).
timeout "${CMD_TIMEOUT_S:-10800}" cmd -p "$Q" -m deepseek/deepseek-v4.1-flash --effort max --yolo --tools-all --skip-onboarding -t --max-turns "${CMD_MAX_TURNS:-600}" > "$O" 2>&1
echo "exit $?" >> "$O"
