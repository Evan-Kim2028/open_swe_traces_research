#!/usr/bin/env bash
D=/home/evan/devin-tasks; R=/home/evan/Documents/open_swe_traces_research; P=$R/experiments/pipeline
sleep 1200
cd $R
for r in clientgo helm gin; do
  nohup timeout 10800 devin -p "$(cat $D/devin_verifier_$r.md) RESUME NOTE: a previous session was cut off by a rate limit; reuse any suites, task dirs and validation.json already present and finish the remaining units." --model swe-2-max --permission-mode dangerous > $D/devin_verifier_$r.log 2>&1 &
  sleep 180
done
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
PY="scripts/pipeline.py"
nohup setsid uv run python $PY dry-run --repo nats-server --units 5 > $P/logs/dryrun_nats.log 2>&1 < /dev/null &
echo "$(date -u +%H:%M) relaunched 3 verifiers + dry run"
