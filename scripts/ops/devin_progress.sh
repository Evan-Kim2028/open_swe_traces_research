#!/usr/bin/env bash
# Is devin running AND making progress on the climb cohort? Rate and ETA, not just liveness.
#
# "Devin is up" and "devin is getting somewhere" are different questions. A cohort launched
# narrower than the cap keeps its concurrency forever, a top-up loop can die silently, and a
# sweep whose units all get dropped logs "nothing left to trial" and exits looking healthy.
# Each of those reads as alive and delivers nothing, so this reports verdicts landed, the
# measured rate, and what that implies for the finish.
set -u
R=/home/evan/Documents/open_swe_traces_research
COHORT="${1:-sweep_devin_climb_L2}"
cd "$R"
echo "===== DEVIN CHECK $(date '+%Y-%m-%d %H:%M:%S')  cohort=$COHORT"
timeout 200 uv run python scripts/ops/slots.py 2>/dev/null | head -1
echo "  containers on this cohort:"
for c in $(docker ps --format '{{.Names}}' | grep env-main-1); do
  stem=${c%%__env-main-1}
  u=$(echo "$stem" | sed 's/-l[0-9].*//')
  if ls "experiments/dose_response/$COHORT" 2>/dev/null | grep -q "^${u}-L"; then
    echo "    CLIMB $stem"
  else
    echo "    other $stem"
  fi
done
# Which driver SHOULD be running depends on the phase, so report what is there and let the
# reader judge, rather than alarming on the absence of a loop that was stopped on purpose.
# The first version printed "topup loop NOT RUNNING <-- freed slots will idle" immediately
# after the L2 probe was deliberately stopped to prioritise the climb, which is a false alarm
# and the fastest way to train someone to ignore a monitor.
timeout 120 uv run python - <<'PY' 2>/dev/null
import os
want = {"cohort_topup.sh": "L2 probe top-up", "grok_ladder.py": "ladder climb",
        "ladder_matrix.py": "comparison matrix"}
seen = {}
for e in os.listdir("/proc"):
    if not e.isdigit():
        continue
    try:
        argv = [a for a in open(f"/proc/{e}/cmdline", "rb").read()
                .decode(errors="replace").split("\0") if a]
    except OSError:
        continue
    if len(argv) < 2:
        continue
    b = os.path.basename(argv[1])
    if b in want:
        extra = " ".join(x for x in argv[2:] if not x.startswith("-"))[:40]
        seen.setdefault(want[b], []).append(f"pid {e} {extra}".rstrip())
    # A bare sweep_seq (or the harbor run under it) keeps launching from its cohort until the
    # cohort is done, so it IS a driver. Omitting it printed "NONE RUNNING -- nothing will
    # launch new trials" while a conc=4 sweep was mid-flight, which is a false alarm on the
    # one line that is supposed to mean "intervene".
    if b == "sweep_seq.sh":
        seen.setdefault("cohort sweep", []).append(f"pid {e} {' '.join(argv[2:])[:40]}")
    if any("harbor" in x for x in argv[:2]) and "--job-name" in argv:
        j = argv[argv.index("--job-name") + 1]
        seen.setdefault("harbor run", []).append(f"pid {e} {j[:44]}")
if seen:
    for k, v in sorted(seen.items()):
        print(f"  driver: {k} -- {'; '.join(v)}")
else:
    print("  driver: NONE RUNNING  <-- nothing will launch new trials")
PY
timeout 600 uv run python - "$COHORT" <<'PY' 2>/dev/null
import sys, os, glob, time
sys.path.insert(0, "scripts/ops")
import trial_ledger as TL
cohort = sys.argv[1]
bys = TL.ledger_by_solver()
units = sorted({os.path.basename(d.rstrip("/")).rsplit("-L", 1)[0]
                for d in glob.glob(f"experiments/dose_response/{cohort}/*/")})
