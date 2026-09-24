#!/usr/bin/env python3
"""Is each unit's ESCALATION LADDER the work of one model?

The single-solver rule in solver_match answers a narrower question than it looks. It asks
whether the solver taking rung k has FAILED this unit somewhere below k — enough to stop a
certificate recording "model B is stronger than model A", which is what it was written
for. It does not ask whether the same model did every rung.

So a unit can have Composer at L0, L2 and L3 and Devin at L4, pass that check, and still
have a ladder assembled from two models. That matters because the ladder is the difficulty
measurement: "fails at L4, flips at L5" is a claim about ONE agent's competence curve. Mix
two models into it and the rung where a unit flips may just be the rung where the stronger
model happened to take over.

Three kinds of "mixed" are not the same thing, and lumping them together overstates the
problem by more than 2x:

  ladder-mixed   an ESCALATION rung (L3+) done by a solver other than the one that owns
                 the rest of the ladder. This is the real defect.
  screen-mixed   two solvers at L0 only. That is the second-screen experiment working as
                 designed — a deliberate falsification test, not contamination.
  L2-mixed       two solvers attempted L2. Harmless: the certificate binds to whoever
                 passed, and trial_ledger already labels the single/cross distinction.

    ladder_purity.py            # full report
    ladder_purity.py --brief    # one line
    ladder_purity.py --fix-plan # what to re-run, and by whom, to make ladders pure
"""

from __future__ import annotations

import collections
import os
import sys



def classify():
    from openswe_traces.ladder import ledger as TL
    bys = TL.ledger_by_solver()
    certs = TL.certificates()
    out = {"pure": [], "ladder_mixed": [], "screen_mixed": [], "l2_mixed": [], "flat": []}
    detail = {}
    for base in sorted(certs):
        byrung = collections.defaultdict(set)
        for solver, rungs in bys.get(base, {}).items():
            for r, v in rungs.items():
                if r.isdigit() and v:
                    byrung[int(r)].add(solver)
        detail[base] = {r: sorted(s) for r, s in sorted(byrung.items())}
        esc = {r: s for r, s in byrung.items() if r >= 3}
        if not esc:
            out["flat"].append(base)
            continue
        # who owns the ladder: the solver appearing at the most escalation rungs
        tally = collections.Counter()
        for s in esc.values():
            tally.update(s)
        owner = tally.most_common(1)[0][0]
        stray = {r: sorted(s - {owner}) for r, s in esc.items() if s - {owner}}
        if stray:
            out["ladder_mixed"].append((base, owner, stray))
        elif len(byrung.get(0, set())) > 1:
            out["screen_mixed"].append(base)
        elif len(byrung.get(2, set())) > 1:
            out["l2_mixed"].append(base)
        else:
            out["pure"].append(base)
    return out, detail


def fix_plan(out, detail):
    """For each ladder-mixed unit: which rungs the ladder owner still owes."""
    plan = []
    for base, owner, stray in out["ladder_mixed"]:
        for rung in sorted(stray):
            if owner not in detail[base].get(rung, []):
                plan.append((base, rung, owner, stray[rung]))
    return plan


def main() -> int:
    out, detail = classify()
    n = sum(len(v) for v in out.values())
    if "--brief" in sys.argv:
        print(f"ladder purity: {len(out['ladder_mixed'])} mixed of "
              f"{len(out['pure'])+len(out['ladder_mixed'])+len(out['screen_mixed'])+len(out['l2_mixed'])}"
              f" laddered units ({len(out['flat'])} flat)")
        return 0
    print(f"certified units: {n}")
    print(f"  flat (L0/L2 only, no ladder)      {len(out['flat']):4d}")
    print(f"  PURE single-model ladder          {len(out['pure']):4d}")
    print(f"  screen-mixed (2 solvers at L0)    {len(out['screen_mixed']):4d}  "
          f"<- second screen, by design")
    print(f"  L2-mixed (2 solvers at L2)        {len(out['l2_mixed']):4d}  <- harmless")
    print(f"  LADDER-MIXED (stray L3+ rung)     {len(out['ladder_mixed']):4d}  <- the defect")
    for base, owner, stray in out["ladder_mixed"]:
        print(f"      {base:24s} owner={owner:9s} stray={stray}")
        print(f"        full: {detail[base]}")
    # COVERAGE. Contamination was the question while one merged ladder had to be kept
    # clean; with per-solver ladders it cannot happen, so the live question became how
    # many units actually HAVE a second curve. A unit climbed by one solver tells us it is
    # hard for that solver. Only a unit climbed by both separates "hard for agents" from
    # "hard for this agent".
    from openswe_traces.ladder import ledger as TL
    cbs = TL.certificates_by_solver()
    two = {b: d for b, d in cbs.items() if len(d) > 1}
    print(f"\ncoverage — independent difficulty curves:")
    print(f"  units with 1 curve   {sum(1 for d in cbs.values() if len(d) == 1):4d}")
    print(f"  units with 2 curves  {len(two):4d}  <- the comparison set")
    for b, d in sorted(two.items())[:10]:
        cells = "  ".join(f"{s}:L{v['rung']}{'' if v['rung_established'] else '?'}"
                          for s, v in sorted(d.items()))
        agree = len({v["rung"] for v in d.values()}) == 1
        print(f"      {b:24s} {cells}   {'agree' if agree else 'DIFFER'}")
    if two:
        diff = sum(1 for d in two.values() if len({v['rung'] for v in d.values()}) > 1)
        print(f"  of {len(two)} compared, {diff} put the unit at a DIFFERENT rung per solver")

    if "--fix-plan" in sys.argv:
        plan = fix_plan(out, detail)
        print(f"\nto make every ladder pure, re-run {len(plan)} rung(s):")
        for base, rung, owner, who in plan:
            print(f"  {base:24s} L{rung} by {owner}   (currently only {'/'.join(who)})")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
