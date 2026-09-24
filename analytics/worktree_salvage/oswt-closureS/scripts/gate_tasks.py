#!/usr/bin/env python3
"""Sweep the gate over task dirs. Thin wrapper for gate.sweep.

Examples:
  uv run python scripts/gate_tasks.py experiments/pipeline/tasks_composerver/helm \
      experiments/pipeline/tasks_composerver/kops experiments/pipeline/tasks_composerver/gin \
      --levels 0,2 --out outputs/gate_matrix.json
  uv run python scripts/gate_tasks.py /path/to/dose_response/tasks --no-executed
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from openswe_traces.gate.sweep import iter_task_dirs, kill_counts, new_failures, sweep


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("roots", nargs="+", type=Path, help="task dirs or roots to scan")
    ap.add_argument("--levels", default="", help="comma list, e.g. 0,2 — keep only <unit>-L<n>")
    ap.add_argument("--no-executed", action="store_true", help="static+documentary only (no docker)")
    ap.add_argument("--force", action="store_true", help="ignore cached executed runs")
    ap.add_argument("--out", type=Path, default=None, help="write matrix JSON here")
    args = ap.parse_args(argv)

    dirs = iter_task_dirs(*args.roots)
    if args.levels:
        keep = {int(x) for x in args.levels.split(",")}
        dirs = [d for d in dirs if _level(d) in keep]
    print(f"{len(dirs)} task dirs", file=sys.stderr)

    def log(m: str) -> None:
        print(m, file=sys.stderr, flush=True)

    result = sweep(
        dirs, run_executed=not args.no_executed, force=args.force, progress=log
    )
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(result, indent=2, default=str) + "\n")
        print(f"wrote {args.out}", file=sys.stderr)
    fails = new_failures(result["matrix"])
    for rid, rows in fails.items():
        if rows:
            print(f"{rid}: {len(rows)} failures")
            for r in rows[:10]:
                print(f"  {r}")
    print(json.dumps(kill_counts(result["matrix"]), indent=1), file=sys.stderr)
    return 0


def _level(d: Path) -> int | None:
    import re

    m = re.search(r"-L(\d+)$", d.name)
    return int(m.group(1)) if m else None


if __name__ == "__main__":
    raise SystemExit(main())
