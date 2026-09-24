#!/usr/bin/env python3
"""Diff an excised-file subset against the hand-edited cheat copies -> cheat.patch.

usage: cheat_diff.py <unit_dir>          # unit dir containing _author/excised/tree
                                       # and _author/cheat/  (same rel paths, edited)
Writes <unit_dir>/_author/cheat.patch.
"""
from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path


def diff_dirs(a: Path, b: Path) -> str:
    with tempfile.TemporaryDirectory() as td:
        tda, tdb = Path(td) / "a", Path(td) / "b"
        subprocess.run(["cp", "-al", str(a) + "/.", str(tda)], check=True)
        subprocess.run(["cp", "-al", str(b) + "/.", str(tdb)], check=True)
        p = subprocess.run(["git", "diff", "--no-index", "a", "b"],
                           cwd=td, capture_output=True, text=True)

        def norm(line: str) -> str:
            parts = line.split(" ")
            for i, tok in enumerate(parts):
                t = tok.rstrip("\n")
                for pre in ("a/", "b/"):
                    if t.startswith(pre):
                        rest = t[len(pre):]
                        if rest.startswith(("a/", "b/")):
                            parts[i] = pre + rest[2:] + ("\n" if tok.endswith("\n") else "")
                        break
            return " ".join(parts)

        out = []
        for line in p.stdout.splitlines(keepends=True):
            if line.startswith(("diff --git ", "rename from ", "rename to ",
                                "copy from ", "copy to ", "--- ", "+++ ")):
                line = norm(line)
            out.append(line)
        return "".join(out)


def main() -> int:
    unit = Path(sys.argv[1]).resolve()
    author = unit / "_author"
    excised, cheat = author / "excised" / "tree", author / "cheat"
    patch = diff_dirs(excised, cheat)
    (author / "cheat.patch").write_text(patch)
    added = sum(1 for l in patch.splitlines() if l.startswith("+") and not l.startswith("+++"))
    gold = author / "gold.patch"
    gold_added = sum(1 for l in gold.read_text().splitlines()
                     if l.startswith("+") and not l.startswith("+++")) if gold.is_file() else 0
    ratio = added / gold_added if gold_added else 0
    print(f"cheat +{added} lines, gold +{gold_added}, ratio {ratio:.2f} "
          f"({'OK' if ratio < 0.6 else 'TOO BIG — implementation, not cheat'})")
    return 0 if ratio < 0.6 else 1


if __name__ == "__main__":
    sys.exit(main())
