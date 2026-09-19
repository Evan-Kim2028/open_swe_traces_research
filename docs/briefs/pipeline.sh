#!/usr/bin/env bash
# Orchestration pipeline: run waves of briefs two at a time on or-grunt.
# A wave starts only when every task of the previous wave has finished AND its PR is MERGED/CLOSED
# (reviewer gate), or a release marker /tmp/lor-review/.<id>.ok exists.
D=/home/evan/devin-tasks; W=/home/evan/Documents/lake-of-rage; R=Evan-Kim2028/lake-of-rage
log(){ echo "$(date -Is) $*"; }
pr_of(){ grep -oE 'https://github.com/[^ )]+/pull/[0-9]+' "$D/$1.out" 2>/dev/null | tail -1 | grep -oE '[0-9]+$'; }
wait_task(){ id=$1; until grep -q '^exit' "$D/$id.out" 2>/dev/null; do sleep 60; done; log "$id finished: PR #$(pr_of $id)"; }
wait_gate(){ id=$1; while true; do [ -f /tmp/lor-review/.$id.ok ] && { log "$id released by marker"; return; }
  n=$(pr_of $id); if [ -n "$n" ]; then st=$(gh pr view $n -R $R --json state --jq .state 2>/dev/null); [ "$st" = MERGED ] || [ "$st" = CLOSED ] && { log "$id gate open (PR #$n $st)"; return; }; fi; sleep 90; done; }
run_wave(){ log "WAVE START: $*"; for id in "$@"; do : > "$D/$id.out"; /home/evan/.local/bin/or-grunt "$D/$id.md" "$W" "$D/$id.out"; done
  for id in "$@"; do wait_task $id; done; for id in "$@"; do wait_gate $id; done; log "WAVE DONE: $*"; }
# wave 1 already running (t8/t9): just gate on them
for id in t8-product-overrides t9-parity-plan; do wait_task $id; done; for id in t8-product-overrides t9-parity-plan; do wait_gate $id; done
run_wave w2-2840 w2-2782
run_wave w3-2839 w3-1239
run_wave w4-2837 w4-2841
run_wave w5-2836 w5-2838
log "PIPELINE DONE"
