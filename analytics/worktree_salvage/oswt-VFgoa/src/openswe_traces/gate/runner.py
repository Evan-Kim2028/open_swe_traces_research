"""Gate orchestration: run impls, merge under tier precedence, decide launch.

``gate_task`` is the full sweep (executed + static + documentary).
``evaluate_gate`` is the cheap read path (no docker) used by verifier-stage
code. ``write_task_validation`` is the ONLY path that writes rule_verdicts —
it always merges, so a documentary re-evaluation can never overwrite executed
evidence. ``ensure_gate`` is the launch boundary.
"""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.gate.context import GateContext
from openswe_traces.gate.core import (
    EXECUTED,
    GateError,
    Verdict,
    _now_iso,
    executed_rule_ids,
    load_validation,
    merge_verdict_rows,
    registry,
    rule_ids,
    stored_verdict_rows,
    verdict_from_dict,
    write_validation,
)


@dataclass
class GateReport:
    task_dir: Path
    primaries: list[Verdict]
    secondaries: list[Verdict]
    blockers: list[str]
    seconds: float
    gate_runs: Any = None  # exec_rules.GateRuns when executed impls ran
    extra: dict[str, Any] = field(default_factory=dict)

    @property
    def ok(self) -> bool:
        return not self.blockers


def _run_impls(ctx: GateContext, *, run_executed: bool) -> list[Verdict]:
    fresh: list[Verdict] = []
    for rid in rule_ids():
        for impl in registry().get(rid, []):
            if impl.tier == EXECUTED and not run_executed:
                continue
            try:
                v = impl.run(ctx)
            except Exception as exc:  # noqa: BLE001 - a crashed impl must not skip silently
                v = Verdict(
                    rid,
                    passed=False,
                    skipped=impl.tier != EXECUTED,
                    tier=impl.tier,
                    evidence=f"impl crashed: {type(exc).__name__}: {exc}"[:1500],
                    produced_at=_now_iso(),
                    provenance=impl.provenance,
                )
            if v is not None:
                fresh.append(v)
    return fresh


def _blockers_for(rows: list[Verdict]) -> list[str]:
    by_rule: dict[str, list[Verdict]] = {}
    for v in rows:
        by_rule.setdefault(v.rule_id, []).append(v)
    blockers: list[str] = []
    for rid in sorted(executed_rule_ids()):
        if not any(v.tier == EXECUTED for v in by_rule.get(rid, [])):
            blockers.append(f"{rid}: executed impl registered but no executed verdict on record")
    for rid in rule_ids():
        group = by_rule.get(rid)
        if not group:
            continue
        top_tier = max(_tier_rank(v.tier) for v in group)
        primary = max((v for v in group if _tier_rank(v.tier) == top_tier), key=lambda v: v.produced_at)
        if not primary.skipped and not primary.passed:
            blockers.append(f"{rid} failed ({primary.tier}): {primary.evidence[:200]}")
    return blockers


def _tier_rank(tier: str) -> int:
    from openswe_traces.gate.core import TIER_ORDER

    return TIER_ORDER.get(tier, -1)


def gate_task(
    task_dir: Path | str,
    *,
    docker_run: Any = None,
    force: bool = False,
    run_executed: bool = True,
    extra: dict[str, Any] | None = None,
    write: bool = True,
) -> GateReport:
    """Run every registered impl, merge under tier precedence, write + decide."""
    td = Path(task_dir)
    t0 = time.monotonic()
    ctx = GateContext(task_dir=td, extra=dict(extra or {}), docker_run=docker_run, force=force)
    fresh = _run_impls(ctx, run_executed=run_executed)
    data = load_validation(td)
    primaries_d, secondaries_d = merge_verdict_rows(stored_verdict_rows(data), fresh)
    if write:
        data["rule_verdicts"] = primaries_d
        if secondaries_d:
            data["rule_verdicts_secondary"] = secondaries_d
        else:
            data.pop("rule_verdicts_secondary", None)
        write_validation(td, data)
    primaries = [v for v in (verdict_from_dict(r) for r in primaries_d) if v]
    secondaries = [v for v in (verdict_from_dict(r) for r in secondaries_d) if v]
    blockers = _blockers_for(primaries + secondaries)
    return GateReport(
        task_dir=td,
        primaries=primaries,
        secondaries=secondaries,
        blockers=blockers,
        seconds=time.monotonic() - t0,
        gate_runs=ctx._runs,
    )


