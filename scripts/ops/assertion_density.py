#!/usr/bin/env python3
"""Count each unit's distinct behavioural commitments and compare with its L2 pass rate.

Closure size was refuted as a difficulty lever. This tests the successor hypothesis that
emerged from the double-failure audit: difficulty tracks the number of independent
commitments the solver must satisfy simultaneously -- the specification bandwidth the
contract has to carry.
"""
import json
import re
from collections import defaultdict
from pathlib import Path

FATAL_RE = re.compile(r"t\.(?:Fatalf|Errorf|Fatal|Error)\(\s*\"([^\"]{6,160})")
TESTFUNC_RE = re.compile(r"^func (Test\w+)\(", re.MULTILINE)
ROOT = Path("experiments/dose_response")


def commitments(unit: Path) -> tuple[int, int, int]:
    """(distinct assertion messages, test functions, hidden test lines)"""
    msgs, funcs, lines = set(), 0, 0
    for h in (unit / "tests/hidden").rglob("*_test.go"):
        src = h.read_text(errors="replace")
        msgs |= set(FATAL_RE.findall(src))
        funcs += len(TESTFUNC_RE.findall(src))
        lines += len(src.splitlines())
    return len(msgs), funcs, lines


def main() -> int:
    # L2 outcomes per family
    out = defaultdict(list)
    for job in sorted((ROOT / "jobs").iterdir()):
        if not job.is_dir():
            continue
        void = set()
        vf = job / "void_attempts.txt"
        if vf.is_file():
            void = {x for x in vf.read_text().split() if x}
        for t in sorted(job.iterdir()):
            r = t / "result.json"
            if not r.is_file() or t.name in void:
                continue
            try:
                d = json.loads(r.read_text())
            except Exception:
                continue
            rew = (d.get("verifier_result") or {}).get("rewards", {}).get("reward")
            name = d.get("task_name") or t.name.split("__")[0]
            if rew is None or not name.endswith("-L2"):
                continue
            out[name[:-3]].append(1 if rew == 1.0 else 0)

    rows = []
    seen = set()
    for unit in sorted(ROOT.glob("sweep_*/*-L2")):
        fam = unit.name[:-3]
        if fam in seen or fam not in out or not (unit / "tests/hidden").is_dir():
            continue
        seen.add(fam)
        n_msg, n_fn, n_ln = commitments(unit)
        res = out[fam]
        rows.append((fam, n_msg, n_fn, n_ln, sum(res), len(res)))

    rows.sort(key=lambda r: r[1])
    print(f"{'family':30} {'assertions':>10} {'testfns':>8} {'lines':>7} {'L2 pass':>9}")
    for fam, m, f, ln, p, n in rows:
        print(f"{fam:30} {m:>10} {f:>8} {ln:>7} {p:>4}/{n:<4}")

    # split at the median assertion count
    med = sorted(r[1] for r in rows)[len(rows) // 2]
    lo = [r for r in rows if r[1] <= med]
    hi = [r for r in rows if r[1] > med]
    for label, grp in (("<= median", lo), ("> median", hi)):
        p = sum(r[4] for r in grp)
        n = sum(r[5] for r in grp)
        u = sum(1 for r in grp if r[4] > 0)
        print(f"\n{label} ({med} assertions): {len(grp)} units, {u} ever passed, "
              f"{p}/{n} trials = {p / max(1, n):.0%}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
