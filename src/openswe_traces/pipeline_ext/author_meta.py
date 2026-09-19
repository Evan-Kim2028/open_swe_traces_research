"""Parse ``difficulty.md`` predicted flip and score authors against measured flips.

Required line in every authored unit's ``difficulty.md``::

    predicted_flip: L<k>

``k`` is an integer 0..6. Surrounding prose is allowed. Example::

    # Difficulty
    predicted_flip: L2
    Justification: single-file missing-return; in-tree tests already describe it.

The author brief template (``/home/evan/devin-tasks/openswe_bigL0_author.md``)
lists this line as required.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import defaultdict
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

PREDICTED_FLIP_LINE_RE = re.compile(
    r"^[ \t]*predicted_flip:[ \t]*L([0-6])[ \t]*$",
    re.IGNORECASE | re.MULTILINE,
)
CONTROL_LINE_RE = re.compile(
    r"^[ \t]*control:[ \t]*(true|yes|1)\b",
    re.IGNORECASE | re.MULTILINE,
)

DIFFICULTY_MD_TEMPLATE = """# Difficulty

predicted_flip: L{level}

{justification}
"""

AUTHOR_BRIEF_DIFFICULTY = (
    "difficulty.md — REQUIRED line exactly `predicted_flip: L<k>` with k in 0..6 "
    "(the ladder level you expect the solver's flip point to fall at), plus one "
    "short justification paragraph. L2 = full contract; L0 = bug report only."
)


@dataclass(frozen=True)
class PredictedFlip:
    level: int
    raw: str
    path: str = ""


@dataclass(frozen=True)
class AuthorUnit:
    author_backend: str
    unit: str
    repo: str = ""
    predicted: int | None = None
    measured: int | None = None
    confirmed: bool = False


@dataclass(frozen=True)
class AuthorCalibration:
    backend: str
    n: int
    n_with_both: int
    mean_abs_error: float | None
    exact: int
    off_by_one: int
    rows: tuple[AuthorUnit, ...]

    def as_dict(self) -> dict[str, Any]:
        return {
            "backend": self.backend,
            "n": self.n,
            "n_with_both": self.n_with_both,
            "mean_abs_error": self.mean_abs_error,
            "exact": self.exact,
            "off_by_one": self.off_by_one,
            "rows": [
                {
                    "unit": r.unit,
                    "repo": r.repo,
                    "predicted": r.predicted,
                    "measured": r.measured,
                    "confirmed": r.confirmed,
                }
                for r in self.rows
            ],
        }


def predicted_flip_line(level: int) -> str:
    if level not in range(7):
        raise ValueError(f"predicted_flip level must be 0..6, got {level}")
    return f"predicted_flip: L{level}"


def parse_predicted_flip(text: str, *, path: str = "") -> PredictedFlip | None:
    """Return the last ``predicted_flip: L<k>`` line, or None if missing."""
    matches = list(PREDICTED_FLIP_LINE_RE.finditer(text or ""))
    if not matches:
        return None
    m = matches[-1]
    return PredictedFlip(level=int(m.group(1)), raw=m.group(0).strip(), path=path)


def parse_control_flag(text: str) -> bool:
    """True when difficulty.md has ``control: true`` (or yes/1)."""
    return CONTROL_LINE_RE.search(text or "") is not None


def parse_difficulty_md(path: Path | str) -> PredictedFlip | None:
    p = Path(path)
    try:
        text = p.read_text(encoding="utf-8")
    except OSError:
        return None
    return parse_predicted_flip(text, path=str(p))


def parse_author_dir(author_dir: Path | str) -> PredictedFlip | None:
    root = Path(author_dir)
    for cand in (root / "difficulty.md", root / "DIFFICULTY.md"):
        hit = parse_difficulty_md(cand)
        if hit is not None:
            return hit
    return None


def author_calibration(units: Sequence[AuthorUnit]) -> list[AuthorCalibration]:
    """Predicted vs measured flip, grouped by author backend."""
    by_backend: dict[str, list[AuthorUnit]] = defaultdict(list)
    for u in units:
        by_backend[u.author_backend or "unknown"].append(u)
    out: list[AuthorCalibration] = []
    for backend, rows in sorted(by_backend.items()):
        paired = [r for r in rows if r.predicted is not None and r.measured is not None]
        errs = [abs(int(r.predicted) - int(r.measured)) for r in paired]
        mae = (sum(errs) / len(errs)) if errs else None
        exact = sum(1 for e in errs if e == 0)
        off1 = sum(1 for e in errs if e == 1)
        out.append(
            AuthorCalibration(
                backend=backend,
                n=len(rows),
                n_with_both=len(paired),
                mean_abs_error=mae,
                exact=exact,
                off_by_one=off1,
                rows=tuple(rows),
            )
        )
    return out


def author_unit_from_dict(row: dict[str, Any]) -> AuthorUnit:
    return AuthorUnit(
        author_backend=str(row.get("author_backend") or "unknown"),
        unit=str(row["unit"]),
        repo=str(row.get("repo") or ""),
        predicted=None if row.get("predicted") is None else int(row["predicted"]),
        measured=None if row.get("measured") is None else int(row["measured"]),
        confirmed=bool(row.get("confirmed", False)),
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Parse predicted_flip / author calibration")
    parser.add_argument("--difficulty", type=Path, help="difficulty.md path")
    parser.add_argument("--author-dir", type=Path, help="unit _author dir")
    parser.add_argument("--units", type=Path, help="JSON list of AuthorUnit dicts")
    args = parser.parse_args(argv)
    if args.difficulty:
        pred = parse_difficulty_md(args.difficulty)
    elif args.author_dir:
        pred = parse_author_dir(args.author_dir)
    else:
        pred = None
    if args.difficulty or args.author_dir:
        json.dump(
            None if pred is None else {"level": pred.level, "raw": pred.raw, "path": pred.path},
            sys.stdout,
            indent=2,
        )
        sys.stdout.write("\n")
        return 0 if pred is not None else 1
    if args.units:
        data = json.loads(args.units.read_text(encoding="utf-8"))
        units = [author_unit_from_dict(r) for r in data]
        json.dump([c.as_dict() for c in author_calibration(units)], sys.stdout, indent=2)
        sys.stdout.write("\n")
        return 0
    parser.error("need --difficulty, --author-dir, or --units")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
