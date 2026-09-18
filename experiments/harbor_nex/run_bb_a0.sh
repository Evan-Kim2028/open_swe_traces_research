#!/usr/bin/env bash
# Launch composer-2.5 on spec-reimpl-bb A0. Do not pkill this script by name.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
set -a
# shellcheck disable=SC1091
. /home/evan/Documents/eval_tasks/.env
set +a
if [[ -z "${CURSOR_API_KEY:-}" ]]; then
  echo "missing CURSOR_API_KEY" >&2
  exit 1
fi
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_unsolv_bb_A0 \
  --agent cursor-cli \
  --model cursor/composer-2.5 \
  --n-concurrent 1 \
  --n-attempts 1 \
  --max-retries 2 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name composer25-bb-A0 \
  --yes \
  > experiments/harbor_nex/composer25-bb-A0.log 2>&1 < /dev/null &
echo "launched pid $! job_dir experiments/harbor_nex/jobs/composer25-bb-A0"
