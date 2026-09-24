#!/usr/bin/env python3
"""Give an L0 bug report the reproduce command it is missing.

B6 is the instruction floor: an L0 unit whose bug report names no way to reproduce the
failure is refused by task_lint, which means the sweep drops it and the cohort trials
nothing. Both Grok-authored cohorts are in that state — 10 of 10 units blocked in each,
20 units that cost real tokens to author and cannot be trialled as written.

It also wastes launches. orchestrate counts a lint-blocked unit as runnable, so
sweep_grokgogit launched three times at concurrency 8, dropped all ten units each time,
and exited — while sweep_escalate_L4 waited behind it.

The command is derived, never invented: ``tests/test.sh`` already runs the graded suite,
so the package under test is recorded there. What the repair must NOT copy is the
``-run '^(TestDetail01|...)$'`` filter — naming the hidden tests in an L0 bug report
raises the rung, and a unit that is easy because we told it the test names certifies
nothing. Compare a healthy L0:

    tests/test.sh   go test -count=1 -timeout 15m -run '^(TestDetail01|...)$' ./upup/pkg/fi/cloudup/awsup
    instruction.md  go test -count=1 ./upup/pkg/fi/cloudup/awsup/

Package only. That is the convention this reproduces.

Usage::

    repair_b6.py                    # what would change
    repair_b6.py --apply
    repair_b6.py --apply <dir>...
"""

from __future__ import annotations
from openswe_traces.paths import REPO as _REPO

import argparse
import glob
import pathlib
import re
import subprocess
import sys

REPO = _REPO
# Same regex task_lint gates on, so "repaired" means exactly "passes the gate".
REPRO_RE = re.compile(r"go test|pytest|npm test|cargo test|tests/test\.sh|make test",
                      re.IGNORECASE)
GO_TEST_RE = re.compile(r"go test\s+(?P<flags>.*?)(?P<pkgs>(?:\./|\S*/)\S*)\s*(?:;|$)")
NO_WEB = "IMPORTANT: This repository is fully self-contained."


def package_of(unit: pathlib.Path) -> str | None:
    """The package the graded suite runs, taken from the unit's own test.sh."""
    sh = unit / "tests" / "test.sh"
    if not sh.is_file():
        return None
    for line in sh.read_text(errors="replace").splitlines():
        if "go test" not in line:
            continue
        # Drop the -run filter FIRST: it contains the hidden test names, and the package
        # sits after it.
        stripped = re.sub(r"-run\s+'[^']*'", "", line)
        stripped = re.sub(r"-run\s+\"[^\"]*\"", "", stripped)
        m = GO_TEST_RE.search(stripped)
        if m and m.group("pkgs"):
            pkg = m.group("pkgs").strip().rstrip(";").strip()
            if pkg.startswith("./") or pkg.startswith("."):
                return pkg
    return None


def repair(unit: pathlib.Path, apply: bool) -> tuple[bool, str]:
    instr = unit / "instruction.md"
    if not instr.is_file():
        return False, "no instruction.md"
    text = instr.read_text(errors="replace")
    if REPRO_RE.search(text):
        return False, "already has a reproduce command"
    pkg = package_of(unit)
    if not pkg:
        return False, "no go-test package found in tests/test.sh"
    block = f"\nReproduce with:\n\n```\ngo test -count=1 {pkg}\n```\n"
    if NO_WEB in text:
        new = text.replace(NO_WEB, block + "\n" + NO_WEB, 1)
    else:
        new = text.rstrip() + "\n" + block
    if apply:
        instr.write_text(new)
    return True, f"go test -count=1 {pkg}"


def lint_ok(unit: pathlib.Path) -> bool:
    r = subprocess.run(["uv", "run", "python", "scripts/ops/task_lint.py", str(unit)],
                       cwd=REPO, capture_output=True, text=True, timeout=600)
    return "BLOCK" not in r.stdout


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("units", nargs="*")
    ap.add_argument("--apply", action="store_true")
    args = ap.parse_args()

    units = [pathlib.Path(u) for u in args.units]
    if not units:
        units = [pathlib.Path(d.rstrip("/")) for d in
                 glob.glob(str(REPO / "experiments/dose_response/sweep_*/*-L0/"))]
    todo = []
    for u in sorted(units):
        ok, why = repair(u, apply=False)
        if ok:
            todo.append((u, why))
    print(f"L0 units missing a reproduce command: {len(todo)}")
    for u, why in todo[:8]:
        print(f"  {u.name:28s} -> {why}")
    if len(todo) > 8:
        print(f"  ... and {len(todo) - 8} more")
    if not args.apply:
        print("\n(dry run — rerun with --apply)")
        return 0

    fixed = failed = 0
    for u, _ in todo:
        repair(u, apply=True)
        if lint_ok(u):
            fixed += 1
        else:
            failed += 1
            print(f"  STILL BLOCKED after repair: {u.name}")
    print(f"\nrepaired {fixed}, still blocked {failed}")
    return 1 if failed else 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
