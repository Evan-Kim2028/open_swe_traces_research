#!/usr/bin/env python3
"""Generate the dose-response Arm-1 grid and package Harbor task dirs.

Thin CLI over ``openswe_traces.synth.dose_response``: writes fabricated units
to ``--fab-root``, ``grid.csv``, and ``<unit>-L0``/``<unit>-L2`` Harbor task
dirs under ``--tasks-root`` (OpenRouter-only agent allowlist).

Example:

    uv run python scripts/dose_response_fab.py
"""

from __future__ import annotations

import sys
from pathlib import Path

from openswe_traces.synth.dose_response import generate_grid, package_grid


def main() -> int:
    import argparse

    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--fab-root", type=Path, default=Path("experiments/dose_response/fab"))
    p.add_argument("--tasks-root", type=Path, default=Path("experiments/dose_response/tasks"))
    p.add_argument("--grid-csv", type=Path, default=Path("experiments/dose_response/grid.csv"))
    p.add_argument("--log", type=Path, default=Path("outputs/closure_I.log"))
    p.add_argument("--no-proof", action="store_true")
    p.add_argument("--no-package", action="store_true")
    args = p.parse_args()

    rows = generate_grid(args.fab_root, args.grid_csv, log_path=args.log, proof=not args.no_proof)
    reached = sum(1 for r in rows if r["proof"] in {"ok", "skipped"})
    failed = sum(1 for r in rows if r["proof"] == "failed")
    unreach = sum(1 for r in rows if r["proof"] == "unreachable")
    print(f"grid: {reached} reached, {failed} proof-failed, {unreach} unreachable -> {args.grid_csv}")
    if args.no_package:
        return 0 if failed == 0 else 1
    dirs = package_grid(args.fab_root, args.tasks_root)
    print(f"packaged {len(dirs)} task dirs -> {args.tasks_root}")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
