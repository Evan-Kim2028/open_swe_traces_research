"""How Composer and Devin work a task, read from their trial traces.

    uv run python -m openswe_traces.analysis.traces [--json | --rows]

--rows prints one JSON object per trial, for plotting from another project.

Each trial is reduced to the same handful of measures whichever agent ran it:

  calls      tool calls, split into read, search, edit, run and test (a run whose command
             invokes `go test` or tests/test.sh)
  first_edit fraction of the trial's calls made before its first edit: how long it looks
             before it touches code
  minutes    agent wall time, from result.json agent_execution
  passed     the verifier's verdict

Composer's calls come from harbor's ATIF trajectory. Devin's ATIF trajectory keeps only the
tail of a session (29 of 89 calls on the trial checked), so its calls come from its own
session database, tool_call_state, which records every call.

Two comparisons: every graded trial by model and level, and the (task, level) cells both
models ran, where the task is held fixed.
"""
import collections
import json
import sqlite3
import statistics
import sys
from datetime import datetime

from openswe_traces.ladder import ledger as TL

TEST = ("go test", "test.sh")
COMPOSER_KIND = {"readToolCall": "read", "grepToolCall": "search", "globToolCall": "search",
                 "lsToolCall": "search", "editToolCall": "edit", "writeToolCall": "edit",
                 "deleteToolCall": "edit", "shellToolCall": "run"}
DEVIN_KIND = {"read": "read", "grep": "search", "find_file_by_name": "search",
              "edit": "edit", "write": "edit", "exec": "run"}


SEARCH_CMDS = ("grep", "rg", "find", "ls", "tree")
READ_CMDS = ("cat", "head", "tail", "sed -n", "less", "wc", "nl")


def _kind(kind, command):
    """Shell commands are classified by what they run, so a shell grep counts as a search
    for both agents whether or not the agent has a dedicated search tool."""
    if kind == "run" and command:
        cmd = command.strip()
        if cmd.startswith("cd ") and "&&" in cmd:
            cmd = cmd.split("&&", 1)[1].strip()
        if any(t in command for t in TEST):
            return "test"
        if cmd.startswith(SEARCH_CMDS):
            return "search"
        if cmd.startswith(READ_CMDS):
            return "read"
    return kind or "other"


def composer_calls(tdir):
    try:
        steps = json.load(open(f"{tdir}/agent/trajectory.json")).get("steps") or []
    except (OSError, ValueError):
        return None
    out = []
    for s in steps:
        for tc in s.get("tool_calls") or []:
            args = tc.get("arguments") or {}
            out.append(_kind(COMPOSER_KIND.get(tc.get("function_name")),
                             args.get("command") if isinstance(args, dict) else None))
    return out


def devin_calls(tdir):
    """Calls in the order Devin made them. tool_call_state is rewritten as calls complete, so
    its row order is not chronological; the assistant messages in message_nodes are, and
    each call appears on several nodes, so it is counted once by id."""
    try:
        con = sqlite3.connect(f"file:{tdir}/agent/sessions.db?mode=ro", uri=True)
        rows = con.execute("select chat_message from message_nodes order by node_id").fetchall()
        con.close()
    except sqlite3.Error:
        return None
    out, seen = [], set()
    for (m,) in rows:
        try:
            msg = json.loads(m)
        except ValueError:
            continue
        for tc in msg.get("tool_calls") or []:
            if tc.get("id") in seen:
                continue
            seen.add(tc.get("id"))
            args = tc.get("arguments") or {}
            out.append(_kind(DEVIN_KIND.get(tc.get("name")), args.get("command") if isinstance(args, dict) else None))
    return out


def _minutes(tdir):
    try:
        ae = json.load(open(f"{tdir}/result.json")).get("agent_execution") or {}
        a, b = (datetime.fromisoformat(ae[k].replace("Z", "+00:00")) for k in ("started_at", "finished_at"))
        return (b - a).total_seconds() / 60
    except (OSError, ValueError, KeyError, AttributeError):
        return None


def trials(jobs_dir=TL.JOBS):
    """One record per graded Composer or Devin trial that kept its trace."""
    for t in TL.trials(jobs_dir):
        if t["reward"] is None or t["errored"] or t["rung"] == "1":
            continue
        model = TL.solver_of(t["model"])
        calls = {"composer": composer_calls, "devin": devin_calls}.get(model, lambda _: None)(t["dir"])
        if not calls:
            continue
        kinds = collections.Counter(calls)
        first = next((i for i, k in enumerate(calls) if k == "edit"), None)
        yield {"model": model, "base": t["base"], "rung": t["rung"], "passed": t["reward"] > 0,
               "calls": len(calls), **{k: kinds.get(k, 0) for k in ("read", "search", "edit", "run", "test")},
               "first_edit": None if first is None else first / len(calls),
               "minutes": _minutes(t["dir"])}


def _median(xs):
    xs = [x for x in xs if x is not None]
    return round(statistics.median(xs), 2) if xs else None


