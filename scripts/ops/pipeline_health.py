#!/usr/bin/env python3
"""Every invariant the pipeline is supposed to hold, checked in one place.

Written because the failure mode is always the same: agents look busy, the bank does not
move, and it takes a human noticing to find out why. Five utilisation bugs were found that
way today. Each became a check here so the next one is caught by a script, not by luck.

Exit 0 all clear, 1 warnings, 2 something needs fixing now.
"""
import os, sys, json, glob, time, subprocess, collections

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
if starting and agents < 4:
    OK.append(f"Composer {agents}/12 — a sweep started {newest_sweep}s ago, environments still coming up")
elif agents < 4 and untrialled > 8:
    FAIL.append(f"Composer at {agents}/12 with {untrialled} units staged and untrialled — "
                f"a sweep should be running. Check the sweep cap and container headroom.")
else:
    OK.append(f"Composer {agents}/12 agents, {untrialled} units of runway, queue={queue or 'empty'}")

# --- 4. Devin slots + throttle ---------------------------------------------------
dev = sh("pgrep -af '[d]evin --model' | grep -oP 'closure_\\w+' | sort -u").split()
# Trials run the CLI inside a container: invisible to pgrep, but they spend the same
# account quota. Reporting sessions only showed "4/4" while the real load was 5.
dev_trials = int(sh("ps -eo args | awk '/harbor run/ && !/awk/ {a=\"\";c=1;j=\"\"; "
                    "for(i=1;i<NF;i++){if($i==\"--agent\")a=$(i+1); "
                    "if($i==\"--n-concurrent\")c=$(i+1); if($i==\"--job-name\")j=$(i+1)} "
                    "if(a==\"devin\" && !(j in seen)){seen[j]=1; tot+=c}} END{print tot+0}'") or 0)
throttled = sh("grep -l 'Reached free model rate limit' /home/evan/Documents/oswt-*/outputs/*.log "
               "2>/dev/null | xargs -r stat -c %Y 2>/dev/null | sort -rn | head -1")
recent_throttle = throttled and (time.time() - int(throttled)) < 1800
if recent_throttle:
    OK.append(f"Devin {len(dev)}/4 — throttle within 30m, backoff correctly holding refills")
elif len(dev) < 3:
    WARN.append(f"Devin only {len(dev)}/4 and no recent throttle — refill may be stuck; "
                f"check pipeline_autogen output in supervisor.log")
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

print(f"{'='*66}\nPIPELINE HEALTH  {time.strftime('%H:%M:%S')}\n{'='*66}")
for m in FAIL:
    print(f"  FAIL  {m}")
for m in WARN:
    print(f"  WARN  {m}")
for m in OK:
    print(f"  ok    {m}")
print(f"{'='*66}")
sys.exit(2 if FAIL else (1 if WARN else 0))
