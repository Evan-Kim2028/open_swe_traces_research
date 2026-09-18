#!/usr/bin/env bash
# Launch grok-xhigh-iter9. Do not pkill this script by name.
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
  --path experiments/harbor_nex/tasks_iter9 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter9 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter9.log 2>&1 < /dev/null &
echo "launched pid $! job_dir experiments/harbor_nex/jobs/grok-xhigh-iter9"
