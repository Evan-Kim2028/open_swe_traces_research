#!/bin/bash
# pilot_arm2_fix — rerun the 3 helm/kops L0 pilot trials after the module-path +
# unexcised-src repair. Exactly 3 Composer trials, n-attempts 1.
# Run:  nohup bash docs/briefs/pilot_arm2_fix.sh > experiments/dose_response/pilot_arm2_fix.log 2>&1 &
set -euo pipefail
cd "$(dirname "$0")/../.."

set -a
# shellcheck source=/dev/null
source /home/evan/Documents/eval_tasks/.env
set +a

exec harbor run \
  --path experiments/dose_response/pilot_fix \
  --agent cursor-cli \
  --model cursor/composer-2.5 \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 1 \
  --jobs-dir experiments/dose_response/jobs \
  --job-name pilot_arm2_fix \
  --yes
