#!/usr/bin/env python3
import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
"""Refuse a trial whose verdict is already decided.

Measured 2026-09-20 over 1632 trials: 883 of them (54%) re-measured a unit whose
outcome was already known. 18.8 trials per certified unit against a floor of 2.6.

A certificate needs exactly two facts: fails L0, passes L2. Every trial beyond the
one that establishes each fact buys nothing. The ladder rungs (L1/L3/L4/L5) answer a
population question, not a per-unit one, so they run on a sample.

  trial_guard.py <unit>            -> exit 0 run it, 1 skip (reason on stdout)
  trial_guard.py --report          -> what the ledger currently believes
"""
import sys, os, glob, json, collections, hashlib

JOBS = "experiments/dose_response/jobs"
LADDER_SAMPLE = 0.15      # fraction of certified units that also walk the full ladder
NONFLIP_CAP   = 3         # L2 failures before a unit is declared non-flipping

# Every rung that is not L0 or L2. Written as a set, not the string "1345", because that
# literal silently omitted L6: kops-clustervalid accumulated NINETEEN passing L6 trials
# because every ladder check skipped it. Deriving membership from "not a certifying rung"
# means a new rung cannot be forgotten.
CERTIFYING_RUNGS = {"0", "2"}

# A rung answers one question, so it needs one verdict. Repeat trials buy nothing and were
# 15-39% of all spend. The cap is per rung and absolute - it holds even when a contract
# repair makes prior verdicts stale, which is what let single units run 7, 11, 19 times.
RUNG_TRIAL_CAP = 3

# The solvers we actually screen with. A unit is only "screened" once each of these has
# had its turn at L0, because condemnation is model-agnostic: it takes just one of them
# to solve the bug report to prove the unit was never hard.
SCREENERS = ("composer", "devin")

# Absolute ceiling on recorded L0 trials for one unit, across all screeners. The
# second-screen rule below bypasses the per-rung cap, and an ERRORED trial records no
# verdict — so a solver that keeps erroring would stay forever "unscreened" and re-queue
# without limit. This is the backstop for that.
L0_SCREEN_CAP = len(SCREENERS) * RUNG_TRIAL_CAP

# Contract repair resets L2 staleness, so a unit could be repaired and retried forever.
# Two repairs is the budget: if a contract still cannot carry the unit after two rewrites,
# the defect is the unit, not the prose.
REPAIR_BUDGET = 2


def ledger():
    """Delegated: trial_ledger is the single correct reader. The old inline version scored
    every result.json as 0 (the reward is nested at verifier_result.rewards.reward) and
    counted each trial twice, once real and once as a false zero."""
    import trial_ledger
    return trial_ledger.ledger()


def contract_fingerprint(base):
    """A verdict binds only while the artifact it judged is unchanged. Repair rewrites
    instruction.md — the L2 contract — which makes every prior L2 failure stale. Without
    this the guard would veto exactly the retrial that tests the repair.

    Newest copy wins: the staged unit is what the next trial will actually run."""
    cands = []
    for root in ("experiments/dose_response",):
        cands += glob.glob(f"{root}/*/{base}-L2/instruction.md")
        cands += glob.glob(f"{root}/*/*/{base}-L2/instruction.md")
        cands += glob.glob(f"{root}/*/*/*/{base}-L2/instruction.md")
    if not cands:
        return None
    newest = max(cands, key=os.path.getmtime)
    try:
        return hashlib.sha256(open(newest, "rb").read()).hexdigest()[:12]
    except Exception:
        return None


def verdict_is_stale(base):
    """True when the contract has changed since the verdict ledger was last stamped."""
    cur = contract_fingerprint(base)
    if cur is None:
        return False
    stamp = os.path.join(".guard_stamps", base)
    prev = open(stamp).read().strip() if os.path.exists(stamp) else None
    return prev is not None and prev != cur


