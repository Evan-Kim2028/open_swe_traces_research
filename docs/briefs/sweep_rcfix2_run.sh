#!/usr/bin/env bash
# Reconciled-contract test: helm-repindex L2 with a contract derived from the hidden assertions.
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:-sweep_rcfix2}"
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 8 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name "$D" --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" "$D"
