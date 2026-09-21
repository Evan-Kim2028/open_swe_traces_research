#!/usr/bin/env python3
"""Every invariant the pipeline is supposed to hold, checked in one place.

Written because the failure mode is always the same: agents look busy, the bank does not
move, and it takes a human noticing to find out why. Five utilisation bugs were found that
way today. Each became a check here so the next one is caught by a script, not by luck.

Exit 0 all clear, 1 warnings, 2 something needs fixing now.
"""
import os, sys, re, json, glob, time, subprocess, collections

R = "/home/evan/Documents/open_swe_traces_research"
os.chdir(R)
sys.path.insert(0, "scripts/ops")
FAIL, WARN, OK = [], [], []


def sh(c, t=60):
    try:
        return subprocess.run(c, shell=True, capture_output=True, text=True, timeout=t).stdout.strip()
    except Exception:
        return ""


# --- 1. the loop itself is alive -------------------------------------------------
pid = sh("cat outputs/supervisor/pid 2>/dev/null")
alive = pid and sh(f"kill -0 {pid} 2>/dev/null && echo y") == "y"
logage = int(time.time() - os.path.getmtime("outputs/supervisor/supervisor.log")) if \
    os.path.exists("outputs/supervisor/supervisor.log") else 99999
if not alive:
    FAIL.append("supervisor DEAD — restart: setsid bash -c 'echo $$ > outputs/supervisor/pid; "
                "exec bash scripts/ops/supervisor.sh 300' &")
elif logage > 900:
    FAIL.append(f"supervisor alive but log is {logage//60}m stale — it is wedged, not working")
else:
    OK.append(f"supervisor alive (pid {pid}, log {logage}s old)")

# --- 2. is the bank moving? a flat bank with healthy agents IS the bug -----------
rows = []
if os.path.exists("outputs/rolling.jsonl"):
    for ln in open("outputs/rolling.jsonl"):
        try:
            rows.append(json.loads(ln))
        except Exception:
            pass
recent = [r for r in rows if r.get("t", 0) > time.time() - 3600]
if len(recent) >= 3:
    d = recent[-1]["certified"] - recent[0]["certified"]
    dt = recent[-1]["trials"] - recent[0]["trials"]
    if d == 0 and dt >= 5:
        FAIL.append(f"STALL: {dt} trials in the last hour, certified did not move. "
                    f"Check for re-trialling of decided units and for idle container slots.")
    elif d == 0:
        WARN.append(f"bank flat for an hour ({dt} trials) — low throughput, check capacity")
    else:
        OK.append(f"bank +{d} in the last hour ({dt} trials, {dt/max(1,d):.1f}/cert)")

# --- 3. Composer fed? ------------------------------------------------------------
agents = int(sh("docker ps --format '{{.Names}}' | grep -c env-main") or 0)
queue = [l.strip() for l in open("outputs/supervisor/sweep_queue.txt").read().split("\n")
         if l.strip()] if os.path.exists("outputs/supervisor/sweep_queue.txt") else []
try:
    import trial_ledger
    per = trial_ledger.ledger()
    # "Not in the ledger" is not the same as "runnable". Most such units are L2 dirs whose
    # L0 was never run, and the guard correctly refuses them (run the cheaper verdict
    # first). Counting them as idle capacity produced a FAIL on 14 units of which exactly
    # one could actually be trialled.
    import trial_guard
    runnable = 0
    for d in glob.glob("experiments/dose_response/sweep_*/*/"):
        if not os.path.isdir(d):
            continue
        name = os.path.basename(d.rstrip("/"))
        if name.rsplit("-L", 1)[0] in per:
            continue
        try:
            ok, _ = trial_guard.decide(name, per)
        except Exception:
            ok = True
        if ok:
            runnable += 1
    untrialled = runnable
except Exception:
    per, untrialled = {}, -1
if agents == 0 and not queue and untrialled == 0:
    FAIL.append("Composer STARVED: no agents, empty queue, nothing staged untrialled. "
                "The Devin->Composer handoff has stalled; check STAGE jobs.")
newest_sweep = sh("ps -eo etimes,args | awk '/sweep_seq[.]sh/ && !/awk/ {print $1}' | sort -n | head -1")
starting = newest_sweep.isdigit() and int(newest_sweep) < 240
_budget_left = subprocess.run(
    "uv run python scripts/ops/composer_budget.py --quiet", shell=True,
    capture_output=True).returncode == 0

