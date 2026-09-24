#!/usr/bin/env python3
"""Would a SECOND solver also have failed this unit at L0?

Every certificate rests on one L0 failure, and screening has been routed by free capacity
rather than coverage, so in practice each unit has exactly one screener's opinion:
Composer ran 447 L0 trials to Devin's 51, and only two units have an L0 verdict from both.
"The solvers agree" has never been measured; its evidence is an absence of trials.

What a second verdict means changed when condemnation went per-solver, and this file has
been relabelled to match. It was built under a model-agnostic rule: a unit any frontier
agent fixed from the bug report alone was not hard, full stop, so a second screener's PASS
CONDEMNED the unit and took its certificate with it — p disagreements meant p inflated
headline count. That rule is retired. A certificate now says "this solver failed L0 and
passed L<k>", which a different solver's L0 pass cannot contradict, so nothing is
condemned and no certificate is withdrawn.

The trials are the same and they are worth more than before, because they answer the
question the retired rule assumed away: DOES HARDNESS TRANSFER BETWEEN MODELS? Both fail
L0 and the unit is hard in a way that outlives the model that screened it. One passes and
the hardness is a fact about a model, not about the task — the unit stays certified for
its screener and gains a second, much lower curve for the other. At rate p, p of the
dataset is model-specific difficulty. That is a headline result either way it comes out,
and it is the same cheap trial we were already buying.

This enrols units on the roster `trial_guard` consults, which opens their existing L0
directory to the solver that has not screened it. Nothing is copied and nothing is
renamed: the verdict lands on the real unit's ledger row, where it becomes the first point
of that solver's own ladder.

    second_screen.py --plan 20      # who would be screened, and by whom
    second_screen.py --enrol 20     # open them (orchestrate routes them next tick)
    second_screen.py --report       # disagreement rate once verdicts land
    second_screen.py --clear        # close the experiment
"""

from __future__ import annotations
from openswe_traces.paths import REPO as _REPO

import argparse
import collections
import glob
import os
import pathlib
import random
import sys

REPO = _REPO
SWEEPS = REPO / "experiments" / "dose_response"


REPO_PREFIXES = ("go-github-", "client-go-", "kops-", "helm-", "gin-", "goa-")


def _clopper_pearson(k: int, n: int, alpha: float = 0.05):
    """Exact two-sided binomial confidence interval, no scipy.

    Bisection on the binomial CDF. Needed because the interesting quantity here is a
    small count over a small sample, where the normal approximation is simply wrong:
    1/18 has a 95% upper bound near 27%, not the 16% a wald interval reports."""
    from math import comb

    def cdf(x, p):
        return sum(comb(n, i) * p ** i * (1 - p) ** (n - i) for i in range(0, x + 1))

    if n == 0:
        return 0.0, 1.0
    lo = 0.0
    if k > 0:
        a, b = 0.0, 1.0
        for _ in range(200):
            m = (a + b) / 2
            if cdf(k - 1, m) > 1 - alpha / 2:
                a = m
            else:
                b = m
        lo = a
    a, b = 0.0, 1.0
    for _ in range(200):
        m = (a + b) / 2
        if cdf(k, m) > alpha / 2:
            a = m
        else:
            b = m
    return lo, a


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
    from openswe_traces.ladder import guard as TG
    from openswe_traces.ladder import ledger as TL
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
    from openswe_traces.ladder import guard as TG
    return REPO / TG.SECOND_SCREEN_ROSTER


def enrol(n: int, seed: int) -> int:
    from openswe_traces.ladder import guard as TG
    cands = candidates()
    random.Random(seed).shuffle(cands)
    pick = cands[:n]
    p = roster_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    existing = TG.enrolled_for_second_screen()
    with open(p, "a") as fh:
        if not existing:
            fh.write("# units open for a second L0 screen by the solver that never saw\n"
                     "# them. A PASS here does NOT remove the certificate (that is\n"
                     "# per-solver); it starts the second solver's own ladder and marks\n"
                     "# this unit's difficulty as model-specific.\n")
        for base, _seen, _want in pick:
            if base not in existing:
                fh.write(base + "\n")
    added = len({b for b, _, _ in pick} - existing)
    print(f"enrolled {added} unit(s) for a second L0 screen -> {p}")
    print(f"  roster now {len(TG.enrolled_for_second_screen())} unit(s); "
          f"orchestrate will route them to the unused screener next tick")
    return 0


