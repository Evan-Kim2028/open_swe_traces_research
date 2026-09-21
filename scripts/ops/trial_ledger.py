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
        base, rung = name.rsplit("-L", 1)
        rung = rung[:1]

        reward = err = None
        tok = cost = 0.0
        # Who produced this verdict. "Hard at L0" is solver-dependent, so a bank screened by
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
               "rung": rung, "reward": reward, "errored": err is not None,
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

    The bank is deliberately multi-model: tasks are trialled by whichever solver has
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
        passed_l2 = {s for s, d in bysolver.items() if d.get("2") and max(d["2"]) > 0}
        if not (failed_l0 and passed_l2):
            continue
        shared = failed_l0 & passed_l2
        out[base] = {"l0": sorted(failed_l0), "l2": sorted(passed_l2),
                     "kind": "single" if shared else "cross",
                     "binds_for": sorted(shared) if shared else sorted(passed_l2)}
    return out


def summary(jobs_dir=JOBS, nonflip_cap=3):
    per = ledger(jobs_dir)
    allt = list(trials(jobs_dir))
    cert = [u for u, d in per.items()
            if d.get("0") and max(d["0"]) == 0 and d.get("2") and max(d["2"]) > 0]
    easy = [u for u, d in per.items() if d.get("0") and max(d["0"]) > 0]
    nonflip = [u for u, d in per.items()
               if d.get("0") and max(d["0"]) == 0
               and len(d.get("2", [])) >= nonflip_cap and max(d.get("2") or [1]) == 0]
    return {
        "units": len(per),
        "trials_total": len(allt),
        "trials_valid": sum(1 for t in allt if t["reward"] is not None and not t["errored"]),
        "trials_errored": sum(1 for t in allt if t["errored"] or t["reward"] is None),
        "certified": sorted(cert), "too_easy": sorted(easy), "nonflip": sorted(nonflip),
        "by_model": collections.Counter(t["model"] or "?" for t in allt),
        "tokens": sum(t["tokens"] for t in allt),
        "cost_usd": sum(t["cost"] for t in allt),
    }




def report_multimodel(jobs_dir=JOBS):
    """Print the bank as what it is: a multi-model bank."""
    import collections as _c
    c = certificates(jobs_dir)
    kind = _c.Counter(v["kind"] for v in c.values())
    binds = _c.Counter(s for v in c.values() for s in v["binds_for"])
    print(f"certificates     {len(c)}")
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
    print(f"non-flipping     {len(s['nonflip'])}")
    print(f"tokens           {s['tokens']/1e9:.2f}B")
    print(f"cost             ${s['cost_usd']:.2f}")
    print(f"by solver        {dict(s['by_model'])}")
    if s["certified"]:
        print(f"trials/certified {s['trials_valid']/len(s['certified']):.1f}")
    report_multimodel()
