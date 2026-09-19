"""Thin CLI: openswe-pipeline run | dry-run | status."""

from __future__ import annotations

import argparse
import sys

from openswe_traces.pipeline.config import load_config
from openswe_traces.pipeline.run import dry_run, run_pipeline, status_text


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="openswe-pipeline")
    parser.add_argument("--config", default=None, help="experiments/pipeline/config.yaml")
    parser.add_argument("--repos", default=None, help="experiments/pipeline/repos.yaml")
    sub = parser.add_subparsers(dest="cmd", required=True)

    run = sub.add_parser("run", help="full overnight set (resumable)")
    run.add_argument("--host", default=None, help="laptop (default) or vps")

    dry = sub.add_parser("dry-run", help="one repo, N units, print the results table")
    dry.add_argument("--repo", required=True)
    dry.add_argument("--units", type=int, default=5)

    sub.add_parser("status", help="print sqlite resume state")
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    cfg = load_config(args.config, args.repos)
    if args.cmd == "status":
        sys.stdout.write(status_text(cfg))
        return 0
    if args.cmd == "dry-run":
        table = dry_run(args.repo, args.units, cfg=cfg)
        sys.stdout.write(table.rstrip() + "\n")
        return 0
    if args.cmd == "run":
        dest = run_pipeline(cfg, host=getattr(args, "host", None))
        sys.stdout.write(f"results: {dest}\n")
        return 0
    parser.print_help()
    return 2


def _script_main() -> None:
    raise SystemExit(main())


if __name__ == "__main__":
    _script_main()
