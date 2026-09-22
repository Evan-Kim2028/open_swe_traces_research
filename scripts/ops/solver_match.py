#!/usr/bin/env python3
"""May THIS solver trial this unit? The rule that keeps certificates single-solver.

A certificate is "fails at L0, passes at L2 or above". It is only evidence about the
AFFORDANCE when the same solver does both. When one model fails L0 and a different one
passes L2, the flip may record nothing but the second model being stronger — a real
certificate, but a weaker claim, and `trial_ledger.certificates()` labels it `cross`.

Cross-solver certificates reached 25 of 193 and were climbing, because L0 screening and
L2 certification were routed by whoever had a free slot. attachsvc: Composer failed L0,
Devin passed L2. channelver: the reverse. Neither says anything about the contract.

The rule, stated once here and enforced at the sweep gate:

    L0  anyone may screen — there is no prior failure to be consistent with
    L2+ only a solver that has FAILED this unit at a lower rung

which is the same rule escalate.py already applies to L3-L6, moved down to the rung
where most certificates are actually earned.

The cost is the L2 overflow valve: when Devin is full, Composer can no longer absorb
Devin's L2 units. That valve existed because Devin's cap of 4 left 24 units queued while
Composer sat idle with budget. It is a throughput loss taken deliberately — a certificate
that does not isolate the affordance is not worth the slot it saved.

Usage::

    solver_match.py <agent> <unit-dir-or-name> [...]   # prints KEEP/DROP per unit
    solver_match.py --audit                            # any RUNNING sweep about to
                                                       # contaminate? exit 1 if so
"""

from __future__ import annotations

import os
import pathlib
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# harbor's --agent value -> the solver name trial_ledger records
AGENT_TO_SOLVER = {"devin": "devin", "cursor": "composer", "cursor-cli": "composer",
                   "cursor-agent": "composer", "composer": "composer", "grok": "grok"}


def solver_for(agent: str) -> str:
    return AGENT_TO_SOLVER.get((agent or "").strip().lower(), "composer")


def failed_low(base: str, bys, below: int) -> set[str]:
    """Solvers that have a recorded FAILURE for this unit at any rung under `below`."""
    out = set()
    for solver, rungs in bys.get(base, {}).items():
        for r, rewards in rungs.items():
            if r.isdigit() and int(r) < below and rewards and max(rewards) == 0:
                out.add(solver)
    return out


def decide(unit: str, agent: str, bys=None) -> tuple[bool, str]:
    """-> (may this agent trial this unit, why)."""
    name = pathlib.Path(unit).name
    if "-L" not in name:
        return True, "not a rung-suffixed unit"
    base, suffix = name.rsplit("-L", 1)
    rung = suffix[:1]
    if not rung.isdigit():
        return True, "unparsed rung"
    if int(rung) < 2:
        return True, "L0/L1: screening, no prior failure to match"
    if bys is None:
        import trial_ledger as TL
        bys = TL.ledger_by_solver()
    me = solver_for(agent)
    failed = failed_low(base, bys, int(rung))
    if not failed:
        # No failure below this rung yet. trial_guard refuses L2 before L0 anyway; if it
        # somehow gets here, allowing it cannot create a cross certificate on its own.
        return True, f"no recorded failure below L{rung}"
    if me in failed:
        return True, f"{me} failed this unit below L{rung}"
    return False, (f"cross-solver: {me} would certify a unit failed by "
                   f"{'/'.join(sorted(failed))} — routes to them instead")


def audit() -> int:
    """Is any RUNNING sweep holding units its own solver may not certify?

    This is the signal worth watching, and it is not "did the gate fire". In the healthy
    case the gate fires NEVER: orchestrate steers each sweep to a cohort its solver can
    certify, so sweep_seq's gate is a backstop with nothing to do. Counting gate hits
    therefore reads zero both when everything is right and when the gate is broken.

    Asking instead whether a live sweep is CARRYING something it should not separates
    those two states. Non-zero means orchestrate mis-steered, or a sweep predates the
    gate and needs its pending set cleaned by hand — which is exactly what ten queued
    Devin trials needed when the rule first landed.
    """
    import re
    import slots
    import trial_ledger as TL
    bys = TL.ledger_by_solver()
    per = TL.ledger()
    owner = {}
    for agent in ("devin", "cursor"):
        for t in slots.trials(agent):
            owner[re.sub(r"_r\d+(_\d+)?$", "", t["job"])] = agent
    total = 0
    for cohort, agent in sorted(owner.items()):
        d = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(
            os.path.abspath(__file__)))), "experiments/dose_response", cohort)
        if not os.path.isdir(d):
            continue
        risky = []
        for u in sorted(os.listdir(d)):
            if "-L" not in u:
                continue
            base, suf = u.rsplit("-L", 1)
            rung = suf[:1]
            if not rung.isdigit() or int(rung) < 2:
                continue
            if per.get(base, {}).get(rung):          # already decided
                continue
            if not decide(u, agent, bys)[0]:
                risky.append(u)
        if risky:
            total += len(risky)
            print(f"AT RISK {cohort} ({agent}): {len(risky)} unit(s) it may not certify "
                  f"— e.g. {risky[0]}")
    if not total:
        print("audit: no running sweep holds a unit its solver may not certify")
    return 1 if total else 0


def main() -> int:
    if "--audit" in sys.argv:
        return audit()
    if len(sys.argv) < 3:
        print(__doc__.strip().splitlines()[-1])
        return 2
    agent, units = sys.argv[1], sys.argv[2:]
    import trial_ledger as TL
    bys = TL.ledger_by_solver()
    bad = 0
    for u in units:
        ok, why = decide(u, agent, bys)
        bad += not ok
        print(f"{'KEEP' if ok else 'DROP'} {pathlib.Path(u).name:30s} {why}")
    return 1 if bad else 0


if __name__ == "__main__":
    raise SystemExit(main())
