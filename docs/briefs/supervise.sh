#!/usr/bin/env bash
# supervise.sh -- keep every runner alive; restart any that died.
# Runners are plain background bash. A crash or reboot silently ends the
# pipeline, and the only symptom is "nothing is happening" — which is exactly
# the failure you do not notice. Also surfaces a tripped circuit breaker,
# since a halted runner looks identical to an idle one.
set -u
D=/home/evan/devin-tasks
while true; do
  for r in dq cq mq auto-review; do
    [ -f "$D/$r.sh" ] || continue
    if ! pgrep -f "devin-tasks/$r\.sh" >/dev/null 2>&1; then
      echo "$(date -Is) [supervise] $r.sh dead -> restarting"
      setsid nohup bash "$D/$r.sh" >> "$D/$r.log" 2>&1 < /dev/null &
      sleep 3
    fi
  done
  # Devin is retired but dq.sh still generates review briefs into queue/.
  # With DQ_HALTED set those would pile up unlaunched, which looks exactly like
  # "reviews stopped happening" and is the kind of silent stall nobody notices.
  # Rehome them: reviews -> cursor, everything else -> deepseek.
  if [ -f "$D/DQ_HALTED" ]; then
    for f in "$D"/queue/*.md; do
      [ -f "$f" ] || continue
      b=$(basename "$f")
      case "$b" in
        90-review-*|89-review-batch-*) mv "$f" "$D/cqueue/" 2>/dev/null && echo "$(date -Is) [supervise] rehomed $b -> cursor" ;;
        *) mv "$f" "$D/mqueue/" 2>/dev/null && echo "$(date -Is) [supervise] rehomed $b -> deepseek" ;;
      esac
    done
  fi
  for h in CQ_HALTED MQ_HALTED; do
    [ -f "$D/$h" ] && echo "$(date -Is) [supervise] $h is set — launches stopped until removed"
  done
  sleep 120
done