def report() -> int:
    from openswe_traces.ladder import guard as TG
    from openswe_traces.ladder import ledger as TL
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
        # "DISAGREE/NOT HARD/condemned" was the vocabulary of the model-agnostic rule and
        # is now simply wrong: the second screener's pass withdraws nothing. What the two
        # buckets separate is whether the difficulty is a property of the TASK or of the
        # MODEL that screened it, so they are named that way.
        print(f"  MODEL-SPECIFIC (second screener passed L0 — hard for its screener "
              f"only): {len(disagreed)}/{len(done)} = {rate:.0%}")
        # This rate is DIRECTIONAL and calling it "the solvers disagree" overstates it.
        # Screening was routed by free capacity and Composer had almost all of it, so
        # Composer is the first screener in 41 of the 42 pairs: every divergence so far is
        # "Devin solved at L0 what Composer could not", and the reverse direction is
        # measured by a single unit. Whether Composer would rescue Devin's L0 failures at
        # the same rate is unmeasured, so the number below is a one-way capability gap,
        # not a symmetric disagreement rate. Printed because it is the first thing a
        # reader will otherwise assume away.
        firsts = collections.Counter()
        for base in done:
            for s_, v in sorted(bys.get(base, {}).items()):
                if v.get("0") and max(v["0"]) <= 0:
                    firsts[s_] += 1
        if firsts:
            top, n_top = firsts.most_common(1)[0]
            if n_top >= 0.8 * len(done):
                print(f"    NB directional: {top} is the L0 failer in {n_top}/{len(done)} "
                      f"pairs, so this reads 'what the OTHER solver rescues from {top}';"
                      f" the reverse is ~unmeasured")
        print(f"  TRANSFERS      (both screeners failed L0 — hard beyond one model)"
              f"     : {len(agreed)}")
        # Per unit, say what the FAILING solver actually holds, because "the certificate
        # stands" is not true everywhere: hfs-dot was one of the four cross certificates
        # that the per-solver rewrite dropped, so its L0 failer has no certificate to
        # stand. A divergence on a unit nobody has certified is a different and weaker
        # observation than a divergence against a completed curve, and the report should
        # not flatten the two.
        bysolver = TL.certificates_by_solver()
        for base, r in disagreed:
            failers = [s_ for s_, v in r.items() if v <= 0]
            held = {s_: (bysolver.get(base) or {}).get(s_, {}).get("rung")
                    for s_ in failers}
            certs = [f"{s_} L{k}" for s_, k in held.items() if k is not None]
            note = ("certified " + ", ".join(certs) if certs
                    else "no certificate: the failer never passed a rung")
            print(f"    DIVERGES  {base:26s} {r}   ({note})")
        # POOLED IS THE WRONG NUMBER, and reporting it alone was actively misleading.
        # The first three divergences were all go-github, against zero from eleven units
        # across every other repo — Fisher's exact ~0.003. That is not a dataset-wide rate,
        # it is one repo whose bug reports give the fix away to one of the two solvers.
        # Extrapolating the pooled rate over the whole dataset predicted ~46 affected units
        # when the true exposure was 7. Break it down first, then extrapolate within a
        # family only if the families actually agree.
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
        # units in one repo. Whether a family is 100% or 80% divergent is not the
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
            print(f"    {fam:12s} {k}/{n} model-specific{flag}")
        n_cert = len(TL.certificates())
        if len(per) > 1 and worst:
            n_fam = sum(1 for b in TL.certificates() if repo_of(b) == worst)
            print(f"\n  the pooled {rate:.0%} is driven by '{worst}'. Do NOT extrapolate it "
                  f"across the dataset:")
            # These certificates are NOT at risk — that phrasing outlived the rule that
            # made it true. What is local to this family is the CLAIM THAT THE DIFFICULTY
            # GENERALISES; the certificates themselves are per-solver and stand.
            print(f"    {worst}: {n_fam} certificate(s) = {n_fam/n_cert*100:.0f}% of the "
                  f"dataset; they stand, but read their difficulty as screener-specific")
            others = sum(n for f, (n, k) in per.items() if f != worst)
            ok = sum(k for f, (n, k) in per.items() if f != worst)
            print(f"    elsewhere: {ok}/{others} model-specific — "
                  f"{'hardness transferred every time it was tested' if ok == 0 else 'see above'}")
            # Once the divergent family is identified and separated, the dataset-wide
            # claim rests on THIS column alone, and "0 out of n" is not a result until it
            # is attached to a bound. With zero divergences the exact one-sided 95% upper
            # bound is 1 - 0.05**(1/n) — the honest way to say how much model-specific
            # difficulty the sample could still be hiding. Printed here so the stopping point is visible
            # rather than recomputed by hand every time someone reads the report.
            if others:
                # A rate needs an interval, and the interval has to survive the count
                # becoming non-zero. The first draft only printed a bound while ok == 0,
                # so the statistics disappeared from the report at exactly the moment the
                # column stopped being clean and the number started to matter — the one
                # divergence outside go-github (hfs-dot) made the output LESS informative
                # than when there were none.
                lo, hi = _clopper_pearson(ok, others)
                print(f"    -> {ok}/{others} = {ok/others*100:.1f}%, 95% CI "
                      f"[{lo*100:.1f}%, {hi*100:.1f}%]")
                n_cert2 = len(TL.certificates())
                print(f"       over {n_cert2} certificates: ~{round(ok/others*n_cert2)} "
                      f"model-specific (point), up to {round(hi*n_cert2)} (upper bound)")
                if ok == 0:
                    need = 0
                    while _clopper_pearson(0, others + need)[1] > 0.05 and need <= 500:
                        need += 1
                    print(f"       {need} more clean screen(s) would put the bound "
                          f"under 5%")
                else:
                    # With a divergence banked, more clean screens still help but the
                    # arithmetic is much worse; say how much worse rather than implying
                    # the original target is still reachable.
                    need = 0
                    while _clopper_pearson(ok, others + need)[1] > 0.05 and need <= 2000:
                        need += 1
                    reach = f"{need} more clean screen(s)" if need <= 2000 else "not reachable"
                    print(f"       to get the bound under 5% from here: {reach}")
        else:
            print(f"\n  extrapolated to {n_cert} certificates: ~{round(rate * n_cert)} "
                  f"are hard for their screener only, not for both")
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


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
