#!/usr/bin/env bash
# Cross-repo closures at L0, k=1: first solve data on whether a dependency boundary adds difficulty.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_xrepo --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 2 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_xrepo --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" sweep_xrepo
