#!/usr/bin/env python3
"""Is the pipeline stable enough that scale is purely a budget question?

Three bugs surfaced in one hour on 2026-09-21 — a ladder rule that omitted L6 (one unit
ran nineteen identical trials), no repair budget (units re-ran 7-19 times), and 75% of
trials re-measuring an already-decided rung. The fixes are cheap; the evidence that they
worked is not. This measures that evidence instead of asserting it.

The gate, over a rolling window:

  1. over-cap rate < 5%       - trials on a rung that ALREADY had RUNG_TRIAL_CAP verdicts.
                               The first draft measured ANY repeat against a <15% target,
                               which the guard can never satisfy: the cap deliberately
                               allows up to 3 verdicts per rung (contract repair needs a
                               retry), so a perfectly-working cap still shows ~67% repeats.
                               Measuring the cap breach is measuring the actual waste.
  2. >= 20 Devin trials       - the solver had only ever run 19 trials, which is too thin
                               to call stable. 40 was the first number chosen and it was
                               arbitrary: recent Devin trials run ~42 min (not the 23 min
                               an older, easier sample suggested), so 40 meant ~24h on one
                               trial slot. 20 still quadruples the evidence base at half
                               the wall-clock, and the over-cap criterion is what actually
                               tests the guard fixes.
  3. no NEW failure mode      - a novel exception signature means an unknown unknown.
                               Known ones (already catalogued) do not reopen the gate.

Passing does not mean "no bugs". It means the known leaks stayed shut for a full window
under real load, which is the most that can be claimed without waiting forever.

  stability_gate.py [--hours N] [--brief]
"""
import sys, os, re, glob, json, time, collections

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import trial_ledger

HOURS = float(sys.argv[sys.argv.index("--hours") + 1]) if "--hours" in sys.argv else 6.0
BRIEF = "--brief" in sys.argv
JOBS = "experiments/dose_response/jobs"
CATALOG = "outputs/supervisor/known_failures.json"

REPEAT_MAX = 0.05   # over-cap trials, not mere repeats
DEVIN_MIN = 20

# Signatures seen and understood. A new one is what the gate is listening for.
SEED_KNOWN = [
    "Unknown model",                  # egress allowlist missing server.codeium.com
    "Named models unavailable",       # stale free-plan CURSOR_API_KEY
    "already exists and cannot be resumed",   # harbor job-name collision
    "CancelledError",                 # our own timeout cutting a live solve
]


def _known():
    try:
        return set(json.load(open(CATALOG)))
    except Exception:
        return set(SEED_KNOWN)


def _remember(sigs):
    os.makedirs(os.path.dirname(CATALOG), exist_ok=True)
    json.dump(sorted(_known() | set(sigs)), open(CATALOG, "w"), indent=1)


