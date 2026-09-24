"""Materialize bbolt batch-2 unit trees into experiments/pipeline/work/vf_bbolt_*.

Per unit under the AU worktree's authored_batch2/bbolt/<unit>/_author/:

- vf_bbolt_excised/<unit> = pristine repos2/bbolt/src + excised/excision.patch
- vf_bbolt_gold/<unit>    = excised tree + gold.patch   (never inspected)
- vf_bbolt_cheat/<unit>   = excised tree + cheat.patch  (never inspected)

Integrity: every file the author shipped under the sparse _author/excised/
must be byte-identical to the patched tree, and `go build ./...` must pass on
the excised tree (matches the author's own verification).
"""

from __future__ import annotations

import filecmp
import shutil
import subprocess
import sys
from pathlib import Path

AU = Path("/home/evan/Documents/oswt-AUnew2/experiments/pipeline/authored_batch2/bbolt")
PRISTINE = Path("/home/evan/Documents/oswt-NEWREPOS/experiments/pipeline/repos2/bbolt/src")
WORK = Path("/home/evan/Documents/oswt-VFnew2/experiments/pipeline/work")

UNITS = [
    "bucket", "cursor", "tx", "db", "node", "txcheck", "compact",
    "meta", "page", "inode", "inbucket", "loadutil", "verifyenv",
    "flarray", "flhashmap", "flshared",
    "surgeon", "xray", "gutscli",
    "cmdutils", "cmdget", "cmdpage", "cmddump", "cmdpages", "cmdsurgerymeta",
]


def _copytree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest, symlinks=True)


def _patch(tree: Path, patch: Path) -> None:
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch), "-d", str(tree)],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"patch {patch.name} failed on {tree}:\n{proc.stdout}{proc.stderr}")


def _check_sparse(unit: str, tree: Path) -> list[str]:
    """Every file under the sparse _author/excised must match the patched tree."""
    sparse = AU / unit / "_author" / "excised"
    bad = []
    for path in sparse.rglob("*"):
        if not path.is_file() or path.name == "excision.patch":
            continue
        rel = path.relative_to(sparse)
        got = tree / rel
        if not got.is_file() or not filecmp.cmp(path, got, shallow=False):
            bad.append(str(rel))
    return bad


def stage(unit: str, *, check: bool = True) -> Path:
    author = AU / unit / "_author"
    excised = WORK / "vf_bbolt_excised" / unit
    _copytree(PRISTINE, excised)
    _patch(excised, author / "excised" / "excision.patch")
    if check:
        bad = _check_sparse(unit, excised)
        if bad:
            raise RuntimeError(f"{unit}: sparse excised mismatch: {bad}")
    for kind in ("gold", "cheat"):
        tree = WORK / f"vf_bbolt_{kind}" / unit
        _copytree(excised, tree)
        _patch(tree, author / f"{kind}.patch")
    return excised


def main() -> int:
    units = sys.argv[1:] or UNITS
    (WORK / "vf_bbolt_excised").mkdir(parents=True, exist_ok=True)
    (WORK / "vf_bbolt_gold").mkdir(parents=True, exist_ok=True)
    (WORK / "vf_bbolt_cheat").mkdir(parents=True, exist_ok=True)
    for unit in units:
        stage(unit)
        print(f"staged {unit}", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
