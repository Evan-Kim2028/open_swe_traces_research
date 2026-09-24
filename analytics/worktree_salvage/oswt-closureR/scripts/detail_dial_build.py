"""Build the detail-count x verification-affordance grid (closure-R).

Generates the 40 fabricated units (d in {1,2,4,6,8} x v in {low,high} x 4
seeds, store domain, fixed |S|), proves each locally, runs the per-detail trap
audit, writes grid.csv, then packages Harbor L0 task dirs with the CURSOR
agent allowlist and the per-detail verifier.

Example:

    uv run python scripts/detail_dial_build.py \
        --fab experiments/detail_dial/fab \
        --tasks experiments/detail_dial/tasks \
        --grid experiments/detail_dial/grid.csv \
        --log outputs/closure_R.log
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from openswe_traces.synth.detail_dial import build_grid, package_grid


def main_cli(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--fab", type=Path, default=Path("experiments/detail_dial/fab"))
    p.add_argument("--tasks", type=Path, default=Path("experiments/detail_dial/tasks"))
    p.add_argument("--grid", type=Path, default=Path("experiments/detail_dial/grid.csv"))
    p.add_argument("--proofs", type=Path, default=None, help="scratch root for proofs/audits")
    p.add_argument("--no-proof", action="store_true")
    p.add_argument("--no-audit", action="store_true")
    p.add_argument("--no-package", action="store_true")
    p.add_argument("--log", type=Path, default=Path("outputs/closure_R.log"))
    args = p.parse_args(argv)

    rows = build_grid(
        args.fab,
        args.grid,
        proof=not args.no_proof,
        audit=not args.no_audit,
        scratch_root=args.proofs,
        log_path=args.log,
    )
    print(f"grid: {len(rows)} cells written to {args.grid}")
    bad = [r for r in rows if not r.proof_ok]
    if bad:
        print(f"PROOF FAILURES: {[r.unit for r in bad]}")
        return 1
    if args.no_audit:
        print("audit skipped (--no-audit)")
    else:
        bad_audit = [r.unit for r in rows if r.audit_ok is not True]
        print(f"audit: {len(rows) - len(bad_audit)}/{len(rows)} units OK" + (f"; FAILURES: {bad_audit}" if bad_audit else ""))
        if bad_audit:
            return 1
    if not args.no_package:
        task_dirs = package_grid(args.fab, args.tasks)
        print(f"packaged {len(task_dirs)} Harbor task dirs under {args.tasks}")
    return 0


if __name__ == "__main__":
    sys.exit(main_cli())
