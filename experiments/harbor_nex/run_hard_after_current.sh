#!/usr/bin/env bash
# Wait for grok-xhigh to finish, then launch grok-xhigh-hard detached.
# Do not pkill/pgrep this script's name.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
RESULT="$ROOT/experiments/harbor_nex/jobs/grok-xhigh/result.json"
LOG="$ROOT/experiments/harbor_nex/grok-xhigh-hard-wait.log"
ENV_FILE="/home/evan/Documents/eval_tasks/.env"

cd "$ROOT"

while true; do
  if [[ -f "$RESULT" ]]; then
    finished="$(python3 -c "import json,sys; d=json.load(open(sys.argv[1])); print(d.get('finished_at') or '')" "$RESULT")"
    if [[ -n "$finished" && "$finished" != "None" ]]; then
      echo "$(date -Is) grok-xhigh finished_at=$finished" >>"$LOG"
      break
    fi
  fi
  echo "$(date -Is) waiting for grok-xhigh finished_at" >>"$LOG"
  sleep 120
done

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

if [[ -z "${CURSOR_API_KEY:-}" ]]; then
  echo "$(date -Is) CURSOR_API_KEY missing after sourcing env" >>"$LOG"
  exit 1
fi

setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_hard \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 3 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-hard \
  --yes \
  >>"$ROOT/experiments/harbor_nex/grok-xhigh-hard-run.log" 2>&1 &

echo "$(date -Is) launched grok-xhigh-hard pid=$!" >>"$LOG"
