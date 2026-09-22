#!/usr/bin/env python3
"""One reconciliation pass: observe the whole pipeline, decide, act.

Written after an evening of patching symptoms one at a time. The failures were never
exotic - two launchers fighting over one manifest, a serial gate holding twelve container
slots idle, sessions crowding out the trials that actually certify, a counter that could
not see in-container work. Each got a bespoke fix and the next one appeared somewhere else.

So this is a reconciler, not a script of imperatives. It reads the real state, compares it
to the invariants below, and takes the smallest action that closes the gap. Running it
twice changes nothing the second time.

INVARIANTS
  1. Devin runs up to slots.CAP concurrent, counting host sessions AND in-container trials, because
     both spend the same account quota.
  2. A session that has written nothing and logged no tool call for 25 minutes is stuck,
     not thinking. Restart it. Never kill a session that is progressing: dq2.sh's own
     comment records five relaunches lost their work that way.
  3. Trials must always be running somewhere. Trials are the only thing that certifies.
  4. L2 goes to Devin (free, low volume, where certification happens). L0 goes to Composer
     (high volume) until its 250M budget is spent, then to Devin too.
  5. Composer never exceeds its token budget.

  orchestrate.py            report what it would do
  orchestrate.py --apply    do it
"""
import json, os, re, subprocess, sys, time, glob
from cohorts import barren   # one definition; the monitor imports the same one
from slots import CAP as DEVIN_CAP   # never hardcode the cap; slots.py owns it
import solver_match                  # who may certify what; see solver_match.py

R = "/home/evan/Documents/open_swe_traces_research"
os.chdir(R)
sys.path.insert(0, "scripts/ops")
APPLY = "--apply" in sys.argv
ACTIONS, NOTES = [], []


def sh(c, t=90):
    """A failing shell command must not look like a legitimate empty answer. An
    unterminated quote in the trial counter returned '' for hours; int('' or 0) read as
    zero trials, which would have let this launch past the Devin cap all night."""
    try:
        r = subprocess.run(c, shell=True, capture_output=True, text=True, timeout=t)
        if r.returncode != 0 and not r.stdout.strip():
            NOTES.append(f"SHELL FAILED rc={r.returncode}: {r.stderr.strip()[:80]}")
        return r.stdout.strip()
    except Exception as e:
        NOTES.append(f"SHELL EXC: {e}")
        return ""


# Frozen sweep snapshots must outlive the session that created them. This pointed at
# a per-session scratch directory that is deleted with the job, which would have taken
# the frozen script out from under every sweep still reading it.
FROZEN = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                      "..", "..", "outputs", "supervisor", "frozen",
                      "sweep_seq.frozen.sh")
FROZEN = os.path.normpath(FROZEN)
os.makedirs(os.path.dirname(FROZEN), exist_ok=True)
# Certification before ladder study. Reversible: delete the file to resume.
LADDER_PAUSED = os.path.exists("outputs/supervisor/ladder_paused")
# Cohorts built by escalate.py. Exempt from the ladder pause; see the use site.
ESCALATION_PREFIX = "sweep_escalate"


LOCKDIR = "outputs/supervisor/sweep_locks"


def sweep_locked(cohort, max_age=5400):
    """True if this cohort was launched recently and may still be running.

    Relaunching a sweep re-uses the SAME harbor job name, so the new run wipes the previous
    run's trials: 24 gated units produced exactly 1 surviving verdict because orchestrate
    relaunched the cohort every tick. Process detection was supposed to prevent that and
    failed twice - once because sweeps run from a renamed frozen copy, once returning empty
    while a sweep was demonstrably alive. A lock file is not clever, but it cannot be
    misread."""
    os.makedirs(LOCKDIR, exist_ok=True)
    f = os.path.join(LOCKDIR, cohort)
    if not os.path.exists(f):
        return False
    age = time.time() - os.path.getmtime(f)
    if age >= max_age:
        return False
    # A blind 90-minute timer is safe against double-launch but wastes the machine: four
    # cohorts stayed locked by sweeps that had already exited, and every container sat
    # idle with runnable units waiting. The lock now records the launched process's own
    # PID, so liveness is a kill(pid, 0) on a number we wrote ourselves - not the `pgrep
    # -f` pattern matching that misfired twice before. No PID (old-format lock) falls back
    # to the timer.
    try:
        pid = int((open(f).read().split() + [""])[1])
    except (ValueError, IndexError, OSError):
        return True
    try:
        os.kill(pid, 0)
        return True                    # owner alive -> genuinely locked
    except ProcessLookupError:
        os.remove(f)                   # owner gone -> cohort is free again
        return False
    except PermissionError:
        return True


