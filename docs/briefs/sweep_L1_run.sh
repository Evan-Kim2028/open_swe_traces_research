#!/usr/bin/env bash
# L1 gapped-contract causal test: 16 dirs (8 units x {L1binding, L1other}), k=1.
# Holds the L2 contract constant and deletes exactly one invariant, so a flip
# attributes the L0->L2 gain to a specific sentence rather than to prose volume.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_L1 --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 4 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_L1 --yes
"$R/scripts/ops/post_sweep.sh" sweep_L1
