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
import os, re, sys, statistics, collections

from openswe_traces.ladder import ledger as trial_ledger

# Used only when a rung has too little history to measure. Deliberately pessimistic: an
# under-estimate spends real money, an over-estimate only delays a launch.
FALLBACK = {"0": 2_500_000, "2": 7_000_000, "other": 5_000_000}
MIN_SAMPLES = 4


def _rung_of(job_or_unit: str) -> str:
    """The rung a job or unit is at. Cohort names carry it as a _L<n> SUFFIX.

    This only ever recognised _L2 and fell through to "0" for everything else, so every
    escalation cohort — sweep_escalate_L3, _L4, _L5, _L6 — was priced as L0. With no L0
    sample meeting MIN_SAMPLES it then took the 6.00M fallback, and at the 1.8x margin
    quoted 10.8M per unit. The measured escalation cost is 2.13M mean / 1.20M median over
    63 trials, so a 31-unit L3 cohort was refused at ~335M when it is worth ~119M — the
    gate starving the exact work it should have been letting through.
    """
    s = job_or_unit or ""
    if "-L" in s:                       # a UNIT: "<base>-L<rung>"
        return s.rsplit("-L", 1)[1][:1]
    m = re.search(r"_L(\d)(?:_|$)", s)  # a COHORT: "sweep_..._L<rung>" or "..._L<rung>_r1"
    if m:
        return m.group(1)
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


COHORT_MIN = 5          # trials of a cohort's OWN before trusting its own price


def cohort_observed(cohort: str, model_prefix="composer"):
    """(price, n) from THIS cohort family's own trials, or (None, 0).

    A rung price averages workloads that are not alike. L3 holds 24 old sweep_climb
    trials (median 2.29M, p90 27.93M) and 12 escalation trials (median 0.94M, p90 2.21M)
    — two populations differing 24x at the quantile the guard prices on. Mixed, they put
    a 31-unit escalation cohort at 1333M and the gate refused work worth about 52M, with
    Composer sitting on free slots and the ladder stalled.

    A cohort's own recent history is the better predictor of its next trial. This does
    NOT relax the guard: same window, same quantile, same 1.8x margin, applied to a
    population that actually resembles what is about to run. Falls back to the rung when
    a cohort has fewer than COHORT_MIN trials of its own, so a brand-new cohort is still
    priced conservatively.
    """
    fam = re.sub(r"_r\d+(_\d+)?$", "", cohort or "")
    vals = []
    for t in trial_ledger.trials():
        if not (t.get("model") or "").startswith(model_prefix):
            continue
        tok = t.get("tokens") or 0
        if tok <= 0 or t.get("errored"):
            continue
        job = re.sub(r"_r\d+(_\d+)?$", "", t.get("job") or "")
        if job != fam:
            continue
        try:
            vals.append((os.path.getmtime(os.path.join(t.get("dir") or "",
                                                       "result.json")), tok))
        except OSError:
            continue
    if len(vals) < COHORT_MIN:
        return None, len(vals)
    vals.sort()
    recent = sorted(tok for _, tok in vals[-WINDOW:])
    # A quantile is only a quantile when the index it picks is INTERIOR. With 7 samples
    # int(7 * 0.9) == 6, which is the last element, so "the 90th percentile" was literally
    # the most expensive trial ever seen in the cohort — and SAFETY then multiplied that
    # single worst case by 1.8. sweep_escalate_composer_L4 was priced at 21.8M/unit from
    # one 12.11M outlier against a cohort median of 2.00M, and the gate refused to launch
    # 20 units at ~436M while 134M sat unspent.
    #
    # Clamp the index below the maximum so the estimate is an order statistic of the body
    # rather than of the tail. The tail is not ignored — it still raises the median-to-
    # quantile gap and SAFETY still applies on top — it just stops being the whole
    # estimate when the sample is too small for the quantile to mean anything.
    n = len(recent)
    idx = min(int(n * QUANTILE), n - 2 if n >= 3 else n - 1)
    return recent[idx], n


def cost_per_trial(rung: str, model_prefix="composer", cohort: str = "") -> float:
    if cohort:
        price, n = cohort_observed(cohort, model_prefix)
        if price is not None:
            return price
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



def rung_moments(rung: str, model_prefix="composer"):
    """(mean, sd, n) over every trial at this RUNG, or None when too few.

    The cohort-level equivalent of cohort_moments, for cohorts too new to have a history
    of their own. A rung with 74 observations is a far better estimator than a flat
    multiple of a tail quantile."""
    import math
    vals = []
    for t in trial_ledger.trials():
        if not (t.get("model") or "").startswith(model_prefix):
            continue
        tok = t.get("tokens") or 0
        if tok <= 0 or t.get("errored") or t["rung"] != str(rung):
            continue
        vals.append(tok)
    if len(vals) < MIN_SAMPLES:
        return None
    n = len(vals)
    mean = sum(vals) / n
    var = sum((x - mean) ** 2 for x in vals) / (n - 1) if n > 1 else 0.0
    return mean, math.sqrt(var), n


