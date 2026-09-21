#!/usr/bin/env python3
"""REPAIR2 round 2: residual-driven contract repair + re-verify.

  repair2_round2.py <staged unit dir> [<staged unit dir> ...]

Round 1 repaired the audited gaps in outputs/repair2/gaps.jsonl. The
verification re-audit then flagged a second, smaller set of residuals —
some real, some noise. outputs/repair2/round2_gaps.json holds the VETTED
residual list (each item confirmed against the staged contract text and
the hidden suite by hand; reaudit hallucinations excluded).

For each unit: repair2.repair_contract on the staged dir (same mechanical
guards as round 1), then repair2_verify.verify_one -> append results row.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "src"))

from openswe_traces.gate import repair2  # noqa: E402
import repair2_verify  # noqa: E402

SPEC = Path("outputs/repair2/round2_gaps.json")


def main(argv: list[str]) -> int:
    spec = json.loads(SPEC.read_text())
    key = repair2.load_api_key()
    for u in [Path(a) for a in argv]:
        name = u.name
        gaps = spec.get(name, [])
        print(f"=== {name} ({len(gaps)} residual gaps) ===", flush=True)
        if not gaps:
            print("  no vetted residuals — skip", flush=True)
            continue
        ro = repair2.repair_contract(u, gaps, key, f"repair2r2/{name}")
        print(f"  applied={ro.applied} violations={ro.violations} "
              f"note={ro.note}", flush=True)
        if not ro.applied:
            continue
        try:
            row = repair2_verify.verify_one(u)
        except Exception as e:
            print(f"  verify failed: {e}", flush=True)
            row = {"unit": name, "error": str(e), "round": 2}
        row["round"] = 2
        with repair2.RESULTS_PATH.open("a") as fh:
            fh.write(json.dumps(row) + "\n")
        print(json.dumps({k: v for k, v in row.items()
                          if k != "reaudit_answer"}, indent=1)[:1800],
              flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
