"""Per-repo control unit: the author's easiest predicted-L2 task.

The control must go 3/3 at L2; if it does not, the repo's ladder is untrusted
(see calibration flag ``control_miss``).
"""

from __future__ import annotations

import argparse
import json
import sys
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from openswe_traces.pipeline_ext.author_meta import PredictedFlip
from openswe_traces.pipeline_ext.calibration import AttemptRecord, attempt_from_dict


@dataclass(frozen=True)
class UnitMeta:
    repo: str
    name: str
    predicted_flip: int | None = None
    n_lines: int = 0
    n_files: int = 0
    family: str = ""
    author_backend: str = ""
    is_control: bool = False


@dataclass(frozen=True)
class ControlStatus:
    repo: str
    unit: str | None
    present: bool
    l2_passes: int
    l2_attempts: int
    ok: bool
    detail: str

    def as_dict(self) -> dict[str, Any]:
        return {
            "repo": self.repo,
            "unit": self.unit,
            "present": self.present,
            "l2_passes": self.l2_passes,
            "l2_attempts": self.l2_attempts,
            "ok": self.ok,
            "detail": self.detail,
        }


def pick_control(units: Sequence[UnitMeta], repo: str | None = None) -> UnitMeta | None:
    """Author's easiest predicted-L2 unit (fewest lines, then name).

    An explicit ``control: true`` on difficulty.md wins over the heuristic.
    """
    pool = [u for u in units if repo is None or u.repo == repo]
    marked = [u for u in pool if u.is_control]
    if marked:
        return min(marked, key=lambda u: (u.n_lines if u.n_lines else 10**9, u.name))
    l2 = [u for u in pool if u.predicted_flip == 2]
    if l2:
        return min(l2, key=lambda u: (u.n_lines if u.n_lines else 10**9, u.name))
    with_pred = [u for u in pool if u.predicted_flip is not None]
    if with_pred:
        return min(
            with_pred,
            key=lambda u: (int(u.predicted_flip or 9), u.n_lines or 10**9, u.name),
        )
    if not pool:
        return None
    return min(pool, key=lambda u: (u.n_lines or 10**9, u.name))


def is_control(unit: str, units: Sequence[UnitMeta], *, repo: str | None = None) -> bool:
    ctrl = pick_control(units, repo=repo)
    return ctrl is not None and ctrl.name == unit and (repo is None or ctrl.repo == repo)


def control_unit_name(units: Sequence[UnitMeta], repo: str) -> str | None:
    ctrl = pick_control(units, repo=repo)
    return None if ctrl is None else ctrl.name


def control_status(
    repo: str,
    units: Sequence[UnitMeta],
    attempts: Sequence[AttemptRecord],
    *,
    solver: str | None = None,
) -> ControlStatus:
    ctrl = pick_control(units, repo=repo)
    if ctrl is None:
        return ControlStatus(
            repo=repo,
            unit=None,
            present=False,
            l2_passes=0,
            l2_attempts=0,
            ok=False,
            detail="no units; cannot mark a control",
        )
    rows = [
        r
        for r in attempts
        if r.repo == repo
        and r.unit == ctrl.name
        and r.level == 2
        and not r.timeout
        and (solver is None or r.solver == solver)
    ]
    p = sum(1 for r in rows if r.passed)
    n = len(rows)
    ok = n >= 3 and p >= 3
    return ControlStatus(
        repo=repo,
        unit=ctrl.name,
        present=True,
        l2_passes=p,
        l2_attempts=n,
        ok=ok,
        detail=f"control {repo}/{ctrl.name} L2 {p}/{n} (need 3/3)",
    )


def units_from_predicted(
    repo: str,
    names_and_flips: Sequence[tuple[str, PredictedFlip | None, int]],
) -> list[UnitMeta]:
    """``(name, parsed flip, n_lines)`` helper for tests and CLIs."""
    out: list[UnitMeta] = []
    for name, pred, n_lines in names_and_flips:
        out.append(
            UnitMeta(
                repo=repo,
                name=name,
                predicted_flip=None if pred is None else pred.level,
                n_lines=n_lines,
            )
        )
    return out


def unit_from_dict(row: dict[str, Any]) -> UnitMeta:
    return UnitMeta(
        repo=str(row["repo"]),
        name=str(row.get("name") or row.get("unit") or ""),
        predicted_flip=None if row.get("predicted_flip") is None else int(row["predicted_flip"]),
        n_lines=int(row.get("n_lines") or 0),
        n_files=int(row.get("n_files") or 0),
        family=str(row.get("family") or ""),
        author_backend=str(row.get("author_backend") or ""),
        is_control=bool(row.get("is_control", False)),
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Per-repo control unit")
    parser.add_argument("--units", type=Path, required=True, help="JSON list of UnitMeta")
    parser.add_argument("--repo", required=True)
    parser.add_argument("--attempts", type=Path, help="JSON list of AttemptRecord")
    parser.add_argument("--unit", help="Check is_control for this name")
    args = parser.parse_args(argv)
    units = [unit_from_dict(r) for r in json.loads(args.units.read_text(encoding="utf-8"))]
    out: dict[str, Any] = {
        "control": None if pick_control(units, args.repo) is None else pick_control(units, args.repo).name,
        "is_control": is_control(args.unit, units, repo=args.repo) if args.unit else None,
    }
    if args.attempts:
        attempts = [attempt_from_dict(r) for r in json.loads(args.attempts.read_text(encoding="utf-8"))]
        out["status"] = control_status(args.repo, units, attempts).as_dict()
    json.dump(out, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
