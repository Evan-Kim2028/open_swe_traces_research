#!/usr/bin/env bash
# cq.sh -- cursor-agent (composer-2.5) queue runner. Parallel capacity beside
# dq.sh, which is capped at 3 devin sessions. Separate queue/running dirs so the
# two runners can never claim the same brief.
set -u
D=/home/evan/devin-tasks; Q=$D/cqueue; R=$D/crunning; DONE=$D/cdone; FAILED=$D/cfailed
HB=$D/cq.heartbeat; RES=$D/cq.results.tsv; HALT=$D/CQ_HALTED
MAXN="${CQ_MAXN:-3}"; POLL="${CQ_POLL:-60}"; BREAK_AFTER="${CQ_BREAK_AFTER:-3}"
mkdir -p "$Q" "$R" "$DONE" "$FAILED"
[ -f "$RES" ] || printf 'ts\tname\tdur_s\texit\tverdict\tdetail\n' > "$RES"

# Count OUR jobs via pidfiles, never `ps | grep cursor-agent`: this box runs
# unrelated cursor-agent work, and grepping the process table both counts those
# and self-matches the grep. Both bugs stalled this queue at live=7/MAXN=3.
live() {
  local n=0 f pid
  for f in /home/evan/devin-tasks/cqpids/*.pid; do
    [ -f "$f" ] || continue
    pid=$(cat "$f" 2>/dev/null)
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then n=$((n+1)); else rm -f "$f"; fi
  done
  echo "$n"
}

classify() { # <out> <dur> <rc>
  local out="$1" dur="$2" rc="$3"
  grep -qiE 'rate.?limit|too many requests|quota exceeded|overloaded' "$out" 2>/dev/null && { echo RATE_LIMIT; return; }
  grep -qiE 'unauthorized|not logged in|invalid api key' "$out" 2>/dev/null && { echo AUTH; return; }
  grep -qE 'https://github\.com/[^ ]+/pull/[0-9]+|NO PR NEEDED|MERGED [0-9a-f]{7}' "$out" 2>/dev/null && { echo OK; return; }
  [ "$dur" -lt 120 ] && { echo FAST_FAIL; return; }
  [ "$rc" -ne 0 ] && { echo ERROR; return; }
  echo NO_PR
}

consecutive_bad() { tail -n 6 "$RES" | awk -F'\t' 'NR>0{print $5}' | tac | awk '/^OK$/{exit} /^(RATE_LIMIT|AUTH|FAST_FAIL|ERROR|NO_PR)$/{c++} END{print c+0}'; }

launch() {
  local brief="$1" name; name=$(basename "$brief" .md)
  mv "$brief" "$R/$name.md" 2>/dev/null || return 1
  # Repo is per-brief: a brief states its own checkout on a "Repo:" line.
  # Hardcoding lake-of-rage silently ran silphco work in the wrong tree.
  local src wt
  src=$(grep -m1 -oE '/home/evan/(Documents/)?[a-z-]+' "$R/$name.md" 2>/dev/null | head -1)
  case "$src" in
    */silphcoanalytics) src=/home/evan/Documents/silphcoanalytics ;;
    *) src=/home/evan/lake-of-rage ;;
  esac
  wt="$D/wt/$name"
  mkdir -p "$D/wt"; rm -rf "$wt"
  git -C "$src" worktree add -q --detach "$wt" origin/main 2>/dev/null || wt="$src"
  setsid nohup bash -c "
    s=\$(date +%s)
    /home/evan/.local/bin/cq-grunt '$R/$name.md' '$wt' '$D/$name.cq.out' >/dev/null 2>&1
    while ! grep -q '^exit ' '$D/$name.cq.out' 2>/dev/null; do sleep 20; done
    rc=\$(grep -oE '^exit [0-9]+' '$D/$name.cq.out' | tail -1 | awk '{print \$2}')
    e=\$(date +%s); dur=\$((e-s))
    v=\$($D/cq-classify.sh '$D/$name.cq.out' \$dur \${rc:-1})
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \"\$(date -Is)\" '$name' \"\$dur\" \"\${rc:-1}\" \"\$v\" \"\$(grep -oE 'https://github.com/[^ ]+/pull/[0-9]+' '$D/$name.cq.out' | tail -1)\" >> $RES
    if [ \"\$v\" = OK ]; then mv '$R/$name.md' '$DONE/$name.md' 2>/dev/null; else mv '$R/$name.md' '$FAILED/$name.md' 2>/dev/null; fi
  " >/dev/null 2>&1 </dev/null &
  echo "[cq] launched $name (composer-2.5) wt=$wt"
}

echo "[cq] runner start maxn=$MAXN poll=${POLL}s"
while true; do
  bad=$(consecutive_bad)
  printf 'ts=%s live=%s queued=%s running=%s consec_bad=%s halted=%s\n' \
    "$(date -Is)" "$(live)" "$(ls "$Q" 2>/dev/null | wc -l)" "$(ls "$R" 2>/dev/null | wc -l)" \
    "$bad" "$([ -f "$HALT" ] && echo yes || echo no)" > "$HB"
  if [ "$bad" -ge "$BREAK_AFTER" ] && [ ! -f "$HALT" ]; then
    echo "[cq] CIRCUIT BREAKER: $bad consecutive non-OK. Investigate $RES then rm $HALT" > "$HALT"
  fi
  if [ ! -f "$HALT" ]; then
    while [ "$(live)" -lt "$MAXN" ]; do
      nxt=$(ls "$Q"/*.md 2>/dev/null | sort | head -1); [ -n "$nxt" ] || break
      launch "$nxt" || break
      sleep 10
    done
  fi
  sleep "$POLL"
done