def sweep_lock(cohort, pid=None, conc=0, agent="cursor"):
    """Record WHICH solver reserved the concurrency, not just how much.

    Devin sweeps and Composer sweeps share this directory, so a lock that does not say
    who owns it makes the two caps eat each other: with Devin's four slots full, four of
    Composer's twelve were reserved against work Composer was not doing."""
    os.makedirs(LOCKDIR, exist_ok=True)
    open(os.path.join(LOCKDIR, cohort), "w").write(
        f"{int(time.time())} {pid or ''} {conc} {agent}")


def reserved_slots(agent="cursor"):
    """Concurrency already promised to sweeps that are launched but not yet visible.

    A container takes minutes to build, so `docker ps` under-reports a sweep that has
    just started. orchestrate only added its own launches within a single tick, and the
    supervisor re-runs it every five minutes - so tick after tick each saw a near-empty
    machine and launched again. Eight sweeps ended up holding 39 concurrency against a
    cap of 12. Counting what live locks have reserved closes the window."""
    tot = 0
    for f in glob.glob(os.path.join(LOCKDIR, "*")):
        try:
            parts = open(f).read().split()
            pid = int(parts[1]); c = int(parts[2]) if len(parts) > 2 else 0
            # A lock written before locks carried an agent counts as this one: the old
            # behaviour, so an in-flight sweep is never under-counted while they rotate.
            who = parts[3] if len(parts) > 3 else agent
            if who != agent:
                continue
            os.kill(pid, 0)
            tot += c
        except (ValueError, IndexError, OSError):
            continue
    return tot




def freeze():
    """Run sweeps from a snapshot. Editing scripts/ops/sweep_seq.sh while a sweep was
    executing shifted bash's read offset and killed the trial phase with a syntax error
    after the gate had already run.

    The snapshot is named by its own content hash. A single fixed path only moved the
    problem: a running sweep reads from the frozen file for its whole lifetime, so the
    next tick's refresh-in-place would corrupt it exactly the way editing the source did.
    Content addressing means a given sweep's file is never rewritten - a changed source
    produces a NEW path and leaves running sweeps on the bytes they started with."""
    try:
        import shutil, hashlib
        src = "scripts/ops/sweep_seq.sh"
        h = hashlib.sha256(open(src, "rb").read()).hexdigest()[:12]
        dst = FROZEN.replace(".frozen.sh", f".{h}.sh")
        if not os.path.exists(dst):
            shutil.copy(src, dst)
            # keep the directory from growing without bound; never touch a file a live
            # sweep could still be reading, so only prune ones older than a long sweep
            import glob as _g, time as _t
            for old_f in _g.glob(FROZEN.replace(".frozen.sh", ".*.sh")):
                if old_f != dst and _t.time() - os.path.getmtime(old_f) > 6 * 3600:
                    try:
                        os.remove(old_f)
                    except OSError:
                        pass
        return dst
    except Exception:
        return "./scripts/ops/sweep_seq.sh"


def run(c):
    """Fire and forget; the reconciler must not block on a sweep. Returns the pid so the
    cohort lock can be tied to the process that actually owns it."""
    if APPLY:
        return subprocess.Popen(c, shell=True, cwd=R, start_new_session=True,
                                stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL).pid
    return None


# ---------- observe ----------------------------------------------------------------
sys.path.insert(0, os.path.join(R, "scripts/ops"))
from slots import occupancy as _occ   # one definition of occupancy; see slots.py
_o = _occ("devin")
sessions = _o["sessions"]
trials = sum(t["conc"] for t in _o["trials"])
# COMPOSER's occupancy, not every trial container. `grep -c env-main` counts Devin
# trials too, so with Devin's own cap of 4 full, four of Composer's twelve slots were
# being spent on work Composer is not doing. Devin is capped separately by devin_cap.py;
# these two ceilings must not consume each other.
try:
    import slots as _slots
    containers = sum(t["conc"] for t in _slots.trials("cursor"))
