#!/usr/bin/env python3
"""Thin CLI over openswe_traces.cg_coverage — call-graph contract coverage gaps.

  uv run python scripts/cg_coverage.py universe          # build unit roster
  uv run python scripts/cg_coverage.py scan [--workers N] [--force]
  uv run python scripts/cg_coverage.py map               # LLM prose->symbol pass
  uv run python scripts/cg_coverage.py report            # confusion matrix + repindex
  uv run python scripts/cg_coverage.py unit <dir>        # one unit, print sets
"""
from __future__ import annotations

import argparse
import json
import sys
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

from openswe_traces import cg_coverage as cc


def _universe(args) -> list[dict]:
    uj = cc.OUT / "universe.json"
    if uj.is_file() and not args.rebuild:
        return json.loads(uj.read_text())
    units = cc.build_universe()
    uj.parent.mkdir(parents=True, exist_ok=True)
    uj.write_text(json.dumps(units, indent=2))
    return units


def cmd_universe(args) -> int:
    units = cc.build_universe()
    (cc.OUT / "universe.json").write_text(json.dumps(units, indent=2))
    for u in units:
        print(("FLIP   " if u["flipped"] else "nonflip") + "  " + u["rel"])
    return 0


def cmd_scan(args) -> int:
    units = _universe(args)
    todo = [u for u in units if args.force or not cc.scan_path(Path(u["path"])).is_file()]
    cc.log(f"scan: {len(todo)} to do of {len(units)}")
    done = 0
    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {ex.submit(cc.run_scan, Path(u["path"]), cc.scan_path(Path(u["path"])), args.force): u for u in todo}
        for f in as_completed(futs):
            u = futs[f]
            try:
                f.result()
                done += 1
                cc.log(f"scan ok {u['unit']} ({done}/{len(todo)})")
            except Exception as e:  # noqa: BLE001 (keep going; unit stays unscanned)
                cc.log(f"scan FAIL {u['unit']}: {e}")
    return 0


def cmd_map(args) -> int:
    units = _universe(args)
    api_key = cc.load_api_key() if not args.no_llm else None
    budget = [0]
    lock = threading.Lock()
    todo = []
    for u in units:
        up = Path(u["path"])
        if not cc.scan_path(up).is_file():
            cc.log(f"map skip (no scan): {u['unit']}")
            continue
        if cc.sets_path(up).is_file() and not args.force:
            continue
        todo.append(u)
    cc.log(f"map: {len(todo)} to do of {len(units)}")

    def one(u):
        up = Path(u["path"])
        try:
            cc.analyze_unit(up, api_key, budget, force_scan=args.force,
                            llm=not args.no_llm, budget_lock=lock)
            cc.log(f"map ok {u['unit']} (reqs={budget[0]})")
        except Exception as e:  # noqa: BLE001 (keep going; unit stays unmapped)
            cc.log(f"map FAIL {u['unit']}: {e}")

    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        list(ex.map(one, todo))
    return 0


def cmd_unit(args) -> int:
    up = Path(args.unit).resolve()
    api_key = cc.load_api_key() if not args.no_llm else None
    sets = cc.analyze_unit(up, api_key, [0], force_scan=args.force, llm=not args.no_llm)
    print(json.dumps(sets, indent=2))
    return 0


def _repindex_units() -> list[dict]:
    out = []
    for sweep, name, label in cc.REPINDEX_SERIES:
        p = cc.DR / sweep / name
        if p.is_dir():
            out.append({"unit": name, "sweep": sweep, "path": str(p), "label": label})
    return out


def cmd_repindex(args) -> int:
    api_key = cc.load_api_key() if not args.no_llm else None
    budget = [0]
    for u in _repindex_units():
        up = Path(u["path"])
        if not cc.sets_path(up).is_file() or args.force:
            cc.analyze_unit(up, api_key, budget, force_scan=args.force, llm=not args.no_llm)
        sets = json.loads(cc.sets_path(up).read_text())
        gap = sets["gap_deterministic"] if args.no_llm else sets["gap"]
        print(f"{u['label']:22s} cand={len(sets['candidates']):3d} gap={len(gap):3d}  {gap}")
    cc.log(f"repindex done (reqs={budget[0]})")
    return 0


def _print_conf(tag: str, c: dict) -> None:
    print(f"  {tag:34s} n={c['n']:3d} flagged={c['flagged']:3d} "
          f"tp={c['tp']:3d} fp={c['fp']:3d} fn={c['fn']:3d} tn={c['tn']:3d} "
          f"prec={c['precision']:.1%} rec={c['recall']:.1%} "
          f"base={c['base_rate']:.1%}")


def cmd_report(args) -> int:
    units = _universe(args)
    primary = [u for u in units if not u.get("pending")]
    pending = [u for u in units if u.get("pending")]
    have = [u for u in primary if cc.sets_path(Path(u["path"])).is_file()]
    have_p = [u for u in pending if cc.sets_path(Path(u["path"])).is_file()]
    cc.log(f"report: {len(have)} primary + {len(have_p)} pending with sets")

    print("\n=== gap-nonempty -> did-not-flip ===")
    print(f"primary cohort: {len(primary)} units "
          f"(linter documented 119 units, 47 nonflip = 39% base rate)")
    for field, tag in (("gap_rows_only", "rows+literal only"),
                       ("gap_deterministic", "rows+literal+hidden-named"),
                       ("gap", "full (+LLM map)")):
        _print_conf(tag, cc.confusion(have, field))
    if have_p:
        print(f"\nrobustness: +{len(have_p)} /tmp-staged clientgo units")
        for field, tag in (("gap_deterministic", "det"), ("gap", "full")):
            _print_conf(tag, cc.confusion(have + have_p, field))

    print("\n=== double-failure families ===")
    for fam in cc.DOUBLE_FAILURE:
        hit = [u for u in have if u["unit"] == fam + "-L2"]
        if not hit:
            hit = [u for u in have if u["unit"].startswith(fam)]
        if not hit:
            print(f"  {fam:32s} (no unit)")
            continue
        sets = json.loads(cc.sets_path(Path(hit[0]["path"])).read_text())
        print(f"  {hit[0]['unit']:32s} flipped={hit[0]['flipped']} "
              f"cand={len(sets['candidates'])} gap={sets['gap']}")

    print("\n=== helm-repindex contract revisions ===")
    for u in _repindex_units():
        sp = cc.sets_path(Path(u["path"]))
        if not sp.is_file():
            print(f"  {u['label']:22s} (not scanned)")
            continue
        sets = json.loads(sp.read_text())
        print(f"  {u['label']:22s} cand={len(sets['candidates']):3d} "
              f"gap={len(sets['gap']):3d} (det={len(sets['gap_deterministic'])}, "
              f"rows={len(sets['gap_rows_only'])})  {sets['gap']}")

    rep = {"confusion": {
               "rows_only": cc.confusion(have, "gap_rows_only"),
               "deterministic": cc.confusion(have, "gap_deterministic"),
               "full": cc.confusion(have, "gap"),
           },
           "units": have}
    (cc.OUT / "report.json").write_text(json.dumps(rep, indent=2))
    return 0


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("universe", "scan", "map", "report", "repindex", "unit"):
        p = sub.add_parser(name)
        p.add_argument("--force", action="store_true")
        p.add_argument("--rebuild", action="store_true")
        p.add_argument("--no-llm", action="store_true")
        p.add_argument("--workers", type=int, default=8)
        p.add_argument("unit", nargs="?")
        p.set_defaults(fn=globals()["cmd_" + name])
    args = ap.parse_args(argv)
    return args.fn(args)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
