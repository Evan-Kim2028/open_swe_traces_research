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
  1. Devin runs 2-4 concurrent, counting host sessions AND in-container trials, because
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


FROZEN = "/home/evan/.claude/jobs/a1eaeb86/tmp/sweep_seq.frozen.sh"


def freeze():
    """Run sweeps from a snapshot. Editing scripts/ops/sweep_seq.sh while a sweep was
    executing shifted bash's read offset and killed the trial phase with a syntax error
    after the gate had already run. A frozen copy cannot be corrupted mid-flight."""
    try:
        import shutil
        if (not os.path.exists(FROZEN)
                or os.path.getmtime("scripts/ops/sweep_seq.sh") > os.path.getmtime(FROZEN)):
            shutil.copy("scripts/ops/sweep_seq.sh", FROZEN)
    except Exception:
        return "./scripts/ops/sweep_seq.sh"
    return FROZEN


def run(c):
    """Fire and forget; the reconciler must not block on a sweep."""
    if APPLY:
        subprocess.Popen(c, shell=True, cwd=R, start_new_session=True,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


# ---------- observe ----------------------------------------------------------------
sessions = [x for x in sh("pgrep -af '[d]evin --model' | grep -oP 'closure_\\w+' | sort -u").split() if x]
trials = int(sh("ps -eo args | awk '/harbor run/ && !/awk/ {a=\"\";c=1;j=\"\"; "
                "for(i=1;i<NF;i++){if($i==\"--agent\")a=$(i+1); if($i==\"--n-concurrent\")c=$(i+1); "
                "if($i==\"--job-name\")j=$(i+1)} if(a==\"devin\" && !(j in seen)){seen[j]=1; t+=c}} "
                "END{print t+0}\'") or 0)
containers = int(sh("docker ps --format '{{.Names}}' | grep -c env-main") or 0)
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
import trial_ledger, trial_guard
per = trial_ledger.ledger()
runnable = {}
for d in glob.glob("experiments/dose_response/sweep_*/*/"):
    if not os.path.isdir(d):
        continue
    cohort = d.split(os.sep)[2]
    name = os.path.basename(d.rstrip("/"))
    if name.rsplit("-L", 1)[0] in per:
        continue
    try:
        ok, _ = trial_guard.decide(name, per)
    except Exception:
        ok = True
    if ok:
        runnable[cohort] = runnable.get(cohort, 0) + 1

print("=" * 70)
print(f"ORCHESTRATE  {time.strftime('%H:%M:%S')}   {'APPLY' if APPLY else 'dry run'}")
print("=" * 70)
print(f"  devin        {devin_total} run(s) = {len(sessions)} session(s) + {trials} trial(s)   [target 2-4]")
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
        if cohort in active_trial_cohorts:
            continue
        is_l2 = cohort.endswith("_L2")
        if is_l2 or not budget_ok:
            room = 4 - devin_total
            if room < 1:
                NOTES.append(f"{cohort}: {n} unit(s) waiting, no devin headroom ({devin_total}/4)")
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
            ACTIONS.append(("sweep-composer", cohort, f"{n} unit(s), concurrency {room}"))
            containers += room
        if len(ACTIONS) >= 3:
            break

# 3b. nothing runnable anywhere means the trial pipeline is starved at the source: units
# are authored and verified but never converted into task directories. Staging is
# mechanical and free, so the reconciler does it rather than waiting for a human.
if sum(runnable.values()) == 0:
    staged_names = {os.path.basename(d.rstrip("/")) for d in glob.glob("experiments/dose_response/sweep_*/")}
    cand = None
    for src in sorted(glob.glob("/home/evan/Documents/oswt-*/experiments/pipeline/authored_batch*/*/")
                      + glob.glob("experiments/pipeline/authored_batch*/*/")):
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
            continue
        # skip cohorts already in the bank under a repo-prefixed name
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
        tag = f"RC{repo.replace('-','')}{batch[-1]}"
        mani = open("/home/evan/devin-tasks/queue/manifest.tsv").read() if os.path.exists(
            "/home/evan/devin-tasks/queue/manifest.tsv") else ""
        if re.search(rf"^{re.escape(tag)}\t", mani, re.M):
            NOTES.append(f"{batch}/{repo}: {n} unit(s) need contracts; {tag} already queued")
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
    NOTES.append(f"devin at {len(sessions)+trials}/4 (sessions drain via dq2 MAXN=2) — "
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
        run(f"AGENT=devin bash {freeze()} {target} {rounds} {room} "
            f"> outputs/{target}.log 2>&1")
        sh(f"sed -i '/^{target}$/d' outputs/supervisor/sweep_queue.txt")
    elif kind == "sweep-composer":
        room = int(re.search(r"concurrency (\d+)", why).group(1))
        run(f"bash {freeze()} {target} 1 {room} > outputs/{target}.log 2>&1")
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
            run(f"uv run python scripts/ops/stage_units.py '{src.rstrip('/')}' "
                f"experiments/dose_response/{target} --rungs {globals().get('_stage_rungs','0,2')} --jobs 4 "
                f"> outputs/stage_{target}.log 2>&1")
    elif kind == "restart-stuck" and APPLY:
        # dq2.sh relaunches from the manifest on its next pass; just clear the stuck one.
        sh(f"pgrep -f 'prompt-file /home/evan/devin-tasks/{target}.md' | xargs -r kill")

for n in NOTES:
    print(f"  note: {n}")
print("=" * 70)
