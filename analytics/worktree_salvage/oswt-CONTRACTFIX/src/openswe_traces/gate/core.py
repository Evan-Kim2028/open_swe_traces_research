"""Registry, verdict model, tiered merge, and validation.json IO for the gate.

Evidence tiers: ``executed`` (ran in the task image or in a real trial) >
``static`` (derived from packaged files) > ``documentary`` (copied from notes,
recorded rewards, or hand-maintained fields). The merge rule is monotone: a
lower tier never overwrites a higher-tier verdict for the same rule; it is
retained in ``rule_verdicts_secondary`` for audit.
"""

from __future__ import annotations

import json
from collections.abc import Callable, Iterable, Mapping
from dataclasses import asdict, dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.safety import TaskSafetyError

DOCUMENTARY = "documentary"
STATIC = "static"
EXECUTED = "executed"

TIER_ORDER: dict[str, int] = {DOCUMENTARY: 0, STATIC: 1, EXECUTED: 2}

VALIDATION_FILE = "validation.json"


def _now_iso() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat()


@dataclass(frozen=True)
class Verdict:
    """One rule evaluation. ``skipped`` means the impl could not evaluate."""

    rule_id: str
    passed: bool
    skipped: bool
    tier: str
    evidence: str
    produced_at: str
    provenance: str

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class RuleImpl:
    """A registered implementation of one rule at one tier."""

    rule_id: str
    tier: str
    run: Callable[[Any], Verdict | None]
    provenance: str
    description: str = ""


_REGISTRY: dict[str, list[RuleImpl]] = {}


def register(
    rule_id: str,
    tier: str,
    *,
    provenance: str,
    description: str = "",
) -> Callable[[Callable[[Any], Verdict | None]], Callable[[Any], Verdict | None]]:
    """Register ``fn(ctx) -> Verdict | None`` as an implementation of ``rule_id``."""
    if tier not in TIER_ORDER:
        raise ValueError(f"unknown tier {tier!r}")

    def deco(fn: Callable[[Any], Verdict | None]) -> Callable[[Any], Verdict | None]:
        _REGISTRY.setdefault(rule_id, []).append(
            RuleImpl(rule_id=rule_id, tier=tier, run=fn, provenance=provenance, description=description)
        )
        return fn

    return deco


def registry() -> dict[str, list[RuleImpl]]:
    return _REGISTRY


def rule_ids() -> list[str]:
    """Canonical display order: lettered rules, then named packaging checks."""
    def key(rid: str) -> tuple[int, str]:
        if len(rid) >= 2 and rid[0].isalpha() and rid[1:].isdigit():
            return (0, f"{rid[0]}{int(rid[1:]):04d}")
        return (1, rid)

    return sorted(_REGISTRY, key=key)


def executed_rule_ids() -> set[str]:
    return {rid for rid, impls in _REGISTRY.items() if any(i.tier == EXECUTED for i in impls)}


# --- verdict (de)serialisation -----------------------------------------------


def verdict_to_dict(v: Verdict) -> dict[str, Any]:
    return v.to_dict()


def verdict_from_dict(row: Mapping[str, Any]) -> Verdict | None:
    """Parse a stored row. Rows without ``tier`` are legacy → documentary."""
    rid = str(row.get("rule_id") or "")
    if not rid:
        return None
    tier = str(row.get("tier") or DOCUMENTARY)
    if tier not in TIER_ORDER:
        tier = DOCUMENTARY
    return Verdict(
        rule_id=rid,
        passed=bool(row.get("passed")),
        skipped=bool(row.get("skipped")),
        tier=tier,
        evidence=str(row.get("evidence") or "")[:2000],
        produced_at=str(row.get("produced_at") or ""),
        provenance=str(row.get("provenance") or "legacy"),
    )


# --- merge --------------------------------------------------------------------


def _primary_of(rows: list[Verdict]) -> Verdict:
    """Highest tier wins; within a tier the newest produced_at wins."""

    def key(v: Verdict) -> tuple[int, str]:
        return (TIER_ORDER[v.tier], v.produced_at or "")

    return max(rows, key=key)


def merge_verdict_rows(
    stored: Iterable[Mapping[str, Any]],
    new: Iterable[Verdict],
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    """Merge stored rows + fresh verdicts → (primaries, secondaries) as dicts.

    ``stored`` may mix primary and secondary rows (both lists are read).
    Ordering of the primary list follows ``rule_ids()`` registration order,
    then alphabetical for unregistered ids.
    """
    by_rule: dict[str, list[Verdict]] = {}
    for row in stored:
        v = verdict_from_dict(row)
        if v is not None:
            by_rule.setdefault(v.rule_id, []).append(v)
    for v in new:
        by_rule.setdefault(v.rule_id, []).append(v)

    order = {rid: i for i, rid in enumerate(rule_ids())}
    primaries: list[dict[str, Any]] = []
    secondaries: list[dict[str, Any]] = []
    for rid in sorted(by_rule, key=lambda r: (order.get(r, 10_000), r)):
        rows = by_rule[rid]
        prim = _primary_of(rows)
        primaries.append(prim.to_dict())
        secondaries.extend(v.to_dict() for v in rows if v is not prim)
    return primaries, secondaries


# --- validation.json IO --------------------------------------------------------


def load_validation(task_dir: Path | str) -> dict[str, Any]:
    path = Path(task_dir) / VALIDATION_FILE
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def stored_verdict_rows(data: Mapping[str, Any]) -> list[Mapping[str, Any]]:
    rows: list[Mapping[str, Any]] = []
    for key in ("rule_verdicts", "rule_verdicts_secondary"):
        for row in data.get(key) or []:
            if isinstance(row, Mapping):
                rows.append(row)
    return rows


def write_validation(task_dir: Path | str, data: Mapping[str, Any]) -> Path:
    td = Path(task_dir)
    td.mkdir(parents=True, exist_ok=True)
    path = td / VALIDATION_FILE
    path.write_text(json.dumps(dict(data), indent=2, default=str) + "\n", encoding="utf-8")
    return path


def record_verdicts(task_dir: Path | str, verdicts: Iterable[Verdict]) -> Path:
    """Merge ``verdicts`` into ``validation.json`` under tier precedence."""
    td = Path(task_dir)
    data = load_validation(td)
    primaries, secondaries = merge_verdict_rows(stored_verdict_rows(data), verdicts)
    data["rule_verdicts"] = primaries
    if secondaries:
        data["rule_verdicts_secondary"] = secondaries
    else:
        data.pop("rule_verdicts_secondary", None)
    return write_validation(td, data)


class GateError(TaskSafetyError):
    """The gate refuses this task dir for launch."""

    def __init__(self, message: str, blockers: list[str] | None = None) -> None:
        super().__init__(message)
        self.blockers = blockers or []