if starting and agents < 4:
    OK.append(f"Composer {agents}/12 — a sweep started {newest_sweep}s ago, environments still coming up")
elif agents < 4 and untrialled > 8 and _budget_left:
    FAIL.append(f"Composer at {agents}/12 with {untrialled} units staged and untrialled — "
                f"a sweep should be running. Check the sweep cap and container headroom.")
elif agents < 4 and not _budget_left:
    # An idle Composer is correct, not broken, once the 250M cap is reached: orchestrate
    # routes every sweep to Devin from then on. Without this the monitor screams FAIL for
    # the rest of the run and the real signals get lost in it.
    OK.append(f"Composer idle at {agents}/12 — budget spent, all work routed to Devin")
else:
    OK.append(f"Composer {agents}/12 agents, {untrialled} units of runway, queue={queue or 'empty'}")

# --- 4. Devin slots + throttle ---------------------------------------------------
sys.path.insert(0, "scripts/ops")
from slots import occupancy as _occ   # shared definition; see slots.py
_o = _occ("devin")
dev = _o["sessions"]
# Trials run the CLI inside a container: invisible to pgrep, but they spend the same
# account quota. Reporting sessions only showed "4/4" while the real load was 5.
dev_trials = sum(t["conc"] for t in _o["trials"])
throttled = sh("grep -l 'Reached free model rate limit' /home/evan/Documents/oswt-*/outputs/*.log "
               "2>/dev/null | xargs -r stat -c %Y 2>/dev/null | sort -rn | head -1")
recent_throttle = throttled and (time.time() - int(throttled)) < 1800
if recent_throttle:
    OK.append(f"Devin {len(dev)}/4 — throttle within 30m, backoff correctly holding refills")
elif len(dev) + dev_trials < 2:
    # Branch on TOTAL, not sessions. This tested len(dev) alone and fired "Devin only
    # 1/4" while one session and three in-container trials were saturating the cap - the
    # very number the next line computes. A trial occupies a Devin slot exactly as a
    # session does.
    WARN.append(f"Devin only {len(dev) + dev_trials}/4 ({len(dev)} session(s) + "
                f"{dev_trials} trial(s)) and no recent throttle — dq2 shepherds ONE job "
                f"per instance, so start another launcher: "
                f"MAXN=4 setsid nohup bash /home/evan/devin-tasks/dq2.sh >> "
                f"/home/evan/devin-tasks/dq2.log 2>&1 &")
else:
    total = len(dev) + dev_trials
    names = ' '.join(x.replace('closure_', '') for x in dev)
    msg = f"Devin {total} run(s) = {len(dev)} session(s) + {dev_trials} trial(s)"
    # Devin must be DOING something: 2-4 concurrent runs. Below 2 is as much a failure as
    # above 4 -- an idle Devin is wasted free capacity, and nobody notices an absence.
    if total > 4:
        FAIL.append(msg + " — OVER CAP. dq2.sh owns sessions (MAXN=2); do not kill a "
                          "running session, it restarts from scratch. Wait for drain.")
    elif total < 2:
        FAIL.append(msg + " — UNDER FLOOR. Devin is idle and free. Check: is dq2.sh alive "
                          "(pgrep -f dq2), does the manifest have pending jobs "
                          "(pipeline_autogen.py --status), is a throttle cooloff active "
                          "(.devin_cooldown)? Queue an _L2 cohort to use the trial slots.")
    else:
        OK.append(msg + (f": {names}" if names else ""))

# --- 5. autogen is generating, and not duplicating -------------------------------
mani = {}
if os.path.exists("/home/evan/devin-tasks/queue/manifest.tsv"):
    for ln in open("/home/evan/devin-tasks/queue/manifest.tsv"):
        if ln.startswith("#") or not ln.strip():
            continue
        f = ln.split("\t")
        if len(f) >= 7:
            mani[f[0]] = f
pend = [n for n, f in mani.items()
        if not os.path.exists(os.path.join(f[2], f[6].strip())) and os.path.exists(f[1])]
if len(pend) == 0:
    WARN.append("no pending Devin jobs — autogen should be appending one per tick")
elif len(pend) > 10:
    WARN.append(f"{len(pend)} pending Devin jobs — queue is becoming a wishlist")
