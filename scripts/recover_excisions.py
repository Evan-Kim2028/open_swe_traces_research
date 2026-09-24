#!/usr/bin/env python3
"""Thin CLI over openswe_traces.synth.excision_recover."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.excision_recover import (
    ExcisionRecoveryError,
    reconstruct_excision,
)

UPSTREAM = {
    "gin": "experiments/pipeline/repos2/gin/src",
    "nats-server": "experiments/pipeline/repos2/nats-server/src",
    "client-go": "experiments/harbor_nex/base/src",
    "helm": "experiments/pipeline/repos/helm/src",
    "kops": "experiments/pipeline/repos/kops/src",
    "goa": "experiments/pipeline/repos/goa/src",
}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Rebuild gitignored _author/excised/excision.patch from gold.patch + contract.md"
    )
    parser.add_argument("--repo", required=True, help="repo name, e.g. gin")
    parser.add_argument("--units", nargs="*", help="subset of unit names (default: all)")
    parser.add_argument("--upstream", default=None, help="override the upstream src tree")
    parser.add_argument("--force", action="store_true", help="rewrite an existing excision.patch")
    args = parser.parse_args(argv)

    rel = args.upstream or UPSTREAM.get(args.repo)
    if not rel:
        parser.error(f"no upstream tree known for {args.repo}; pass --upstream")
    upstream = Path(rel) if Path(rel).is_absolute() else ROOT / rel

    authored = ROOT / "experiments/pipeline/authored" / args.repo
    if not authored.is_dir():
        parser.error(f"no authored units at {authored}")
    names = args.units or sorted(p.name for p in authored.iterdir() if (p / "_author").is_dir())

    rows: list[dict[str, object]] = []
    failures = 0
    for name in names:
        author = authored / name / "_author"
        out = author / "excised" / "excision.patch"
        if out.is_file() and not args.force:
            rows.append({"unit": name, "status": "exists"})
            continue
        try:
            info = reconstruct_excision(author, upstream)
        except ExcisionRecoveryError as exc:
            failures += 1
            rows.append({"unit": name, "status": "failed", "error": str(exc)})
            continue
        rows.append(
            {
                "unit": name,
                "status": "rebuilt",
                "bytes": info["bytes"],
                "removed_tests": info["removed_tests"],
                "missing_tests": info["missing_tests"],
            }
        )
    print(json.dumps({"repo": args.repo, "units": rows, "failures": failures}, indent=2))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
