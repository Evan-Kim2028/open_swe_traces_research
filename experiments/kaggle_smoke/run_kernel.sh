#!/usr/bin/env bash
# Push a kernel, poll until it finishes, pull the results.
# The KERNEL id and timeout are the two levers for "don't waste GPU quota":
#   - a script kernel auto-terminates when the code finishes (no idle burn)
#   - TIMEOUT caps a hung/runaway kernel so it can't eat the whole session
# Default TIMEOUT is the platform session cap (12h) so it NEVER kills a legit
# long run; Kaggle clamps to its real max. Tighten it only to fail-fast.
#
# Usage: ./run_kernel.sh [DIR]                 # DIR holds kernel-metadata.json (default: script dir)
#        ./run_kernel.sh experiments/nesting_check
#        TIMEOUT=3600 ./run_kernel.sh          # smoke test: kill if it hangs past 1h
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
DIR="$(cd "${1:-$HERE}" && pwd)"        # folder containing kernel-metadata.json + code
RUN="uv run --project /home/evan/Documents/kaggle_e4b kaggle"     # uv-managed kaggle CLI (project lives at repo root)
OUTDIR="$DIR/out"
TIMEOUT="${TIMEOUT:-43200}"             # seconds; 12h = Kaggle GPU session cap

# Read the kernel id straight from the metadata so the two never drift.
KERNEL="$(python3 -c "import json;print(json.load(open('$DIR/kernel-metadata.json'))['id'])")"

echo ">> Pushing kernel ($KERNEL) from $DIR, timeout=${TIMEOUT}s ($((TIMEOUT/3600))h) ..."
$RUN kernels push -p "$DIR" -t "$TIMEOUT"

echo ">> Polling status (Ctrl-C is safe; the kernel keeps running on Kaggle) ..."
while true; do
  STATUS="$($RUN kernels status "$KERNEL" 2>/dev/null | tr -d '\r')"
  echo "   $(date +%H:%M:%S)  $STATUS"
  if echo "$STATUS" | grep -qiE "complete|error|cancel"; then
    break
  fi
  sleep 20
done

echo ">> Downloading output to $OUTDIR ..."
mkdir -p "$OUTDIR"
$RUN kernels output "$KERNEL" -p "$OUTDIR"

echo ">> ===== output JSON ====="
shopt -s nullglob
JSONS=("$OUTDIR"/*.json)
if (( ${#JSONS[@]} )); then
  for j in "${JSONS[@]}"; do echo "--- $j ---"; cat "$j"; echo; done
else
  echo "No JSON output — dumping tail of the run log:"
  ls -la "$OUTDIR"
  cat "$OUTDIR"/*.log 2>/dev/null | tail -40 || true
fi