else:
    OK.append(f"{len(pend)} Devin jobs pending")

# --- 6. resources ----------------------------------------------------------------
free = int((sh("df --output=avail -BG / | tail -1") or "0G").strip().rstrip("G") or 0)
load = float((sh("uptime").split("load average:")[-1].split(",")[0] or 0))
cores = int(sh("nproc") or 1)
if free < 100:
    FAIL.append(f"disk {free}G < 100G prune floor — confirm prune_worktrees ran and never "
                f"touched ladder-base:* or environment/src")
elif free < 140:
    WARN.append(f"disk {free}G, approaching the 100G floor")
else:
    OK.append(f"disk {free}G free")
if load > cores * 2.5:
    WARN.append(f"load {load:.0f} on {cores} cores — CPU saturated; do NOT raise container "
                f"concurrency, trials will just take longer")
else:
    OK.append(f"load {load:.0f} on {cores} cores")

# --- 7. nobody is re-trialling a decided unit ------------------------------------
if per:
    live_jobs = set()
    for ln in sh("ps -eo args").split("\n"):
        if "sweep_seq.sh" in ln and "awk" not in ln:
            parts = ln.split()
            for i, w in enumerate(parts[:-1]):
                if w.endswith("sweep_seq.sh"):
                    live_jobs.add(parts[i + 1])
    waste = 0
    for t in trial_ledger.trials():
        if t["reward"] is None or t["errored"] or t["mtime"] < time.time() - 3600:
            continue
        d = per.get(t["base"], {})
        # Attribute only to sweeps still running. A finished sweep's re-trials are history
        # and cannot be prevented now; flagging them made the check cry wolf on its first run.
        job_stem = t["job"].rsplit("_r", 1)[0]
        if t["rung"] == "0" and len(d.get("0", [])) > 1 and job_stem in live_jobs:
            waste += 1
    if waste >= 4:
        FAIL.append(f"{waste} re-trials of already-decided L0 units in the last hour — "
                    f"the between-round re-gate in sweep_seq is not firing")
    elif waste:
        WARN.append(f"{waste} L0 re-trial(s) in the last hour")
    else:
        OK.append("no re-trialling of decided units")

# --- 9. finished work stranded in worktrees --------------------------------------
# No monitor could see this and it was the single biggest drag on the bank: a VF job writes
# its hidden suites inside its own worktree and a RC job writes contracts inside its own.
# Nothing moved them to the main checkout, so 74 finished units were unstageable while
# autogen - whose census takes the max across roots - reported them verified and queued
# more authoring behind them. harvest.py closes the gap; this asserts it stays closed.
try:
    debt = sh("uv run python scripts/ops/harvest.py 2>/dev/null | head -1", t=300)
    n = re.search(r"(\d+) recoverable", debt)
    if n and int(n.group(1)) >= 10:
        FAIL.append(f"{n.group(1)} finished unit(s) stranded in worktrees — "
                    f"run: uv run python scripts/ops/harvest.py --apply")
    elif n and int(n.group(1)):
        WARN.append(f"{n.group(1)} unit(s) stranded in worktrees (harvest will pick them up)")
    else:
        OK.append("no finished work stranded in worktrees")
except Exception:
    pass

# --- 10. authored-but-unverified backlog -----------------------------------------
# Authoring generates its own backlog; verification is what drains it. Ask autogen for the
# figure rather than re-deriving it: a local count included cohorts banked through the old
# path and reported 154 where the real backlog was 30, which would have had the monitor
# screaming about a bottleneck that did not exist.
try:
    u = sh("uv run python scripts/ops/pipeline_autogen.py --unverified 2>/dev/null", t=300)
    unver = int(u.strip().split()[-1])
    if unver >= 60:
        WARN.append(f"{unver} authored unit(s) unverified — verification is the bottleneck")
    else:
        OK.append(f"{unver} authored unit(s) awaiting verification")
except Exception:
    pass

