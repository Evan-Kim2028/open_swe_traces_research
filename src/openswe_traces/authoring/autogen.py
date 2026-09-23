#!/usr/bin/env python3
"""Keep the pipeline fed without a human in the loop.

The pipeline is author -> verify -> stage -> trial. Every stage was hand-queued, so it
stalled whenever nobody noticed a stage had finished. This closes the loop: each tick it
looks at what exists on disk, works out the next missing artifact, and appends exactly one
job to the Devin manifest, newest stage first so work drains rather than piles up.

Stage order matters. Verifying beats authoring, and staging beats verifying, because an
authored unit that nobody verifies is worth nothing while a staged unit is worth a trial.

  pipeline_autogen.py            append at most one job if a slot will free
  pipeline_autogen.py --status   print the pipeline census and do nothing
"""
import os, sys, glob, json, subprocess, re

R = "/home/evan/Documents/open_swe_traces_research"
MANIFEST = "/home/evan/devin-tasks/queue/manifest.tsv"
BRIEFS = "/home/evan/devin-tasks"
TARGET_CERTIFIED = 500

# Repos ranked by measured yield, best first. gin and helm are deliberately last:
# 34%/33% yield and helm costs 18 trials per certified unit against bbolt's 3.2.
REPO_ROTATION = ["go-github", "kops", "go-git", "bbolt", "nats-server", "client-go", "goa"]


def sh(c):
    try:
        return subprocess.run(c, shell=True, capture_output=True, text=True, timeout=60, cwd=R).stdout
    except Exception:
        return ""


def manifest_jobs():
    jobs = {}
    if os.path.exists(MANIFEST):
        for ln in open(MANIFEST):
            if ln.startswith("#") or not ln.strip():
                continue
            f = ln.rstrip("\n").split("\t")
            if len(f) >= 7:
                jobs[f[0]] = {"brief": f[1], "wt": f[2], "deliv": f[6]}
    return jobs


def pending(jobs):
    """Jobs queued but not finished — work already in the pipe."""
    out = []
    for n, j in jobs.items():
        d = os.path.join(j["wt"], j["deliv"])
        if j["deliv"] not in ("", "-") and os.path.exists(d):
            continue
        if not os.path.exists(j["brief"]):
            continue
        out.append(n)
    return out


def census():
    """What exists at each pipeline stage, per cohort."""
    c = {"authored": {}, "verified": {}, "staged": {}}
    # Scan the main checkout AND every worktree. A VF job writes its suites inside its own
    # worktree, so a main-only census reports 0 verified for a cohort already being verified
    # and would queue a duplicate job against it.
    roots = [R] + sorted(glob.glob("/home/evan/Documents/oswt-*"))
    for root in roots:
        # Glob authored* not authored_batch*: the per-repo au5 batches and the Grok
        # output (authored_grokgogit, ...) do not carry the word "batch", so a
        # batch-only census silently ignored 122 then 20 units - they read as
        # "not authored" and verification was never queued for them.
        for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/"):
            repo = os.path.basename(d.rstrip("/"))
            batch = os.path.basename(os.path.dirname(d.rstrip("/")))
            if repo.startswith("_"):
                continue
            # The per-repo au5 batches were consolidated into authored_batch5. Counting
            # both queues a second VF job for work already in flight, so fold the alias
            # exactly as roots.py does rather than keeping two answers.
            if batch.startswith("authored_au5"):
                batch = "authored_batch5"
            units = [u for u in glob.glob(d + "*/") if os.path.isdir(u)]
            withtests = [u for u in units
                         if glob.glob(u + "_author/tests/**/*_test.go", recursive=True)
                         or glob.glob(u + "tests/**/*_test.go", recursive=True)
                         or glob.glob(u + "**/*_test.go", recursive=True)]
            key = f"{batch}/{repo}"
            # best observation across roots wins: verification may live in a worktree
            c["authored"][key] = max(c["authored"].get(key, 0), len(units))
            c["verified"][key] = max(c["verified"].get(key, 0), len(withtests))
    for d in glob.glob(f"{R}/experiments/dose_response/sweep_*/"):
        n = os.path.basename(d.rstrip("/"))
        u = [x for x in glob.glob(d + "*/") if os.path.isdir(x)]
        if u:
            c["staged"][n] = len(u)
    return c


def certified_count():
    sys.path.insert(0, os.path.join(R, "scripts/ops"))
    try:
        from openswe_traces.ladder import ledger as trial_ledger
        return len(trial_ledger.summary()["certified"])
    except Exception:
        return -1


