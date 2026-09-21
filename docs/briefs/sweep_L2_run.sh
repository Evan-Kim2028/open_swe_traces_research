#!/usr/bin/env bash
# L1/L2 escalation: L0 -> L1 (gapped contract, one invariant removed) -> L2 (full contract). k=3 on escalation.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_L2 --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_L2 --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" sweep_L2
