#!/usr/bin/env bash
# One digest answering "are we moving toward the goal", for a periodic self-check.
#
# watch.sh answers "is anything broken". That is not the same question. A pipeline can be
# perfectly healthy and making no progress at all -- composer idle with budget, devin's
# queue draining into units nobody can certify, a ladder driver blocked on an in-flight
# trial that died. Those show up as silence in an alarm-only monitor, which is exactly how
# the 143-unit missing-src backlog stayed invisible for hours.
#
# So this prints RATES and REMAINING WORK, not just state.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd "$R"
echo "=========== DIGEST $(date '+%Y-%m-%d %H:%M:%S') ==========="

echo "--- goal 1: grok's ladder on the units composer exhausted"
timeout 400 python3 scripts/ops/grok_ladder.py --plan 2>/dev/null | tail -7
# Match argv PROPERLY, not with pgrep -f on a pattern this script also contains. The
# first cut used `pgrep -f "grok_ladder.py --run"` and reported TWO drivers when one was
# running: the pattern appears in this file, so the subshell running the pgrep matched
# itself. That is the same class of error that once reported four supervisors and read
# exactly like the duplicate-launcher bug that put devin at 7 against a cap of 4 --
# a phantom duplicate is indistinguishable from a real one, and a real one is serious
# here, because two drivers race to launch the same rung.
timeout 60 python3 - <<'PY'
import os
found = []
for e in os.listdir("/proc"):
    if not e.isdigit():
        continue
    try:
        argv = [a for a in open(f"/proc/{e}/cmdline", "rb").read()
                .decode(errors="replace").split("\0") if a]
    except OSError:
        continue
    if (len(argv) >= 3 and "python" in os.path.basename(argv[0])
            and argv[1].endswith("grok_ladder.py") and "--run" in argv):
        found.append(e)
if not found:
    print("  !! NO LADDER DRIVER RUNNING")
elif len(found) > 1:
    print(f"  !! {len(found)} LADDER DRIVERS RUNNING (pids {', '.join(found)}) — they race")
else:
    print(f"  driver pid {found[0]} alive")
PY
tail -2 outputs/supervisor/grok_ladder_driver.log 2>/dev/null | sed 's/^/  /'

echo "--- goal 2: throughput (are verdicts landing?)"
timeout 300 python3 - <<'PY'
import glob, json, os, time, collections
now = time.time()
for label, win in (("last 1h", 3600), ("last 6h", 21600)):
    n = p = 0
    per = collections.Counter()
    for f in glob.glob('experiments/dose_response/jobs/*/*/result.json'):
        if os.path.getmtime(f) < now - win:
            continue
        try:
            d = json.load(open(f))
        except Exception:
            continue
        r = (d.get('verifier_result') or {}).get('rewards', {}).get('reward')
        if r is None:
            continue
        n += 1
        p += (r > 0)
        m = (d.get('agent_result') or {}).get('model') or '?'
        per[m.split('/')[-1]] += 1
    print(f"  {label}: {n} verdict(s), {p} pass  {dict(per)}")
PY

echo "--- goal 3: dataset growth + budget"
timeout 400 python3 scripts/ops/trial_ledger.py --report 2>/dev/null | grep -E "^certificates|^  above L2|^  climbed|^tokens|^cost|^  binds"
timeout 300 python3 scripts/ops/budget_forecast.py --brief 2>/dev/null | tail -2

echo "--- goal 4: capacity actually in use"
timeout 200 python3 scripts/ops/slots.py 2>/dev/null | head -1
echo "  composer containers: $(docker ps --format '{{.Names}}' | grep -c cursor || true)"
timeout 500 python3 - <<'PY'
import sys, os, glob, collections
sys.path.insert(0, "scripts/ops")
import trial_ledger, trial_guard as TG, cohorts
per = trial_ledger.ledger()
c = collections.Counter()
for d in glob.glob("experiments/dose_response/sweep_*/*/"):
    if not os.path.isdir(d) or not glob.glob(d + "tests/hidden/**/*", recursive=True):
        continue
    if cohorts.barren(d.split(os.sep)[2]):
        continue
    if not os.path.isdir(os.path.join(d, "environment", "src")):
        continue
    b = os.path.basename(d.rstrip("/"))
    for s in ("composer", "devin", "grok"):
        try:
            if TG.decide(b, per, solver=s)[0]:
                c[s] += 1
        except Exception:
            pass
print(f"  runnable with a source tree: {dict(c)}")
PY

echo "--- goal 5: alarms"
timeout 600 bash scripts/ops/watch.sh 2>&1 | sed 's/^/  /'
echo "  disk: $(df -h . | tail -1 | awk '{print $4" free ("$5" used)"}')"
echo "=========== END DIGEST ==========="