except Exception:
    containers = int(sh("docker ps --format '{{.Names}}' | grep -c env-main") or 0)
# A just-launched sweep has no containers yet but has already claimed its slots. This is
# the guard that stopped eight sweeps holding 39 concurrency against a cap of 12, and it
# is what makes the narrower count above safe.
containers = max(containers, reserved_slots())

# Container count is not the only ceiling: this machine is CPU-bound and a staging pass
# compiles gold, cheat and bare variants of every unit, so load can sit at 45 on 32 cores
# while only THREE containers are up. Orchestrate would read that as nine free slots and
# launch into an already-saturated box. Treat sustained load above ~1.4x cores as full.
try:
    _cores = os.cpu_count() or 32
    _load5 = float(open("/proc/loadavg").read().split()[1])
    if _load5 > 1.25 * _cores:
        containers = max(containers, 12)
        LOADED = f"load {_load5:.0f} on {_cores} cores"
    else:
        LOADED = ""
except Exception:
    LOADED = ""
# Match sweep_seq.sh AND sweep_seq.frozen.sh. The freeze fixed one bug and introduced
# this one: the detector saw no sweeps and would have launched duplicates all night.
sweeps = [x for x in sh("ps -eo args | awk '/sweep_seq[^ ]*[.]sh/ && !/awk/ "
                        "{for(i=1;i<NF;i++) if($i ~ /sweep_seq[^ ]*[.]sh$/){print $(i+1); break}}' | sort -u").split() if x]
devin_total = len(sessions) + trials
throttle_age = int(sh("f=$(grep -lE 'Reached free model rate limit' /home/evan/Documents/oswt-*/outputs/*.log "
                      "2>/dev/null | xargs -r stat -c %Y 2>/dev/null | sort -rn | head -1); "
                      "if [ -n \"$f\" ]; then echo $(( $(date +%s) - f )); else echo 999999; fi") or 999999)
budget_ok = subprocess.run("uv run python scripts/ops/composer_budget.py --quiet",
                           shell=True, cwd=R, capture_output=True).returncode == 0

# per-session progress, from the CLI's own database
import sqlite3
prog = {}
try:
    db = os.path.expanduser("~/.local/share/devin/cli/sessions.db")
    c = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
    now = time.time()
    for sid, wd, la in c.execute("select id,working_directory,last_activity_at from sessions "
                                 "order by created_at desc limit 12"):
        k = os.path.basename(wd or "?")
        if k in prog:
            continue
        v = float(la); v = v / 1000 if v > 1e12 else v
        t = c.execute("select count(*) from tool_call_state where session_id=?", (sid,)).fetchone()[0]
        prog[k] = {"idle_min": (now - v) / 60, "tools": t}
except Exception:
    pass

