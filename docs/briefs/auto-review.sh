#!/usr/bin/env bash
# auto-review.sh -- fire a grok batch review when enough unreviewed PRs pile up.
#
# Reviews used to happen only when a human remembered. This watches merge
# activity and triggers on accumulation, which is also the right granularity:
# cluster review finds cross-PR interactions, and those only exist once there
# are several PRs to interact. Today's batch of 14 found 8 issues that every
# individual review had missed, including two critical ones.
set -u
D=/home/evan/devin-tasks
STATE="$D/.auto_review_state"
THRESHOLD="${AUTO_REVIEW_THRESHOLD:-4}"   # PRs merged since last review
MAX_WAIT_H="${AUTO_REVIEW_MAX_WAIT_H:-6}" # ...or this long with >=1 PR
POLL="${AUTO_REVIEW_POLL:-600}"
REPOS="Evan-Kim2028/lake-of-rage Evan-Kim2028/silphcoanalytics"
mkdir -p "$D"

last_for() { grep -E "^$1 " "$STATE" 2>/dev/null | tail -1 | awk '{print $2}'; }
set_for()  { local r="$1" v="$2"; grep -vE "^$r " "$STATE" 2>/dev/null > "$STATE.tmp" || true;
             echo "$r $v" >> "$STATE.tmp"; mv "$STATE.tmp" "$STATE"; }

while true; do
  for repo in $REPOS; do
    last=$(last_for "$repo"); last="${last:-0}"
    now=$(date +%s)
    # count merges since the last review watermark
    n=0
    while read -r at; do
      [ -z "$at" ] && continue
      e=$(date -d "$at" +%s 2>/dev/null) || continue
      [ "$e" -gt "$last" ] && n=$((n+1))
    done < <(gh pr list -R "$repo" --state merged --limit 40 --json mergedAt -q '.[].mergedAt' 2>/dev/null)

    age_h=$(( (now - last) / 3600 ))
    fire=0
    [ "$n" -ge "$THRESHOLD" ] && fire=1
    [ "$n" -ge 1 ] && [ "$last" -gt 0 ] && [ "$age_h" -ge "$MAX_WAIT_H" ] && fire=1

    if [ "$fire" -eq 1 ]; then
      echo "$(date -Is) [auto-review] $repo: $n unreviewed merges -> firing batch review"
      hours=$(( age_h > 0 ? age_h + 1 : 12 ))
      bash "$D/grok-batch-review.sh" "$hours" "$repo" >> "$D/auto-review.log" 2>&1 || true
      set_for "$repo" "$now"
    fi
  done
  sleep "$POLL"
done
