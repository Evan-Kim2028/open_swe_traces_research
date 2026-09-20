#!/usr/bin/env bash
# Kill harbor processes whose job dir has stopped growing and that own no containers.
#
# Six finished jobs had live harness processes 8-20 hours later, each holding its docker
# project networks. That is a plausible contributor to the address-pool exhaustion that
# silently zeroed 21 trials.
set -u
R=/home/evan/Documents/open_swe_traces_research
for p in $(pgrep -f '[h]arbor run'); do
  jn=$(tr '\0' ' ' < /proc/$p/cmdline 2>/dev/null | grep -o 'job-name [a-zA-Z0-9_]*' | awk '{print $2}')
  [ -n "$jn" ] || continue
  d="$R/experiments/dose_response/jobs/$jn"
  [ -d "$d" ] || continue
  # a job still working writes into its dir; 45 min of silence means it is finished or wedged
  if [ -z "$(find "$d" -newermt '-45 minutes' 2>/dev/null | head -1)" ]; then
    echo "killing stale harbor: $jn (pid $p, no writes in 45min)"
    kill -9 "$p" 2>/dev/null
  fi
done
