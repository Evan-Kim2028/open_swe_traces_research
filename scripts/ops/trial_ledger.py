#!/usr/bin/env python3
"""The one place that reads trial outcomes. Everything else imports this.

Two bugs made the previous readers untrustworthy and both are fixed here.

result.json has no top-level "reward" — it is at verifier_result.rewards.reward. The old
reader did .get("reward", 0), so every result.json scored 0 and invented a failure. And
because <trial>/verifier/reward.txt and <trial>/result.json sit in different directories,
the same trial was counted twice, once real and once as a false zero.

A trial is a DIRECTORY. It has at most one outcome, and a trial that raised before the
verifier ran has no outcome at all — that is an error, not a failure, and it must not count
against a unit.
"""
import json, os, glob, collections

JOBS = "experiments/dose_response/jobs"


def trials(jobs_dir=JOBS):
    """Yield one record per trial directory."""
    seen = set()
    for f in glob.glob(jobs_dir + "/*/*/result.json") + glob.glob(jobs_dir + "/*/*/verifier/reward.txt"):
        tdir = os.path.dirname(f)
        if tdir.endswith("/verifier"):
            tdir = os.path.dirname(tdir)
        if tdir in seen:
            continue
        seen.add(tdir)
        name = os.path.basename(tdir).split("__")[0]
        if "-L" not in name or not name.rsplit("-L", 1)[-1][:1].isdigit():
            continue
        base, suffix = name.rsplit("-L", 1)
        # The rung is the leading digit; the rest names WHICH variant of that rung.
        # "-L1binding" and "-L1other" are two different gaps in the same contract and
        # are scored against different hidden suites, yet both collapse to rung "1"
        # under one base, where max() then reads "passed L1" if either gap passed.
        # Harmless today - L1 is excluded from certification and from escalation - but
        # it would quietly corrupt the affordance study, so the variant is carried.
        rung = suffix[:1]
        variant = suffix

        reward = err = None
        tok = cost = 0.0
        # Who produced this verdict. "Hard at L0" is solver-dependent, so a dataset screened by
        # two different solvers is not one population unless we can stratify it later.
        agent = model = None
        rj = os.path.join(tdir, "result.json")
        if os.path.exists(rj):
            try:
                d = json.load(open(rj))
                vr = (d.get("verifier_result") or {}).get("rewards") or {}
                if "reward" in vr:
                    reward = float(vr["reward"])
                err = d.get("exception_info")
                ar = d.get("agent_result") or {}
                # Harbor fills the TOP-LEVEL token fields for some agents and leaves them
                # None for others, putting the real numbers only in the per-model
                # breakdown. Devin is the second kind: every one of its trials reported
                # n_input_tokens=None while model_usage["devin/swe-2-max"] held 1.16M.
                # Reading only the top level scored all 161 devin trials at zero, so the
                # project's entire token and cost accounting has been composer-only
                # without ever saying so.
                tok = (ar.get("n_input_tokens") or 0) + (ar.get("n_output_tokens") or 0)
                if not tok:
                    for mu in (ar.get("model_usage") or {}).values():
                        if isinstance(mu, dict):
                            tok += (mu.get("n_input_tokens") or 0) + \
                                   (mu.get("n_output_tokens") or 0)
                cost = ar.get("cost_usd") or 0.0
                ai = d.get("agent_info") or {}
                agent = ai.get("name")
                model = ((ai.get("model_info") or {}).get("name"))
            except Exception:
                pass
        if reward is None:
            rt = os.path.join(tdir, "verifier", "reward.txt")
            if os.path.exists(rt):
                try:
                    reward = float(open(rt).read().strip() or 0)
                except Exception:
                    pass
        yield {"dir": tdir, "job": tdir.split(os.sep)[-2], "unit": name, "base": base,
               "rung": rung, "variant": variant,
               "reward": reward, "errored": err is not None,
               "tokens": tok, "cost": cost, "agent": agent, "model": model,
               "mtime": os.path.getmtime(tdir)}


def solver_of(model):
    """Which solver family produced a trial. Difficulty is a property of a task RELATIVE
    to a solver, so a flip certificate that spans two of them is a different claim from
    one that does not - and 15 of the first 140 certificates did span two."""
    m = (model or "")
    if m.startswith("swe"):
        return "devin"
    if m.startswith("composer") or m.startswith("cursor"):
        return "composer"
    if m.startswith("grok"):
        return "grok"
    return "other"


def ledger(jobs_dir=JOBS):
    """base -> rung -> [reward]. Errored trials and trials with no verdict are excluded."""
    per = collections.defaultdict(lambda: collections.defaultdict(list))
    for t in trials(jobs_dir):
        if t["reward"] is None or t["errored"]:
            continue
        per[t["base"]][t["rung"]].append(t["reward"])
    return per


