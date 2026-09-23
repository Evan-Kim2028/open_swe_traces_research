"""Did each verdict measure the model? Sort every trial by what its grader printed, and scan
every trajectory for web tools. The findings of 2026-09-23 are in
analytics/research/harness_audit_2026-09-23.md.

    uv run python -m openswe_traces.analysis.harness_audit      # run from the data repo root
"""
from __future__ import annotations

import collections
import json
import pathlib
import re
import sqlite3

from openswe_traces.ladder import ledger as T

OK_RAN = re.compile(r"^ok\s+\S+\s+[\d.]+s\s*$", re.M)          # a package that ran tests
FAIL_TEST = re.compile(r"^\s*--- FAIL", re.M)
WEB = re.compile(r"web|fetch|browse|url", re.I)


def verdict_class(t: dict) -> str:
    d = pathlib.Path(t["dir"])
    if t["void"]:
        return "void: " + t["void"]
    if t["errored"] or t["reward"] is None:
        return "no verdict (harness error)"
    p = d / "verifier/test-stdout.txt"
    if not p.exists():
        return f"reward {t['reward']} without grader output"
    out = p.read_text(errors="replace")
    if t["reward"] == 1.0:
        patch = d / "artifacts/logs/artifacts/agent.patch"
        text = patch.read_text(errors="replace") if patch.exists() else ""
        if not OK_RAN.search(out):
            return "PASS vacuous: no package ran a test"
        if re.search(r"^\+\+\+ b/\S*_test\.go", text, re.M):
            return "PASS, agent also edited test files"
        return "PASS"
    if "panic: test timed out" in out:
        return "FAIL timeout"
    if FAIL_TEST.search(out):
        return "FAIL tests ran and failed"
    if "[build failed]" in out:
        return "FAIL agent code does not build"
    if "[setup failed]" in out:
        return "FAIL setup failed (unexplained)"
    if re.search(r"^FAIL\s", out, re.M):
        return "FAIL package panic or exit"
    return "FAIL zero with no failing test (suspicious)"


def tool_calls(agent_dir: pathlib.Path) -> list[tuple[str, str]]:
    """(tool name, arguments) from an ATIF trajectory, or Devin's sessions.db."""
    calls: list[tuple[str, str]] = []
    try:
        for s in json.load(open(agent_dir / "trajectory.json")).get("steps", []):
            for c in s.get("tool_calls") or []:
                calls.append((c.get("function_name") or "?", json.dumps(c.get("arguments"))[:300]))
    except (OSError, ValueError):
        pass
    db = agent_dir / "sessions.db"
    if not calls and db.exists():
        try:
            con = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
            for (m,) in con.execute("select message from message_nodes"):
                j = json.loads(m)
                for c in (j.get("chat_message") or j).get("tool_calls") or []:
                    f = c.get("function") or c
                    calls.append((f.get("name", "?"), str(f.get("arguments"))[:300]))
        except (sqlite3.Error, ValueError):
            pass
    return calls


def audit() -> dict:
    classes = collections.Counter()
    by_solver = collections.defaultdict(collections.Counter)
    web, n_calls = [], collections.Counter()
    for t in T.trials():
        s = T.solver_of(t["model"])
        k = verdict_class(t)
        classes[k] += 1
        by_solver[s][k] += 1
        calls = tool_calls(pathlib.Path(t["dir"]) / "agent")
        n_calls[s] += len(calls)
        web += [{"trial": f"{t['job']}/{pathlib.Path(t['dir']).name}", "solver": s,
                 "reward": t["reward"], "tool": n, "args": a}
                for n, a in calls if WEB.search(n)]
    return {"verdicts": dict(classes.most_common()),
            "verdicts by solver": {s: dict(c) for s, c in by_solver.items()},
            "tool calls scanned": dict(n_calls), "web tool calls": web}


def main() -> None:
    r = audit()
    for k, n in r["verdicts"].items():
        print(f"{n:6d}  {k}")
    print("\ntool calls scanned:", r["tool calls scanned"])
    print("web tool calls:", len(r["web tool calls"]))
    for w in r["web tool calls"]:
        print("  ", w)


if __name__ == "__main__":
    main()
