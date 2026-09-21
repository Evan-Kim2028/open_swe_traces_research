#!/usr/bin/env bash
# Adaptive climb on the 4 L2 non-flippers. Arg = rung dir (sweep_climb_L3 / _L5). k=3 (escalation policy).
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:-sweep_climb_L3}"
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name "$D" --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" "$D"
