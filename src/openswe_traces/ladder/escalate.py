#!/usr/bin/env python3
"""Escalate units that fail L0 AND L2 to a higher affordance rung.

Why
---
A unit that fails L0 and passes L2 is a flip certificate: hard from a bug report,
solvable from a contract. A unit that fails BOTH was being written off as
"non-flipping" - 48 of them. The evidence says that verdict is wrong. Nine of them
were escalated by hand and ALL NINE flipped higher up:

    gin-enginecfg L5 | gin-negotiate L5 | helm-chartdl L5 | helm-chartrepo L5
    helm-depresolver L3,L5 | helm-httpgetter L5 | kops-difftext L5
    kops-taintparse L5 | kops-clustervalid L3 fail, L4 fail, L5 PASS, L6 PASS

11 passes against 2 fails. They are not broken tasks - they are the HARDEST tasks
in the dataset, and the rung at which a unit finally flips is a difficulty measure
rather than a failure.

Search policy
-------------
L0 then L2 is the certifying path and is unchanged; escalation only begins once L2
has failed. From there it climbs ONE RUNG AT A TIME: L2 -> L3 -> L4 -> L5 -> L6.

An earlier version probed L5 first and bisected downward, on the grounds that 9 of
9 hand-escalated units had flipped at L5 and that it reached the same answer in
~2.6 trials per unit instead of 4. That is cheaper but it is not the same
experiment: jumping the ladder means the rung a unit lands on was never shown to be
the rung it needs, only a rung that works. The affordance ladder is the measurement
here, so every step gets walked.

    known_fail = highest rung with a fail       (starts at 2)
    known_pass = lowest rung above it with a pass, if any
    gap of 1   -> settled, the minimum flipping rung is known_pass
    otherwise  -> trial known_fail + 1, the next rung up
    past L6    -> the unit genuinely does not flip at any affordance

Staging
-------
Rungs of one unit differ in three files - instruction.md, affordance.json,
validation.json - and from L5 up in which hidden tests are restored into the tree.
Everything else is shared, so a higher rung is built from the unit's existing L2
directory via ``build_affordance_levels``; no authored source or re-excision is
needed. Where reclaim_disk has already dropped that L2's ``environment/src``, this
calls restore_env_src.py first.

Usage::

    escalate.py                      # what would be escalated, and to which rung
    escalate.py --apply              # stage the next rung for every candidate
    escalate.py --apply --limit 10
    escalate.py --report             # rung at which each escalated unit flipped
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import subprocess
import sys

from openswe_traces.ladder import ledger as TL
from openswe_traces.paths import REPO

SWEEPS = REPO / "experiments" / "dose_response"
# Rungs an escalation may use. L1 is a ladder-study rung, not an escalation step.
LADDER = (3, 4, 5, 6)
ESCALATION_DEST = "sweep_escalate"
# Devin is free, so when both solvers failed a unit low, Devin gets the escalation.
SOLVER_PREFERENCE = ("devin", "composer")


def history(d: dict) -> dict[int, bool]:
    """rung -> passed?, for the rungs that matter to an escalation."""
    out = {}
    for r, rewards in d.items():
        if not r.isdigit() or not rewards:
            continue
        out[int(r)] = max(rewards) > 0
    return out


def next_rung(h: dict[int, bool]) -> tuple[int | None, str]:
    """-> (rung to trial next, why). None means nothing left to learn."""
    if h.get(0) is not False:
        return None, "not a candidate: L0 did not fail"
    fails = [r for r, ok in h.items() if not ok and r >= 2]
    passes = [r for r, ok in h.items() if ok and r >= 2]
    if not fails:
        return None, "not a candidate: L2 has no fail on record"
    known_fail = max(fails)
    if known_fail >= max(LADDER):
        return None, f"exhausted: fails at L{known_fail}, the top rung"
    above = [r for r in passes if r > known_fail]
    if above and min(above) - known_fail == 1:
        return None, f"settled: flips at L{min(above)}, fails at L{known_fail}"
    # One rung at a time, whether or not something higher is already known to pass.
    # A unit that passed L5 with L3 and L4 untried has not yet been shown to NEED L5.
    nxt = known_fail + 1
    if nxt in h:
        return None, f"settled: L{nxt} already has a verdict"
    gap = f" (L{min(above)} passes, narrowing)" if above else ""
    return nxt, f"step to L{nxt} (fails through L{known_fail}){gap}"


def failing_solver(base: str, bys) -> str:
    """Which solver should be asked the higher rung: the one that FAILED this unit low.

    A certificate is strongest when the same solver fails at L0 and passes higher up —
    the flip then isolates the affordance. Escalating on a solver that never failed the
    unit produces a cross-solver certificate instead, which may only record that the
    second model is stronger. So escalation follows the failure.

    Of 46 candidates, 42 failed on composer alone and 4 on both. When both failed, the
    free solver takes it.
    """
    d = bys.get(base, {})
    failed = set()
    for solver, rungs in d.items():
        for r, rewards in rungs.items():
            if r.isdigit() and int(r) <= 2 and rewards and max(rewards) == 0:
                failed.add(solver)
    for pref in SOLVER_PREFERENCE:
        if pref in failed:
            return pref
    return "composer"


def dest_root_for(solver: str, rung: int) -> pathlib.Path:
    """Cohorts are per-solver so orchestrate can route them without per-unit logic."""
    return SWEEPS / f"{ESCALATION_DEST}_{solver}_L{rung}"


def l2_dir(base: str) -> pathlib.Path | None:
    """The unit's L2 staging dir - the A0-shaped source every higher rung is cut from."""
    cands = sorted(SWEEPS.glob(f"*/{base}-L2"))
    # prefer one that still has its tree; restoring is cheap but not free
    for c in cands:
        if (c / "environment" / "src").is_dir() and any((c / "environment" / "src").iterdir()):
            return c
    return cands[0] if cands else None