def summarise(rs):
    keys = ("calls", "read", "search", "edit", "run", "test", "first_edit", "minutes")
    by = collections.defaultdict(list)
    for r in rs:
        by[(r["model"], r["passed"])].append(r)
        by[(r["model"], "all")].append(r)
    table = {f"{m} {p if p == 'all' else ('pass' if p else 'fail')}": {"trials": len(v), **{k: _median(x[k] for x in v) for k in keys}}
             for (m, p), v in sorted(by.items(), key=lambda kv: str(kv[0]))}
    mix = {}
    for m in ("composer", "devin"):
        tot = collections.Counter()
        for r in rs:
            if r["model"] == m:
                tot.update({k: r[k] for k in ("read", "search", "edit", "run", "test")})
        n = sum(tot.values()) or 1
        mix[m] = {k: round(100 * v / n) for k, v in tot.items()}
    return {"medians": table, "call mix %": mix}


STEP = {"0": "L1", "2": "L2", "3": "L3-4", "4": "L3-4", "5": "L5-6", "6": "L5-6"}


def by_level(rs):
    """Passing and failing runs at each ladder step, per model: how much work, and its shape.

    explore  share of calls that read or search the repository
    test     share of calls that run the tests
    """
    groups = collections.defaultdict(list)
    for r in rs:
        groups[(r["model"], STEP.get(r["rung"], "?"), "pass" if r["passed"] else "fail")].append(r)
    out = {}
    for (m, step, v), g in sorted(groups.items()):
        explore = [(x["read"] + x["search"]) / x["calls"] for x in g if x["calls"]]
        test = [x["test"] / x["calls"] for x in g if x["calls"]]
        out[f"{m} {step} {v}"] = {"trials": len(g), "calls": _median(x["calls"] for x in g),
                                  "explore": _median(explore), "test": _median(test),
                                  "edits": _median(x["edit"] for x in g),
                                  "first_edit": _median(x["first_edit"] for x in g),
                                  "minutes": _median(x["minutes"] for x in g)}
    return out


def flips(rs):
    """Same model, same task: the failed run just below its first pass against that pass.

    The by-level view mixes tasks, since only tasks that failed lower down reach higher
    levels. Holding the task fixed isolates what the extra information changed."""
    fam = collections.defaultdict(list)
    for r in rs:
        fam[(r["model"], r["base"])].append(r)
    pairs = collections.defaultdict(list)
    for (m, _), g in fam.items():
        passes = [r for r in g if r["passed"]]
        if not passes:
            continue
        p = min(passes, key=lambda r: int(r["rung"]))
        lower = [r for r in g if not r["passed"] and int(r["rung"]) < int(p["rung"])]
        if lower:
            pairs[m].append((max(lower, key=lambda r: int(r["rung"])), p))
    explore = lambda r: (r["read"] + r["search"]) / r["calls"] if r["calls"] else None
    out = {}
    for m, ps in pairs.items():
        out[m] = {"tasks": len(ps)}
        for k in ("calls", "edit", "test", "first_edit", "minutes"):
            out[m][k] = (_median(f[k] for f, _ in ps), _median(p[k] for _, p in ps))
        out[m]["explore"] = (_median(explore(f) for f, _ in ps), _median(explore(p) for _, p in ps))
    return out


def paired(rs):
    """Cells (task, level) both models ran: each model's median measure on the same cells."""
    cell = collections.defaultdict(dict)
    for r in rs:
        cell[(r["base"], r["rung"])].setdefault(r["model"], r)
    both = [v for v in cell.values() if "composer" in v and "devin" in v]
    out = {"cells": len(both),
           "both pass": sum(v["composer"]["passed"] and v["devin"]["passed"] for v in both),
           "only composer passes": sum(v["composer"]["passed"] and not v["devin"]["passed"] for v in both),
           "only devin passes": sum(v["devin"]["passed"] and not v["composer"]["passed"] for v in both),
           "both fail": sum(not v["composer"]["passed"] and not v["devin"]["passed"] for v in both)}
    for k in ("calls", "read", "edit", "test", "first_edit", "minutes"):
        out[k] = {m: _median(v[m][k] for v in both) for m in ("composer", "devin")}
    return out


def cli():
    if "--rows" in sys.argv:
        for r in trials():
            print(json.dumps(r))
        return
    rs = list(trials())
    res = {"trials": collections.Counter(r["model"] for r in rs), **summarise(rs),
           "by level": by_level(rs), "flips": flips(rs), "paired": paired(rs)}
    if "--json" in sys.argv:
        print(json.dumps(res, indent=1, default=str))
        return
    print("trials with a trace:", dict(res["trials"]))
    print("\nmedians per trial")
    for k, v in res["medians"].items():
        print(f"  {k:16s} {v}")
    print("\ncall mix %", res["call mix %"])
    print("\nby ladder step")
    for k, v in res["by level"].items():
        print(f"  {k:22s} {v}")
    print("\nsame task, failed run below the first pass -> the pass:", res["flips"])
    print("\nsame task, same level:", res["paired"])


if __name__ == "__main__":
    cli()
