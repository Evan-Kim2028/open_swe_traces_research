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
pgrep -f "cohort_topup.sh $COHORT" >/dev/null && echo "  topup loop: alive" \
  || echo "  topup loop: NOT RUNNING  <-- freed slots will idle"
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
