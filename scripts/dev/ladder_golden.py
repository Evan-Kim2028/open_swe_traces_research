#!/usr/bin/env python3
"""Characterization check for the ladder code: prove a refactor changed no decision.

The ladder, guard and certificate logic has no unit tests, and its inputs (the jobs tree,
the supervisor rosters, docker) move while an experiment runs. Comparing old and new code
against live state would report real progress as a regression. This pins the inputs:

  snapshot  build a frozen root: trial results hard-linked, rosters copied, everything
            else in the repo symlinked read-only. Nothing in the snapshot is written.
  dump      import the ladder modules from --ops, run them with the snapshot as cwd, and
            write every certificate, ledger row and guard/match decision as JSON.

Usage:
  ladder_golden.py snapshot <repo> <root>
  ladder_golden.py dump <root> <ops-dir> <out.json> [--src <src-dir>]

Two dumps from the same snapshot must be identical: `diff old.json new.json`.
"""
import json
import os
import shutil
import sys

JOBS = "experiments/dose_response/jobs"
ROSTERS = ("ladder_backfill", "second_screen_enrolled", "devin_cap_override",
           "known_failures.json")


def snapshot(repo, root):
    """Freeze the inputs the ladder code reads, relative to cwd."""
    repo, root = os.path.abspath(repo), os.path.abspath(root)
    if os.path.exists(root):
        shutil.rmtree(root)
    os.makedirs(root)
    for name in os.listdir(repo):
        if name not in ("experiments", "outputs"):
            os.symlink(os.path.join(repo, name), os.path.join(root, name))
    for name in os.listdir(os.path.join(repo, "experiments")):
        if name != "dose_response":
            os.makedirs(os.path.join(root, "experiments"), exist_ok=True)
            os.symlink(os.path.join(repo, "experiments", name),
                       os.path.join(root, "experiments", name))
    dr = os.path.join(repo, "experiments", "dose_response")
    for name in os.listdir(dr):
        if name != "jobs":
            os.makedirs(os.path.join(root, "experiments", "dose_response"), exist_ok=True)
            os.symlink(os.path.join(dr, name),
                       os.path.join(root, "experiments", "dose_response", name))
    n = 0
    for job in os.scandir(os.path.join(repo, JOBS)):
        if not job.is_dir():
            continue
        for trial in os.scandir(job.path):
            if not trial.is_dir():
                continue
            for rel in ("result.json", "verifier/reward.txt"):
                src = os.path.join(trial.path, rel)
                if os.path.exists(src):
                    dst = os.path.join(root, JOBS, job.name, trial.name, rel)
                    os.makedirs(os.path.dirname(dst), exist_ok=True)
                    os.link(src, dst)
                    n += 1
    sup = os.path.join(root, "outputs", "supervisor")
    os.makedirs(sup)
    for name in ROSTERS:
        src = os.path.join(repo, "outputs", "supervisor", name)
        if os.path.exists(src):
            shutil.copy2(src, sup)
    print(f"snapshot {root}: {n} trial files")


def _plain(x):
    """defaultdicts, sets and tuples to sorted plain JSON."""
    if isinstance(x, dict):
        return {str(k): _plain(v) for k, v in sorted(x.items(), key=lambda kv: str(kv[0]))}
    if isinstance(x, (set, frozenset)):
        return sorted(_plain(v) for v in x)
    if isinstance(x, (list, tuple)):
        return [_plain(v) for v in x]
    return x


def dump(root, ops, out, src=None):
    os.chdir(root)
    sys.path.insert(0, os.path.abspath(ops))
    if src:
        sys.path.insert(0, os.path.abspath(src))
    import trial_ledger as TL
    import trial_guard as TG
    import solver_match as SM
    import escalate as ES

    TG.inflight_cells = lambda: frozenset()   # docker state is live; not part of the check
    trials = sorted((dict(t, mtime=None) for t in TL.trials(JOBS)), key=lambda t: t["dir"])
    per = TL.ledger(JOBS)
    bys = TL.ledger_by_solver(JOBS)
    summary = TL.summary(JOBS)
    summary.pop("by_model")                   # Counter order is insertion order
    decisions = {}
    for base in sorted(bys):
        for rung in ("0", "2", "3", "4", "5", "6"):
            unit = f"{base}-L{rung}"
            for solver in ("composer", "devin"):
                decisions[f"guard {unit} {solver}"] = list(TG.decide(unit, per, solver))
            for agent in ("cursor", "devin"):
                decisions[f"match {unit} {agent}"] = list(SM.decide(unit, agent, bys))
        decisions[f"escalate {base}"] = list(ES.next_rung(ES.history(per[base])))
    consts = {k: repr(getattr(m, k)) for m, ks in (
        (TG, ("RUNG_TRIAL_CAP", "SCREENERS", "SOLVERS", "CERTIFYING_RUNGS", "REPAIR_BUDGET")),
        (SM, ("AGENT_TO_SOLVER",)),
        (ES, ("LADDER", "SOLVER_PREFERENCE", "ESCALATION_DEST"))) for k in ks}
    import slots
    # Absolute repo paths differ by checkout by design; record them relative to it. A wrong
    # _OVERRIDE silently ignores the devin cap override, so it is checked, not assumed.
    checkout = os.path.dirname(os.path.dirname(os.path.abspath(ops)))
    consts.update({k: os.path.relpath(str(v), checkout) for k, v in (
        ("escalate.REPO", ES.REPO), ("escalate.SWEEPS", ES.SWEEPS),
        ("slots._OVERRIDE", slots._OVERRIDE))})
    doc = {"trials": trials, "ledger": per, "ledger_by_solver": bys,
           "certificates": TL.certificates(JOBS),
           "certificates_by_solver": TL.certificates_by_solver(JOBS),
           "summary": summary, "decisions": decisions, "constants": consts}
    with open(out, "w") as fh:
        json.dump(_plain(doc), fh, indent=1, sort_keys=True)
    print(f"{out}: {len(trials)} trials, {len(decisions)} decisions")


if __name__ == "__main__":
    cmd, *rest = sys.argv[1:] or ["-h"]
    if cmd == "snapshot" and len(rest) == 2:
        snapshot(*rest)
    elif cmd == "dump" and len(rest) >= 3:
        src = rest[rest.index("--src") + 1] if "--src" in rest else None
        dump(rest[0], rest[1], rest[2], src)
    else:
        sys.exit(__doc__)
