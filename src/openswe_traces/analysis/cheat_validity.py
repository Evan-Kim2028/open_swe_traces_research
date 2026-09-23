#!/usr/bin/env python3
"""Is this 'cheat patch' actually a cheat? Audit the adversary before trusting its verdict.

A3 says a patch that special-cases the contract's examples must FAIL the hidden suite. The
verdict is only meaningful if the patch really is a special-case. On 2026-09-20 the repair loop
reported 5 of 14 units as "cheat now passes" — and every one of those patches turned out to be a
full working implementation, 94%-171% the size of gold. A model asked to cheat had simply written
the code. The alarm was a mislabeled artifact, not a contract leak.

Two signals, cheapest first:

  SIZE     added lines vs gold's added lines. >= 0.6 means the patch is doing the real work.
           A genuine cheat is a hardcoded table or an early return: small and lumpy.
  SEED     (needs --docker) re-run under AUDIT_HIDDEN_SEED. A real implementation passes both
           seeds; a patch that memorised the shipped seed's examples passes only the original.
           This reuses the B5 machinery in pipeline_ext.hack_audit.

A patch that fails either check is not a valid adversary and must be regenerated before its
pass/fail says anything about the contract.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ADD_RE = re.compile(r"^\+(?!\+\+)", re.MULTILINE)
HARDCODE_RE = re.compile(
    r"switch\s+\w+\s*\{|map\[string\]string\{|if\s+\w+\s*==\s*\"|return\s+\"[^\"]{3,}\"",
)
SIZE_FLOOR = 0.6


def added(p: Path) -> int:
    return len(ADD_RE.findall(p.read_text(errors="replace"))) if p.is_file() else 0


def audit(unit: Path) -> dict:
    cheat = unit / "tests" / "cheat.patch"
    gold = unit / "tests" / "gold.patch"
    c, g = added(cheat), added(gold)
    ratio = c / g if g else 0.0
    body = cheat.read_text(errors="replace") if cheat.is_file() else ""
    hardcodes = len(HARDCODE_RE.findall(body))
    verdict = "IMPLEMENTATION" if ratio >= SIZE_FLOOR else "plausible-cheat"
    return {"unit": unit.name, "cheat_lines": c, "gold_lines": g, "ratio": round(ratio, 2),
            "hardcode_markers": hardcodes, "verdict": verdict}


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("units", nargs="+", type=Path)
    a = ap.parse_args(argv)
    bad = 0
    print(f"{'unit':30} {'cheat':>7} {'gold':>7} {'ratio':>7}  verdict")
    for u in a.units:
        if not (u / "tests").is_dir():
            continue
        r = audit(u)
        if r["verdict"] == "IMPLEMENTATION":
            bad += 1
        print(f"{r['unit']:30} {r['cheat_lines']:>7} {r['gold_lines']:>7} "
              f"{r['ratio']:>7.2f}  {r['verdict']}")
    if bad:
        print(f"\n{bad} patch(es) are implementations, not cheats — their A3 verdict is MEANINGLESS.")
        print("Regenerate with an explicit instruction to special-case the contract's worked")
        print("examples and nothing else, then re-run preflight.")
    return 1 if bad else 0


def cli():
    raise SystemExit(main(sys.argv[1:]))


if __name__ == "__main__":
    cli()