RULES = """
## Two rules from measurement, not taste

**1. Never re-excise a closure the dataset already has.** Run `scripts/check_unit_overlap.py` and
grep `experiments/pipeline/authored*/*/*/_author/closure.md` for the file and symbol you mean to
cut. A duplicate re-measures a unit we already own. List every candidate you rejected for
overlap and what it collided with.

**2. Report your own diminishing returns.** Number units in authoring order and record how many
candidates you rejected before each acceptance. Pooled across the dataset, later units currently
yield BETTER than earlier ones (42% early half, 52% late half) because briefs improved faster
than closures ran out. The moment your search cost per accepted unit starts climbing, say so and
stop rather than padding the count — that inflection is worth more than three weak units.

## Surface selection, from the per-repo funnel

Parsing / predicate / serialisation closures certify far better than orchestration glue:
go-github 71%, kops 60%, go-git 56%, bbolt 52% against gin 34% and helm 33% — and helm costs 18
trials per certified unit where bbolt costs 3.2. Prefer a pure function over a method that
touches a client, a config tree or a scheduler.

**`DETAILS.md` carries an `Inferable:` line per commitment** (yes / doc / partially / no). Never
list an arbitrary choice as a behavioural commitment: an exact error string, a type spelling, a
print layout or punctuation is `Inferable: no`, and the verifier asserts only its shape.
"""


def write_author_job(repo, tag):
    # Tell the author what the dataset has already taken. Nothing did, so every job saw an
    # empty repo and went for the same obvious targets: 20% of all units excise line
    # ranges that overlap another unit's, and one vendored gin file alone carries 16.
    import subprocess as _sp
    try:
        _claimed = _sp.run(["uv", "run", "python", "scripts/ops/excision_registry.py",
                            repo, "--md"], cwd=R, capture_output=True, text=True,
                           timeout=300).stdout.strip()
    except Exception:
        _claimed = ""
    if not _claimed:
        _claimed = "## Already claimed\n\nNothing yet — the repo is open."
    brief = f"{BRIEFS}/closure_{tag}.md"
    open(brief, "w").write(f"""# Job: author 20 NEW units in {repo}

Worktree: /home/evan/Documents/oswt-{tag} (branch {tag.lower()}). Read AGENTS.md. Use `uv run`.
Docker allowed. Never run: git stash, git reset, git checkout -- <path>, git restore, git clean,
git commit. NO solver trials.

Read `analytics/research/PIPELINE.md` first.
{RULES}
## Deliverables

`experiments/pipeline/authored_{tag.lower()}/{repo}/<unit>/_author/` with gold.patch, cheat.patch,
api.md, DETAILS.md (annotated), bugreport.md, closure.md. The cheat special-cases the worked
examples ONLY — validate with `scripts/ops/cheat_validity.py`; a size ratio >= 0.6 against gold
means you wrote an implementation, not a cheat.

Writeup `analytics/research/authored_{tag.lower()}.md`: per unit in authoring order — closure,
surface type, rejections before acceptance, overlap collisions, Inferable breakdown, cheat ratio.
End with whether search cost per accepted unit was rising. Log to outputs/{tag}.log. Do not commit.

{_claimed}
""")
    return brief


def write_verify_job(batch, repo, tag):
    brief = f"{BRIEFS}/closure_{tag}.md"
    open(brief, "w").write(f"""# Job: hidden test suites for the {repo} units in {batch}

Worktree: /home/evan/Documents/oswt-{tag} (branch {tag.lower()}). Read AGENTS.md. Use `uv run`.
Docker allowed. Never run: git stash, git reset, git checkout -- <path>, git restore, git clean,
git commit. NO solver trials — you write tests, you never solve.

Units: `experiments/pipeline/{batch}/{repo}/<unit>/_author/`. Read `analytics/research/PIPELINE.md`.

## The rule that decides whether these units are worth anything

**You are blind to gold.patch. Do not open it.** You grade what DETAILS.md commits to and
nothing more. That wall is what makes a unit fair, and it is where every bad unit in the dataset
came from: a verifier that could not see gold invented an assertion no solver could derive.

**Obey the `Inferable:` annotation on every DETAILS line.**
- `yes` / `doc` — assert the behaviour exactly.
- `partially` — assert the derivable part; do not pin the rest.
- `no` — **assert SHAPE, never the literal.** Assert that an error is returned naming the
  offending field; never that it equals a specific sentence.

Measured: reading solver traces, units passed 7 of 8 tests and failed on one unstated
convention — an exact Print layout, panic-on-unsorted-input. Across the dataset 51% of failing L0
trials fail exactly one test, and 90% of those are legitimately solvable at L2. A unit whose
difficulty is one unguessable literal measures spec-guessing, not engineering.

## Deliverables per unit

`tests/hidden/<pkg>_bb_test.go` with `TestDetail01..NN` numbered to match DETAILS lines (the
numbering is load-bearing — it traces a failure back to a commitment), plus `tests/test.sh`.

**And `_author/contract.md`, if and only if it does not already exist.** This is not optional:
`stage_units.py` reads contract.md unconditionally, so a unit without one cannot be staged at
ANY rung and can never certify — certification requires an L2 pass and L2 *is* the contract.
Four cohorts (90+ units) were authored with suites and no contract and were dead on arrival.

Write it AFTER the suite, from DETAILS.md and the suite you just wrote — never from gold:
one prose commitment per assertion, plus a coverage table pairing each hidden test name with
the row that justifies it. A contract row with no test is a lie; a test with no row is an
ambush. For any DETAILS line marked `Inferable: no`, state the SHAPE ("returns an error naming
the offending field"), never the literal. No file names or line numbers in the prose (B7).
Verify each in Docker: excised+hidden -> FAIL, gold -> PASS, cheat -> FAIL, gold touches no test
file (A12). Report any unit that cannot satisfy all four rather than bending a test to fit.

Writeup `analytics/research/verified_{tag.lower()}.md`: per unit — assertions written, the
Inferable breakdown and what you asserted for each `no`, the four Docker results, and any
DETAILS line you refused to assert, with the reason. Log to outputs/{tag}.log. Do not commit.
""")
    return brief


