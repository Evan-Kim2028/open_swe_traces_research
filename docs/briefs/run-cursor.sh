#!/usr/bin/env bash
# run-cursor.sh <brief.md> <outfile> [model] -- launch a cursor-agent run.
# Brief is read into a var, never inlined, to avoid the nested-quoting failure.
set -u
B="$1"; O="$2"; M="${3:-composer-2.5}"
[ -s "$B" ] || { echo "brief missing or empty: $B" > "$O"; exit 1; }
cd /home/evan/Documents
Q="$(cat "$B")"
timeout 7200 cursor-agent -p -f --model "$M" "$Q" > "$O" 2>&1
echo "exit $?" >> "$O"
