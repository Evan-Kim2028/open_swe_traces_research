#!/usr/bin/env bash
# Push arms one at a time whenever fewer than 2 Kaggle batch GPU sessions are active.
# Kaggle rejects a 3rd concurrent push ("Maximum batch GPU session count of 2 reached")
# instead of queueing, so this loop does the queueing. Usage: nohup ./queue_arms.sh arm1 arm2 &
set -uo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"; ROOT="$(cd "$HERE/../.." && pwd)"
KG="uv run --project $ROOT kaggle"
ALL=(random top_within_task bottom_within_task random_masked)
PENDING=("$@")
slug() { echo "evandekim/openswe-curve-${1//_/-}"; }
active() { local n=0; for a in "${ALL[@]}"; do s=$($KG kernels status "$(slug "$a")" 2>/dev/null || true); echo "$s" | grep -qE "RUNNING|QUEUED|STARTING" && n=$((n+1)); done; echo $n; }
while ((${#PENDING[@]})); do
  n=$(active); echo "$(date -u +%H:%M) active=$n pending=${PENDING[*]}"
  if (( n < 2 )); then
    a=${PENDING[0]}; d="$HERE/arms/$a"; mkdir -p "$d"
    cp "$HERE/train_curve.py" "$d/train_curve.py"
    sed -i "s/\"MANIFEST\": \"random\"/\"MANIFEST\": \"$a\"/" "$d/train_curve.py"
    out=$($KG kernels push -p "$d" -t 43200 2>&1); echo "$out" | tail -1
    if echo "$out" | grep -q "successfully pushed"; then PENDING=("${PENDING[@]:1}"); sleep 120; continue; fi
  fi
  sleep 600
done
echo "$(date -u +%H:%M) all arms pushed"
