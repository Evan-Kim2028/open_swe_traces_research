#!/usr/bin/env python3
"""REPAIR2 shadow gate: implement each repaired unit's excised functions
from the contract ALONE, run the hidden suite, record the outcome.

  repair2_shadow.py <unit dir> [<unit dir> ...] [--results PATH]

Default results file: outputs/shadow_gate/results.jsonl (staged units).
Use --results outputs/shadow_gate/results_orig.jsonl when gating the
ORIGINAL unit dirs — the preexisting-vs-repair-caused comparison.

Resume-safe: a unit already in the results file with a non-"error" outcome
is skipped. One row per unit, appended as each finishes.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "src"))

from openswe_traces.gate import shadow  # noqa: E402


def done_units(path: Path) -> set[str]:
    out = set()
    if path.is_file():
        for line in path.read_text().splitlines():
            try:
                d = json.loads(line)
            except json.JSONDecodeError:
                continue
            if d.get("outcome") != "error":
                out.add(d["unit"])
    return out


def main(argv: list[str]) -> int:
    results = shadow.RESULTS_PATH
    units = []
    force = False
    i = 0
    while i < len(argv):
        if argv[i] == "--results":
            results = Path(argv[i + 1])
            i += 2
        elif argv[i] == "--force":
            force = True
            i += 1
        else:
            units.append(Path(argv[i]))
            i += 1
    if not units:
        print(__doc__)
        return 2

    key = shadow.load_api_key()
    results.parent.mkdir(parents=True, exist_ok=True)
    done = done_units(results)
    for u in units:
        name = u.name
        if name in done and not force:
            shadow.log(f"{name}: already gated, skipping")
            continue
        shadow.log(f"gating {name}")
        try:
            r = shadow.gate_unit(u, key)
        except Exception as e:
            shadow.log(f"  {name}: gate crashed: {e}")
            r = shadow.UnitResult(name, "error", None, "", 0, 0, 0.0,
                                  "", [], {}, [], [], [], [],
                                  f"{type(e).__name__}: {e}")
        with results.open("a") as fh:
            fh.write(shadow._result_to_json(r) + "\n")
        shadow.log(f"  {name}: {r.outcome} reward={r.reward} "
                   f"model={r.model} req={r.requests} tok={r.tokens} "
                   f"{r.seconds:.0f}s failing={r.failing_tests[:4]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
