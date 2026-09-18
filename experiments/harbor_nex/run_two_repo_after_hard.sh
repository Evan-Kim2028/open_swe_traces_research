#!/usr/bin/env bash
# Wait for grok-xhigh-hard (or grok-xhigh fallback), then launch the two-repo Harbor job.
# Do not pkill this script by name.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

HARD_RESULT="experiments/harbor_nex/jobs/grok-xhigh-hard/result.json"
XHIGH_RESULT="experiments/harbor_nex/jobs/grok-xhigh/result.json"
LOG="experiments/harbor_nex/run_two_repo_after_hard.log"
JOB_LOG="experiments/harbor_nex/grok-xhigh-two-repo.log"
DEADLINE_SECS=$((6 * 60 * 60))
POLL_SECS=120

finished_at_set() {
  local path=$1
  [[ -f "$path" ]] || return 1
  python3 - "$path" <<'PY'
import json, sys
p = sys.argv[1]
try:
    data = json.load(open(p))
except Exception:
    sys.exit(1)
val = data.get("finished_at")
sys.exit(0 if val not in (None, "", 0) else 1)
PY
}

wait_for() {
  local path=$1
  while true; do
    if finished_at_set "$path"; then
      echo "$(date -Is) ready: $path" | tee -a "$LOG"
      return 0
    fi
    echo "$(date -Is) waiting for finished_at in $path" | tee -a "$LOG"
    sleep "$POLL_SECS"
  done
}

mkdir -p experiments/harbor_nex
echo "$(date -Is) two-repo waiter start" | tee -a "$LOG"

start_ts=$(date +%s)
hard_seen=0
while true; do
  now=$(date +%s)
  elapsed=$((now - start_ts))
  if [[ -e "$HARD_RESULT" || -d "experiments/harbor_nex/jobs/grok-xhigh-hard" ]]; then
    hard_seen=1
  fi
  if finished_at_set "$HARD_RESULT"; then
    echo "$(date -Is) grok-xhigh-hard finished" | tee -a "$LOG"
    break
  fi
  if (( elapsed >= DEADLINE_SECS )) && (( hard_seen == 0 )); then
    echo "$(date -Is) grok-xhigh-hard did not appear within 6h; waiting for grok-xhigh" | tee -a "$LOG"
    wait_for "$XHIGH_RESULT"
    break
  fi
  echo "$(date -Is) waiting for grok-xhigh-hard (elapsed ${elapsed}s)" | tee -a "$LOG"
  sleep "$POLL_SECS"
done

ENV_FILE="/home/evan/Documents/eval_tasks/.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing $ENV_FILE" | tee -a "$LOG"
  exit 1
fi
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a
if [[ -z "${CURSOR_API_KEY:-}" ]]; then
  echo "CURSOR_API_KEY not set after sourcing env" | tee -a "$LOG"
  exit 1
fi

echo "$(date -Is) launching harbor grok-xhigh-two-repo" | tee -a "$LOG"
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_two_repo \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-two-repo \
  --yes \
  >> "$JOB_LOG" 2>&1 &
echo "$(date -Is) harbor pid $!" | tee -a "$LOG"
