"""Pass-rate curve, flip nearest 50%, inter-attempt agreement, early-warning flags.

The calibration curve (pass rate vs ladder level, per repo × solver) is the
primary measurement the pipeline exists to produce.
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

PASS_NEED = 2
PASS_OF = 3

FLAG_ACTIONS: dict[str, str] = {
    "gate_rejection": (
        "Gate rejection > 40% for this repo (by rule). Pause the repo and rewrite "
        "or drop the failing gate; do not keep generating units until the kill rate is down."
    ),
    "control_miss": (
        "Control unit is not 3/3 at L2. Stop the repo: the control is the author's "
        "easiest predicted-L2 unit and must be trivial for the solver. Author or verifier is broken."
    ),
    "too_easy": (
        "First 15 valid tasks are all 3/3 at L2. Tasks are too easy; raise the author "
        "quality bar (harder units, more state/sequence/dynamic) and discard the easy batch from the curve."
    ),
    "l6_fail": (
        "A solver failed L6 (all tests in tree). Audit before counting (C1): verifier too "
        "narrow, gold/instruction broken, or infra (class d). Legitimate L6 fails void the unit."
    ),
    "noisy_splits": (
        "More than 1/3 of scored levels show 2-1 splits. Too noisy for a flip point; add "
        "attempts or drop the unit. C6 requires a stable 2/3."
    ),
    "contamination": (
        "More than 3 contamination or test-edit incidents on this host. Inspect B2/B1 leaks, "
        "tighten the allowlist, rerun the affected trials."
    ),
    "author_yield": (
        "Author yield < 3 valid units/session. Switch author backend or raise session minutes; "
        "the brief or model is too weak for this repo."
    ),
    "attempt_time": (
        "Attempt wall time grew > 2x across attempts. Treat as infra (class d), not a fail; "
        "check hung tests, load, and Harbor timeouts."
    ),
}


@dataclass(frozen=True)
class AttemptRecord:
    """One scored attempt. Timeouts (class d) set ``timeout=True`` and are dropped from rates."""

    repo: str
    unit: str
    solver: str
    level: int
    attempt: int
    passed: bool
    timeout: bool = False
    wall_seconds: float | None = None
    audit_class: str = "clean"
    rejected_rule: str | None = None
    host: str = ""
    author_backend: str = ""
    author_session: str = ""
    is_control: bool = False
    valid_unit: bool = True


@dataclass(frozen=True)
class PassRate:
    repo: str
    solver: str
    level: int
    passes: int
    attempts: int

    @property
    def rate(self) -> float:
        return self.passes / self.attempts if self.attempts else 0.0


@dataclass(frozen=True)
class EarlyWarning:
    flag: str
    fired: bool
    detail: str
    action: str
    repo: str = ""
    solver: str = ""


@dataclass
class CalibrationReport:
    curve: list[PassRate] = field(default_factory=list)
    nearest_50: dict[tuple[str, str], int | None] = field(default_factory=dict)
    split_fraction: float = 0.0
    flags: list[EarlyWarning] = field(default_factory=list)

    def as_dict(self) -> dict[str, Any]:
        return {
            "curve": [
                {
                    "repo": r.repo,
                    "solver": r.solver,
                    "level": r.level,
                    "passes": r.passes,
                    "attempts": r.attempts,
                    "rate": r.rate,
                }
                for r in self.curve
            ],
            "nearest_50": {f"{k[0]}|{k[1]}": v for k, v in self.nearest_50.items()},
            "split_fraction": self.split_fraction,
            "flags": [
                {
                    "flag": f.flag,
                    "fired": f.fired,
                    "detail": f.detail,
                    "action": f.action,
                    "repo": f.repo,
                    "solver": f.solver,
                }
                for f in self.flags
            ],
        }


def scored(records: Sequence[AttemptRecord]) -> list[AttemptRecord]:
    return [r for r in records if not r.timeout and r.audit_class in {"", "clean", "a"}]


def pass_rate_curve(records: Sequence[AttemptRecord]) -> list[PassRate]:
    buckets: dict[tuple[str, str, int], list[AttemptRecord]] = defaultdict(list)
    for row in scored(records):
        buckets[(row.repo, row.solver, row.level)].append(row)
    out: list[PassRate] = []
    for (repo, solver, level), rows in sorted(buckets.items()):
        out.append(
            PassRate(
                repo=repo,
                solver=solver,
                level=level,
                passes=sum(1 for r in rows if r.passed),
                attempts=len(rows),
            )
        )
    return out


def nearest_50(curve: Sequence[PassRate], *, repo: str, solver: str) -> int | None:
    """Level whose pass rate is closest to 50%. None if that solver has no curve."""
    rows = [r for r in curve if r.repo == repo and r.solver == solver and r.attempts]
    if not rows:
        return None
    best = min(rows, key=lambda r: (abs(r.rate - 0.5), r.level))
    return best.level


def _level_key(row: AttemptRecord) -> tuple[str, str, str, int]:
    return (row.repo, row.unit, row.solver, row.level)


def inter_attempt_agreement(records: Sequence[AttemptRecord]) -> float:
    """Fraction of (repo, unit, solver, level) groups with a 2-1 split (n=3)."""
    groups: dict[tuple[str, str, str, int], list[AttemptRecord]] = defaultdict(list)
    for row in scored(records):
        groups[_level_key(row)].append(row)
    if not groups:
        return 0.0
    splits = 0
    n = 0
    for rows in groups.values():
        if len(rows) < 3:
            continue
        n += 1
        # Use the first 3 scored attempts.
        trio = rows[:3]
        p = sum(1 for r in trio if r.passed)
        if p in {1, 2}:
            splits += 1
    return splits / n if n else 0.0


def _gate_rejection_flags(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    # Units with rejected_rule, unique per (repo, unit, rule)
    by_repo_rule: dict[tuple[str, str], set[str]] = defaultdict(set)
    units_by_repo: dict[str, set[str]] = defaultdict(set)
    for row in records:
        units_by_repo[row.repo].add(row.unit)
        if row.rejected_rule:
            by_repo_rule[(row.repo, row.rejected_rule)].add(row.unit)
    flags: list[EarlyWarning] = []
    for repo, units in units_by_repo.items():
        n = len(units)
        if n == 0:
            continue
        for (r, rule), killed in by_repo_rule.items():
            if r != repo:
                continue
            rate = len(killed) / n
            fired = rate > 0.40
            flags.append(
                EarlyWarning(
                    flag="gate_rejection",
                    fired=fired,
                    detail=f"{repo} rule {rule}: {len(killed)}/{n} units rejected ({rate:.0%})",
                    action=FLAG_ACTIONS["gate_rejection"],
                    repo=repo,
                )
            )
    return flags


def _control_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    flags: list[EarlyWarning] = []
    controls = {(r.repo, r.unit, r.solver) for r in records if r.is_control}
    for repo, unit, solver in sorted(controls):
        rows = [
            r
            for r in scored(records)
            if r.repo == repo and r.unit == unit and r.solver == solver and r.level == 2
        ]
        p = sum(1 for r in rows if r.passed)
        n = len(rows)
        fired = n >= 3 and p < 3
        flags.append(
            EarlyWarning(
                flag="control_miss",
                fired=fired,
                detail=f"{repo}/{unit} {solver} L2 {p}/{n} (need 3/3)",
                action=FLAG_ACTIONS["control_miss"],
                repo=repo,
                solver=solver,
            )
        )
    return flags


def _too_easy_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    flags: list[EarlyWarning] = []
    # First 15 valid units per (repo, solver), chronological by min attempt index.
    valid_units: dict[tuple[str, str], dict[str, list[AttemptRecord]]] = defaultdict(lambda: defaultdict(list))
    for row in scored(records):
        if not row.valid_unit:
            continue
        valid_units[(row.repo, row.solver)][row.unit].append(row)
    for (repo, solver), units in valid_units.items():
        ordered = list(units.items())[:15]
        if len(ordered) < 15:
            continue
        all_easy = True
        for _unit, rows in ordered:
            l2 = [r for r in rows if r.level == 2]
            if not (len(l2) >= 3 and sum(1 for r in l2 if r.passed) >= 3):
                all_easy = False
                break
        flags.append(
            EarlyWarning(
                flag="too_easy",
                fired=all_easy,
                detail=f"{repo}/{solver}: first 15 valid units all 3/3 at L2={all_easy}",
                action=FLAG_ACTIONS["too_easy"],
                repo=repo,
                solver=solver,
            )
        )
    return flags


def _l6_fail_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    flags: list[EarlyWarning] = []
    hits: dict[tuple[str, str], list[AttemptRecord]] = defaultdict(list)
    for row in scored(records):
        if row.level == 6 and not row.passed:
            hits[(row.repo, row.solver)].append(row)
    for (repo, solver), rows in sorted(hits.items()):
        flags.append(
            EarlyWarning(
                flag="l6_fail",
                fired=True,
                detail=f"{repo}/{solver}: {len(rows)} L6 fail(s) on {sorted({r.unit for r in rows})[:8]}",
                action=FLAG_ACTIONS["l6_fail"],
                repo=repo,
                solver=solver,
            )
        )
    return flags


def _split_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    frac = inter_attempt_agreement(records)
    return [
        EarlyWarning(
            flag="noisy_splits",
            fired=frac > 1 / 3,
            detail=f"2-1 split fraction={frac:.2f} (threshold 1/3)",
            action=FLAG_ACTIONS["noisy_splits"],
        )
    ]


def _contamination_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    by_host: dict[str, int] = defaultdict(int)
    for row in records:
        if row.audit_class in {"contaminated", "checksum", "test-edit"}:
            host = row.host or "unknown"
            by_host[host] += 1
    flags: list[EarlyWarning] = []
    for host, n in sorted(by_host.items()):
        flags.append(
            EarlyWarning(
                flag="contamination",
                fired=n > 3,
                detail=f"host {host}: {n} contamination/test-edit incidents (threshold 3)",
                action=FLAG_ACTIONS["contamination"],
            )
        )
    return flags


def _author_yield_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    # Unique valid units per (author_backend, author_session)
    sessions: dict[tuple[str, str], set[str]] = defaultdict(set)
    for row in records:
        if not row.author_session:
            continue
        key = (row.author_backend or "unknown", row.author_session)
        sessions.setdefault(key, set())
        if row.valid_unit and not row.rejected_rule:
            sessions[key].add(f"{row.repo}/{row.unit}")
    flags: list[EarlyWarning] = []
    for (backend, session), units in sorted(sessions.items()):
        n = len(units)
        flags.append(
            EarlyWarning(
                flag="author_yield",
                fired=n < 3,
                detail=f"{backend} session {session}: {n} valid units (need >= 3)",
                action=FLAG_ACTIONS["author_yield"],
                solver=backend,
            )
        )
    return flags


def _attempt_time_flag(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    groups: dict[tuple[str, str, str, int], list[AttemptRecord]] = defaultdict(list)
    for row in records:
        if row.wall_seconds is None:
            continue
        groups[_level_key(row)].append(row)
    flags: list[EarlyWarning] = []
    for key, rows in groups.items():
        rows = sorted(rows, key=lambda r: r.attempt)
        times = [r.wall_seconds for r in rows if r.wall_seconds and r.wall_seconds > 0]
        if len(times) < 2:
            continue
        first, last = times[0], times[-1]
        if last > 2 * first:
            repo, unit, solver, level = key
            flags.append(
                EarlyWarning(
                    flag="attempt_time",
                    fired=True,
                    detail=(
                        f"{repo}/{unit} {solver} L{level}: wall {first:.0f}s → {last:.0f}s "
                        f"(>{2}x)"
                    ),
                    action=FLAG_ACTIONS["attempt_time"],
                    repo=repo,
                    solver=solver,
                )
            )
    return flags


def early_warning_flags(records: Sequence[AttemptRecord]) -> list[EarlyWarning]:
    out: list[EarlyWarning] = []
    out.extend(_gate_rejection_flags(records))
    out.extend(_control_flag(records))
    out.extend(_too_easy_flag(records))
    out.extend(_l6_fail_flag(records))
    out.extend(_split_flag(records))
    out.extend(_contamination_flag(records))
    out.extend(_author_yield_flag(records))
    out.extend(_attempt_time_flag(records))
    return out


def calibrate(records: Sequence[AttemptRecord]) -> CalibrationReport:
    curve = pass_rate_curve(records)
    nearest: dict[tuple[str, str], int | None] = {}
    pairs = {(r.repo, r.solver) for r in curve}
    for repo, solver in sorted(pairs):
        nearest[(repo, solver)] = nearest_50(curve, repo=repo, solver=solver)
    return CalibrationReport(
        curve=curve,
        nearest_50=nearest,
        split_fraction=inter_attempt_agreement(records),
        flags=early_warning_flags(records),
    )


def attempt_from_dict(row: dict[str, Any]) -> AttemptRecord:
    return AttemptRecord(
        repo=str(row["repo"]),
        unit=str(row["unit"]),
        solver=str(row["solver"]),
        level=int(row["level"]),
        attempt=int(row.get("attempt") or 0),
        passed=bool(row.get("passed")),
        timeout=bool(row.get("timeout", False)),
        wall_seconds=None if row.get("wall_seconds") is None else float(row["wall_seconds"]),
        audit_class=str(row.get("audit_class") or "clean"),
        rejected_rule=row.get("rejected_rule"),
        host=str(row.get("host") or ""),
        author_backend=str(row.get("author_backend") or ""),
        author_session=str(row.get("author_session") or ""),
        is_control=bool(row.get("is_control", False)),
        valid_unit=bool(row.get("valid_unit", True)),
    )


def load_attempt_records(path: Path | str) -> list[AttemptRecord]:
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    if isinstance(data, dict) and "records" in data:
        data = data["records"]
    if not isinstance(data, list):
        raise TypeError("attempt records JSON must be a list or {records: [...]}")
    return [attempt_from_dict(row) for row in data if isinstance(row, dict)]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Pass-rate curve and early-warning flags")
    parser.add_argument("--records", required=True, help="JSON list of AttemptRecord dicts")
    args = parser.parse_args(argv)
    report = calibrate(load_attempt_records(args.records))
    json.dump(report.as_dict(), sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
