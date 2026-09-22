#!/usr/bin/env python3
"""Would a SECOND solver also have failed this unit at L0?

Every certificate rests on one L0 failure. Condemnation is model-agnostic by design — a
unit any frontier agent fixes from the bug report alone is not hard — but screening has
been routed by free capacity rather than coverage, so in practice each unit has exactly
one screener's opinion: Composer ran 447 L0 trials to Devin's 51, and only two units have
an L0 verdict from both. "The solvers agree" has never been measured; its evidence is an
absence of trials.

If the two disagree at rate p, roughly p of the certified units were never hard and the
headline count is inflated by that much. p is cheap to measure and expensive to guess.

This enrols units on the roster `trial_guard` consults, which opens their existing L0
directory to the solver that has not screened it. Nothing is copied and nothing is
renamed: the verdict lands on the real unit's ledger row, so a pass condemns the unit for
real and it leaves the dataset. That is the measurement, not a side effect of it.

    second_screen.py --plan 20      # who would be screened, and by whom
    second_screen.py --enrol 20     # open them (orchestrate routes them next tick)
    second_screen.py --report       # disagreement rate once verdicts land
    second_screen.py --clear        # close the experiment
"""

from __future__ import annotations

import argparse
import glob
import os
import pathlib
import random
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
REPO = pathlib.Path(__file__).resolve().parents[2]
SWEEPS = REPO / "experiments" / "dose_response"


REPO_PREFIXES = ("go-github-", "client-go-", "kops-", "helm-", "gin-", "goa-")


def repo_of(base: str) -> str:
    for p in REPO_PREFIXES:
        if base.startswith(p):
            return p.rstrip("-")
    return "other"


def candidates():
    """(base, first screeners, the screener that has not tried it) for certified units.

    Restricted to certified units on purpose. The question that matters is not whether
    the solvers agree in general, it is how many of the certificates we are claiming
    would survive a second opinion — so the sample is drawn from the claim itself."""
    import trial_guard as TG
    import trial_ledger as TL
    certs = TL.certificates()
    bys = TL.ledger_by_solver()
    out = []
    for base in sorted(certs):
        if not glob.glob(str(SWEEPS / f"*/{base}-L0/tests/hidden")):
            continue                      # no runnable L0 dir left on disk
        seen = {s for s, r in bys.get(base, {}).items() if r.get("0")}
        want = [s for s in TG.SCREENERS if s not in seen]
        if want:
            out.append((base, sorted(seen), want))
    return out


def roster_path():
    import trial_guard as TG
    return REPO / TG.SECOND_SCREEN_ROSTER


def enrol(n: int, seed: int) -> int:
    import trial_guard as TG
    cands = candidates()
    random.Random(seed).shuffle(cands)
    pick = cands[:n]
    p = roster_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    existing = TG.enrolled_for_second_screen()
    with open(p, "a") as fh:
        if not existing:
            fh.write("# units open for a second L0 screen by the solver that never saw\n"
                     "# them. A PASS here condemns the unit and removes its certificate.\n")
        for base, _seen, _want in pick:
            if base not in existing:
                fh.write(base + "\n")
    added = len({b for b, _, _ in pick} - existing)
    print(f"enrolled {added} unit(s) for a second L0 screen -> {p}")
    print(f"  roster now {len(TG.enrolled_for_second_screen())} unit(s); "
          f"orchestrate will route them to the unused screener next tick")
    return 0