# staged cohorts with runnable units
import trial_ledger, trial_guard, token_cost
per = trial_ledger.ledger()
runnable = {}
runnable_by = {}   # solver -> cohort -> count
for d in glob.glob("experiments/dose_response/sweep_*/*/"):
    if not os.path.isdir(d):
        continue
    cohort = d.split(os.sep)[2]
    name = os.path.basename(d.rstrip("/"))
    # Skip only the rung that already has a verdict, not every unit that has ever been
    # trialled. The coarse check hid 12 units that failed L0 and had never been run at L2
    # - the exact rung the harvest had just made stageable - because their L0 history put
    # them "in the ledger". Re-trial protection is what this guards, and a rung with no
    # verdict has nothing to protect. trial_guard still arbitrates the rest (condemned,
    # non-flipping, ladder sampling).
    base = name.rsplit("-L", 1)[0]
    # trial_ledger normalises "2rc"/"2auto" to "2"; parsing the raw suffix here meant
    # the skip never matched for repair-contract cohorts and they re-trialled freely.
    rung = (name.rsplit("-L", 1)[1][:1] if "-L" in name else "0")
    #
    # This fast path is a SHADOW of trial_guard's rule, and it runs before the guard, so
    # anywhere the two disagree the guard loses silently. That is what happened to the
    # second screen: enrolling 20 certified units opened them in the guard and changed
    # orchestrate's runnable count by exactly zero, because each one has an L0 verdict
    # and was dropped here before trial_guard.decide was ever called. Any future rule
    # that reopens a decided rung will hit the same wall, so the exemption is asked of
    # the guard rather than restated here.
    if (base in per and per[base].get(rung)
            and not (rung == "0" and trial_guard.unscreened(base))):
        continue
    # A unit with no hidden suite can never be trialled - task_lint BLOCKs it inside the
    # sweep, so counting it as runnable made orchestrate relaunch four go-github cohorts
    # every tick, each exiting immediately with "guard kept 0 unit(s)" while burning a
    # launch and a lock. Five units dataset-wide are in this state (auditcoerce staged into
    # four cohorts with no tests/ dir at all, plus auditentry); they need a verifier, not
    # a trial.
    if not glob.glob(d + "tests/hidden/**/*", recursive=True):
        continue

    # A cohort whose LAST sweep kept zero units is barren: every unit in it is refused
    # downstream, by task_lint rather than by trial_guard. sweep_grokgogit's ten units all
    # BLOCK on B6 ("L0 bug report has no reproduce command"), so orchestrate counted them
    # runnable, launched at concurrency 8, the lint dropped all ten, the sweep exited, and
    # the next tick did it again — three launches, zero trials, while sweep_escalate_L4
    # waited behind it. Same shape as the tests/hidden bug above; a second way to be
    # un-trialable that the same check does not cover.
    #
    # Self-healing on purpose: the skip lapses as soon as the cohort is touched, so
    # repairing a unit makes it eligible again without anyone clearing state.
    if barren(cohort):
        continue

    # Ladder rungs (L1, L3-L6) are affordance-study data: they can never certify, by
    # construction. They were taking ~half of Composer's budget while the dataset was the
    # goal, so they pause behind a flag file rather than being deleted - the staged
    # cohorts and every verdict already collected stay exactly where they are.
    #   pause : touch outputs/supervisor/ladder_paused
    #   resume: rm outputs/supervisor/ladder_paused
    # The pause is about affordance-study sampling on units that already certified.
    # An escalation cohort wears the same rung suffixes but is the opposite thing: the
    # only route by which a unit that failed both L0 and L2 ever certifies. trial_guard
    # arbitrates which single rung each escalating unit may take next.
    if (LADDER_PAUSED and rung in ("1", "3", "4", "5", "6")
            and not cohort.startswith(ESCALATION_PREFIX)):
        continue
    try:
        ok, _ = trial_guard.decide(name, per)
    except Exception:
        ok = True
    if ok:
        runnable[cohort] = runnable.get(cohort, 0) + 1
        # ...and separately per solver. A cohort can be fully runnable and still have
        # nothing THIS solver may take, since the no-contamination rule admits only a
        # solver that failed the unit low.
        for _a in ("devin", "cursor"):
            try:
                if solver_match.decide(name, _a)[0]:
                    runnable_by.setdefault(_a, {})[cohort] = \
                        runnable_by.setdefault(_a, {}).get(cohort, 0) + 1
            except Exception:
                runnable_by.setdefault(_a, {})[cohort] = \
                    runnable_by.setdefault(_a, {}).get(cohort, 0) + 1

print("=" * 70)
print(f"ORCHESTRATE  {time.strftime('%H:%M:%S')}   {'APPLY' if APPLY else 'dry run'}")
print("=" * 70)
print(f"  devin        {devin_total} run(s) = {len(sessions)} session(s) + "
      f"{trials} trial(s)   [cap {DEVIN_CAP}]")
for s in sessions:
    k = "oswt-" + s.replace("closure_", "")
    p = prog.get(k, {})
    print(f"                 {s.replace('closure_',''):22s} idle {p.get('idle_min',0):5.1f}m  tools {p.get('tools',0)}")
print(f"  composer     {containers} container(s), budget {'OK' if budget_ok else 'SPENT'}")
print(f"  sweeps       {' '.join(sweeps) if sweeps else 'NONE'}")
print(f"  throttle     {throttle_age//60}m ago" if throttle_age < 99999 else "  throttle     none")
print(f"  runnable     {sum(runnable.values())} unit(s) across {len(runnable)} cohort(s)")

# ---------- decide -----------------------------------------------------------------
# 2. stuck sessions: no tool call and no activity for 25 min
for s in sessions:
    k = "oswt-" + s.replace("closure_", "")
    p = prog.get(k)
    if p and p["idle_min"] > 25:
        ACTIONS.append(("restart-stuck", s, f"idle {p['idle_min']:.0f}m — stuck, not thinking"))

