#!/usr/bin/env python3
"""Move a scripts/ops module into the package and leave a shim at the old path.

Every scripts/ops/*.py used to be both a library (other scripts `import trial_ledger` after
putting scripts/ops on sys.path) and a command (`uv run python scripts/ops/trial_guard.py`).
Running shell loops, frozen copies and cron lines call those paths, so a move must keep
both working. For each module named on the command line this:

  1. git-mvs scripts/ops/<old>.py to src/openswe_traces/<new path>.py
  2. rewrites, in every module already in MOVES, bare imports of moved modules into
     package imports under the SAME local name, so no function body changes
  3. drops the sys.path.insert lines that only existed to make bare imports resolve
  4. turns the `if __name__ == "__main__":` block into `def cli():`
  5. writes a shim at the old path that registers the package module under the old name
     (one module object, so private names and mutable state are shared) and runs cli()

Usage: move_module.py <old-stem> [<old-stem> ...]   (run from the repo root)
"""
import pathlib
import re
import subprocess
import sys

# old scripts/ops stem -> new dotted module. Grows one wave at a time.
MOVES = {
    "trial_ledger": "openswe_traces.ladder.ledger",
    "trial_guard": "openswe_traces.ladder.guard",
    "solver_match": "openswe_traces.ladder.match",
    "escalate": "openswe_traces.ladder.escalate",
    "slots": "openswe_traces.ops.slots",
    "roots": "openswe_traces.ops.roots",
}

OPS = pathlib.Path("scripts/ops")
SRC = pathlib.Path("src")

SHIM = '''#!/usr/bin/env python3
"""Moved to {new}. This path keeps old commands and `import {old}` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from {pkg} import {leaf} as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
'''


def target(old):
    return SRC / (MOVES[old].replace(".", "/") + ".py")


def rewrite_imports(text):
    """Bare imports of moved modules become package imports under the same local name."""
    names = "|".join(sorted(MOVES, key=len, reverse=True))

    def plain(m):
        indent, old, alias = m.group(1), m.group(2), m.group(3)
        pkg, leaf = MOVES[old].rsplit(".", 1)
        return f"{indent}from {pkg} import {leaf} as {alias or old}"

    text = re.sub(rf"^(\s*)import ({names})(?: as (\w+))?(?=\s*(?:#.*)?$)", plain, text,
                  flags=re.M)
    text = re.sub(rf"^(\s*)from ({names}) import ",
                  lambda m: f"{m.group(1)}from {MOVES[m.group(2)]} import ", text, flags=re.M)
    return text


def drop_path_hacks(text):
    """sys.path.insert(...) lines that point at scripts/ops or src are no longer needed."""
    text = re.sub(r"^(import [\w, ]+); sys\.path\.insert\(0, .*\)\n", r"\1\n", text, flags=re.M)
    return re.sub(r"^sys\.path\.insert\(0, .*(?:__file__|REPO).*\)\n", "", text, flags=re.M)


def main_to_cli(text):
    m = re.search(r'^if __name__ == "__main__":\n', text, flags=re.M)
    if not m:
        return text
    return (text[:m.start()] + "def cli():\n" + text[m.end():].rstrip("\n")
            + '\n\n\nif __name__ == "__main__":\n    cli()\n')


def ensure_package(path):
    path.parent.mkdir(parents=True, exist_ok=True)
    for d in [path.parent, *path.parent.parents]:
        if d == SRC:
            break
        init = d / "__init__.py"
        if not init.exists():
            init.write_text("")
            subprocess.run(["git", "add", str(init)], check=True)


def move(old):
    src, dst = OPS / f"{old}.py", target(old)
    ensure_package(dst)
    subprocess.run(["git", "mv", str(src), str(dst)], check=True)
    dst.write_text(main_to_cli(drop_path_hacks(dst.read_text())))
    pkg, leaf = MOVES[old].rsplit(".", 1)
    src.write_text(SHIM.format(new=MOVES[old], old=old, pkg=pkg, leaf=leaf))
    subprocess.run(["git", "add", str(src), str(dst)], check=True)


def main(argv):
    unknown = [a for a in argv if a not in MOVES]
    if not argv or unknown:
        sys.exit(f"usage: move_module.py <stem>...; unknown: {unknown}; known: {list(MOVES)}")
    for old in argv:
        move(old)
    for old in MOVES:
        p = target(old)
        if p.exists():
            p.write_text(rewrite_imports(p.read_text()))
    print("moved:", ", ".join(f"{o} -> {MOVES[o]}" for o in argv))


if __name__ == "__main__":
    main(sys.argv[1:])
