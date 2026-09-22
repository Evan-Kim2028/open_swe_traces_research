#!/usr/bin/env bash
# Periodic heartbeat. watch.sh prints only on CHANGE or ALARM, so a quiet log is the
# system being fine -- but it has to actually be RUNNING for that to mean anything, and an
# absence of output from a loop nobody started is indistinguishable from an absence of
# problems. That is why this exists as a supervised loop with a timestamped tick rather
# than something remembered and re-run by hand.
#
# Usage: monitor_loop.sh [interval_sec]   (default 300)
set -u
R=/home/evan/Documents/open_swe_traces_research
INT="${1:-300}"
LOG="$R/outputs/supervisor/watch.log"
cd "$R"
while true; do
  {
    echo "--- $(date '+%Y-%m-%d %H:%M:%S')"
    timeout 600 bash scripts/ops/watch.sh 2>&1
  } >> "$LOG"
  sleep "$INT"
done
