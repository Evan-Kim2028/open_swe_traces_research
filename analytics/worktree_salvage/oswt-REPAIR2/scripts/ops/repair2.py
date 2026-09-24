#!/usr/bin/env python3
"""REPAIR2 per-unit driver. Thin CLI over openswe_traces.gate.repair2.

  repair2.py stage  <src unit dir> <stage root>     copy unit into sweep_repair2
  repair2.py gaps   <unit>                          print the unit's audit gaps
  repair2.py contract <staged unit dir>             draft+guard contract repair
  repair2.py reaudit <staged unit dir>              contract-gap re-read (fresh)
  repair2.py lint    <staged unit dir>              task_lint + literal budget
  repair2.py preflight <staged unit dir>            bare/gold/fresh-cheat
  repair2.py verify  <staged unit dir>              reaudit + lint + preflight
  repair2.py record  <staged unit dir> <json>       append a results row

All LLM calls go through Composer via the gate cache. Nothing writes back to
the source unit dir — repairs land only under the stage root.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "src"))

from openswe_traces.gate import repair2  # noqa: E402


def main(argv: list[str]) -> int:
    cmd, args = argv[0], argv[1:]
    if cmd == "stage":
        import shutil
        src, root = Path(args[0]), Path(args[1])
        dest = root / src.name
        if not dest.exists():
            root.mkdir(parents=True, exist_ok=True)
            shutil.copytree(src, dest, symlinks=True)
        print(dest)
        return 0
    if cmd == "gaps":
        per = repair2.load_gaps()
        for g in per.get(args[0], []):
            print(f"[{g['kind']:>9}] {g['gtype']:8} {g['text']}")
        return 0
    unit = Path(args[0])
    if cmd == "contract":
        per = repair2.load_gaps()
        gaps = [g for g in per.get(unit.name, [])
                if g["kind"] in ("DERIVABLE", "COUNTER", "UNKNOWN")]
        if not gaps:
            print("no DERIVABLE/COUNTER gaps — nothing to repair")
            return 0
        before = repair2.literal_budget(unit)
        ro = repair2.repair_contract(unit, gaps, repair2.load_api_key(),
                                     f"repair2/{unit.name}")
        after = repair2.literal_budget(unit)
        print(f"applied={ro.applied} violations={ro.violations} note={ro.note}")
        print(f"literals {before[0]}->{after[0]} grounded "
              f"{before[2]:.0%}->{after[2]:.0%}")
        return 0 if ro.applied else 1
    if cmd == "reaudit":
        r = repair2.reaudit(unit)
        print(json.dumps({k: v for k, v in r.items() if k != "answer"}, indent=1))
        print(r["answer"])
        return 0
    if cmd == "lint":
        sys.path.insert(0, str(Path(__file__).parent))
        import task_lint
        findings = task_lint.lint(unit)
        n_lit, n_gr, frac = repair2.literal_budget(unit)
        for sev, code, msg in findings:
            print(f"[{sev}] {code}: {msg}")
        print(f"literals={n_lit} grounded={n_gr} ({frac:.0%})")
        return 1 if any(f[0] == "BLOCK" for f in findings) else 0
    if cmd == "preflight":
        pf = repair2.preflight(unit, repair2.load_api_key())
        print(f"bare={pf.bare_reward} gold={pf.gold_reward} "
              f"cheat={pf.cheat_reward} verdict={pf.verdict} note={pf.note}")
        if pf.cheat_patch:
            (unit / "tests" / "cheat.patch").write_text(pf.cheat_patch)
            p2 = unit / "patches" / "cheat.patch"
            if (unit / "patches").is_dir():
                p2.write_text(pf.cheat_patch)
        return 0 if pf.verdict == "ok" else 1
    if cmd == "verify":
        r = repair2.reaudit(unit)
        print(json.dumps({k: v for k, v in r.items() if k != "answer"}))
        pf = repair2.preflight(unit, repair2.load_api_key())
        print(f"preflight bare={pf.bare_reward} gold={pf.gold_reward} "
              f"cheat={pf.cheat_reward} verdict={pf.verdict}")
        return 0
    if cmd == "record":
        row = json.loads(args[1])
        repair2.RESULTS_PATH.parent.mkdir(parents=True, exist_ok=True)
        with repair2.RESULTS_PATH.open("a") as fh:
            fh.write(json.dumps(row) + "\n")
        return 0
    print(__doc__)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