# 3+4. trials must be running
active_trial_cohorts = set(sweeps)
if throttle_age < 1800:
    NOTES.append(f"throttle {throttle_age//60}m ago — inside the 30m cooloff, not starting devin work")
else:
    for cohort, n in sorted(runnable.items(), key=lambda kv: -kv[1]):
        if cohort in active_trial_cohorts or sweep_locked(cohort):
            continue
        is_l2 = cohort.endswith("_L2")
        # escalate.py names a cohort after the solver that FAILED its units low, because
        # a certificate is strongest when the same solver fails at L0 and passes higher:
        # the flip then isolates the affordance rather than a capability gap. Route by
        # that name instead of sending every ladder rung to Composer by default.
        wants_devin = "_devin_" in cohort or cohort.endswith("_devin")
        wants_composer = "_composer_" in cohort or cohort.endswith("_composer")
        # L2 prefers Devin because Devin is free, but "prefers" must not mean "only":
        # Devin is capped at 4 and every certifiable unit in the dataset is now at L2, so a
        # Devin-only rule left 24 units queued behind the cap while Composer sat idle with
        # budget in hand. L2 is the rung that certifies; when Devin is full and there is
        # budget, Composer takes the overflow. Ladder rungs never qualify - they cannot
        # certify - and are filtered out before this point when paused.
        # DEVIN_CAP, not a literal 4. slots.CAP was raised to 6 and this kept routing to
        # 4, so Devin sat at 4/6 looking like it was "filling gradually" when the router
        # was simply never offering it the last two slots. The cap being defined in one
        # place is worth nothing if the consumer hardcodes it anyway.
        devin_full = (DEVIN_CAP - devin_total) < 1
        if wants_composer:
            to_devin = False
        elif wants_devin:
            to_devin = True
        else:
            # L0/L1 is SCREENING, open to any solver, and Devin is free. Sending it to
            # Composer by default created a self-reinforcing starvation: Composer screened
            # 447 L0 trials to Devin's 51, so Composer came to own 225 of the 250 units
            # whose L0 failed — and L2 ownership follows the L0 failure. Devin was then
            # left with ~22 units it was ever eligible for and ran dry at 5/6 slots.
            # Screening on the free solver also conserves Composer's paid budget for the
            # ladder work only Composer may do.
            is_screen = not is_l2 and not re.search(r"_L[3-6](_|$)", cohort)
            if is_screen and not devin_full:
                to_devin = True
            else:
                to_devin = (is_l2 or not budget_ok) and not (is_l2 and devin_full and budget_ok)
        # Never launch a sweep its solver may take nothing from. It would trial nothing,
        # log "guard kept 0 unit(s)", and cohorts.barren() would then mark the cohort
        # barren for EVERY solver — stranding work that the other one could do. Two
        # fixes landing in the same hour, interacting badly.
        eligible = runnable_by.get("devin" if to_devin else "cursor", {}).get(cohort, 0)
        if eligible < 1:
            other = "composer" if to_devin else "devin"
            NOTES.append(f"{cohort}: {n} unit(s) but none this solver may certify "
                         f"— belongs to {other}")
            continue
        if to_devin:
            room = DEVIN_CAP - devin_total
            if room < 1:
                NOTES.append(f"{cohort}: {n} unit(s) waiting, no devin headroom ({devin_total}/{DEVIN_CAP})")
                continue
            room = min(room, 2)
            ACTIONS.append(("sweep-devin", cohort,
                            f"{n} unit(s), concurrency {room}, {'2' if is_l2 else '1'} round(s)"))
            devin_total += room
        else:
            room = min(8, 12 - containers)
            if room < 2:
                NOTES.append(f"{cohort}: {n} unit(s) waiting, composer at {containers}/12")
                continue
            # Price the launch before making it. Container count was the only cap, so one
            # decision at concurrency 6 committed ~61M against a 100M budget and overshot
            # it by 23%. A sweep runs every runnable unit in the cohort, so the commitment
            # is units x per-trial cost, not concurrency x cost.
            room, why_budget = token_cost.can_afford(cohort, n, room)
            if room < 1:
                NOTES.append(f"{cohort}: {why_budget}")
                continue
            why = f"{n} unit(s), concurrency {room}"
            if is_l2:
                why += " [L2 overflow: devin full, spending budget on certification]"
            ACTIONS.append(("sweep-composer", cohort, why))
            containers += room
        if len(ACTIONS) >= 3:
            break

