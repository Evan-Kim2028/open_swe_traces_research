"""Every number *Difficulty is an information gap* states, computed from the ledger.

    uv run python -m openswe_traces.reports.paper_numbers [--json]

Each line is labelled with the phrase the write-up uses, in the write-up's level numbering
(the bug report is L1; the code calls it rung 0). Devin's dollar cost is not in the trial
results: Devin bills in ACUs, so its cost row comes from the account export and is printed
here as missing rather than guessed.
"""
import collections
import json
import sys

from openswe_traces.ladder import ledger as TL

# The authoring census, taken 2026-09-22 across the main checkout and every oswt-*
# worktree. roots.units() can no longer reproduce it: worktree_gc retired those worktrees
# into analytics/worktree_salvage. Restate it here if a new census is taken.
AUTHORED = 591

# The write-up merges L4 into L3 and L6 into L5: the cut keeps the signatures and nearly
# every task has one hidden test file, so those rungs are usually the same task.
STEP = {"2": "L2 full description", "3": "L3-4 test names", "4": "L3-4 test names",
        "5": "L5-6 test file", "6": "L5-6 test file"}
LEVEL_GROUP = {"0": "L1 the screen", "2": "L2 the certificate"}


def _solver_tokens(t_dir):
    """(input+output, cache) tokens for one trial, whichever field its agent fills."""
    try:
        ar = json.load(open(f"{t_dir}/result.json")).get("agent_result") or {}
    except (OSError, ValueError):
        return 0, 0
    tok = (ar.get("n_input_tokens") or 0) + (ar.get("n_output_tokens") or 0)
    cache = ar.get("n_cache_tokens") or 0
    if not tok:
        for mu in (ar.get("model_usage") or {}).values():
            if isinstance(mu, dict):
                tok += (mu.get("n_input_tokens") or 0) + (mu.get("n_output_tokens") or 0)
                cache += mu.get("n_cache_tokens") or 0
    return tok, cache


def _scale(trials, valid, n_certs, by_solver, s):
    """Tokens and dollars for everything Stage 1 ran, and what the next measurements cost.

    Composer and Grok report tokens and cost_usd per trial, cached tokens inside input.
    Devin reports neither to harbor reliably; its exact per-request counts come from its
    session databases (reports.devin_usage), cached tokens on top of input, priced at the
    SWE-2 promotional rate that matched its ACU bill. Host Devin sessions are authoring;
    host Grok authoring sessions come from reports.grok_usage.
    """
    from openswe_traces.reports import devin_usage as DU

    tok, cache, usd, runs = (collections.Counter() for _ in range(4))
    for t in trials:
        m = TL.solver_of(t["model"])
        if m == "devin":
            continue
        a, c = _solver_tokens(t["dir"])
        tok[m] += a
        cache[m] += c
        usd[m] += t["cost"]
        runs[m] += 1
    dv = DU.exact_report()
    if not dv["trials"]["sessions"]:
        raise SystemExit(f"no Devin session databases under {DU.JOBS}; "
                         "set OPENSWE_REPO to the checkout that holds the jobs")
    for part, name in (("trials", "devin trials"), ("host", "devin host, mostly authoring")):
        c = dv[part]
        tok[name] = c["input"] + c["cache_read"] + c["output"]
        cache[name] = c["cache_read"]
        usd[name] = DU.cost(c, DU.PRICE_PROMO)
        runs[name] = c["sessions"]
    from openswe_traces.reports import grok_usage as GU
    ga = GU.report().get("authoring", collections.Counter())
    tok["grok authoring"] = ga["input"] + ga["output"]
    cache["grok authoring"] = ga["cache_read"]
    usd["grok authoring"] = ga["usd"]
    runs["grok authoring"] = ga["sessions"]
    total_tok, total_usd = sum(tok.values()), sum(usd.values())
    authoring = usd["devin host, mostly authoring"] + usd["grok authoring"]
    grading = total_usd - authoring

    # Observed means per trial run, for pricing measurements Stage 1 could not afford.
    per_run = {"composer": (usd["composer"] / runs["composer"], tok["composer"] / runs["composer"]),
               "devin": (usd["devin trials"] / runs["devin trials"],
                         tok["devin trials"] / runs["devin trials"])}
    both = (per_run["composer"][0] + per_run["devin"][0], per_run["composer"][1] + per_run["devin"][1])
    graded_runs = collections.Counter(TL.solver_of(t["model"]) for t in valid if t["rung"] != "1")
    climb = len(valid) / max(1, len(s["certified"]) + len(s["too_easy"]) + len(s["nonflip"]))
    scenarios = {
        "rerun variance: 3 repeats x 50 tasks x L1,L2": 3 * 50 * 2,
        "no selection: both models climb the same 100 tasks": round(100 * climb),
        "full grid: 591 tasks x 6 levels x 3 repeats, per model": AUTHORED * 6 * 3,
    }
    return {
        "tokens by source": {k: f"{v / 1e9:.2f}B" for k, v in tok.items()},
        "tokens total": f"{total_tok / 1e9:.2f}B",
        "cache reads share": f"{100 * sum(cache.values()) / total_tok:.0f}%",
        "runs or sessions": dict(runs),
        "cost_usd by source (Devin at SWE-2 promo)": {k: round(v) for k, v in usd.items()},
        "cost_usd total": round(total_usd),
        "authoring cost per authored task (Devin and Grok host)": round(authoring / AUTHORED, 2),
        "grading cost per certificate": round(grading / n_certs, 2),
        "per run, cost and tokens": {m: (round(c, 2), f"{t_ / 1e6:.1f}M") for m, (c, t_) in per_run.items()},
        "runs per graded task, all models": round(climb, 1),
        "next measurements, cost for both models": {
            k: (f"{n} runs per model", f"${n * both[0]:,.0f}", f"{n * both[1] / 1e9:.0f}B tokens")
            for k, n in scenarios.items()},
        "graded runs by model": dict(graded_runs),
    }