# --- 11. staged tasks vs the agent registry ---------------------------------------
# A host an agent needs but a task does not allow is invisible from the outside: the CLI
# in the container gets an empty model list and the trial dies with "Unknown model", which
# reads as a bad slug. That cost 98 of 102 Devin trials in one hour before anyone looked
# at exception.txt. Compare what is staged against scripts/ops/agents.py rather than a
# literal, so adding an agent cannot silently leave old cohorts unable to run it.
try:
    sys.path.insert(0, "scripts/ops")
    from agents import egress_hosts
    need = set(egress_hosts())
    bad = []
    for tf in glob.glob("experiments/dose_response/sweep_*/*/task.toml"):
        m = re.search(r"allowed_hosts\s*=\s*\[([^\]]*)\]", open(tf).read())
        have = set(x.strip().strip('"') for x in (m.group(1).split(",") if m else []))
        if need - have:
            bad.append((tf, sorted(need - have)))
    if bad:
        miss_hosts = sorted({h for _, hs in bad for h in hs})
        FAIL.append(f"{len(bad)} staged task(s) miss solver host(s) {' '.join(miss_hosts[:4])}"
                    f" — trials for that agent will die with a misleading 'Unknown model'")
    else:
        OK.append(f"all staged tasks allow every registry host ({len(need)})")
except Exception:
    pass

# --- 12. repeat-rung trials ------------------------------------------------------
# 73% of all tokens went to trials beyond the minimum L0+L2 path, and the leak was silent:
# nothing counted how often a rung was re-measured. A rung answers one question, so a
# second verdict on it buys nothing. This is the metric that would have surfaced
# kops-clustervalid running nineteen L6 trials.
try:
    sys.path.insert(0, "scripts/ops")
    import trial_ledger as _tl
    _seen, _rep, _tot = collections.defaultdict(set), 0, 0
    _cut = time.time() - 6 * 3600
    _rows = []
    for _t in _tl.trials():
        _u = _t.get("unit") or ""
        _b = _u.rsplit("-L", 1)[0]
        _r = _u.rsplit("-L", 1)[1][:1] if "-L" in _u else "0"
        _d = _t.get("dir") or ""
        try:
            _w = os.path.getmtime(os.path.join(_d, "result.json"))
        except OSError:
            continue
        _rows.append((_w, _b, _r))
    for _w, _b, _r in sorted(_rows):
        if _w < _cut:
            _seen[_b].add(_r)
            continue
        _tot += 1
        if _r in _seen[_b]:
            _rep += 1
        _seen[_b].add(_r)
    if _tot:
        _pct = _rep / _tot * 100
        _msg = f"{_rep}/{_tot} trial(s) in the last 6h re-measured a decided rung ({_pct:.0f}%)"
        if _pct >= 40:
            FAIL.append(_msg + " — trial_guard is leaking; check RUNG_TRIAL_CAP and rung parsing")
        elif _pct >= 15:
            WARN.append(_msg)
        else:
            OK.append(_msg)
except Exception:
    pass

# --- 13. stability gate ----------------------------------------------------------
# The standing objective: one clean 6h window (repeat-rung <15%, >=40 devin trials, no new
# failure mode) before scaling spend. Reported here so every health check shows progress
# toward it rather than only the day's incidents.
try:
    import subprocess as _sp
    _g = _sp.run(["uv", "run", "python", "scripts/ops/stability_gate.py", "--brief"],
                 capture_output=True, text=True, timeout=300)
    _line = (_g.stdout or "").strip()
    if _line:
        (OK if _g.returncode == 0 else WARN).append(_line)
except Exception:
    pass

# --- 14. multi-model certificate split -------------------------------------------
# The bank is deliberately multi-model (analytics/research/MULTIMODEL_BANK.md). A
# certificate whose L0 failure and L2 pass came from different models is a weaker claim,
# so the split is reported rather than collapsed into one "certified" number.
try:
    import trial_ledger as _tlx
    _c = _tlx.certificates()
    _cross = sum(1 for v in _c.values() if v["kind"] == "cross")
    _msg = (f"{len(_c)} certificate(s): {len(_c)-_cross} single-solver, "
            f"{_cross} cross-solver")
    (WARN if _cross > 0.25 * max(len(_c), 1) else OK).append(_msg)
except Exception:
    pass

print(f"{'='*66}\nPIPELINE HEALTH  {time.strftime('%H:%M:%S')}\n{'='*66}")
for m in FAIL:
    print(f"  FAIL  {m}")
for m in WARN:
    print(f"  WARN  {m}")
for m in OK:
    print(f"  ok    {m}")
print(f"{'='*66}")
sys.exit(2 if FAIL else (1 if WARN else 0))
