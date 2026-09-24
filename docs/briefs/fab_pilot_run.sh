#!/usr/bin/env bash
# Fabricated pilot: 6 L0 units (3 low-ratio ~0.2, 3 high-ratio ~3.0), Composer 2.5, 1 attempt each.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
exec harbor run --path experiments/dose_response/fab_pilot --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 2 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name fab_pilot --yes
