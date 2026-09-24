#!/usr/bin/env python3
"""Shadow-implementation gate CLI. Thin wrapper over openswe_traces.gate.shadow.

Examples:
  uv run python scripts/shadow_gate.py UNIT_DIR [UNIT_DIR ...]
  uv run python scripts/shadow_gate.py --batch SWEEP_DIR [--only name ...]
  uv run python scripts/shadow_gate.py --batch SWEEP_DIR --keep-trees --max-requests 200

Each unit dir needs environment/src (excised tree), environment/Dockerfile,
instruction.md or contract.md, and tests/test.sh + tests/hidden. Results
append to outputs/shadow_gate/results.jsonl; units already there are skipped.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from openswe_traces.gate.shadow import (
    MAX_REQUESTS,
    SHADOW_MODELS,
    gate_batch,
    log,
)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("units", nargs="*", type=Path,
                    help="unit dirs to gate")
    ap.add_argument("--batch", type=Path, default=None,
                    help="dir of unit dirs (e.g. a sweep dir)")
    ap.add_argument("--only", nargs="*", default=None,
                    help="restrict --batch to these unit dir names")
    ap.add_argument("--models", nargs="*", default=list(SHADOW_MODELS),
                    help="cursor-agent models, tried in order (weak first)")
    ap.add_argument("--workers", type=int, default=1,
                    help="units gated concurrently (docker builds/tests)")
    ap.add_argument("--max-requests", type=int, default=MAX_REQUESTS)
    ap.add_argument("--keep-trees", action="store_true",
                    help="keep the shadowed trees under outputs/shadow_gate/trees/")
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

    log(f"shadow gate: {len(unit_dirs)} unit(s), models={args.models}")
    results = gate_batch(unit_dirs, models=args.models,
                         max_requests=args.max_requests,
                         keep_trees=args.keep_trees, workers=args.workers)
    n_pass = sum(1 for r in results if r.outcome == "pass")
    log(f"done: {n_pass}/{len(results)} shadow(s) passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
