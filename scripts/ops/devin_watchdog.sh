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
alive=$(timeout 300 python3 - <<'PY' 2>/dev/null
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
# A FAILED PROBE IS NOT ZERO. This read `${alive:-0}`, so a probe that timed out or errored
# reported "no launchers" and the watchdog restarted the queue on top of a climb driver that was
# running perfectly well. Same shape as occupancy treating an unreadable docker as empty: the
# unknown case has to fail toward doing nothing, because the action here is to START things.
if ! [ "${alive:-}" -ge 0 ] 2>/dev/null; then
  echo "    launcher probe FAILED — assuming something is running, taking no action"
  alive=99
fi

# 3. is there work left for devin anywhere in the two phases?
work=$(timeout 600 python3 - <<'PY' 2>/dev/null
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
# Two numbers, because they answer different questions and conflating them produced three
# wrong counters in a row.
#
#   REMAINING  the queue's own work: missing L3-L6 cells for each climb candidate, plus
#              rostered holes like advrefs (which jumped L2 -> L6, so its L3/L4/L5 are open by
#              roster). This is what counts down to zero.
#   ADMISSIBLE what the guard would let launch right now. It is much smaller, because the climb
#              takes one rung at a time -- L5 stays refused until L4 resolves -- so a small
#              number here is normal, not a stall.
#
# Everything else the guard admits is excluded on purpose. dupexpr, httpclienterr, reqidgen,
# sampler and traceopts are all admissible at L3-L6 only because contract repairs made their old
# verdicts stale, which is unrelated work and was inflating "outstanding" by 20 cells.
remaining = 0
admissible = 0
cells = set()
for b in GL.l2_failures_of("devin", bys):
    d = (bys.get(b) or {}).get("devin") or {}
    for r in ("3", "4", "5", "6"):
        if not d.get(r):
            cells.add((b, r))
for b, pr in bys.items():                      # rostered holes only
    d = pr.get("devin") or {}
    for r in ("3", "4", "5", "6"):
        if not d.get(r):
            try:
                if TG.backfill_wanted(b, r):
                    cells.add((b, r))
            except Exception:
                pass
remaining = len(cells)
for b, r in cells:
    u = f"{b}-L{r}"
    try:
        if TG.decide(u, per, solver="devin")[0] and SM.decide(u, "devin")[0]:
            admissible += 1
    except Exception:
        pass
print(f"{remaining} {admissible}")
PY
)
set -- ${work:-}
remaining="${1:-}"; admissible="${2:-}"
if ! [ "${remaining:-}" -ge 0 ] 2>/dev/null; then
  echo "    work probe FAILED — taking no action this cycle"
  remaining=0; admissible=0; alive=99
fi
echo "    launchers alive: $alive   cells remaining: $remaining   admissible now: $admissible"

if [ "$alive" -eq 0 ] && [ "$remaining" -gt 0 ]; then
  echo "    !! nothing is launching and $remaining cell(s) remain — restarting the queue"
  nohup bash scripts/ops/devin_queue.sh "$FEEDER" \
    >> outputs/supervisor/devin_queue.log 2>&1 &
  echo "    queue restarted pid $!"
elif [ "$remaining" -eq 0 ]; then
  echo "    devin work COMPLETE — nothing outstanding in either phase"
else
  echo "    on course"
fi
