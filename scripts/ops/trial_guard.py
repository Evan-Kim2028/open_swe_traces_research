#!/usr/bin/env python3
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


def ledger():
    per = collections.defaultdict(lambda: collections.defaultdict(list))
    pats = (JOBS + "/*/**/reward.txt", JOBS + "/*/**/result.json")
    for pat in pats:
        for r in glob.glob(pat, recursive=True):
            unit = None
            for p in r.split(os.sep):
                if "-L" in p and p.rsplit("-L", 1)[-1][:1].isdigit():
                    unit = p
            if not unit:
                continue
            try:
                if r.endswith("reward.txt"):
                    v = float(open(r).read().strip() or 0)
                else:
                    v = float(json.load(open(r)).get("reward", 0) or 0)
            except Exception:
                continue
            base, rung = unit.rsplit("-L", 1)
            per[base][rung[:1]].append(v)
    return per


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


def stamp_verdict(base):
    cur = contract_fingerprint(base)
    if cur is None:
        return
    os.makedirs(".guard_stamps", exist_ok=True)
    open(os.path.join(".guard_stamps", base), "w").write(cur)


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
    if verdict_is_stale(base):
        return True, "contract changed since last verdict — prior trials no longer bind"
    d = per.get(base, {})
    l0, l2 = d.get("0", []), d.get("2", [])

    if l0 and max(l0) > 0:
        return False, f"condemned: passed L0 ({len(l0)} trial(s)) — not a hard unit"
    if l0 and max(l0) == 0 and l2 and max(l2) > 0:
        return False, "certified: L0 fail + L2 pass already on record"
    if rung == "0" and l0:
        return False, f"L0 already decided ({len(l0)} trial(s), max={max(l0)})"
    if rung == "2":
        if not l0:
            return False, "L2 before L0 — run L0 first, it is the cheaper verdict"
        if len(l2) >= NONFLIP_CAP and max(l2) == 0:
            return False, f"non-flipping: {len(l2)} L2 failures — repair the contract, do not retry"
    if rung in "1345":
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