def repair_count(base):
    """How many times this unit's contract has been rewritten since first stamped.

    Each distinct fingerprint we have stamped is one repair. Recorded alongside the stamp
    so it survives restarts and is visible to anyone reading .guard_stamps/."""
    hist = os.path.join(".guard_stamps", base + ".history")
    if not os.path.exists(hist):
        return 0
    try:
        return max(0, len({l.strip() for l in open(hist) if l.strip()}) - 1)
    except OSError:
        return 0


def stamp_verdict(base):
    cur = contract_fingerprint(base)
    if cur is None:
        return
    os.makedirs(".guard_stamps", exist_ok=True)
    hist = os.path.join(".guard_stamps", base + ".history")
    try:
        seen = {l.strip() for l in open(hist)} if os.path.exists(hist) else set()
        if cur not in seen:
            with open(hist, "a") as fh:
                fh.write(cur + "\n")
    except OSError:
        pass
    open(os.path.join(".guard_stamps", base), "w").write(cur)


_BYS = None

SECOND_SCREEN_ROSTER = "outputs/supervisor/second_screen_enrolled"


def enrolled_for_second_screen():
    """Units deliberately signed up for a second L0 screen, one base name per line.

    Enrolment exists because the guard is read by orchestrate to decide which cohorts
    have work left. Opening the second screen for everyone at once does not just permit
    212 trials, it makes almost every retired cohort look runnable again and orchestrate
    relaunches them wholesale — the entire Devin cap spent re-screening before anyone has
    seen a single disagreement. The roster keeps the sample the size we chose."""
    try:
        with open(SECOND_SCREEN_ROSTER) as fh:
            return {l.strip() for l in fh if l.strip() and not l.startswith("#")}
    except OSError:
        return set()


def unscreened(base):
    """Screeners with no recorded L0 verdict for this ENROLLED unit.

    Screening has been routed by free capacity, not by coverage, and the result is that
    almost every unit has been screened exactly once: Composer ran 447 L0 trials to
    Devin's 51, and only two units have an L0 verdict from both. So the dataset's central
    claim — that these tasks are hard for frontier agents — rests on ONE agent's opinion
    per unit, and "the solvers agree" has never actually been measured.

    A unit nobody has condemned but only one solver has tried is not a settled L0. It is
    an untested one."""
    if base not in enrolled_for_second_screen():
        return []
    global _BYS
    if _BYS is None:
        # Memoized: orchestrate asks this once per staged directory (thousands), and
        # ledger_by_solver rescans every result.json in jobs/. Uncached it turned a
        # sub-second planning pass into minutes.
        import trial_ledger as _tl
        _BYS = _tl.ledger_by_solver()
    seen = {s for s, rungs in _BYS.get(base, {}).items() if rungs.get("0")}
    return [s for s in SCREENERS if s not in seen]


def in_ladder_sample(base):
    """Stable membership: the same units always walk the ladder, so the sample is
    comparable across runs instead of drifting every time it is recomputed."""
    h = int(hashlib.sha256(base.encode()).hexdigest()[:8], 16)
    return (h % 1000) < LADDER_SAMPLE * 1000


