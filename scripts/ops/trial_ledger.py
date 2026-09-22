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
                tok = (ar.get("n_input_tokens") or 0) + (ar.get("n_output_tokens") or 0)
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
    """base -> {'solvers': [...], 'kind': 'single'|'cross', 'l0': [...], 'l2': [...]}.

    A unit is certified when some solver fails it at L0 and some solver passes it at L2.
    'single' means one solver did both, so the flip isolates the affordance. 'cross' means
    the L2 pass came from a different model than the L0 failure, so the flip may reflect a
    capability gap rather than the contract - a real certificate, but a weaker claim.
    """
    out = {}
    bys = ledger_by_solver(jobs_dir)
    for base, bysolver in bys.items():
        failed_l0 = {s for s, d in bysolver.items() if d.get("0") and max(d["0"]) == 0}
        if not failed_l0:
            continue
        # CONDEMNATION IS MODEL-AGNOSTIC, and this check was missing: failed_l0 is built
        # per solver, so one solver failing L0 was enough to certify no matter what any
        # other solver did. trial_guard has always used the merged ledger and refuses a
        # unit with max(l0) > 0 as "not a hard unit", so the two have disagreed silently
        # since the rule went in — it simply never showed, because until the second-screen
        # roster almost no unit had an L0 verdict from two solvers.
        #
        # go-github-customprop is the first: composer failed it at L0, devin solved it
        # from the bug report alone. It was listed as a certificate AND in too_easy at the
        # same time. The dataset's claim is that a task is hard for frontier agents, so one
        # agent solving it without the contract disqualifies it however many others failed.
        if {sv for sv, d in bysolver.items() if d.get("0") and max(d["0"]) > 0}:
            continue
        # The flip does not have to happen at L2. A unit that fails L0 and L2 and then
        # passes at L5 is still hard-and-solvable; the rung it needs IS its difficulty.
        # Certifying only at L2 wrote off 46 of the hardest units in the dataset as
        # "non-flipping". The lowest passing rung is the one that binds.
        passed = {}
        for solver, d in bysolver.items():
            for r, rewards in d.items():
                if r.isdigit() and int(r) >= 2 and rewards and max(rewards) > 0:
                    passed.setdefault(int(r), set()).add(solver)
        if not passed:
            continue
        rung = min(passed)
        at_rung = passed[rung]
        shared = failed_l0 & at_rung
        # How much evidence the flip rests on. A certificate is `max(reward) > 0` at the
        # binding rung, which is the same rule the whole dataset uses - but 1 pass in 6 is
        # not the same claim as 1 in 1, and helm-depresolver binds at L3 on 1 of 6.
        # Recording it keeps that difference auditable instead of invisible.
        per_rung = {}
        for d in bys[base].values():
            for rk, rv in d.items():
                if rk.isdigit():
                    per_rung.setdefault(int(rk), []).extend(rv)
        rewards = per_rung.get(rung, [])
        n_pass = sum(1 for r in rewards if r > 0)
        # Was this rung SHOWN to be the one the unit needs, or merely the first one
        # tried that worked? An earlier policy probed L5 directly, so 32 units bind at
        # L5 having never been asked whether L3 or L4 would have done. "Flips by L5" is
        # a weaker statement than "needs L5" and the histogram must not blur them.
        # L2 is established by construction: L0 failed and the contract is the rung.
        below = per_rung.get(rung - 1, [])
        established = rung == 2 or bool(below and max(below) == 0)
        out[base] = {"l0": sorted(failed_l0), "l2": sorted(at_rung),
                     "rung": rung, "escalated": rung > 2,
                     "rung_established": established,
                     "kind": "single" if shared else "cross",
                     "n_trials_at_rung": len(rewards), "n_pass_at_rung": n_pass,
                     "thin": len(rewards) >= 3 and n_pass == 1,
                     "binds_for": sorted(shared) if shared else sorted(at_rung)}
    return out


def summary(jobs_dir=JOBS, nonflip_cap=3):
    per = ledger(jobs_dir)
    allt = list(trials(jobs_dir))
    certs = certificates(jobs_dir)
    cert = sorted(certs)
    easy = [u for u, d in per.items() if d.get("0") and max(d["0"]) > 0]
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
