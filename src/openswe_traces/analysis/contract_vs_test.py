#!/usr/bin/env python3
"""Surface a unit's hidden-test oracle next to its L2 contract, for a contradiction read.

The reconciler (in progress) automates deriving the contract from the tests. This is the
manual instrument that found helm-repindex's three contract defects: hidden tests written
by our verifier pass state their oracle in comments ("// oracle: ...", "// S3: ...") and in
assertion messages, and those are exactly the commitments the contract has to match.

Usage: contract_vs_test.py <unit-dir> [...]
"""
import re
import sys
from pathlib import Path

ORACLE_RE = re.compile(r"^\s*//\s*(S\d+.*|oracle:.*|gold-observed.*)$", re.MULTILINE)
FATAL_RE = re.compile(r"t\.(?:Fatalf|Errorf|Fatal|Error)\(\s*\"([^\"]{8,120})")


def main(argv: list[str]) -> int:
    for arg in argv:
        unit = Path(arg)
        print("=" * 100)
        print(unit.name)
        instr = unit / "instruction.md"
        if instr.is_file():
            text = instr.read_text()
            print("\n--- CONTRACT ---")
            print(text.split("Reproduce with:")[0].strip()[:3000])
        hidden = sorted((unit / "tests/hidden").rglob("*_test.go"))
        for h in hidden:
            src = h.read_text()
            print(f"\n--- HIDDEN ORACLE  {h.name}  ({len(src.splitlines())} lines) ---")
            for m in ORACLE_RE.finditer(src):
                print("  //", m.group(1).strip())
            print("  -- assertion messages --")
            seen = set()
            for m in FATAL_RE.finditer(src):
                msg = m.group(1)
                if msg not in seen:
                    seen.add(msg)
                    print("   *", msg)
    return 0


def cli():
    raise SystemExit(main(sys.argv[1:]))


if __name__ == "__main__":
    cli()