# 3b. nothing runnable anywhere means the trial pipeline is starved at the source: units
# are authored and verified but never converted into task directories. Staging is
# mechanical and free, so the reconciler does it rather than waiting for a human.
# Stage on a LOW-WATER mark, not on empty. Waiting for zero meant 28 fully-stageable
# go-github units sat idle because one runnable unit remained somewhere else; by the time
# runway hits zero the trial slots are already starving.
if sum(runnable.values()) < 5:
    staged_names = {os.path.basename(d.rstrip("/")) for d in glob.glob("experiments/dose_response/sweep_*/")}
    # The same cohort exists in several worktrees: the authoring copy has no contracts,
    # the reconciler copy does. Picking whichever glob landed first meant reading
    # AU4gogithub2's contract-less copy while RCgogithub4 held 28 finished contracts, so
    # 28 stageable units stayed "blocked". Prefer the most complete copy of each cohort.
    by_cohort = {}
    for src in sorted(glob.glob("/home/evan/Documents/oswt-*/experiments/pipeline/authored_batch*/*/")
                      + glob.glob("experiments/pipeline/authored_batch*/*/")):
        repo = os.path.basename(src.rstrip("/"))
        batch = os.path.basename(os.path.dirname(src.rstrip("/")))
        if repo.startswith("_"):
            continue
        score = len(glob.glob(src + "*/_author/contract.md")) + len(
            glob.glob(src + "*/tests/test.sh"))
        key = (batch, repo)
        if key not in by_cohort or score > by_cohort[key][1]:
            by_cohort[key] = (src, score)
    cand = None
    for src in [v[0] for v in by_cohort.values()]:
        repo = os.path.basename(src.rstrip("/"))
        batch = os.path.basename(os.path.dirname(src.rstrip("/")))
        if repo.startswith("_"):
            continue
        units = [u for u in glob.glob(src + "*/") if os.path.isdir(u)]
        # Match the stager's own definition exactly: _author/ plus a verifier-written
        # tests/ suite (tests/test.sh and tests/hidden/*_test.go). A recursive glob for
        # *_test.go also matches the repo's own tests inside environment/src, which made
        # this claim 15 verified goa units and hand the stager a cohort it rejected.
        # The stager reads a fixed set of files per unit and dies on the first missing one.
        # Encode ALL of them, per rung, rather than discovering each precondition by crash:
        # L0 needs the bug report; L2 additionally needs the contract. The goa cohort has
        # hidden suites but no contract.md, so it is L0-only and can never certify until a
        # contract is written for it.
        # stage_units reads contract.md unconditionally (line 244, before the rung branch),
        # so it is required even for an L0-only stage. A cohort with hidden suites but no
        # contract is NOT stageable at any rung - it needs the reconciler stage first.
        # Staging it anyway just fails on the first unit, which it did twice.
        def ready(u):
            need = ["_author/gold.patch", "_author/bugreport.md",
                    "_author/contract.md", "tests/test.sh"]
            return (os.path.isdir(u + "_author")
                    and all(os.path.exists(u + n) for n in need)
                    and glob.glob(u + "tests/hidden/**/*_test.go", recursive=True))
        withtests = [u for u in units if ready(u)]
        needs_contract = [u for u in units
                          if os.path.isfile(u + "tests/test.sh")
                          and glob.glob(u + "tests/hidden/**/*_test.go", recursive=True)
                          and not os.path.exists(u + "_author/contract.md")]
        if len(withtests) < 8 and len(needs_contract) >= 8:
            globals()["_needs_contract"] = (src, repo, batch, len(needs_contract))
        if len(withtests) < 8:
            continue
        tag = f"sweep_{repo.replace('-','')}_{batch[-1]}"
        if tag in staged_names:
            # A cohort staged earlier may hold far fewer units than are now ready: an
            # earlier VF pass staged 2 go-github units, then the reconciler finished 28.
            # Treating the tag as "done" left 26 stageable units invisible. Stage the
            # remainder under a suffixed tag rather than skipping the cohort.
            have = len([d for d in glob.glob(f"experiments/dose_response/{tag}/*/")
                        if os.path.isdir(d)])
            if len(withtests) < have + 8:
                continue
            for suf in "bcdef":
                if f"{tag}{suf}" not in staged_names:
                    tag = f"{tag}{suf}"
                    break
            else:
                continue
        # skip cohorts already in the dataset under a repo-prefixed name
        sample = os.path.basename(withtests[0].rstrip("/"))
        if sample in per or f"{repo}-{sample}" in per:
            continue
        cand = (src, tag, len(withtests))
        break
    if cand:
        ACTIONS.append(("stage", cand[1], f"{cand[2]} verified unit(s) from {cand[0].split('oswt-')[-1][:40]}"))
        globals()["_stage_src"] = cand[0]
    elif globals().get("_needs_contract"):
        src, repo, batch, n = globals()["_needs_contract"]
        # Only report it if no reconciler job is already queued for this cohort, otherwise
        # the action repeats every tick forever and looks like progress when it is not.
        base = f"RC{repo.replace('-','')}{batch[-1]}"
        MANI = "/home/evan/devin-tasks/queue/manifest.tsv"
        mani = open(MANI).read() if os.path.exists(MANI) else ""
        # A queued tag is not a finished tag. dq2's done-check is "does the report file
        # exist", so a reconciler that writes 8 of 38 contracts and files its report is
        # retired as complete and the other 30 units stay unstageable forever - which is
        # exactly how 74 units accumulated. If the tag's report exists but contracts are
        # still missing, the job is done AND wrong: queue a fresh suffixed tag for the
        # remainder instead of reporting "already queued" every tick.
        tag = None
        for cand_tag in [base] + [base + s for s in "bcdef"]:
            row = re.search(rf"^{re.escape(cand_tag)}\t(?:[^\t]*\t){{2}}[^\t]*\t[^\t]*\t[^\t]*\t([^\t]*)",
                            mani, re.M)
            if not row:
                tag = cand_tag
                break                      # never queued -> use it
            wt_c = f"/home/evan/Documents/oswt-{cand_tag}"
            if not os.path.exists(os.path.join(wt_c, row.group(1))):
                tag = None                 # queued and still running -> wait
                NOTES.append(f"{batch}/{repo}: {n} unit(s) need contracts; "
                             f"{cand_tag} queued and not yet finished")
                break
            # finished but contracts still missing -> fall through to the next suffix
        if tag is None:
            pass
        else:
            ACTIONS.append(("needs-contract", f"{batch}/{repo}",
                        f"{n} unit(s) have hidden suites but NO contract.md — "
                        f"cannot stage at any rung; queue a reconciler job"))
    else:
        NOTES.append("no runnable units and nothing verified left to stage — "
                     "waiting on VF jobs to finish writing hidden suites")

