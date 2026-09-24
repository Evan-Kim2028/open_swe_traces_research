#!/usr/bin/env bash
# nats-server L2 escalation, k=3 (escalation policy). Concurrency 6: docker's default
# address pool ran out at 4 sweeps x 12, which errors trials before the agent starts.
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:-sweep_gogit_L2}"
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 6 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name "$D" --yes
"$R/scripts/ops/post_sweep.sh" "$D"
