#!/usr/bin/env bash
# Diagnostic: every remaining double-failure at L5 (one hidden test file restored into the tree).
# helm-repindex and kops-clustervalid both go 0/3 at L0-L4 and 3/3 at L5. If the rest do the same,
# no double-failure in the dataset is a capability limit -- they are all specification failures.
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:-sweep_dblL5}"
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 14 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name "$D" --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" "$D"
