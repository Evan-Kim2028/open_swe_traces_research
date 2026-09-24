#!/usr/bin/env bash
# L0 screen, batch 2: the 20 repaired client-go + goa units, k=1 (screening policy), Composer 2.5.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_L0_batch2 --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_L0_batch2 --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" sweep_L0_batch2
