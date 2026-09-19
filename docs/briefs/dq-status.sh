#!/usr/bin/env bash
D=/home/evan/devin-tasks
echo "== heartbeat =="; if [ -f "$D/dq.heartbeat" ]; then cat "$D/dq.heartbeat"; age=$(( $(date +%s) - $(stat -c %Y "$D/dq.heartbeat") )); echo "heartbeat_age=${age}s $([ $age -gt 180 ] && echo '>>> STALE: runner may be dead')"; else echo "NO HEARTBEAT — runner never started"; fi
echo "== runner alive =="; c=$(ls -d /proc/[0-9]* 2>/dev/null | while read -r d; do tr "\0" " " < "$d/cmdline" 2>/dev/null | grep -q 'devin-tasks/dq.sh' && echo x; done | wc -l); echo "processes=$c $([ "$c" -eq 0 ] && echo '>>> DEAD: restart with setsid nohup ~/devin-tasks/dq.sh > ~/devin-tasks/dq.log 2>&1 &')"
[ -f "$D/DQ_HALTED" ] && { echo "== HALTED =="; cat "$D/DQ_HALTED"; }
echo "== last 8 outcomes =="; [ -f "$D/dq.results.tsv" ] && tail -8 "$D/dq.results.tsv" | column -t -s$'\t' 2>/dev/null || echo "(none yet)"
echo "== outcome tally =="; [ -f "$D/dq.results.tsv" ] && tail -n +2 "$D/dq.results.tsv" | awk -F'\t' '{c[$6]++} END{for(k in c) printf "  %-12s %s\n", k, c[k]}'
echo "== queue =="; ls "$D/queue" 2>/dev/null | tr '\n' ' '; echo; echo "== failed briefs =="; ls "$D/failed" 2>/dev/null | tr '\n' ' '; echo
