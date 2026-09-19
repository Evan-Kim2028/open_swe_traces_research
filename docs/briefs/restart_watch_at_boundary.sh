#!/bin/bash
# Restart solve-watch (to pick up new code) once the current unit's solve-unit reports done.
R=/home/evan/Documents/open_swe_traces_research; P=$R/experiments/pipeline
pat="solve""-watch"
start=$(wc -l < $P/logs/solve_watch.log)
while ! tail -n +$start $P/logs/solve_watch.log | grep -q "solve-unit done\|solve-unit error"; do sleep 60; done
sleep 5
pkill -f "$pat --interval" ; sleep 3
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a && nohup setsid uv run python scripts/pipeline.py solve-watch --interval 180 --host laptop > $P/logs/solve_watch_stdout.log 2>&1 &
echo "$(date -u +%H:%M) restarted solve-watch after unit boundary" >> $P/logs/restart_watch.log