def report() -> int:
    import trial_guard as TG
    import trial_ledger as TL
    bys = TL.ledger_by_solver()
    roster = TG.enrolled_for_second_screen()
    if not roster:
        print("second screen: nobody enrolled")
        return 0
    done, pending, agreed, disagreed = [], [], [], []
    for base in sorted(roster):
        seen = {s: r["0"] for s, r in bys.get(base, {}).items() if r.get("0")}
        if len(seen) < 2:
            pending.append(base)
            continue
        done.append(base)
        (disagreed if any(max(v) > 0 for v in seen.values()) else agreed).append(
            (base, {s: max(v) for s, v in seen.items()}))
    print(f"second screen: {len(done)} verdict(s) in, {len(pending)} pending")
    if done:
        rate = len(disagreed) / len(done)
        print(f"  DISAGREE (second screener passed — unit was never hard): "
              f"{len(disagreed)}/{len(done)} = {rate:.0%}")
        print(f"  agree    (both failed L0 — hardness holds)            : {len(agreed)}")
        for base, r in disagreed:
            print(f"    NOT HARD  {base:26s} {r}")
        # POOLED IS THE WRONG NUMBER, and reporting it alone was actively misleading.
        # The first three condemnations were all go-github, against zero from eleven units
        # across every other repo — Fisher's exact ~0.003. That is not a dataset-wide
        # softness rate, it is one repo whose bug reports give the fix away. Extrapolating
        # the pooled rate over the whole dataset predicted ~46 bad certificates when the
        # true exposure was 7. Break it down first, then extrapolate within a family only
        # if the families actually agree.
        import collections
        per = collections.defaultdict(lambda: [0, 0])
        for base in done:
            per[repo_of(base)][0] += 1
        for base, _r in disagreed:
            per[repo_of(base)][1] += 1
        print("\n  by repo:")
        # Flag on HETEROGENEITY, not on a family being perfectly bad. The first cut of
        # this required k == n, so the moment one go-github unit agreed (4/5 instead of
        # 5/5) the whole warning switched off and the misleading pooled extrapolation
        # came back — "~51 of 217 would not survive" while the real exposure was six
        # units in one repo. Whether a family is 100% or 80% condemned is not the
        # question; whether the families DIFFER is.
        worst = None
        ranked = sorted(per.items(), key=lambda kv: (-(kv[1][1] / max(1, kv[1][0])),
                                                     -kv[1][0]))
        top_fam, (top_n, top_k) = ranked[0]
        rest_n = sum(n for f, (n, k) in per.items() if f != top_fam)
        rest_k = sum(k for f, (n, k) in per.items() if f != top_fam)
        top_rate = top_k / max(1, top_n)
        rest_rate = rest_k / max(1, rest_n)
        # enough of a gap, on enough units, to be worth separating
        if top_n >= 3 and rest_n >= 3 and top_k and top_rate >= max(0.5, 3 * rest_rate):
            worst = top_fam
        for fam, (n, k) in sorted(per.items(), key=lambda kv: -kv[1][0]):
            flag = ""
            if fam == worst:
                flag = f"  <-- {k/n*100:.0f}%, vs {rest_rate*100:.0f}% elsewhere"
            elif k:
                flag = f"  ({k/n*100:.0f}%)"
            print(f"    {fam:12s} {k}/{n} condemned{flag}")
        n_cert = len(TL.certificates())
        if len(per) > 1 and worst:
            at_risk = sum(1 for b in TL.certificates() if repo_of(b) == worst)
            print(f"\n  the pooled {rate:.0%} is driven by '{worst}'. Do NOT extrapolate it "
                  f"across the dataset:")
            print(f"    {worst}: {at_risk} certificate(s) still standing, all at risk "
                  f"= {at_risk/n_cert*100:.0f}% of the dataset")
            others = sum(n for f, (n, k) in per.items() if f != worst)
            ok = sum(k for f, (n, k) in per.items() if f != worst)
            print(f"    elsewhere: {ok}/{others} condemned — "
                  f"{'no evidence of softness' if ok == 0 else 'see above'}")
        else:
            print(f"\n  extrapolated to {n_cert} certificates: ~{round(rate * n_cert)} "
                  f"would not survive a second screen")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--plan", nargs="?", type=int, const=20)
    ap.add_argument("--enrol", type=int)
    ap.add_argument("--report", action="store_true")
    ap.add_argument("--clear", action="store_true")
    ap.add_argument("--seed", type=int, default=0)
    a = ap.parse_args()
    if a.clear:
        p = roster_path()
        if p.exists():
            p.unlink()
        print(f"cleared {p} — second screen closed, guard back to single-screen")
        return 0
    if a.report:
        return report()
    if a.enrol:
        return enrol(a.enrol, a.seed)
    n = a.plan or 20
    cands = candidates()
    random.Random(a.seed).shuffle(cands)
    print(f"certified units with a runnable L0 dir and an unused screener: {len(cands)}")
    print(f"would enrol {min(n, len(cands))}:")
    for base, seen, want in cands[:min(n, 10)]:
        print(f"  {base:26s} screened by {'/'.join(seen):10s} -> second {'/'.join(want)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
