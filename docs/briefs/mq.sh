#!/usr/bin/env bash
# mq.sh -- deepseek (cmd) queue runner. Third lane beside dq.sh (devin) and
# cq.sh (cursor). Separate queue dirs so no two runners can claim one brief.
#
# Job counting uses pidfiles, never `ps | grep <model>`: that self-matches the
# grep AND counts unrelated work on this box. Both bugs stalled cq.sh at
# live=7/MAXN=3 earlier today.
set -u
D=/home/evan/devin-tasks; Q=$D/mqueue; R=$D/mrunning; DONE=$D/mdone; FAILED=$D/mfailed
PIDDIR=$D/mqpids; HB=$D/mq.heartbeat; RES=$D/mq.results.tsv; HALT=$D/MQ_HALTED
MAXN="${MQ_MAXN:-2}"; POLL="${MQ_POLL:-60}"; BREAK_AFTER="${MQ_BREAK_AFTER:-3}"
mkdir -p "$Q" "$R" "$DONE" "$FAILED" "$PIDDIR"
[ -f "$RES" ] || printf 'ts\tname\tdur_s\texit\tverdict\tdetail\n' > "$RES"

live() {
  local n=0 f pid
  for f in "$PIDDIR"/*.pid; do
    [ -f "$f" ] || continue
    pid=$(cat "$f" 2>/dev/null)
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then n=$((n+1)); else rm -f "$f"; fi
  done
  echo "$n"
}

consecutive_bad() { tail -n 6 "$RES" | awk -F'\t' 'NR>0{print $5}' | tac | awk '/^OK$/{exit} /^(RATE_LIMIT|AUTH|FAST_FAIL|ERROR|NO_PR)$/{c++} END{print c+0}'; }

launch() {
  local brief="$1" name; name=$(basename "$brief" .md)
  mv "$brief" "$R/$name.md" 2>/dev/null || return 1
  setsid nohup bash -c "
    echo \$\$ > '$PIDDIR/$name.pid'
    s=\$(date +%s)
    bash $D/run-cmd.sh '$R/$name.md' '$D/$name.mq.out'
    rc=\$(grep -oE '^exit [0-9]+' '$D/$name.mq.out' 2>/dev/null | tail -1 | awk '{print \$2}')
    e=\$(date +%s); dur=\$((e-s))
    v=\$($D/cq-classify.sh '$D/$name.mq.out' \$dur \${rc:-1})
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \"\$(date -Is)\" '$name' \"\$dur\" \"\${rc:-1}\" \"\$v\" \"\$(grep -oE 'https://github.com/[^ ]+/pull/[0-9]+' '$D/$name.mq.out' | tail -1)\" >> $RES
    if [ \"\$v\" = OK ]; then mv '$R/$name.md' '$DONE/$name.md' 2>/dev/null; else mv '$R/$name.md' '$FAILED/$name.md' 2>/dev/null; fi
    rm -f '$PIDDIR/$name.pid'
  " >/dev/null 2>&1 </dev/null &
  echo "[mq] launched $name (deepseek-v4.1-flash)"
}

echo "[mq] runner start maxn=$MAXN poll=${POLL}s"
while true; do
  bad=$(consecutive_bad)
  printf 'ts=%s live=%s queued=%s running=%s consec_bad=%s halted=%s\n' \
    "$(date -Is)" "$(live)" "$(ls "$Q" 2>/dev/null|wc -l)" "$(ls "$R" 2>/dev/null|wc -l)" \
    "$bad" "$([ -f "$HALT" ] && echo yes || echo no)" > "$HB"
  [ "$bad" -ge "$BREAK_AFTER" ] && [ ! -f "$HALT" ] && \
    echo "[mq] CIRCUIT BREAKER: $bad consecutive non-OK. Investigate $RES then rm $HALT" > "$HALT"
  if [ ! -f "$HALT" ]; then
    while [ "$(live)" -lt "$MAXN" ]; do
      nxt=$(ls "$Q"/*.md 2>/dev/null | sort | head -1); [ -n "$nxt" ] || break
      launch "$nxt" || break
      sleep 8
    done
  fi
  sleep "$POLL"
done
