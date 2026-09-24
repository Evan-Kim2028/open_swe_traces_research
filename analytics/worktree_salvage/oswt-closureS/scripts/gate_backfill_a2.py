#!/usr/bin/env python
"""Backfill executed A2 verdicts from recorded passed trials.

Usage:
    uv run python scripts/gate_backfill_a2.py <tasks-root>... [--db state.db] [--jobs jobs_root] [--dry]

Every trial that PASSED and has a saved agent patch is compared to the task's
gold.patch; a structurally different patch records an executed A2 verdict on
the task dir.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from openswe_traces.gate.alt_fix import backfill_jobs, backfill_state_db


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("tasks_roots", nargs="+", help="task roots (flat or <repo>/<unit>-L<n>)")
    ap.add_argument("--db", nargs="*", default=[], help="pipeline state.db paths")
    ap.add_argument("--jobs", nargs="*", default=[], help="job dirs to scan for trial patches")
    ap.add_argument("--dry", action="store_true", help="report without writing verdicts")
    args = ap.parse_args()

    roots = [Path(r) for r in args.tasks_roots]
    report: dict[str, dict] = {}
    for db in args.db:
        report[f"db:{db}"] = backfill_state_db(Path(db), roots, dry=args.dry)
    for jobs in args.jobs:
        report[f"jobs:{jobs}"] = backfill_jobs(Path(jobs), roots, dry=args.dry)

    for src, res in report.items():
        print(f"{src}: recorded={len(res['recorded'])} equivalent={len(res['equivalent'])} skipped={len(res['skipped'])}")
    print(json.dumps(report, indent=2, default=str))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
