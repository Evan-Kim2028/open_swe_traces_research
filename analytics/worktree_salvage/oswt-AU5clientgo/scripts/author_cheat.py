#!/usr/bin/env python3
"""Emit cheat.patch for a unit: diff excised/tree/<rel> vs a hand-edited worktree.

Usage: author_cheat.py UNIT WORKTREE REL [REL ...]
WORKTREE is the excised copy the author edited in place (e.g. /tmp/excise_X/app).
Writes _author/cheat.patch and _author/cheat/<rel> copies, prints line counts.
"""
from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def udiff(old: Path, new: Path, rel: str) -> str:
    r = subprocess.run(
        ["diff", "-u", "--label", f"a/{rel}", "--label", f"b/{rel}", str(old), str(new)],
        capture_output=True, text=True)
    if r.returncode == 0:
        return ""
    if r.returncode != 1:
        raise RuntimeError(r.stderr)
    return f"diff --git a/{rel} b/{rel}\n" + r.stdout


def added_lines(patch: str) -> int:
    return sum(1 for ln in patch.splitlines() if ln.startswith("+") and not ln.startswith("+++"))


def main(argv: list[str]) -> int:
    unit, worktree = argv[1], Path(argv[2])
    rels = argv[3:]
    out = ROOT / "experiments/pipeline/authored_au5clientgo/client-go" / unit / "_author"
    patch = ""
    for rel in rels:
        patch += udiff(out / "excised" / "tree" / rel, worktree / rel, rel)
        dst = out / "cheat" / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(worktree / rel, dst)
    (out / "cheat.patch").write_text(patch)
    gold = (out / "gold.patch").read_text()
    c, g = added_lines(patch), added_lines(gold)
    print(f"cheat +{c}  gold +{g}  ratio {c/g if g else 0:.2f}  (must be < 0.6)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
