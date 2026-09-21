#!/usr/bin/env python3
"""Repair-loop CLI. Thin wrapper over openswe_traces.gate.repair_loop.

The loop: shadow a contract -> attribute each failing assertion -> repair
only the commitments the contract does not state -> repeat to a fixed
point, then regenerate the cheat patch from the repaired contract and
re-run preflight (bare fails, gold passes, fresh cheat fails).

Examples:
  uv run python scripts/repair_loop.py UNIT_DIR [UNIT_DIR ...] --stage DIR
  uv run python scripts/repair_loop.py --batch SWEEP_DIR --stage DIR
  uv run python scripts/repair_loop.py --batch SWEEP_DIR --only exprhash-L2 --no-preflight

Each unit dir needs environment/src (excised tree), environment/Dockerfile,
instruction.md, and tests/test.sh + tests/hidden. Work copies live under
outputs/repair_loop/work/; per-unit round records land in
outputs/repair_loop/rounds/<unit>.json and one-line summaries append to
outputs/repair_loop/results.jsonl. Terminal-outcome units are skipped on
re-run; units that errored are retried. Source unit dirs are never
written — repaired units are staged under --stage only.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from openswe_traces.gate.repair_loop import (
    JUDGE_MODELS,
    MAX_REQUESTS,
    MAX_ROUNDS,
    log,
    loop_batch,
)
from openswe_traces.gate.shadow import SHADOW_MODELS


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("units", nargs="*", type=Path, help="unit dirs to loop")
    ap.add_argument("--batch", type=Path, default=None,
                    help="dir of unit dirs (e.g. a sweep dir)")
    ap.add_argument("--only", nargs="*", default=None,
                    help="restrict --batch to these unit dir names")
    ap.add_argument("--stage", type=Path, default=None,
                    help="stage repaired units here (one dir per unit, never overwriting)")
    ap.add_argument("--max-rounds", type=int, default=MAX_ROUNDS)
    ap.add_argument("--max-requests", type=int, default=MAX_REQUESTS)
    ap.add_argument("--shadow-models", nargs="*", default=list(SHADOW_MODELS))
    ap.add_argument("--judge-models", nargs="*", default=list(JUDGE_MODELS))
    ap.add_argument("--no-preflight", action="store_true",
                    help="skip cheat-regen + preflight at the end of each unit")
    args = ap.parse_args(argv)

    unit_dirs = list(args.units)
    if args.batch:
        for p in sorted(args.batch.iterdir()):
            if not p.is_dir() or p.name.startswith("_"):
                continue
            if args.only and p.name not in args.only:
                continue
            unit_dirs.append(p)
    unit_dirs = [u for u in unit_dirs if (u / "environment" / "src").is_dir()]
    if not unit_dirs:
        ap.error("no unit dirs given (each needs environment/src)")
    if args.stage:
        args.stage.mkdir(parents=True, exist_ok=True)

    log(f"repair loop: {len(unit_dirs)} unit(s), rounds<={args.max_rounds}, "
        f"stage={args.stage}")
    results = loop_batch(unit_dirs, stage_root=args.stage,
                         max_rounds=args.max_rounds,
                         max_requests=args.max_requests,
                         shadow_models=args.shadow_models,
                         judge_models=args.judge_models,
                         run_preflight=not args.no_preflight)
    conv = sum(1 for r in results if r.outcome.startswith("converged"))
    log(f"done: {conv}/{len(results)} converged")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
