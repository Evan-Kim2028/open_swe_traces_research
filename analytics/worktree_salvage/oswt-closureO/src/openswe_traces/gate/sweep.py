"""Sweep the gate over a set of task dirs; emit a verdict matrix.

Reproducible entry point for the consolidation sweep: iterates Harbor task
dirs, runs ``gate_task`` (executed + static + documentary impls), and collects
a per-task matrix plus the registry table. Resume-safe: executed results are
cached in each dir's ``validation.json`` by content key, so a rerun skips
unchanged packages.
"""

from __future__ import annotations

import time
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.gate.core import Verdict, registry, rule_ids
from openswe_traces.gate.runner import gate_task
from openswe_traces.synth.rules import is_harbor_task


def iter_task_dirs(*roots: Path) -> list[Path]:
    out: list[Path] = []
    for root in roots:
        root = Path(root)
        if not root.is_dir():
            continue
        cands = [root] if is_harbor_task(root) else sorted(
            p
            for p in root.rglob("*")
            if p.is_dir()
            and is_harbor_task(p)
            and not any(part.startswith("_") for part in p.relative_to(root).parts)
        )
        out.extend(cands)
    return sorted(set(out))


def registry_table() -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for rid in rule_ids():
        for impl in registry().get(rid, []):
            rows.append(
                {
                    "rule": rid,
                    "tier": impl.tier,
                    "implemented": "yes",
                    "provenance": impl.provenance,
                    "runs": impl.description or impl.run.__name__,
                }
            )
    return rows


def _mark(v: Verdict | None) -> str:
    if v is None:
        return "—"
    if v.skipped:
        return "skip"
    return ("pass" if v.passed else "FAIL") + f"({v.tier[:1]})"


def sweep(
    task_dirs: list[Path],
    *,
    docker_run: Any = None,
    run_executed: bool = True,
    force: bool = False,
    progress: Callable[[str], None] | None = None,
) -> dict[str, Any]:
    """Run the full gate over every dir; return the verdict matrix."""
    log = progress or (lambda m: None)
    matrix: dict[str, dict[str, Any]] = {}
    t_all = time.monotonic()
    for i, td in enumerate(task_dirs, 1):
        t0 = time.monotonic()
        try:
            rep = gate_task(
                td, docker_run=docker_run, force=force, run_executed=run_executed
            )
            by = {v.rule_id: v for v in rep.primaries}
            matrix[td.as_posix()] = {
                "ok": rep.ok,
                "blockers": rep.blockers,
                "seconds": round(time.monotonic() - t0, 1),
                "verdicts": {
                    rid: {
                        "mark": _mark(by.get(rid)),
                        "passed": by[rid].passed if rid in by else None,
                        "skipped": by[rid].skipped if rid in by else None,
                        "tier": by[rid].tier if rid in by else None,
                        "evidence": (by[rid].evidence if rid in by else "")[:400],
                    }
                    for rid in rule_ids()
                },
            }
            n_fail = sum(1 for v in rep.primaries if not v.skipped and not v.passed)
            log(f"[{i}/{len(task_dirs)}] {td.name}: ok={rep.ok} fails={n_fail} ({rep.seconds:.0f}s)")
        except Exception as exc:  # noqa: BLE001 - one bad dir must not kill the sweep
            matrix[td.as_posix()] = {
                "ok": False,
                "blockers": [f"sweep error: {type(exc).__name__}: {exc}"],
                "seconds": round(time.monotonic() - t0, 1),
                "verdicts": {},
            }
            log(f"[{i}/{len(task_dirs)}] {td.name}: ERROR {exc}")
    return {
        "n_tasks": len(matrix),
        "seconds": round(time.monotonic() - t_all, 1),
        "registry": registry_table(),
        "matrix": matrix,
    }


def new_failures(matrix: dict[str, Any]) -> dict[str, list[str]]:
    """Failures the four added checks found: A5, A10(exec), B7 leaks, nesting, coverage."""
    watch = {"A5", "A10", "B7", "coverage", "nesting"}
    out: dict[str, list[str]] = {k: [] for k in sorted(watch)}
    for td, row in matrix.items():
        for rid, v in (row.get("verdicts") or {}).items():
            if rid in watch and v.get("passed") is False and not v.get("skipped"):
                out[rid].append(f"{Path(td).name}: {v.get('evidence', '')[:200]}")
    return out


def kill_counts(matrix: dict[str, Any]) -> dict[str, dict[str, int]]:
    """rule -> {evaluated, failed, skipped} over primary verdicts."""
    out = {rid: {"evaluated": 0, "failed": 0, "skipped": 0} for rid in rule_ids()}
    for row in matrix.values():
        for rid, v in (row.get("verdicts") or {}).items():
            if rid not in out:
                continue
            if v.get("skipped") or v.get("passed") is None:
                out[rid]["skipped"] += 1
            else:
                out[rid]["evaluated"] += 1
                if not v["passed"]:
                    out[rid]["failed"] += 1
    return out