def append(tag, brief, deliv):
    wt = f"/home/evan/Documents/oswt-{tag}"
    rows = open(MANIFEST).read() if os.path.exists(MANIFEST) else ""
    # A parked row is still a row. Parking is done by commenting the line
    # (`#PARKED-AU5goa\t...`), which `^AU5goa\t` never matches, so append() concluded the
    # tag was absent and wrote a fresh LIVE duplicate - silently resurrecting every job
    # that had been parked. Seven AU5 rows came back this way and held three Devin slots
    # on the lowest-priority stage. Match the tag whether or not it carries a park prefix.
    if re.search(rf"^(?:#\S*\s*)?{re.escape(tag)}\t", rows, re.M):
        return False
    with open(MANIFEST, "a") as fh:
        fh.write(f"{tag}\t{brief}\t{wt}\t{tag.lower()}\tswe-2-max\t14400\t{deliv}\t-\n")
    subprocess.run(f"git worktree add -q -b {tag.lower()} {wt} HEAD", shell=True, cwd=R,
                   capture_output=True)
    os.makedirs(f"{wt}/outputs", exist_ok=True)
    return True


def main():
    c = census()
    cert = certified_count()
    jobs = manifest_jobs()
    pend = pending(jobs)

    if "--status" in sys.argv:
        print(f"certified {cert} / target {TARGET_CERTIFIED}")
        print(f"pending devin jobs ({len(pend)}): {' '.join(sorted(pend)[:8])}")
        print("\nauthored -> verified, per cohort:")
        for k in sorted(c["authored"]):
            print(f"  {k:28s} {c['verified'].get(k,0):3d}/{c['authored'][k]:3d} verified")
        print(f"\nstaged cohorts: {len(c['staged'])}")
        return 0

    if cert >= TARGET_CERTIFIED:
        print(f"target reached ({cert} >= {TARGET_CERTIFIED}) — not generating work")
        return 0
    # Keep a shallow backlog: enough that no slot idles, not so much that the queue
    # becomes a wishlist nobody reaches.
    # Backlog cap reflects TOTAL worker capacity, not one pool. With Devin (4) and Grok
    # running jobs concurrently, a cap of 6 starved Grok: 4 pending jobs were already
    # executing on Devin, leaving only 2 for anyone else.
    # The cap applies to AUTHORING only. It used to gate the whole function, and since
    # authoring is the stage that generates its own backlog, ten pending AU jobs blocked
    # the verify scan below indefinitely: 74 units sat authored-but-unverified (no hidden
    # suite, so unstageable at any rung) while seven more AU jobs queued behind them and
    # the dataset stayed flat. Verification drains the backlog, so it is never blocked by it.
    author_pend = [n for n in pend if n.startswith("AU")]

    # A cohort already in the trial ledger was verified through an earlier path, with its
    # tests living in the staged sweep dir rather than _author/. The census cannot see that
    # and reports it unverified; without this filter autogen would re-verify the whole
    # existing dataset.
    roots_all = [R] + sorted(glob.glob("/home/evan/Documents/oswt-*"))
    sys.path.insert(0, os.path.join(R, "scripts/ops"))
    try:
        from openswe_traces.ladder import ledger as trial_ledger
        trialled = set(trial_ledger.ledger().keys())
    except Exception:
        trialled = set()

    def already_banked(batch, repo):
        # Look across every root: a cohort can live only in a worktree, and checking just
        # the main checkout found no units and wrongly declared it un-recorded.
        units = set()
        for root in roots_all:
            for x in glob.glob(f"{root}/experiments/pipeline/{batch}/{repo}/*/"):
                if os.path.isdir(x):
                    units.add(os.path.basename(x.rstrip("/")))
        if not units:
            return False
        # Staged units carry a repo prefix the authored directory does not:
        # authored_batch2/go-github/auditentry becomes go-github-auditentry in the ledger.
        # Comparing bare names matched nothing and declared the whole dataset unverified.
        def seen(u):
            return (u in trialled
                    or f"{repo}-{u}" in trialled
                    or any(t.endswith("-" + u) for t in trialled))
        hit = sum(1 for u in units if seen(u))
        return hit >= 0.5 * len(units)

    # Report the same number the authoring gate uses, so the monitor and the reconciler
    # never disagree about how deep the backlog really is. It sits here because it needs
    # already_banked, and ahead of the verify scan because that scan can return early.
    if "--unverified" in sys.argv:
        print(sum(max(0, c["authored"][k] - c["verified"].get(k, 0))
                  for k in c["authored"] if not already_banked(*k.split("/"))))
        return 0

    # 1. verification first: an authored unit nobody verifies is worth nothing
    for key in sorted(c["authored"]):
        batch, repo = key.split("/")
        if already_banked(batch, repo):
            continue
        a, v = c["authored"][key], c["verified"].get(key, 0)
        if a - v >= 8:
            base = f"VF{batch.replace('authored_','').replace('_','')}{repo.replace('-','')}"
            # A tag in the manifest is not a cohort that got verified. VFbatch3gogithub
            # filed its report having written suites for 2 of 21 units; `if tag in jobs:
            # continue` then skipped the cohort permanently and the other 19 stayed
            # unstageable. If the tag is finished but units are still unverified, the job
            # is done AND incomplete - rotate to a suffixed tag for the remainder. Only a
            # job still running is a reason to wait.
            tag = None
            for cand in [base] + [base + x for x in "bcdef"]:
                if cand not in jobs:
                    tag = cand
                    break
                j = jobs[cand]
                if not os.path.exists(os.path.join(j["wt"], j["deliv"])):
                    break                      # still running -> wait for it
            if tag is None:
                continue
            brief = write_verify_job(batch, repo, tag)
            if append(tag, brief, f"analytics/research/verified_{tag.lower()}.md"):
                print(f"queued VERIFY {tag}: {a-v} unverified {repo} units in {batch}")
                return 0

    # Do not author while contracts are outstanding. Authoring adds units that cannot be
    # staged or certified until a contract exists, and it competes for the same Devin
    # slots as the RC jobs that unblock ~100 already-authored units. Parking the AU rows
    # by hand did nothing because this function simply generated new ones each tick.
    rc_pending = [n for n in pend if n.startswith("RC")]
    if rc_pending:
        print(f"contracts outstanding ({' '.join(rc_pending)}) — not authoring more units")
        return 0

    # Authoring is the lowest-priority stage and must never outrank verification. An
    # authored unit with no hidden suite cannot be staged, trialled or certified, so
    # queueing one while a VF job waits for a Devin slot actively delays the only stage
    # that creates certifiable stock. This gate was missing: with the backlog at 30 the
    # unverified check passed and autogen queued three AU7 jobs ahead of two pending VFs.
    vf_pending = [n for n in pend if n.startswith("VF")]
    if vf_pending:
        print(f"verification outstanding ({' '.join(vf_pending)}) — not authoring more units")
        return 0

    if len(author_pend) >= 10:
        print(f"{len(author_pend)} authoring job(s) already pending — not authoring more")
        return 0

    # Never author past the unverified backlog. An authored unit with no hidden suite is
    # not an asset, it is a liability that occupies a Devin slot the verifier needs.
    # Count only cohorts that are NOT already recorded. batch2 was verified through an
    # older path whose tests live in the staged sweep dir, so the census reports ~150
    # unverified units that are in fact certified - counting those would block authoring
    # permanently rather than when the backlog is real.
    unverified = sum(max(0, c["authored"][k] - c["verified"].get(k, 0))
                     for k in c["authored"] if not already_banked(*k.split("/")))
    if unverified >= 40:
        print(f"{unverified} authored unit(s) still unverified — not authoring more")
        return 0

    # 2. otherwise author more, rotating by measured yield
    n = 5
    while n < 40:
        for repo in REPO_ROTATION:
            tag = f"AU{n}{repo.replace('-','')}"
            if tag in jobs:
                continue
            brief = write_author_job(repo, tag)
            if append(tag, brief, f"analytics/research/authored_{tag.lower()}.md"):
                print(f"queued AUTHOR {tag}: 20 new {repo} units (certified {cert}/{TARGET_CERTIFIED})")
                return 0
        n += 1
    print("rotation exhausted")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
