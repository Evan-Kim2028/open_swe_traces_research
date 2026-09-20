#!/usr/bin/env bash
# Prune regenerable source trees ONLY under real disk pressure, oldest-first, never
# touching a batch we still owe trials on.
#
# The old rule -- "prune any worktree with no running job, every 30 minutes" -- bit three
# times in one day (nats-server, go-git, client-go/goa), each time deleting sources for a
# batch that was about to be escalated. Regeneration recovered every one, so nothing was
# lost permanently, but it cost ~40 minutes of wall clock for disk we were not short of.
#
# Policy:
#   - do nothing while free space is above THRESHOLD_GB (default 100)
#   - never prune a worktree with a running agent session
#   - never prune a batch that still has unescalated L0-hard units
#   - prune least-recently-modified first, and stop as soon as we are back over threshold
#   - never touch ladder-base images or experiments/pipeline/repos/*/src
set -u
R=/home/evan/Documents/open_swe_traces_research
THRESHOLD_GB="${THRESHOLD_GB:-100}"
free_gb() { df -BG --output=avail / | tail -1 | tr -dc '0-9'; }

avail=$(free_gb)
if [ "$avail" -ge "$THRESHOLD_GB" ]; then
  echo "prune: ${avail}G free, threshold ${THRESHOLD_GB}G — nothing to do"
  exit 0
fi
echo "prune: ${avail}G free, below ${THRESHOLD_GB}G — pruning oldest first"

# batches we still owe trials on: any family that failed L0 and has no L2 verdict
owed=$(cd "$R" && uv run python - <<'PY'
import json
from collections import defaultdict
from pathlib import Path
res = defaultdict(dict)
for vf in Path("experiments/dose_response/jobs").glob("*/verdicts.json"):
    for u, v in json.loads(vf.read_text()).items():
        if "-L" not in u:
            continue
        f, _, lv = u.rpartition("-L")
        r = lv[:1]
        cur = res[f].get(r)
        if cur is None or cur == "UNKNOWN" or (cur == "unsolved" and v["verdict"] == "solved"):
            res[f][r] = v["verdict"]
owed = {f for f, lv in res.items() if lv.get("0") == "unsolved" and "2" not in lv}
# emit every stem spelling, since repo prefixes contain hyphens
out = set()
for f in owed:
    parts = f.split("-")
    out.add(f)
    out.update("-".join(parts[i:]) for i in range(1, len(parts)))
print(" ".join(sorted(out)))
PY
)

for d in $(ls -dt /home/evan/Documents/oswt-* 2>/dev/null | tac); do
  name=$(basename "$d")
  pgrep -f "closure_${name#oswt-}.md" >/dev/null 2>&1 && { echo "  skip $name (session running)"; continue; }
  skip=0
  for stem in $owed; do
    [ -d "$d/experiments/pipeline" ] || break
    if compgen -G "$d/experiments/pipeline/tasks*/*/${stem}-L2" >/dev/null 2>&1; then
      echo "  skip $name (owes trials on $stem)"; skip=1; break
    fi
  done
  [ "$skip" -eq 1 ] && continue
  n=$(find "$d" -type d -path '*/environment/src' 2>/dev/null | wc -l)
  [ "$n" -eq 0 ] && continue
  find "$d" -type d -path '*/environment/src' -prune -exec rm -rf {} + 2>/dev/null
  echo "  pruned $name ($n trees), now $(free_gb)G free"
  [ "$(free_gb)" -ge "$THRESHOLD_GB" ] && { echo "prune: back over threshold, stopping"; break; }
done
