#!/bin/bash
R=/home/evan/Documents/open_swe_traces_research; P=$R/experiments/pipeline
w="solve""-watch --interval"
if pgrep -f "$w" >/dev/null; then echo "watch already running"; exit 0; fi
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a && nohup setsid uv run python scripts/pipeline.py solve-watch --interval 180 --host laptop > $P/logs/solve_watch_stdout.log 2>&1 &
echo "started"