def ledger_by_solver(jobs_dir=JOBS):
    """base -> solver -> rung -> [reward]. The multi-model view.

    The dataset is deliberately multi-model: tasks are trialled by whichever solver has
    capacity, and a certificate records WHO it binds for. Keeping this separate from
    ledger() means existing callers are unchanged while provenance is available to anyone
    who needs it."""
    per = collections.defaultdict(lambda: collections.defaultdict(
        lambda: collections.defaultdict(list)))
    for t in trials(jobs_dir):
        if t["reward"] is None or t["errored"]:
            continue
        per[t["base"]][solver_of(t.get("model"))][t["rung"]].append(t["reward"])
    return per


def certificates(jobs_dir=JOBS):
    """base -> certificate. Derived from the PER-SOLVER view; there is no other kind.

    This used to pool solvers: a unit counted as certified when SOME solver failed L0 and
    SOME solver passed a rung >= 2, with `kind` marking whether one solver did both
    ("single") or two did ("cross"). Cross was described as a real certificate making a
    weaker claim. Looking at the six that remained, it is neither.

    Four were two halves that do not join. `svcexpr` and `verifyenv`: composer failed L0,
    devin passed L2, devin never screened L0 at all. `flhashmap`: grok and devin failed L0,
    composer passed L2, composer never screened it. `hfs-dot`: composer failed L0 while
    devin PASSED L0 and L2. None has a solver that did both halves, so none is evidence
    about an affordance — pooling manufactured a certificate out of unrelated trials.

    The other two were worse: already-complete composer certificates that pooling
    MISREPORTED. `rootval` (composer 0✗ 2✗ 3✗ 4✗✗ 5✓✓) and `svcerrors` (0✗ 2✗ 5✓) both
    need L5 for composer, and both were recorded as binding at L2 because devin passed L2.
    A three-rung understatement, the same error `defval` exposed: taking the lowest passing
    rung ACROSS solvers lets the stronger model erase the weaker model's difficulty, which
    is the one quantity this dataset exists to measure.

    So a certificate is per solver by construction. A unit is certified when some solver
    failed it at L0 and passed a rung >= 2 itself, and it binds at THAT solver's lowest
    passing rung. `kind` is kept as "single" for callers that still read it, and is now
    always "single" because that is the only thing a certificate was ever made of.
    `binds_for` names the solver; `solvers` lists every solver holding a certificate on the
    unit, which is what a multi-model dataset actually wants to know.
    """
    bys = certificates_by_solver(jobs_dir)
    out = {}
    for base, per_solver in bys.items():
        solver, cert = min(per_solver.items(), key=lambda kv: kv[1]["rung"])
        out[base] = {
            "rung": cert["rung"],
            "escalated": cert["escalated"],
            "rung_established": cert["rung_established"],
            "n_trials_at_rung": cert["n_trials_at_rung"],
            "n_pass_at_rung": cert["n_pass_at_rung"],
            "thin": cert["thin"],
            "kind": "single",
            "binds_for": [solver],
            "solvers": sorted(per_solver),
            "l0": [solver],
            "l2": [solver],
        }
    return out


def certificates_by_solver(jobs_dir=JOBS):
    """base -> solver -> certificate, each from that solver's OWN rungs.

    certificates() answers "is this unit hard and solvable", pooling every solver's
    verdicts. That is the dataset's headline and it keeps its meaning. This answers a
    different question: for THIS agent, where on the ladder does the unit stop being
    impossible? Two agents give two curves, and the gap between them is the difference
    between "hard for agents" and "hard for this agent" — which is the thing the dataset
    exists to measure and, pooled, cannot show.

    Condemnation stays model-agnostic here too: a unit any solver fixed from the bug
    report alone is not hard for anyone, so it yields no per-solver certificate either.
    """
    out = {}
    bys = ledger_by_solver(jobs_dir)
    for base, bysolver in bys.items():
        # No global condemnation here either — see certificates(). A solver's curve is
        # judged on that solver's own L0 verdict.
        for solver, d in bysolver.items():
            l0 = d.get("0") or []
            if not l0 or max(l0) != 0:
                continue                  # this solver never failed it at L0
            passed = sorted(int(r) for r, v in d.items()
                            if r.isdigit() and int(r) >= 2 and v and max(v) > 0)
            if not passed:
                continue
            rung = passed[0]
            rewards = d.get(str(rung), [])
            below = d.get(str(rung - 1), [])
            out.setdefault(base, {})[solver] = {
                "rung": rung,
                "escalated": rung > 2,
                "rung_established": rung == 2 or bool(below and max(below) == 0),
                "n_trials_at_rung": len(rewards),
                "n_pass_at_rung": sum(1 for r in rewards if r > 0),
                "thin": len(rewards) >= 3 and sum(1 for r in rewards if r > 0) == 1,
            }
    return out


