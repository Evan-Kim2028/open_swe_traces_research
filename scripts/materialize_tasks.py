"""Rebuild task environment/src trees on a fresh machine (see pipeline/materialize.py)."""

from __future__ import annotations

import argparse
import logging
from pathlib import Path

from openswe_traces.pipeline.config import load_config
from openswe_traces.pipeline.materialize import materialize_all


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="experiments/pipeline/config.yaml")
    ap.add_argument("--repos", default="experiments/pipeline/repos.yaml")
    ap.add_argument("--only", nargs="*", default=None, help="repo names to materialise")
    ap.add_argument("--force", action="store_true", help="rebuild even if environment/src exists")
    ap.add_argument(
        "--root",
        action="append",
        type=Path,
        default=None,
        help="task roots (default tasks/ and tasks_composerver/)",
    )
    args = ap.parse_args()
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    cfg = load_config(args.config, args.repos)
    n = materialize_all(
        cfg,
        roots=args.root,
        only_missing=not args.force,
        repos=set(args.only) if args.only else None,
    )
    print(f"materialised {n} task dir(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
