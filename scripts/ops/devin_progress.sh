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
done = [u for u in units if (bys.get(u, {}).get("devin") or {}).get("2")]
fails = [u for u in done if max((bys[u]["devin"]["2"])) == 0]
print(f"  cohort: {len(done)}/{len(units)} L2 verdicts, {len(fails)} FAILED L2 "
      f"(these are the climb candidates)")
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
left = len(units) - len(done)
rate = r3 if r3 > 0.5 else (r1 if r1 else 0)
if left and rate:
    print(f"  ETA for the remaining {left}: ~{left / rate:.1f} h at {rate:.1f}/h")
elif not left:
    print("  cohort COMPLETE — build the L3-L6 climb for the L2 failures")
else:
    print("  !! no devin verdicts in the last three hours — investigate")
PY
echo "====="