def compute(jobs_dir=TL.JOBS):
    s = TL.summary(jobs_dir)
    certs = TL.certificates(jobs_dir)
    by_solver = TL.certificates_by_solver(jobs_dir)
    bys = TL.ledger_by_solver(jobs_dir)
    trials = list(TL.trials(jobs_dir))
    valid = [t for t in trials if t["reward"] is not None and not t["errored"]]

    authored = AUTHORED
    graded = len(certs) + len(s["too_easy"]) + len(s["nonflip"])
    first = collections.Counter(STEP[str(c["rung"])] for c in certs.values())

    per_solver = {m: collections.Counter(STEP[str(c["rung"])] for b in by_solver.values()
                                         for mm, c in b.items() if mm == m)
                  for m in ("composer", "devin", "grok")}
    two_solver = sorted(b for b, v in by_solver.items() if len(v) > 1)
    def exhausted_by(r):
        """Failed the bug report and every level tried, up to and including the top."""
        return "0" in r and "6" in r and all(max(v) == 0 for v in r.values() if v)

    exhausted = {m: sorted(b for b, d in bys.items() if m in d and exhausted_by(d[m]))
                 for m in ("composer", "devin", "grok")}

    both_l1 = collections.Counter()
    for d in bys.values():
        c, v = d.get("composer", {}).get("0"), d.get("devin", {}).get("0")
        if c and v:
            both_l1[f"composer {'pass' if max(c) > 0 else 'fail'}, "
                    f"devin {'pass' if max(v) > 0 else 'fail'}"] += 1

    runs = collections.Counter(TL.solver_of(t["model"]) for t in trials)
    cost = collections.Counter()
    for t in trials:
        cost[TL.solver_of(t["model"])] += t["cost"]
    # Priced runs only: Composer and Grok report cost_usd per trial, Devin does not.
    level_runs, level_cost = collections.Counter(), collections.Counter()
    for t in trials:
        if t["rung"] == "1" or TL.solver_of(t["model"]) not in ("composer", "grok"):
            continue
        g = LEVEL_GROUP.get(t["rung"], "L3-L6 above L2")
        level_runs[g] += 1
        level_cost[g] += t["cost"]
    tokens = collections.Counter()
    cache = collections.Counter()
    for t in trials:
        tok, ca = _solver_tokens(t["dir"])
        tokens[TL.solver_of(t["model"])] += tok
        cache[TL.solver_of(t["model"])] += ca

    # A second model screening a certified task at L1: any solver other than the one the
    # certificate binds for, with an L1 verdict of its own.
    second = {b: [m for m, d in bys[b].items() if m not in c["binds_for"] and d.get("0")]
              for b, c in certs.items()}
    second = {b: ms for b, ms in second.items() if ms}
    second_pass = sorted(b for b, ms in second.items()
                         if any(max(bys[b][m]["0"]) > 0 for m in ms))
    # Split by the screening model: Composer's second screens are Devin's certificates, which
    # reached Devin only after Composer had failed them, so pooling the two hides the result.
    by_repo = collections.Counter()
    for b, ms in second.items():
        repo = "go-github" if b.startswith("go-github") else "everywhere else"
        for m in ms:
            by_repo[f"{m} on {repo}, screened"] += 1
            by_repo[f"{m} on {repo}, passed"] += max(bys[b][m]["0"]) > 0

    scale = _scale(trials, valid, len(certs), by_solver, s)

    return {
        "scale and cost": scale,
        "funnel": {"authored": authored, "trialled": s["units"], "graded": graded,
                   "never ran": authored - s["units"], "waiting": s["units"] - graded,
                   "solved from the bug report": len(s["too_easy"]),
                   "certified": len(certs), "failed every level": s["nonflip"]},
        "certified binds at (lowest model)": dict(sorted(first.items())),
        "certified share": {k: round(100 * v / len(certs), 1) for k, v in sorted(first.items())},
        "certificates per model": {m: dict(sorted(c.items())) for m, c in per_solver.items()},
        "tasks with certificates from two models": len(two_solver),
        "two-model tasks": two_solver,
        "exhausted per model (fails L1 through L6)": exhausted,
        "both screened at L1": dict(both_l1),
        "trials": {"total": len(trials), "with a verdict": len(valid),
                   "errored or no verdict": len(trials) - len(valid)},
        "runs by model": dict(runs),
        "cost_usd by model (from results; Devin is ACU-billed, see export)":
            {m: round(v, 2) for m, v in cost.items()},
        "per level, Composer and Grok runs": dict(level_runs),
        "per level, cost_usd": {k: round(v, 2) for k, v in level_cost.items()},
        "per level, cost per run": {k: round(level_cost[k] / n, 2) for k, n in level_runs.items()},
        "runs per certificate": round(len(valid) / len(certs), 2),
        "tokens by model, input+output": dict(tokens),
        "tokens by model, cache": dict(cache),
        "certified tasks a second model screened at L1": len(second),
        "of those, the second model passed at L1": len(second_pass),
        "second screen by repository": dict(by_repo),
    }


def cli():
    out = compute()
    if "--json" in sys.argv:
        json.dump(out, sys.stdout, indent=1, sort_keys=True, default=str)
        print()
        return
    for k, v in out.items():
        print(f"{k:55s} {v}")


if __name__ == "__main__":
    cli()
