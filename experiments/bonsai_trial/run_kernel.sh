#!/usr/bin/env bash
# Push a kernel, poll until it finishes, pull the results.
# The KERNEL id and timeout are the two levers for "don't waste GPU quota":
#   - a script kernel auto-terminates when the code finishes (no idle burn)
#   - TIMEOUT caps a hung/runaway kernel so it can't eat the whole session
# TIMEOUT defaults to 2700s for bonsai_trial: the script self-limits to DEADLINE_S=2400
# so bonsai_metrics.json is always written before the platform kill-switch fires.
#
# Usage: ./run_kernel.sh [DIR]                 # DIR holds kernel-metadata.json (default: script dir)
#        POLL=30 TIMEOUT=3600 ./run_kernel.sh  # tighter poll / different cap
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
DIR="$(cd "${1:-$HERE}" && pwd)"        # folder containing kernel-metadata.json + code
RUN="uv run --project /home/evan/Documents/kaggle_e4b kaggle"     # uv-managed kaggle CLI
OUTDIR="$DIR/out"
TIMEOUT="${TIMEOUT:-2700}"              # seconds; script self-limits at DEADLINE_S=2400
POLL="${POLL:-60}"                      # seconds between status checks
MAXWAIT="${MAXWAIT:-3000}"              # stop polling after this many seconds (50 min)

# Read the kernel id straight from the metadata so the two never drift.
KERNEL="$(python3 -c "import json;print(json.load(open('$DIR/kernel-metadata.json'))['id'])")"

echo ">> Pushing kernel ($KERNEL) from $DIR, timeout=${TIMEOUT}s ($((TIMEOUT/60))m) ..."
$RUN kernels push -p "$DIR" -t "$TIMEOUT"

echo ">> Polling status every ${POLL}s (Ctrl-C is safe; the kernel keeps running on Kaggle) ..."
mkdir -p "$OUTDIR"
: >> "$DIR/poll_status.log"
START=$(date +%s)
while true; do
  STATUS="$($RUN kernels status "$KERNEL" 2>/dev/null | tr -d '\r')"
  echo "   $(date +%H:%M:%S)  $STATUS" | tee -a "$DIR/poll_status.log"
  if echo "$STATUS" | grep -qiE "complete|error|cancel"; then
    break
  fi
  if (( $(date +%s) - START > MAXWAIT )); then
    echo ">> poll budget (${MAXWAIT}s) exhausted; pulling whatever exists"
    break
  fi
  sleep "$POLL"
done

echo ">> Downloading output to $OUTDIR ..."
$RUN kernels output "$KERNEL" -p "$OUTDIR"

echo ">> ===== output JSON ====="
shopt -s nullglob
JSONS=("$OUTDIR"/*.json)
if (( ${#JSONS[@]} )); then
  for j in "${JSONS[@]}"; do echo "--- $j ---"; head -c 4000 "$j"; echo; done
else
  echo "No JSON output — dumping tail of the run log:"
  ls -la "$OUTDIR"
  cat "$OUTDIR"/*.log 2>/dev/null | tail -40 || true
fi
