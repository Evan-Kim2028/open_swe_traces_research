#!/usr/bin/env bash
# Kill cursor-agent processes (composer first, then grok) if load stays > 2x cores for 5 min or free mem < 2 GB.
cores=$(nproc); bad=0
while true; do
  load=$(cut -d' ' -f1 /proc/loadavg | cut -d. -f1); freegb=$(free -g | awk 'NR==2{print $7}')
  if [ "$load" -gt $((cores*4)) ] || [ "$freegb" -lt 2 ]; then bad=$((bad+1)); else bad=0; fi
  if [ "$bad" -ge 10 ]; then
    echo "$(date -u +%H:%M) watchdog: load=$load free=${freegb}G -> killing composer agents"
    for p in $(pgrep -f "model composer""-2.5"); do pkill -TERM -P $p 2>/dev/null; kill $p 2>/dev/null; done; bad=0
    sleep 60; load=$(cut -d' ' -f1 /proc/loadavg | cut -d. -f1); [ "$load" -gt $((cores*4)) ] && for p in $(pgrep -f "cursor-grok""-4.6"); do pkill -TERM -P $p 2>/dev/null; kill $p 2>/dev/null; done
  fi
  sleep 60
done
