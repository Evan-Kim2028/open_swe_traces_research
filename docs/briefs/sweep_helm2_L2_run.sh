#!/usr/bin/env bash
# Escalate the 10 batch-2 helm units that failed L0, at L2, k=3 (escalation policy).
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_helm2_L2 --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_helm2_L2 --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" sweep_helm2_L2