def evaluate_gate(
    task_dir: Path | str,
    extra: dict[str, Any] | None = None,
    *,
    docker_run: Any = None,
    run_executed: bool = False,
) -> list[Verdict]:
    """Merged primaries for a task dir; no writes, no docker by default."""
    rep = gate_task(
        task_dir,
        docker_run=docker_run,
        run_executed=run_executed,
        extra=extra,
        write=False,
    )
    return rep.primaries


def _payload_verdict_rows(payload: dict[str, Any]) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for key in ("rule_verdicts", "rule_verdicts_secondary"):
        for row in payload.get(key) or []:
            if isinstance(row, dict):
                rows.append(row)
    return rows


def attach_payload(
    task_dir: Path | str,
    payload: dict[str, Any],
    *,
    docker_run: Any = None,
) -> dict[str, Any]:
    """payload + merged verdict rows, without writing (legacy attach semantics)."""
    td = Path(task_dir)
    existing = load_validation(td)
    ctx = GateContext(
        task_dir=td, extra={**existing, **payload}, docker_run=docker_run
    )
    fresh = _run_impls(ctx, run_executed=False)
    stored = stored_verdict_rows(existing) + _payload_verdict_rows(payload)
    primaries, secondaries = merge_verdict_rows(stored, fresh)
    data = {
        k: v
        for k, v in payload.items()
        if k not in ("rule_verdicts", "rule_verdicts_secondary")
    }
    data["rule_verdicts"] = primaries
    if secondaries:
        data["rule_verdicts_secondary"] = secondaries
    return data


def write_task_validation(
    task_dir: Path | str,
    payload: dict[str, Any] | None = None,
    *,
    docker_run: Any = None,
    run_executed: bool = False,
) -> dict[str, Any]:
    """The ONLY writer of rule_verdicts. Merges payload + fresh impl verdicts
    with stored rows under tier precedence — a documentary re-evaluation can
    never overwrite executed evidence. Returns the written dict."""
    td = Path(task_dir)
    td.mkdir(parents=True, exist_ok=True)
    payload = dict(payload or {})
    existing = load_validation(td)
    data = {
        **existing,
        **{
            k: v
            for k, v in payload.items()
            if k not in ("rule_verdicts", "rule_verdicts_secondary")
        },
    }
    ctx = GateContext(
        task_dir=td, extra={**existing, **payload}, docker_run=docker_run
    )
    fresh = _run_impls(ctx, run_executed=run_executed)
    stored = list(stored_verdict_rows(existing)) + _payload_verdict_rows(payload)
    primaries, secondaries = merge_verdict_rows(stored, fresh)
    data["rule_verdicts"] = primaries
    if secondaries:
        data["rule_verdicts_secondary"] = secondaries
    else:
        data.pop("rule_verdicts_secondary", None)
    write_validation(td, data)
    return data


def gate_blockers(task_dir: Path | str) -> list[str]:
    """Decision from STORED rows only — no impls run, no writes."""
    rows = [
        v
        for v in (verdict_from_dict(r) for r in stored_verdict_rows(load_validation(task_dir)))
        if v
    ]
    return _blockers_for(rows)


def ensure_gate(
    task_dir: Path | str,
    *,
    docker_run: Any = None,
    force: bool = False,
    extra: dict[str, Any] | None = None,
) -> GateReport:
    """The launch boundary: run the full gate, then refuse on any blocker.

    Raises GateError (a TaskSafetyError) when any rule with an executed impl
    lacks an executed verdict, or any rule's primary verdict fails.
    """
    rep = gate_task(
        task_dir, docker_run=docker_run, force=force, run_executed=True, extra=extra
    )
    if rep.blockers:
        raise GateError(
            f"refuse: gate blockers for {task_dir}: " + "; ".join(rep.blockers),
            blockers=rep.blockers,
        )
    return rep