def decide(unit, per):
    if "-L" not in unit or not unit.rsplit("-L", 1)[-1][:1].isdigit():
        return True, "unrecognised unit name"
    base, rung = unit.rsplit("-L", 1)
    rung = rung[:1]
    d = per.get(base, {})
    l0, l2 = d.get("0", []), d.get("2", [])

    # Settled verdicts outrank contract staleness. The expiry check used to run FIRST, so
    # every contract rewrite resurrected units we had already decided - and with ten
    # repair-contract cohorts (rc, rc2..rc5, auto, auto2, auto3, loop, fix) covering the
    # same goa units, eight already-certified units were re-trialled ten to twelve times
    # each. That is what a window of 38 trials and zero new certifications was made of.
    #
    # A flip certificate is a property we already own: L0 failed and L2 passed, both
    # recorded. Rewriting the contract cannot un-earn it. Condemnation is likewise a
    # property of the unit, not of its prose - a solver that fixed the bug from the report
    # alone did not need the contract. Units still OPEN keep expiring on a contract change,
    # which is the whole point of a repair job.
    if l0 and max(l0) > 0:
        # MODEL-AGNOSTIC BY DESIGN, and this is a research position rather than an
        # oversight. `max` spans every solver: one pass at L0 by ANY of them condemns the
        # unit for all. The dataset's claim is that a task is hard for frontier agents, so
        # a task one frontier agent solves from the bug report alone is not hard — even if
        # another would have failed it.
        #
        # This is deliberately NOT symmetric with certification, where a cross-solver flip
        # is the weaker claim and gets routed to the solver that failed the unit. The
        # asymmetry is the point: "some agent can do it" is enough to disqualify, and
        # "this agent can do it with the contract but not without" is what qualifies.
        return False, f"condemned: passed L0 ({len(l0)} trial(s)) — not a hard unit"

    # A SECOND SCREEN is always worth buying, and this check sits above every refusal
    # below it — including "certified" — on purpose.
    #
    # Every other rule here refuses a trial because its verdict is already known. That
    # reasoning does not reach an unused screener: what is on record is that ONE solver
    # failed at L0, and the question of whether the other would have failed too has no
    # answer yet. Refusing it as "already decided" mistakes one solver's opinion for the
    # population claim the dataset actually makes.
    #
    # It is also the one trial that cannot corrupt anything. The outcomes are: the second
    # screener fails, and the certificate is strictly better evidence than before; or it
    # passes, and the unit is condemned by the model-agnostic rule above and leaves the
    # dataset. A second screen can only ever DESTROY a certificate, never mint one — so
    # unlike every trial the cap protects against, there is no incentive to be careful
    # with it. Blocking it only preserves a number we have not earned.
    if rung == "0" and (want := unscreened(base)):
        if len(l0) >= L0_SCREEN_CAP:
            return False, (f"L0 has {len(l0)} trial(s) (ceiling {L0_SCREEN_CAP}) and "
                           f"{'/'.join(want)} still has no verdict — likely erroring, "
                           f"investigate rather than re-queue")
        return True, (f"second screen: L0 failed by "
                      f"{'/'.join(s for s in SCREENERS if s not in want) or 'nobody'}, "
                      f"never tried by {'/'.join(want)} — a pass condemns the unit")

    if l0 and max(l0) == 0 and l2 and max(l2) > 0:
        # A CROSS certificate is not a finished unit. If one solver failed L0 and a
        # different one passed L2, the flip may record only that the second model is
        # stronger. Closing the unit here freezes that weaker claim forever — the solver
        # that actually failed it never gets its turn, and certificates() can never
        # upgrade the row to single-solver.
        #
        # So leave it open. solver_match then admits only the solver that failed it low,
        # and a pass from that solver makes `shared` non-empty and the certificate
        # single. Four trials were mid-flight producing exactly this contamination when
        # the rule went in; this lets their work stand and be corrected rather than
        # killing them.
        import trial_ledger as _tl
        cert = _tl.certificates().get(base)
        if not cert or cert["kind"] == "single":
            return False, "certified: L0 fail + L2 pass already on record"
        # FALL THROUGH, do not return True here. An early return skips the per-rung cap
        # below, and a cross-certified unit with three L2 trials already on it would have
        # re-run without limit — reopening 25 units straight past the one check that took
        # over-cap from 25% to 0%.
        reopened = (f"cross-certified at L{cert['rung']} (failed by "
                    f"{'/'.join(cert['l0'])}, passed by {'/'.join(cert['l2'])}) — "
                    f"open for the solver that failed it")
    else:
        reopened = None
    # Absolute per-rung cap, checked BEFORE staleness. A rung answers one question and
    # needs one verdict; repair resetting staleness is exactly how single units reached
    # 7, 11 and 19 trials on one rung.
    n_this_rung = len(d.get(rung, []))
    if n_this_rung >= RUNG_TRIAL_CAP:
        return False, (f"rung L{rung} already has {n_this_rung} trial(s) "
                       f"(cap {RUNG_TRIAL_CAP}) — no further verdict to buy")

    if reopened:
        return True, reopened

    reps = repair_count(base)
    if verdict_is_stale(base):
        if reps >= REPAIR_BUDGET:
            return False, (f"contract repaired {reps}x (budget {REPAIR_BUDGET}) and the "
                           f"unit still does not flip — retire it, do not repair again")
        return True, (f"contract changed since last verdict (repair {reps + 1} of "
                      f"{REPAIR_BUDGET}) — prior trials no longer bind")
    if rung == "0" and l0:
        # Reached when the unit is not enrolled for a second screen, or when every
        # screener already has a verdict — the branch above returns first otherwise.
        return False, (f"L0 already decided ({len(l0)} trial(s), max={max(l0)}) "
                       f"— enrol in {SECOND_SCREEN_ROSTER} to buy a second screen")
    if rung == "2":
        if not l0:
            return False, "L2 before L0 — run L0 first, it is the cheaper verdict"
        if len(l2) >= NONFLIP_CAP and max(l2) == 0:
            return False, f"non-flipping: {len(l2)} L2 failures — repair the contract, do not retry"
    if rung not in CERTIFYING_RUNGS:
        # Two different things wear the same suffix. A ladder rung on a CERTIFIED unit is
        # affordance-study data, sampled, and can never certify. A ladder rung on a unit
        # that failed L0 AND L2 is an ESCALATION: it is the only way that unit ever
        # certifies, and 9 of 9 hand-escalated units flipped above L2. Rejecting both with
        # "certify first" is what kept 46 of the hardest units in the dataset written off.
        escalating = bool(l0 and max(l0) == 0 and l2 and max(l2) == 0)
        if escalating:
            import escalate
            want, why = escalate.next_rung(escalate.history(d))
            if want is None:
                return False, f"escalation {why}"
            if want != int(rung):
                return False, (f"escalation wants L{want}, not L{rung} — "
                               f"one rung at a time ({why})")
            return True, f"escalation: {why}"
        if not (l0 and max(l0) == 0 and l2 and max(l2) > 0):
            return False, f"ladder rung L{rung} on an uncertified unit — certify first"
        if not in_ladder_sample(base):
            return False, f"ladder rung L{rung} outside the {LADDER_SAMPLE:.0%} sample"
    return True, "verdict still open"


