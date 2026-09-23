#!/usr/bin/env python3
"""Audit a move: every changed line must be one move_module.py is allowed to make.

For each entry in move_module.MOVES whose original is still at <rev>:scripts/ops/<old>.py,
diff it against the package file and print any added or removed line that is not an
import rewrite, a dropped sys.path hack, the REPO substitution, or the main -> cli wrap.
Silence means the move changed no logic.

Usage: moved_diff.py [<rev>]   (default HEAD; run from the repo root)
"""
import difflib
import re
import subprocess
import sys

sys.path.insert(0, "scripts/dev")
from move_module import MOVES, target  # noqa: E402

ALLOWED = [
    r"^\s*import \w+( as \w+)?\s*(#.*)?$",
    r"^\s*from [\w.]+ import .*$",
    r"^\s*import [\w, ]+(; sys\.path\.insert\(0, .*\))?$",
    r"^[ \t]*_?sys\.path\.insert\(0, .*\)$",
    r'^if __name__ == "__main__":$',
    r"^def cli\(\):$",
    r"^    cli\(\)$",
    r"^\s*$",
]


def allowed(line):
    return any(re.match(p, line) for p in ALLOWED)


def repo_line(a, b):
    """A line whose only change is a __file__ root lookup becoming _REPO."""
    norm = re.sub(r"(pathlib\.)?Path\(__file__\)\.resolve\(\)\.parents\[2\]", "_REPO", a)
    norm = re.sub(r"os\.path\.dirname\(os\.path\.dirname\(os\.path\.dirname\(\s*"
                  r"os\.path\.abspath\(__file__\)\)\)\)", "str(_REPO)", norm)
    return norm == b


def main(rev="HEAD"):
    bad = 0
    for old in MOVES:
        try:
            before = subprocess.run(["git", "show", f"{rev}:scripts/ops/{old}.py"],
                                    capture_output=True, text=True, check=True).stdout
        except subprocess.CalledProcessError:
            continue
        if "Moved to openswe_traces" in before[:200]:
            continue
        after = target(old).read_text()
        # Collapse multi-line __file__ lookups so they compare line for line.
        before = re.sub(r"\(\n\s+os\.path\.abspath\(__file__\)", "(os.path.abspath(__file__)",
                        before)
        after = re.sub(r"\(\n\s+str\(_REPO\)", "(str(_REPO)", after)
        rem, add = [], []
        for d in difflib.ndiff(before.splitlines(), after.splitlines()):
            if d.startswith("- "):
                rem.append(d[2:])
            elif d.startswith("+ "):
                add.append(d[2:])
        rem = [line for line in rem if not allowed(line)]
        add = [line for line in add if not allowed(line)]
        for r in list(rem):
            for a in add:
                if repo_line(r, a):
                    rem.remove(r)
                    add.remove(a)
                    break
        if rem or add:
            bad += 1
            print(f"== {old}")
            for line in rem:
                print(f"  - {line}")
            for line in add:
                print(f"  + {line}")
    print(f"{bad} module(s) with changes outside the codemod's rules")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(*sys.argv[1:]))
