#!/usr/bin/env bash
# Report progress, and put it back on course if it has stopped. Checks, then acts.
#
# A report alone is not enough here: the ways this stalls are all silent. The queue can die and
# leave nothing launching; a cohort sweep can exit with units still runnable; orphaned containers
# can hold quota while returning no verdict. Each reads as a quiet system.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd "$R"
FEEDER=sweep_devin_feeder_L2
bash scripts/ops/devin_progress.sh "$FEEDER" 2>&1

echo "  --- watchdog"
# 1. orphans hold quota and return nothing: always safe to reap
orph=$(timeout 300 uv run python scripts/ops/reap_orphans.py 2>/dev/null | grep -c WOULD || true)
if [ "${orph:-0}" -gt 0 ]; then
  echo "    reaping $orph orphan(s)"
  timeout 300 uv run python scripts/ops/reap_orphans.py --apply 2>/dev/null | tail -3 | sed 's/^/      /'
fi

# 2. is anything still going to launch? match argv, never a pattern this script also contains
alive=$(timeout 120 uv run python - <<'PY' 2>/dev/null
import os
n = 0
for e in os.listdir("/proc"):
    if not e.isdigit():
        continue
    try:
        a = [x for x in open(f"/proc/{e}/cmdline", "rb").read()
             .decode(errors="replace").split("\0") if x]
    except OSError:
        continue
    if len(a) < 2:
        continue
    b = os.path.basename(a[1])
    if b in ("devin_queue.sh", "sweep_seq.sh", "grok_ladder.py", "cohort_topup.sh"):
        n += 1
print(n)
PY
)
alive=${alive:-0}

# 3. is there work left for devin anywhere in the two phases?
work=$(timeout 500 uv run python - <<'PY' 2>/dev/null
import sys, os, glob
sys.path.insert(0, "scripts/ops")
import trial_ledger as TL, trial_guard as TG, solver_match as SM, grok_ladder as GL
per = TL.ledger(); bys = TL.ledger_by_solver(); n = 0
for d in sorted(glob.glob("experiments/dose_response/sweep_devin_feeder_L2/*/")):
    u = os.path.basename(d.rstrip("/"))
    try:
        if TG.decide(u, per, solver="devin")[0] and SM.decide(u, "devin")[0]:
            n += 1
    except Exception:
        pass
n += 4 * len(GL.l2_failures_of("devin", bys))     # 4 climb cells per candidate
print(n)
PY
)
work=${work:-0}
echo "    launchers alive: $alive   devin work outstanding: $work"

if [ "$alive" -eq 0 ] && [ "$work" -gt 0 ]; then
  echo "    !! nothing is launching and $work cell(s) remain — restarting the queue"
  nohup bash scripts/ops/devin_queue.sh "$FEEDER" \
    >> outputs/supervisor/devin_queue.log 2>&1 &
  echo "    queue restarted pid $!"
elif [ "$work" -eq 0 ]; then
  echo "    devin work COMPLETE — nothing outstanding in either phase"
else
  echo "    on course"
fi