if len(sessions) + trials > 4:
    # Deliberately not an action: dq2.sh caps NEW launches and never kills, because a
    # killed session restarts from scratch and loses its work. It drains on its own.
    NOTES.append(f"devin at {len(sessions)+trials}/{DEVIN_CAP} (sessions drain via dq2) — "
                 f"holding new devin work until it falls below 4")

if devin_total < 2 and throttle_age >= 1800:
    ACTIONS.append(("devin-idle", "-", f"only {devin_total} devin run(s); check dq2.sh and the manifest"))

# ---------- act --------------------------------------------------------------------
over = len(sessions) + trials > 4
print("\nACTIONS" if ACTIONS else
      ("\nno action available (see notes)" if (NOTES or over) else "\nall invariants hold"))
for kind, target, why in ACTIONS:
    print(f"  {kind:16s} {target:24s} {why}")
    if kind == "sweep-devin":
        rounds = 2 if target.endswith("_L2") else 1
        room = int(re.search(r"concurrency (\d+)", why).group(1))
        pid = run(f"AGENT=devin bash {freeze()} {target} {rounds} {room} "
                  f"> outputs/{target}.log 2>&1")
        if APPLY:
            sweep_lock(target, pid, room, agent="devin")
        sh(f"sed -i '/^{target}$/d' outputs/supervisor/sweep_queue.txt")
    elif kind == "sweep-composer":
        room = int(re.search(r"concurrency (\d+)", why).group(1))
        pid = run(f"bash {freeze()} {target} 1 {room} > outputs/{target}.log 2>&1")
        if APPLY:
            sweep_lock(target, pid, room)
        sh(f"sed -i '/^{target}$/d' outputs/supervisor/sweep_queue.txt")
    elif kind == "needs-contract" and APPLY:
        # Actually queue the reconciler job. Reporting it every tick is not autonomy: two
        # VF jobs are running with pre-patch briefs and will finish without contracts, and
        # a cohort with suites but no contract can never be staged or certified.
        src, repo, batch, n = globals()["_needs_contract"]
        tag = f"RC{repo.replace('-','')}{batch[-1]}"
        brief = f"/home/evan/devin-tasks/closure_{tag}.md"
        wt = f"/home/evan/Documents/oswt-{tag}"
        rel = src.split("experiments/")[-1].rstrip("/")
        open(brief, "w").write(f"""# Job: write contracts for the {n} verified {repo} units in {batch}

Worktree: {wt} (branch {tag.lower()}). Read AGENTS.md. Use `uv run`.
Docker allowed. Never run: git stash, git reset, git checkout -- <path>, git restore,
git clean, git commit. NO solver trials.

## Why this job exists

`experiments/{rel}/<unit>/` has `_author/gold.patch`, `_author/bugreport.md` and a verified
hidden suite in `tests/`, but **no `_author/contract.md`**. stage_units.py reads contract.md
unconditionally, so these units cannot be staged at ANY rung and can never certify -
certification requires an L2 pass and L2 *is* the contract. Copy the units in from
{src.rstrip('/')} first.

## What to write

`_author/contract.md` per unit: prose stating every behavioural commitment the hidden suite
grades, plus a coverage table pairing each hidden test name with the row that justifies it.
Copy the shape from an existing contract under experiments/pipeline/authored_batch3/kops/.

- **One row per assertion, both directions.** A contract row with no test is a lie; a test
  with no row is an ambush. 90% of units failing exactly one test at L0 pass at L2 precisely
  because the contract supplied the missing commitment.
- **Never state an arbitrary literal.** An exact error string, type spelling, print layout or
  punctuation is unguessable: assert its SHAPE, never the sentence. Two solver traces showed
  units passing 7 of 8 tests and failing only on an exact Print layout and a panic choice.
- **Behaviour, not symbols**: no file names, line numbers or private identifiers (B7).
- You MAY read gold.patch; the reconciler is allowed to, unlike the verifier.

## Deliverable

A contract per unit, plus `analytics/research/contracts_{tag.lower()}.md`: per unit the
commitment count, coverage-table size against the hidden assertion count, and any assertion
you could not justify with a derivable commitment (flag it, do not paper over it).
Log to outputs/{tag}.log. Do not commit.
""")
        sh(f"grep -qP '^{tag}\\t' /home/evan/devin-tasks/queue/manifest.tsv || "
           f"printf '{tag}\\t{brief}\\t{wt}\\t{tag.lower()}\\tswe-2-max\\t14400\\t"
           f"analytics/research/contracts_{tag.lower()}.md\\t-\\n' "
           f">> /home/evan/devin-tasks/queue/manifest.tsv")
        sh(f"cd {R} && git worktree add -q -b {tag.lower()} {wt} HEAD 2>/dev/null; "
           f"mkdir -p {wt}/outputs")
    elif kind == "stage":
        src = globals().get("_stage_src")
        if src:
            # Stage unit by unit. The stager raises on the first bad unit, so a single
            # stale excision.patch (auditcoerce) cost 26 of 28 ready go-github units.
            # --units isolates each one; a failure costs that unit only.
            rungs = globals().get("_stage_rungs", "0,2")
            names = sorted(os.path.basename(d.rstrip("/"))
                           for d in glob.glob(src.rstrip("/") + "/*/") if os.path.isdir(d))
            script = (f'for u in {" ".join(names)}; do '
                      f'uv run python scripts/ops/stage_units.py "{src.rstrip("/")}" '
                      f'experiments/dose_response/{target} --rungs {rungs} --jobs 2 '
                      f'--units "$u" >> outputs/stage_{target}.log 2>&1 '
                      f'|| echo "SKIP $u (unstageable)" >> outputs/stage_{target}.log; done')
            run(script)
    elif kind == "restart-stuck" and APPLY:
        # dq2.sh relaunches from the manifest on its next pass; just clear the stuck one.
        sh(f"pgrep -f 'prompt-file /home/evan/devin-tasks/{target}.md' | xargs -r kill")

for n in NOTES:
    print(f"  note: {n}")
print("=" * 70)
