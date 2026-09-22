#!/usr/bin/env python3
"""Does the remaining Composer budget cover the escalation work that is left?

A rate-based runway ("130M left, 72M/h, 1.8h") is the wrong question here and it cried
wolf: it extrapolates the current burn forever, but the escalation is a FINITE queue of
trials. Burn spikes to 138M/h precisely when the queue is draining fastest, so the alarm
is loudest exactly when the work is closest to done.

The useful question is a subtraction. Count the rungs still owed, price them from
observed history, compare to what is left. That answers "will this finish" instead of
"how long until zero if nothing ever ends".

Priced two ways on purpose. The median is what a typical trial costs; the mean is much
higher because the distribution has a heavy tail (L3's mean is 2.3x its median — a few
runaway trials dominate). A plan that only survives on the median is not funded, it is
lucky, so both are reported and the mean is the one to believe when deciding.

    budget_forecast.py            # full report
    budget_forecast.py --brief    # one line, for monitors
    budget_forecast.py --check    # exit 1 if the mean estimate does not fit
"""

from __future__ import annotations

import collections
import os
import statistics
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))


def rung_prices(model_prefix="composer"):
    """rung -> (median, mean, n) over non-errored trials with a token count."""
    import trial_ledger as TL
    per = collections.defaultdict(list)
    for t in TL.trials():
        if not (t.get("model") or "").startswith(model_prefix):
            continue
        tok = t.get("tokens") or 0
        if tok <= 0 or t.get("errored"):
            continue
        per[t["rung"]].append(tok)
    return {r: (statistics.median(v), sum(v) / len(v), len(v)) for r, v in per.items()}


def owed():
    """rung -> how many trials are still owed at it.

    Both populations count: units that jumped a rung and owe the one below (so the
    certificate says NEEDS rather than merely flips-by), and units still climbing."""
    import escalate
    import trial_ledger as TL
    per = TL.ledger()
    certs = TL.certificates()
    need = collections.Counter()
    for base, d in per.items():
        c = certs.get(base)
        if c and c.get("rung_established"):
            continue                     # settled: the rung below is already recorded
        want, _why = escalate.next_rung(escalate.history(d))
        if want is not None:
            need[str(want)] += 1
    return need


def forecast():
    import token_cost as TC
    prices = rung_prices()
    need = owed()
    left = TC.remaining_tokens()
    fallback = statistics.median([p[0] for p in prices.values()]) if prices else 2e6
    med = mean = 0.0
    for r, n in need.items():
        p = prices.get(r)
        med += n * (p[0] if p else fallback)
        mean += n * (p[1] if p else fallback)
    return left, need, prices, med, mean


def main() -> int:
    left, need, prices, med, mean = forecast()
    brief = "--brief" in sys.argv
    fits = mean <= left
    if brief:
        q = " ".join(f"L{r}x{n}" for r, n in sorted(need.items()))
        print(f"{'FITS' if fits else 'SHORT'} budget {left/1e6:.0f}M vs owed "
              f"{med/1e6:.0f}M median / {mean/1e6:.0f}M mean ({q or 'nothing owed'})")
        return 1 if ("--check" in sys.argv and not fits) else 0
    print(f"composer budget left: {left/1e6:.1f}M")
    print("\nobserved cost per trial:")
    for r in sorted(prices):
        m, a, n = prices[r]
        print(f"  L{r}: n={n:4d}  median {m/1e6:6.2f}M  mean {a/1e6:6.2f}M")
    print("\nrungs still owed:")
    for r, n in sorted(need.items()):
        p = prices.get(r)
        print(f"  L{r}: {n:3d} trial(s)"
              + (f"  ~{n*p[0]/1e6:5.1f}M median / {n*p[1]/1e6:5.1f}M mean" if p else ""))
    print(f"\n  total owed  {med/1e6:6.1f}M median / {mean/1e6:6.1f}M mean")
    print(f"  budget left {left/1e6:6.1f}M")
    print(f"  -> {'FITS' if fits else 'SHORT by %.0fM on the mean' % ((mean-left)/1e6)}")
    if fits and med < left < mean * 1.2:
        print("     (fits, but with little margin — a hot tail could exhaust it)")
    return 1 if ("--check" in sys.argv and not fits) else 0


if __name__ == "__main__":
    raise SystemExit(main())