def measure(hours=HOURS):
    cut = time.time() - hours * 3600
    rows = []
    for t in trial_ledger.trials():
        d = t.get("dir") or ""
        try:
            when = os.path.getmtime(os.path.join(d, "result.json"))
        except OSError:
            continue
        u = t.get("unit") or ""
        rows.append({"t": when, "base": u.rsplit("-L", 1)[0],
                     "rung": u.rsplit("-L", 1)[1][:1] if "-L" in u else "0",
                     "model": t.get("model") or "?", "dir": d})
    rows.sort(key=lambda r: r["t"])

    # count PRIOR verdicts per (unit, rung) so we can tell a permitted retry from waste
    prior = collections.Counter()
    total = repeat = devin = 0
    try:
        import trial_guard
        cap = trial_guard.RUNG_TRIAL_CAP
    except Exception:
        cap = 3
    # The key must mirror trial_guard's cap exactly, or this measures something the guard
    # never promised. Keyed by (base, rung) alone, every legitimate second curve scores as
    # an over-cap breach and the gate eventually fails on the cap working as designed --
    # that read 28% until it was keyed per solver, then 8%.
    #
    # The guard's cap is now per (base, rung, solver) at EVERY rung, including the
    # certifying ones. It was merged at L0 and L2 until a second model's certificate turned
    # out to be unbuyable there: grok was refused L2 on exprhash because Composer had spent
    # the rung's three trials, for a verdict grok had never produced. A different solver's
    # answer at the same rung is not a repeat of the same answer.
    #
    # A separate GLOBAL ceiling now bounds the certifying rungs (RUNG_CERT_CEILING, solver
    # count x cap). It is a different question from "did the cap refuse this trial", so it
    # is counted separately rather than folded into the key.
    try:
        import trial_guard as _tg
        ceiling = getattr(_tg, "RUNG_CERT_CEILING", 9)
        certifying = _tg.CERTIFYING_RUNGS
    except Exception:
        ceiling, certifying = 9, {"0", "2"}
    over_ceiling = 0
    merged = collections.Counter()
    for r in rows:
        key = (r["base"], r["rung"], trial_ledger.solver_of(r["model"]))
        if r["rung"] in certifying:
            mkey = (r["base"], r["rung"])
            if merged[mkey] >= ceiling and r["t"] >= cut:
                over_ceiling += 1
            merged[mkey] += 1
        if r["t"] < cut:
            prior[key] += 1
            continue
        total += 1
        if prior[key] >= cap:      # the cap should have refused this one
            repeat += 1
        prior[key] += 1
        if r["model"].startswith("swe"):
            devin += 1

    known, novel = _known(), collections.Counter()
    for p in glob.glob(f"{JOBS}/*/*/exception.txt"):
        try:
            if os.path.getmtime(p) < cut:
                continue
            blob = open(p, errors="replace").read()
        except OSError:
            continue
        if any(k.lower() in blob.lower() for k in known):
            continue
        m = re.findall(r"^(\w+(?:Error|Exception))\b", blob, re.M)
        novel[m[-1] if m else "unrecognised"] += 1

    rate = repeat / total if total else 0.0
    return {"hours": hours, "trials": total, "repeat": repeat, "repeat_rate": rate,
            "over_ceiling": over_ceiling, "ceiling": ceiling,
            "devin": devin, "novel": dict(novel),
            "ok_repeat": total >= 10 and rate < REPEAT_MAX,
            "ok_devin": devin >= DEVIN_MIN,
            "ok_novel": not novel}


def cap_for_display():
    try:
        import trial_guard
        return trial_guard.RUNG_TRIAL_CAP
    except Exception:
        return 3


def main():
    m = measure()
    passed = m["ok_repeat"] and m["ok_devin"] and m["ok_novel"]
    mark = lambda b: "PASS" if b else "....."
    if BRIEF:
        # "devin 5/40" read as 5 concurrent against a cap of 4. It is a SAMPLE COUNT -
        # trials completed toward the 40 needed for the gate - and it sat next to a ratio,
        # which made the misreading the obvious one. Name the unit in the label.
        print(f"STABILITY {'PASS' if passed else 'not yet'} | "
              f"over-cap {m['repeat_rate']*100:.0f}% (need <{REPEAT_MAX*100:.0f}%) "
              f"| devin-trials-sampled {m['devin']} of {DEVIN_MIN} "
              f"| new-failure-modes {len(m['novel'])}")
        return 0 if passed else 1
    print(f"  STABILITY GATE over {m['hours']:.0f}h — "
          f"{'PASS: scale is now a budget question' if passed else 'not yet'}")
    # The global ceiling is a SEPARATE question from the per-solver cap, and computing it
    # without printing it would make it exactly the kind of invisible metric this file
    # already learned about twice. Informational, not gating: it is bounded by solver count
    # and a rostered backfill may legitimately cross it.
    if m.get("over_ceiling"):
        print(f"    ....  {m['over_ceiling']} trial(s) past the {m['ceiling']}-trial "
              f"all-solver ceiling at a certifying rung (rostered backfills may cross it)")
    print(f"    {mark(m['ok_repeat'])}  over-cap trials {m['repeat']}/{m['trials']} "
          f"= {m['repeat_rate']*100:.0f}%  (need <{REPEAT_MAX*100:.0f}%; a rung may have "
          f"to {cap_for_display()} verdicts, beyond that is waste)")
    print(f"    {mark(m['ok_devin'])}  devin trials SAMPLED {m['devin']} of {DEVIN_MIN} "
          f"(sample size for the gate — not concurrency; the cap is 4 and is enforced "
          f"separately by devin_cap.py)")
    print(f"    {mark(m['ok_novel'])}  new failure modes {len(m['novel'])}  (need 0)"
          + (f"  {m['novel']}" if m["novel"] else ""))
    if m["novel"]:
        print(f"    -> investigate, then catalogue with --accept once understood")
    if "--accept" in sys.argv and m["novel"]:
        _remember(m["novel"])
        print(f"    catalogued {len(m['novel'])} signature(s) as known")
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
