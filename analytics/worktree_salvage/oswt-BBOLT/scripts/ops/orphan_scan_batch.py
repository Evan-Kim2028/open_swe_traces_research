#!/usr/bin/env python3
"""Reachability / orphan scan over a batch of unit dirs (gate 2).

Uses GATEALL's cgscan binary and cg_coverage.compute_sets. No LLM.
orphan count = |gap_deterministic| = |reached ∩ gold − implied|, matching
scripts/gate_bank.py. Advisory; >=2 on an L2 unit is the repair threshold.
"""
from __future__ import annotations

import argparse
import json
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

sys.path.insert(0, "/home/evan/Documents/oswt-GATEALL/src")
from openswe_traces import cg_coverage as cc  # noqa: E402

cc.OUT = Path("/home/evan/Documents/oswt-BBOLT/outputs/cg_coverage")
cc.SCAN_BIN = Path("/home/evan/Documents/oswt-GATEALL/outputs/cg_coverage/cgscan")
cc.LOG = Path("/home/evan/Documents/oswt-BBOLT/outputs/BBOLT.log")


def is_exported_func(qname: str) -> bool:
    short = qname.split(").")[-1] if ")." in qname else qname.rsplit(".", 1)[-1]
    return bool(short) and short[0].isupper() and short[0].isalpha()


def hidden_names(unit: Path) -> str:
    blob = []
    hd = unit / "tests" / "hidden"
    if hd.is_dir():
        for f in hd.rglob("*.go"):
            blob.append(f.read_text(errors="replace"))
    return "\n".join(blob)


def scan_unit(unit: Path) -> dict:
    out_json = cc.OUT / "scan" / f"{unit.parent.name}__{unit.name}.json"
    cc.run_scan(unit, out_json, force=False)
    scan = json.loads(out_json.read_text())
    sets = cc.compute_sets(scan, cc.contract_text(unit))
    gold = sets["gold"]
    gold_fn = [s for s in gold if is_exported_func(s)]
    hidden = hidden_names(unit)
    unnamed = []
    for s in gold_fn:
        short = s.split(").")[-1] if ")." in s else s.rsplit(".", 1)[-1]
        if len(short) >= 2 and short not in hidden:
            unnamed.append(s)
    gap = sets["gap_deterministic"]
    return {
        "unit": unit.name,
        "module": sets.get("module"),
        "n_gold": len(gold),
        "n_gold_fn": len(gold_fn),
        "n_candidates": len(sets["candidates"]),
        "n_reached": len(sets["reached"]),
        "orphans": len(gap),
        "orphan_names": gap,
        "unnamed_gold_fn": unnamed,
        "n_unnamed_gold_fn": len(unnamed),
        "n_hidden_tests": sets.get("n_hidden_tests"),
        "warnings": sets.get("warnings") or [],
    }


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--batch", type=Path, required=True)
    ap.add_argument("--out", type=Path, default=Path("outputs/bbolt_orphans.jsonl"))
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--l2-only", action="store_true")
    args = ap.parse_args(argv)
    units = sorted(p for p in args.batch.iterdir() if p.is_dir() and not p.name.startswith("_"))
    if args.l2_only:
        units = [p for p in units if p.name.endswith("-L2")]
    args.out.parent.mkdir(parents=True, exist_ok=True)
    done = set()
    if args.out.is_file():
        for line in args.out.read_text().splitlines():
            try:
                done.add(json.loads(line)["unit"])
            except Exception:
                pass
    todo = [u for u in units if u.name not in done]
    print(f"orphan_scan: {len(todo)} to do of {len(units)}", flush=True)
    with args.out.open("a") as fh, ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {ex.submit(scan_unit, u): u for u in todo}
        for f in as_completed(futs):
            u = futs[f]
            try:
                row = f.result()
            except Exception as exc:  # noqa: BLE001
                row = {"unit": u.name, "error": str(exc)[:400]}
            fh.write(json.dumps(row) + "\n")
            fh.flush()
            if "error" in row:
                print(f"FAIL {u.name}: {row['error'][:160]}", flush=True)
            else:
                print(
                    f"{u.name:24} orphans={row['orphans']:>3} "
                    f"cand={row['n_candidates']:>3} gold={row['n_gold']:>3} "
                    f"gold_fn={row['n_gold_fn']:>3} unnamed={row['n_unnamed_gold_fn']:>3}",
                    flush=True,
                )
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
