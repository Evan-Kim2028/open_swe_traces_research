#!/usr/bin/env python3
"""Author one excision unit: spec -> stubs -> patches -> build check.

Usage:
  author_excise.py UNIT spec.json [--delete rel_test.go ...]

Reads PRISTINE (env or --pristine, default /tmp/cgobase/app), writes into
  experiments/pipeline/authored_au5clientgo/client-go/UNIT/_author/
    excised/excision.patch   real -> stub (+ test deletions)
    excised/excise_spec.json
    excised/tree/<rel>       stubbed source copies
    gold.patch               stub -> real (source files only)
and verifies the excised tree compiles.
"""
from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXCISE = ROOT / "scripts" / "excise_funcs.py"


def udiff(old: Path | None, new: Path | None, rel: str) -> str:
    """git-style unified diff for one file; None means /dev/null side."""
    src = ["diff", "-u"]
    if old is None:
        src += ["--label", f"a/{rel}", "--label", f"b/{rel}", os.devnull, str(new)]
        header = f"diff --git a/{rel} b/{rel}\nnew file mode 100644\n"
    elif new is None:
        src += ["--label", f"a/{rel}", "--label", os.devnull, str(old), os.devnull]
        header = f"diff --git a/{rel} b/{rel}\ndeleted file mode 100644\n"
    else:
        src += ["--label", f"a/{rel}", "--label", f"b/{rel}", str(old), str(new)]
        header = f"diff --git a/{rel} b/{rel}\n"
    r = subprocess.run(src, capture_output=True, text=True)
    if r.returncode not in (0, 1):
        raise RuntimeError(r.stderr)
    if r.returncode == 0:
        return ""
    return header + r.stdout


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("unit")
    ap.add_argument("spec", type=Path)
    ap.add_argument("--pristine", type=Path, default=Path("/tmp/cgobase/app"))
    ap.add_argument("--delete", nargs="*", default=[], help="test files to delete in excision")
    ap.add_argument("--outroot", type=Path,
                    default=ROOT / "experiments/pipeline/authored_au5clientgo/client-go")
    ap.add_argument("--finish", type=Path, default=None,
                    help="skip excision; generate patches from this pre-edited worktree")
    a = ap.parse_args(argv)

    spec = json.loads(a.spec.read_text())
    files = [rel for unit in spec.values() for rel in unit]
    pristine = a.pristine
    if a.finish:
        worktree = a.finish
        work = worktree.parent
    else:
        work = Path(tempfile.mkdtemp(prefix=f"excise_{a.unit}_"))
        shutil.copytree(pristine, work / "app", symlinks=True)
        worktree = work / "app"

        # excise
        r = subprocess.run(["uv", "run", "python", str(EXCISE), str(worktree), str(a.spec)],
                           capture_output=True, text=True, cwd=ROOT)
        sys.stdout.write(r.stdout)
        if r.returncode != 0:
            sys.stderr.write(r.stderr)
            return 1
        if "WARN" in r.stdout or "GOFMT-BAD" in r.stdout:
            print("!! exciser warnings above — fix spec before continuing")

        for rel in a.delete:
            (worktree / rel).unlink()

    out = a.outroot / a.unit / "_author"
    (out / "excised").mkdir(parents=True, exist_ok=True)
    (out / "excised" / "excise_spec.json").write_text(a.spec.read_text())

    # files that differ from pristine: spec files + deletions + any hand-edited file
    changed = []
    seen = set()
    for rel in files + list(a.delete):
        if rel not in seen:
            changed.append(rel); seen.add(rel)
    if a.finish:
        for p in sorted(worktree.rglob("*.go")):
            rel = str(p.relative_to(worktree))
            if rel in seen:
                continue
            ref = pristine / rel
            if not ref.is_file() or ref.read_bytes() != p.read_bytes():
                changed.append(rel); seen.add(rel)

    excision = ""
    for rel in changed:
        if not (worktree / rel).exists():
            excision += udiff(pristine / rel, None, rel)
        else:
            excision += udiff(pristine / rel, worktree / rel, rel)
    (out / "excised" / "excision.patch").write_text(excision)

    gold = ""
    for rel in files:
        gold += udiff(worktree / rel, pristine / rel, rel)
    (out / "gold.patch").write_text(gold)

    for rel in changed:
        if not (worktree / rel).exists():
            continue
        dst = out / "excised" / "tree" / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(worktree / rel, dst)

    # verify compile
    b = subprocess.run(["go", "build", "./..."], cwd=worktree, capture_output=True, text=True)
    print("BUILD(excised):", "OK" if b.returncode == 0 else "FAIL")
    if b.returncode != 0:
        sys.stderr.write(b.stderr[-4000:])
    pkgs = sorted({"/".join(rel.split("/")[:-1]) for rel in files} |
                  {"/".join(rel.split("/")[:-1]) for rel in a.delete})
    for pkg in pkgs:
        t = subprocess.run(["go", "test", "-count=1", "-run", "^$", "./" + pkg + "/"],
                           cwd=worktree, capture_output=True, text=True)
        status = "OK" if t.returncode == 0 else "FAIL"
        print(f"TESTCOMPILE(excised) {pkg}: {status}")
        if t.returncode != 0:
            sys.stderr.write(t.stderr[-2500:] + "\n")
    print("WORKDIR", worktree)
    return 0 if b.returncode == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
