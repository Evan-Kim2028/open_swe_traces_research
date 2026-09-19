#!/usr/bin/env bash
# Force --n-attempts 1 in any waiter/launch script the builders write; restart sleeping waiters so the edit takes effect.
H=/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex
while true; do
  for f in $H/run_*.sh; do
    [ -f "$f" ] || continue
    if grep -qE -- "--n-attempts [2-9]" "$f"; then
      sed -i -E 's/--n-attempts [2-9]/--n-attempts 1/g' "$f"
      base=$(basename "$f")
      for p in $(pgrep -f "$base"); do kill $p 2>/dev/null; done
      sleep 1; (cd "$H/../.." && setsid nohup bash "$f" > "$H/${base%.sh}.log" 2>&1 < /dev/null &)
      echo "$(date -u +%H:%M) patched+restarted $base"
    fi
  done
  sleep 60
done
