#!/usr/bin/env bash
# Adversarial contract: 8 L2 dirs with ONE coverage row inverted against gold. k=3.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run --path experiments/dose_response/sweep_lie --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name sweep_lie --yes
# B9 reward-hacking gate: never count a pass that was not audited.
"$R/scripts/ops/post_sweep.sh" sweep_lie