def main():
    per = ledger()
    if "--stamp" in sys.argv:
        n = 0
        for base in per:
            stamp_verdict(base); n += 1
        print(f"stamped {n} unit(s) at their current contract")
        return 0
    if "--report" in sys.argv:
        cert = [u for u, d in per.items()
                if d.get("0") and max(d["0"]) == 0 and d.get("2") and max(d["2"]) > 0]
        easy = [u for u, d in per.items() if d.get("0") and max(d["0"]) > 0]
        nonflip = [u for u, d in per.items()
                   if d.get("0") and max(d["0"]) == 0
                   and len(d.get("2", [])) >= NONFLIP_CAP and max(d.get("2") or [1]) == 0]
        tot = sum(len(v) for d in per.values() for v in d.values())
        print(f"units {len(per)}  trials {tot}")
        print(f"  certified   {len(cert)}")
        print(f"  too-easy    {len(easy)}")
        print(f"  non-flip    {len(nonflip)}  (contract repair, not more trials)")
        print(f"  ladder sample {sum(in_ladder_sample(u) for u in cert)} of {len(cert)} certified")
        return 0
    if len(sys.argv) < 2:
        print("usage: trial_guard.py <unit> | --report", file=sys.stderr)
        return 2
    ok, why = decide(sys.argv[1], per)
    print(("RUN   " if ok else "SKIP  ") + sys.argv[1] + " — " + why)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