# Count the CELLS the cohort actually holds. This counted L2 verdicts only, so pointed at a
# cohort of L3-L6 cells it reported "4/4 L2 verdicts ... cohort COMPLETE" while 12 of its 16
# cells had never run.
cells = sorted(os.path.basename(d.rstrip("/"))
               for d in glob.glob(f"experiments/dose_response/{cohort}/*/"))
def _has(cell):
    b, _, r = cell.rpartition("-L")
    return bool((bys.get(b, {}).get("devin") or {}).get(r))
cdone = [c for c in cells if _has(c)]
print(f"  cohort: {len(cdone)}/{len(cells)} cell verdicts")
done = [u for u in units if (bys.get(u, {}).get("devin") or {}).get("2")]
fails = [u for u in done if max((bys[u]["devin"]["2"])) == 0]
if fails:
    print(f"    {len(fails)} unit(s) failed L2 in this cohort: {', '.join(sorted(fails))}")
if fails:
    print(f"    {', '.join(sorted(fails))}")
now = time.time()
rows = [t for t in TL.trials() if TL.solver_of(t.get("model")) == "devin"]
r1 = sum(1 for t in rows if t["mtime"] >= now - 3600)
r3 = sum(1 for t in rows if t["mtime"] >= now - 3 * 3600) / 3
print(f"  devin rate: {r1}/h last hour, {r3:.1f}/h over three")
# The climb is the priority now, so report it here too: which units devin failed at both
# L0 and L2, and how many of their L3-L6 cells are measured.
import collections
climb = []
for b, per in bys.items():
    d = per.get("devin") or {}
    l0, l2 = d.get("0"), d.get("2")
    if not (l0 and max(l0) == 0 and l2 and max(l2) == 0):
        continue
    if any(max(v) > 0 for r, v in d.items() if r.isdigit() and int(r) >= 2 and v):
        continue
    got = [r for r in ("3", "4", "5", "6") if d.get(r)]
    climb.append((b, got))
if climb:
    cells = sum(len(g) for _, g in climb)
    print(f"  CLIMB: {len(climb)} unit(s) failed L0+L2, {cells}/{len(climb)*4} "
          f"L3-L6 cells measured")
    for b, got in sorted(climb):
        print(f"    {b:22s} {','.join('L'+x for x in got) or 'none yet'}")
left = len(cells) - len(cdone)
rate = r3 if r3 > 0.5 else (r1 if r1 else 0)
if left and rate:
    print(f"  ETA for the remaining {left}: ~{left / rate:.1f} h at {rate:.1f}/h")
elif not left:
    print("  cohort COMPLETE — every cell has a verdict")
else:
    print("  !! no devin verdicts in the last three hours — investigate")
PY
# --- the failure modes this evening actually produced, each now checked by name -----------
# Every one of these read as healthy while wasting capacity, which is why they are here rather
# than left to be noticed.
echo "  --- integrity"
timeout 300 uv run python scripts/ops/reap_orphans.py 2>/dev/null | head -4 | sed 's/^/    /'
timeout 200 uv run python - <<'PY' 2>/dev/null
import subprocess, collections
out = subprocess.run(["docker","ps","--format","{{.Names}}"],capture_output=True,text=True).stdout.split()
mains = [c for c in out if c.endswith("__env-main-1")]
cells = collections.Counter(c.split("__")[0] for c in mains)
dup = {k: v for k, v in cells.items() if v > 1}
print(f"    duplicate cells in flight: {dup or 'none'}")
import sys
sys.path.insert(0, "scripts/ops")
import slots
o = slots.occupancy("devin")
flag = "" if o["containers"] <= o["declared"] else "  <-- ORPHANS, occupancy understated"
print(f"    occupancy: {o['total']}/{o['cap']}  declared={o['declared']} "
      f"containers={o['containers']}{flag}")
if o["total"] > o["cap"]:
    print(f"    !! OVER CAP by {o['total'] - o['cap']} — throttle risk, errors every in-flight trial")
PY
echo "====="