def cohort_moments(cohort: str, model_prefix="composer"):
    """(mean, sd, n) over a cohort's own recent trials, or None.

    can_afford needs the distribution of a SUM, and a quantile cannot give it one."""
    import math
    import re as _re
    fam = _re.sub(r"_r\d+(_\d+)?$", "", cohort or "")
    vals = []
    for t in trial_ledger.trials():
        if not (t.get("model") or "").startswith(model_prefix):
            continue
        tok = t.get("tokens") or 0
        if tok <= 0 or t.get("errored"):
            continue
        if _re.sub(r"_r\d+(_\d+)?$", "", t.get("job") or "") != fam:
            continue
        try:
            vals.append((os.path.getmtime(os.path.join(t.get("dir") or "",
                                                       "result.json")), tok))
        except OSError:
            continue
    if len(vals) < COHORT_MIN:
        return None
    vals.sort()
    recent = [tok for _, tok in vals[-WINDOW:]]
    n = len(recent)
    mean = sum(recent) / n
    var = sum((x - mean) ** 2 for x in recent) / (n - 1) if n > 1 else 0.0
    return mean, math.sqrt(var), n


def can_afford(cohort: str, n_units: int, want_conc: int) -> tuple[int, str]:
    """May this cohort launch, and at what concurrency?

    Sized on the COHORT TOTAL, not concurrency. A sweep does not stop after `want_conc`
    trials - it runs every runnable unit in the cohort, then re-gates and may run more.
    Concurrency only sets how many happen at once. Bounding on concurrency is why the
    guard's first two drafts both permitted the launch that overspent: 6 x 4.05M looked
    affordable against 48.7M, but the cohort had ~10 units and spent 61M.

    Returns (concurrency, reason); concurrency 0 means do not launch.
    """
    import math
    rung = _rung_of(cohort)
    left = remaining_tokens()
    if left <= 0:
        return 0, "budget spent"

    # Bound the distribution of the TOTAL, not n copies of a per-trial worst case.
    # `n_units x p90 x SAFETY` assumes all n trials hit their 90th percentile at once,
    # which is not a conservative estimate so much as an impossible one: it priced
    # sweep_escalate_composer_L4 at 436M (20 x 21.8M) against a cohort whose median trial
    # is 2.0M and whose mean is 4.0M. 134M of freshly authorized budget sat unspent
    # because of it.
    #
    # A sum of n draws concentrates: its mean is n*mean and its sd is sd*sqrt(n), so the
    # tail of the TOTAL is proportionally much tighter than the tail of one trial. Use a
    # one-sided 95% bound on the sum. That is still conservative — it funds a cohort that
    # runs expensive across the board — without compounding a tail estimate n times.
    mom = cohort_moments(cohort)
    if mom:
        mean, sd, n_obs = mom
        total = n_units * mean + 1.645 * sd * math.sqrt(n_units)
        how = (f"{n_units} units x {mean/1e6:.1f}M mean +95% tail on the sum, "
               f"from {n_obs} observed")
    else:
        # No cohort history — but the RUNG usually has plenty, and the same argument
        # applies: sizing a cohort total as n x (per-trial high quantile) x SAFETY asserts
        # every trial simultaneously lands in its own tail. sweep_escalate_composer_L5 had
        # run fewer than COHORT_MIN trials of its own, so it took this path and priced 4
        # units at 51.6M against an L5 distribution whose mean is 3.55M over 74 trials —
        # refusing work the budget comfortably covered.
        #
        # Use the rung's own mean and spread to bound the SUM, exactly as the cohort path
        # does. Only when the rung is ALSO unmeasured does the flat per-trial margin
        # apply, which is the case where there genuinely is no spread to estimate.
        mom = rung_moments(rung)
        if mom:
            mean, sd, n_obs = mom
            total = n_units * mean + 1.645 * sd * math.sqrt(n_units)
            how = (f"{n_units} units x {mean/1e6:.1f}M mean +95% tail on the sum, "
                   f"from {n_obs} L{rung} trial(s)")
        else:
            per = cost_per_trial(rung, cohort=cohort) * SAFETY
            total = n_units * per
            how = f"{n_units} units x {per/1e6:.1f}M at L{rung}, incl. {SAFETY}x margin"

    if total > left:
        # Partial launches are not safe: the sweep would run the whole cohort anyway.
        return 0, (f"cohort needs ~{total/1e6:.0f}M ({how}) but only "
                   f"{left/1e6:.1f}M left — not launching")
    return want_conc, (f"~{total/1e6:.0f}M of {left/1e6:.1f}M remaining ({how})")


def cli():
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


if __name__ == "__main__":
    cli()
