"""A13 calibration — judge + deterministic findings vs the hand audit.

Joins ``outputs/contract_judge/*.json`` (the LLM judge's cached verdicts) and
the deterministic findings of ``gate.contract_gold`` against the solver-free
hand audit in ``oswt-LIE/outputs/contract_gold_dump/*_verdict.json``.

Usage:

    uv run python scripts/a13_calibration.py [--dump PATH] [--json]
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from openswe_traces.data import ROOT
from openswe_traces.gate.context import GateContext
from openswe_traces.gate.contract_gold import (
    deterministic_findings,
    gather_evidence,
    load_judgement,
)

DUMP = Path("/home/evan/Documents/oswt-LIE/outputs/contract_gold_dump")
TASKS = ROOT / "experiments" / "pipeline" / "tasks_composerver"


_TEST_NAME_RE = re.compile(r"Test[A-Z][A-Za-z0-9_]*")


def _missing_tests(missing: list) -> set[str]:
    out: set[str] = set()
    for m in missing:
        if isinstance(m, dict):
            out.update(_TEST_NAME_RE.findall(str(m.get("hidden_test") or "")))
        else:
            out.update(_TEST_NAME_RE.findall(str(m)))
    return out


def load_answer_key(dump: Path) -> dict[str, dict]:
    """unit -> {defective, false_rows{test}, missing_tests{test}, vague_rows{test}}."""
    out: dict[str, dict] = {}
    for path in sorted(dump.glob("*_verdict.json")):
        repo = path.stem.replace("_verdict", "")
        data = json.loads(path.read_text())
        for u in data.get("units") or []:
            key = f"{repo}/{u['unit']}"
            out[key] = {
                "defective": bool(u.get("defective")),
                "false_rows": {
                    r["original_test"]
                    for r in u.get("rows") or []
                    if r.get("class") == "false"
                },
                "vague_rows": {
                    r["original_test"]
                    for r in u.get("rows") or []
                    if r.get("class") == "vague"
                },
                "missing_tests": _missing_tests(u.get("missing") or []),
                "missing": u.get("missing") or [],
                "rows": u.get("rows") or [],
            }
    return out


def judge_of(unit_key: str, cache: dict[str, dict]) -> dict | None:
    for d in cache.values():
        if d.get("unit") == unit_key:
            return d.get("verdict") or {}
    return None


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--dump", type=Path, default=DUMP)
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()

    key = load_answer_key(args.dump)
    # unit -> judge verdict + deterministic findings (one pass over the bank)
    rows: list[dict] = []
    seen: set[str] = set()
    for td in sorted(TASKS.glob("*/*-L*")):
        unit_key = f"{td.parent.name}/{td.name.rsplit('-L', 1)[0]}"
        if unit_key in seen or unit_key not in key:
            continue
        ev = gather_evidence(GateContext(task_dir=td))
        if not ev.rows:
            continue
        seen.add(unit_key)
        judgement = load_judgement(ev)
        det = deterministic_findings(ev, GateContext(task_dir=td).excised.bare_names)
        jv = (judgement or {}).get("verdict") or {}
        judge_rows = {
            str(r.get("test")): str(r.get("verdict"))
            for r in jv.get("rows") or []
            if isinstance(r, dict)
        }
        judge_missing = {
            str(m.get("test")): str(m.get("assertion") or m.get("property") or "")
            for m in jv.get("missing") or []
            if isinstance(m, dict)
        }
        hand = key[unit_key]
        rows.append(
            {
                "unit": unit_key,
                "judged": judgement is not None,
                "hand_defective": hand["defective"],
                "hand_false": sorted(hand["false_rows"]),
                "hand_missing": sorted(hand["missing_tests"]),
                "hand_vague": sorted(hand["vague_rows"]),
                "judge_false": sorted(t for t, v in judge_rows.items() if v == "false"),
                "judge_vague": sorted(t for t, v in judge_rows.items() if v == "vague"),
                "judge_missing": sorted(judge_missing),
                "det_suspects": sorted(f.subject for f in det if f.cls == "suspect"),
                "det_vague": sorted(f.subject for f in det if f.cls == "vague"),
            }
        )

    # ---- agreement scoring ---------------------------------------------------
    # Gate semantics: with a cached judgement the unit FAILS iff the judge
    # returned a "false" row verdict or any "missing" entry; det suspects are
    # unresolved-fail only when no judgement exists.
    def flagged(r: dict) -> bool:
        if r["judge_false"] or r["judge_missing"]:
            return True
        return not r["judged"] and bool(r["det_suspects"])

    n_key = len(key)
    n_judged = sum(1 for r in rows if r["judged"])
    tp = sum(1 for r in rows if r["hand_defective"] and flagged(r))
    fn = [r["unit"] for r in rows if r["hand_defective"] and not flagged(r)]
    fp_rows = []
    for r in rows:
        if r["hand_defective"]:
            continue
        noise = set(r["judge_false"]) | set(r["judge_missing"])
        if not r["judged"]:
            noise |= set(r["det_suspects"])
        if noise:
            fp_rows.append((r["unit"], sorted(noise)))

    # per-defect agreement
    false_hits = false_miss = 0
    for r in rows:
        for t in r["hand_false"]:
            if t in r["judge_false"]:
                false_hits += 1
            else:
                false_miss += 1
    missing_hits = missing_miss = 0
    for r in rows:
        for t in r["hand_missing"]:
            if t in r["judge_missing"]:
                missing_hits += 1
            else:
                missing_miss += 1

    report = {
        "n_units_in_key": n_key,
        "n_judged": n_judged,
        "unjudged": sorted(set(key) - {r["unit"] for r in rows if r["judged"]}),
        "unit_level": {
            "hand_defective": sum(1 for r in rows if r["hand_defective"]),
            "flagged_defective": tp,
            "missed_defective": fn,
            "clean_flagged": fp_rows,
        },
        "defect_level": {
            "false_rows": {"hand": false_hits + false_miss, "judge_hit": false_hits},
            "missing_assertions": {
                "hand": missing_hits + missing_miss,
                "judge_hit": missing_hits,
            },
        },
        "units": rows,
    }
    if args.json:
        print(json.dumps(report, indent=1))
        return 0

    print(f"units in answer key: {n_key}; judged: {n_judged}; "
          f"unjudged: {report['unjudged']}")
    print(f"hand-defective units: {report['unit_level']['hand_defective']}; "
          f"flagged by A13: {tp}; missed: {fn}")
    print(f"clean units with A13 flags: {len(fp_rows)}")
    for u, noise in fp_rows:
        print(f"  {u}: {noise}")
    print(f"false rows: judge caught {false_hits}/{false_hits + false_miss}")
    print(f"missing assertions: judge caught {missing_hits}/{missing_hits + missing_miss}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
