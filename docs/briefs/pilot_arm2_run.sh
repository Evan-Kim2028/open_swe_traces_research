#!/usr/bin/env bash
# Pilot arm 2: four real excision units at L0, Composer 2.5, one attempt each. ~$0.40.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
exec harbor run --path experiments/dose_response/pilot --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 2 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name pilot_arm2 --yes
