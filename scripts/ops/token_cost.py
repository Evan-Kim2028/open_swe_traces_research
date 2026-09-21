"""What a trial actually costs, measured per rung.

Orchestrate capped Composer by CONTAINER COUNT and nothing else, so a single launch at
concurrency 6 committed ~41M tokens with no check. The budget gate ran only at launch and
could not see what the launch would cost.

The projection that led to the overspend used the BLENDED average across all rungs
(2.05M/trial), which is dominated by cheap L0 screening. L2 puts the full contract in the
prompt and runs on units already known to be hard, so it costs ~3.3x as much. Pricing an
L2 workload at L0 rates predicted 24 trials from the remaining budget; it bought 7, and
overshot a 100M budget by 23%.

Cost is therefore measured PER RUNG, from observed history, never assumed.
"""
from __future__ import annotations
import os, sys, statistics, collections

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import trial_ledger

# Used only when a rung has too little history to measure. Deliberately pessimistic: an
# under-estimate spends real money, an over-estimate only delays a launch.
FALLBACK = {"0": 2_500_000, "2": 7_000_000, "other": 5_000_000}
MIN_SAMPLES = 4


def _rung_of(job_or_unit: str) -> str:
    s = job_or_unit or ""
    if "-L" in s:
        return s.rsplit("-L", 1)[1][:1]
    if s.endswith("_L2") or "_L2_" in s:
        return "2"
    return "0"


# How many recent trials to price from, and which quantile. A LIFETIME mean is the same
# blended-average error one level up: the lifetime L2 mean is 1.67M, while the L2 cohort
# that overspent averaged 6.81M. Old cheap trials drown out what the machine is running
# now, so a lifetime mean would have permitted the exact launch it exists to prevent.
# Price from the recent window, at a high quantile, because an under-estimate spends real
# money while an over-estimate only delays a launch.
WINDOW = 40
QUANTILE = 0.9
# A rung price cannot predict a specific cohort: the L2 price from the recent window was
# 4.05M while the goa_3_L2 cohort ran at 6.81M - a 1.7x miss, because cohorts differ in
# difficulty and prompt size. The margin is set from that measured error, not chosen. An
# under-estimate spends money that was not authorised; an over-estimate only delays a
# launch until a cheaper cohort or more budget is available.
SAFETY = 1.8


def observed(model_prefix="composer", window=WINDOW):
    """rung -> (price, n). Price is a high quantile of the most RECENT trials."""
    by = collections.defaultdict(list)
    for t in trial_ledger.trials():
        if not (t.get("model") or "").startswith(model_prefix):
            continue
        tok = t.get("tokens") or 0
        if tok <= 0:
            continue
        d = t.get("dir") or ""
        rj = os.path.join(d, "result.json")
        try:
            when = os.path.getmtime(rj)
        except OSError:
            continue
        by[_rung_of(t.get("unit") or t.get("job") or "")].append((when, tok))
    out = {}
    for r, v in by.items():
        v.sort()
        recent = [tok for _, tok in v[-window:]]
        recent.sort()
        idx = min(len(recent) - 1, int(len(recent) * QUANTILE))
        out[r] = (float(recent[idx]), len(recent))
    return out


def cost_per_trial(rung: str, model_prefix="composer") -> float:
    o = observed(model_prefix)
    if rung in o and o[rung][1] >= MIN_SAMPLES:
        return o[rung][0]
    return FALLBACK.get(rung, FALLBACK["other"])


def remaining_tokens() -> float:
    """Budget left, from the same source composer_budget.py uses."""
    import json
    f = "outputs/composer_budget.json"
    cfg = json.load(open(f)) if os.path.exists(f) else {"baseline_tokens": 0,
                                                        "budget": 250_000_000}
    tot = sum(t["tokens"] for t in trial_ledger.trials()
              if (t.get("model") or "").startswith("composer"))
    return max(0.0, cfg["budget"] - max(0, tot - cfg["baseline_tokens"]))


def can_afford(cohort: str, n_units: int, want_conc: int) -> tuple[int, str]:
    """May this cohort launch, and at what concurrency?

    Sized on the COHORT TOTAL, not concurrency. A sweep does not stop after `want_conc`
    trials - it runs every runnable unit in the cohort, then re-gates and may run more.
    Concurrency only sets how many happen at once. Bounding on concurrency is why the
    guard's first two drafts both permitted the launch that overspent: 6 x 4.05M looked
    affordable against 48.7M, but the cohort had ~10 units and spent 61M.

    Returns (concurrency, reason); concurrency 0 means do not launch.
    """
    rung = _rung_of(cohort)
    per = cost_per_trial(rung) * SAFETY
    left = remaining_tokens()
    total = n_units * per
    if left <= 0:
        return 0, f"budget spent"
    if total > left:
        # Partial launches are not safe: the sweep would run the whole cohort anyway.
        return 0, (f"cohort needs ~{total/1e6:.0f}M ({n_units} units x {per/1e6:.1f}M at "
                   f"L{rung}, incl. {SAFETY}x margin) but only {left/1e6:.1f}M left — not launching")
    return want_conc, (f"~{total/1e6:.0f}M of {left/1e6:.1f}M remaining")


if __name__ == "__main__":
    o = observed()
    print(f"  {'rung':6}{'mean tokens':>14}{'n':>6}")
    for r in sorted(o):
        print(f"  L{r:<5}{o[r][0]/1e6:13.2f}M{o[r][1]:6}")
    print(f"\n  remaining budget: {remaining_tokens()/1e6:.1f}M")
    print("\n  replay of the launch that overspent (10 units at L2, 48.7M left):")
    import types
    _orig = remaining_tokens
    globals()["remaining_tokens"] = lambda: 48_700_000
    n, why = can_afford("sweep_goa_3_L2", 10, 6)
    print(f"    can_afford -> conc {n}: {why}")
    globals()["remaining_tokens"] = _orig