def ensure_env_src(unit: pathlib.Path) -> bool:
    src = unit / "environment" / "src"
    if src.is_dir() and any(src.iterdir()):
        return True
    if not (unit / ".reclaimed.json").is_file():
        return False
    r = subprocess.run(["uv", "run", "python", "scripts/ops/restore_env_src.py", str(unit)],
                       cwd=REPO, capture_output=True, text=True, timeout=3600)
    return "ok (" in r.stdout


def stage(base: str, rung: int, solver: str = "composer") -> tuple[bool, str]:
    """Build <ESCALATION_DEST>_<solver>_L<rung>/<base>-L<rung> from the unit's L2 dir."""
    dest_root = dest_root_for(solver, rung)
    dest = dest_root / f"{base}-L{rung}"
    # Already staged under the old solver-agnostic name, or the other solver's? Leave it.
    for other in (SWEEPS / f"{ESCALATION_DEST}_L{rung}",
                  *(dest_root_for(sv, rung) for sv in SOLVER_PREFERENCE)):
        d2 = other / f"{base}-L{rung}"
        if d2 != dest and (d2 / "instruction.md").is_file():
            return True, f"already staged at {d2.relative_to(REPO)}"
    if dest.is_dir() and (dest / "instruction.md").is_file():
        return True, f"already staged at {dest.relative_to(REPO)}"
    src = l2_dir(base)
    if src is None:
        return False, "no L2 directory to cut from"
    hidden = sorted((src / "tests" / "hidden").rglob("*_test.go"))
    if not hidden:
        return False, "no hidden suite in the L2 dir"
    if not ensure_env_src(src):
        return False, "L2 environment/src missing and not restorable"

    from openswe_traces.synth.affordance import build_affordance_levels

    dest_root.mkdir(parents=True, exist_ok=True)
    try:
        built = build_affordance_levels(
            src, [str(p.relative_to(src / "tests" / "hidden")) for p in hidden],
            levels=[rung - 2], dest_root=dest_root, family=base, name_scheme="L",
        )
    except Exception as e:  # noqa: BLE001 - one bad unit must not stop the batch
        return False, f"build failed: {type(e).__name__}: {e}"
    made = built.get(rung - 2)
    if made is None or not made.is_dir():
        return False, "builder produced nothing"
    # The directory name is authoritative for the rung; a stale `level` field in
    # affordance.json mis-scored a whole cohort once. Write it to agree.
    for f in ("affordance.json", "validation.json"):
        p = made / f
        if p.is_file():
            try:
                j = json.loads(p.read_text())
                j["level"] = rung
                p.write_text(json.dumps(j, indent=2) + "\n")
            except Exception:
                pass
    return True, f"staged {made.relative_to(REPO)}"


def candidates(per):
    for base, d in sorted(per.items()):
        h = history(d)
        rung, why = next_rung(h)
        if rung is not None:
            yield base, rung, why, h


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--apply", action="store_true")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--report", action="store_true")
    args = ap.parse_args()

    per = TL.ledger()

    if args.report:
        print(f"{'unit':30s} {'flips at':>9s}  history")
        n = 0
        for base, d in sorted(per.items()):
            h = history(d)
            if h.get(0) is not False:
                continue
            passes = [r for r, ok in h.items() if ok and r > 2]
            if not passes:
                continue
            n += 1
            hs = " ".join(f"L{r}={'PASS' if ok else 'fail'}" for r, ok in sorted(h.items()))
            print(f"{base:30s} {'L' + str(min(passes)):>9s}  {hs}")
        print(f"\n{n} unit(s) certified above L2")
        return 0

    bys = TL.ledger_by_solver()
    rows = [(b, r, w, h, failing_solver(b, bys)) for b, r, w, h in candidates(per)]
    if args.limit:
        rows = rows[:args.limit]
    print(f"escalation candidates: {len(rows)}")
    ok = bad = 0
    for base, rung, why, h, solver in rows:
        if args.apply:
            done, msg = stage(base, rung, solver)
            ok, bad = ok + done, bad + (not done)
            print(f"  {'OK  ' if done else 'FAIL'} {base:28s} -> L{rung} [{solver}]  {msg}")
        else:
            print(f"  {base:28s} -> L{rung} [{solver}]   {why}")
    if args.apply:
        print(f"\nstaged {ok}, failed {bad}")
    else:
        print("\n(dry run — rerun with --apply)")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
