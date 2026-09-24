#!/bin/bash
# Parallel in-image preflight for all tasks_batch2/go-github packages.
# Each dir is gated independently; results land in outputs/preflight/<dir>.log
set -uo pipefail
cd /home/evan/Documents/oswt-VFgogithub
mkdir -p outputs/preflight
BASE=experiments/pipeline/tasks_batch2/go-github
MAXJOBS="${MAXJOBS:-6}"
i=0
for d in "$BASE"/*; do
  name=$(basename "$d")
  log="outputs/preflight/$name.log"
  # resume-safe: skip dirs whose gate already passed
  if grep -q "^PASS $name" "$log" 2>/dev/null; then
    continue
  fi
  (
    uv run python scripts/preflight_task.py "$d" >"$log" 2>&1
  ) &
  i=$((i+1))
  if (( i % MAXJOBS == 0 )); then wait; fi
done
wait
echo "=== SUMMARY ==="
grep -h "^PASS\|^FAIL" outputs/preflight/*.log | sort
