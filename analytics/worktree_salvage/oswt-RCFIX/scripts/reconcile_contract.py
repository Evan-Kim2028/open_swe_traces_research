#!/usr/bin/env python3
"""Derive contract.md from DETAILS.md + hidden tests + gold.patch.

Examples:
  uv run python scripts/reconcile_contract.py unit \\
      --details DETAILS.md --hidden tests/hidden --gold gold.patch \\
      --out contract.md --unit refescape

  uv run python scripts/reconcile_contract.py batch \\
      --author-root /path/authored_batch2/go-github \\
      --hidden-root /path/tasks_batch2/go-github \\
      --out-dir outputs/reconcile/go-github \\
      --stage /path/dose_response/sweep_gogithub_rc
"""

from __future__ import annotations

import argparse
import json
import shutil
import sys
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.reconcile import (
    MAX_REQUESTS,
    log,
    reconcile_paths,
    splice_contract,
)


def _cmd_unit(args: argparse.Namespace) -> int:
    result = reconcile_paths(
        unit=args.unit,
        details_path=args.details,
        hidden_dir=args.hidden,
        gold_path=args.gold,
        out_path=args.out,
        llm=not args.no_llm,
        budget=args.max_requests,
    )
    log(
        f"{result.unit}: rows={len(result.rows)} encoding_shape={result.encoding_shape} "
        f"unrec={result.unreconcilable} model={result.model} "
        f"requests={result.requests} cache={result.cache_hit}"
    )
    if result.unreconcilable:
        log(f"  unreconcilable: {result.unreconcilable_reason}")
    if result.b7_leaks:
        log(f"  b7 leaks scrubbed: {result.b7_leaks[:8]}")
    if args.out:
        log(f"  wrote {args.out}")
    else:
        sys.stdout.write(result.contract)
    return 0


def _discover_units(author_root: Path, hidden_root: Path) -> list[str]:
    names: list[str] = []
    for child in sorted(author_root.iterdir()):
        if not child.is_dir():
            continue
        author = child / "_author"
        details = author / "DETAILS.md"
        gold = author / "gold.patch"
        hidden = hidden_root / f"{child.name}-L2" / "tests" / "hidden"
        if details.is_file() and gold.is_file() and hidden.is_dir():
            names.append(child.name)
    return names


def _cmd_batch(args: argparse.Namespace) -> int:
    units = args.units or _discover_units(args.author_root, args.hidden_root)
    if not units:
        log("no units found")
        return 1
    out_dir: Path = args.out_dir
    out_dir.mkdir(parents=True, exist_ok=True)
    summary = []
    spent = 0
    for name in units:
        author = args.author_root / name / "_author"
        hidden = args.hidden_root / f"{name}-L2" / "tests" / "hidden"
        dest = out_dir / name / "contract.md"
        budget = max(0, args.max_requests - spent)
        log(f"reconcile {name} (budget {budget})")
        result = reconcile_paths(
            unit=name,
            details_path=author / "DETAILS.md",
            hidden_dir=hidden,
            gold_path=author / "gold.patch",
            out_path=dest,
            llm=not args.no_llm,
            budget=budget,
        )
        spent += result.requests
        row = {
            "unit": name,
            "n_rows": len(result.rows),
            "encoding_shape": result.encoding_shape,
            "unreconcilable": result.unreconcilable,
            "unreconcilable_reason": result.unreconcilable_reason,
            "model": result.model,
            "requests": result.requests,
            "cache_hit": result.cache_hit,
            "b7_leaks": result.b7_leaks,
            "applied_fixes": result.applied_fixes,
            "contract": str(dest),
        }
        summary.append(row)
        log(
            f"  {name}: rows={len(result.rows)} shape={result.encoding_shape} "
            f"unrec={result.unreconcilable} model={result.model} req={result.requests}"
        )
        if args.stage:
            _stage_unit(name, result.contract, args)
    (out_dir / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    log(f"batch done: {len(summary)} units, {spent} requests, summary {out_dir / 'summary.json'}")
    return 0


def _stage_unit(name: str, contract: str, args: argparse.Namespace) -> None:
    """Copy L0 (unchanged) and L2 (new contract) into the sweep staging dir."""
    stage: Path = args.stage
    stage.mkdir(parents=True, exist_ok=True)
    src_root: Path = args.hidden_root
    for level in (0, 2):
        src = src_root / f"{name}-L{level}"
        dest = stage / src.name
        if not src.is_dir():
            log(f"  skip stage {name}-L{level}: missing {src}")
            continue
        if dest.exists():
            # Keep src tree hardlinked; rewrite instruction.md via unlink.
            pass
        else:
            try:
                shutil.copytree(src, dest, copy_function=shutil.copy2, symlinks=True)
            except OSError:
                shutil.copytree(src, dest, symlinks=True)
        if level == 2:
            instr = dest / "instruction.md"
            orig = instr.read_text(encoding="utf-8")
            # Unlink in case dest shares inodes with src.
            if instr.exists():
                instr.unlink()
            instr.write_text(splice_contract(orig, contract), encoding="utf-8")
            log(f"  staged {dest.name} (new contract)")
        else:
            log(f"  staged {dest.name} (bugreport unchanged)")


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    sub = ap.add_subparsers(dest="cmd", required=True)

    u = sub.add_parser("unit", help="reconcile one unit dir")
    u.add_argument("--unit", required=True)
    u.add_argument("--details", type=Path, required=True)
    u.add_argument("--hidden", type=Path, required=True)
    u.add_argument("--gold", type=Path, required=True)
    u.add_argument("--out", type=Path, default=None)
    u.add_argument("--no-llm", action="store_true")
    u.add_argument("--max-requests", type=int, default=MAX_REQUESTS)

    b = sub.add_parser("batch", help="reconcile every unit under author-root")
    b.add_argument("--author-root", type=Path, required=True)
    b.add_argument("--hidden-root", type=Path, required=True)
    b.add_argument("--out-dir", type=Path, default=ROOT / "outputs" / "reconcile")
    b.add_argument("--stage", type=Path, default=None, help="dose-response sweep dir")
    b.add_argument("--units", nargs="*", default=None)
    b.add_argument("--no-llm", action="store_true")
    b.add_argument("--max-requests", type=int, default=MAX_REQUESTS)

    args = ap.parse_args(argv)
    if args.cmd == "unit":
        return _cmd_unit(args)
    if args.cmd == "batch":
        return _cmd_batch(args)
    ap.print_help()
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