def summary(jobs_dir=JOBS, nonflip_cap=3):
    per = ledger(jobs_dir)
    allt = list(trials(jobs_dir))
    certs = certificates(jobs_dir)
    cert = sorted(certs)
    # Too-easy means NO solver found it hard. Under the old model-agnostic rule this was
    # "some solver passed L0", which put units in too_easy AND certified simultaneously
    # once per-solver certificates arrived. A unit one solver cracked at L0 and another
    # could not is not too easy; it is a unit with two different difficulties.
    _bys = ledger_by_solver(jobs_dir)
    easy = [u for u, d in _bys.items()
            if any(r.get("0") and max(r["0"]) > 0 for r in d.values())
            and not any(r.get("0") and max(r["0"]) == 0 for r in d.values())]
    # "Non-flipping" now means only what it should: fails L0, fails L2, and has been
    # carried up the ladder to the top rung without ever passing. A unit that has simply
    # not been escalated YET is pending work, not a failed task - calling those two things
    # by the same name is what made 46 of the hardest units look like waste.
    import escalate as _esc
    nonflip, pending_esc = [], []
    for u, d in per.items():
        if u in certs or not (d.get("0") and max(d["0"]) == 0):
            continue
        if not (len(d.get("2", [])) >= nonflip_cap and max(d.get("2") or [1]) == 0):
            continue
        rung, _why = _esc.next_rung(_esc.history(d))
        (pending_esc if rung is not None else nonflip).append(u)
    return {
        "units": len(per),
        "trials_total": len(allt),
        "trials_valid": sum(1 for t in allt if t["reward"] is not None and not t["errored"]),
        "trials_errored": sum(1 for t in allt if t["errored"] or t["reward"] is None),
        "certified": sorted(cert), "too_easy": sorted(easy), "nonflip": sorted(nonflip),
        "escalatable": sorted(pending_esc),
        "escalated": sorted(u for u, v in certs.items() if v.get("escalated")),
        "by_model": collections.Counter(t["model"] or "?" for t in allt),
        "tokens": sum(t["tokens"] for t in allt),
        "cost_usd": sum(t["cost"] for t in allt),
    }




def report_multimodel(jobs_dir=JOBS):
    """Print the dataset as what it is: a multi-model dataset."""
    import collections as _c
    c = certificates(jobs_dir)
    kind = _c.Counter(v["kind"] for v in c.values())
    binds = _c.Counter(s for v in c.values() for s in v["binds_for"])
    thin = [k for k, v in c.items() if v["thin"]]
    esc = [k for k, v in c.items() if v.get("escalated")]
    print(f"certificates     {len(c)}")
    print(f"  above L2       {len(esc)}   flip needed more affordance than the contract")
    print(f"  thin evidence  {len(thin)}   one pass in 3+ trials at the binding rung")
    jumped = [k for k, v in c.items() if not v["rung_established"]]
    if jumped:
        print(f"  rung by-jump   {len(jumped)}   binds at a rung never shown to be "
              f"NEEDED — 'flips by L{c[jumped[0]]['rung']}', not 'needs' it")
    print(f"  single-solver  {kind.get('single', 0)}   flip isolates the affordance")
    print(f"  cross-solver   {kind.get('cross', 0)}   L2 passed on a different model than "
          f"L0 failed — a weaker claim, kept and labelled")
    print(f"  binds for      {dict(binds)}")


if __name__ == "__main__":
    s = summary()
    print(f"units            {s['units']}")
    print(f"trial dirs       {s['trials_total']}")
    print(f"  with a verdict {s['trials_valid']}")
    print(f"  errored/no-run {s['trials_errored']}")
    print(f"certified        {len(s['certified'])}")
    print(f"too-easy         {len(s['too_easy'])}")
    print(f"non-flipping     {len(s['nonflip'])}   (fails every rung to the top)")
    print(f"escalatable      {len(s['escalatable'])}   fails L0+L2, has a rung left to try")
    print(f"  of certified, flipped above L2: {len(s['escalated'])}")
    print(f"tokens           {s['tokens']/1e9:.2f}B")
    print(f"cost             ${s['cost_usd']:.2f}")
    print(f"by solver        {dict(s['by_model'])}")
    if s["certified"]:
        print(f"trials/certified {s['trials_valid']/len(s['certified']):.1f}")
    report_multimodel()
