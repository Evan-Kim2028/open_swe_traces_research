#!/usr/bin/env python3
"""Rebuild the ``environment/src`` trees that ``reclaim_disk.py`` removed.

The recipe, in order, and every step matters:

1. Copy ``/app`` out of ``ladder-base:<repo>`` - the PRISTINE tree (bug absent).
2. Reverse-apply ``patches/gold.patch`` - gold turns excised into pristine, so
   its reverse turns pristine back into excised (bug present).
3. Delete the paths in the manifest's ``excision_deleted``. This step is why the
   manifest exists: an excision that removed in-tree ``*_test.go`` files leaves
   gold.patch with nothing to say about them, so after step 2 they are still
   there and the tree would not match what the verifier proved the suite against.
4. Verify the rebuilt file count against the manifest before declaring success.

Usage::

    restore_env_src.py <unit-dir> [<unit-dir> ...]
    restore_env_src.py --sweep experiments/dose_response/sweep_kops3
"""

from __future__ import annotations

import argparse
import json
import pathlib
import shutil
import subprocess
import sys
import tempfile

REPO = pathlib.Path(__file__).resolve().parents[2]


def pristine_copy(image: str, dest: pathlib.Path) -> bool:
    """Extract the image's /app once; callers reuse it across a whole sweep."""
    cid = subprocess.run(["docker", "create", image], capture_output=True,
                         text=True).stdout.strip()
    if not cid:
        return False
    try:
        r = subprocess.run(["docker", "cp", f"{cid}:/app/.", str(dest)],
                           capture_output=True, text=True)
        return r.returncode == 0
    finally:
        subprocess.run(["docker", "rm", cid], capture_output=True)


def restore(unit: pathlib.Path, cache: dict[str, pathlib.Path]) -> str:
    stamp = unit / ".reclaimed.json"
    if not stamp.is_file():
        return "skip: not reclaimed"
    env_src = unit / "environment" / "src"
    if env_src.is_dir() and any(env_src.iterdir()):
        return "skip: already present"
    man = json.loads(stamp.read_text())
    image = man["base_image"]

    if image not in cache:
        tmp = pathlib.Path(tempfile.mkdtemp(prefix="pristine-"))
        if not pristine_copy(image, tmp):
            return f"FAIL: cannot extract {image}"
        cache[image] = tmp

    env_src.parent.mkdir(parents=True, exist_ok=True)
    shutil.rmtree(env_src, ignore_errors=True)
    shutil.copytree(cache[image], env_src, symlinks=True)

    gold = (unit / "patches" / "gold.patch").resolve()
    r = subprocess.run(["patch", "-p1", "-R", "--silent", "-i", str(gold)],
                       cwd=env_src, capture_output=True, text=True)
    if r.returncode != 0:
        return f"FAIL: gold reverse-apply ({r.stderr.strip()[:80]})"

    for rel in man.get("excision_deleted", []):
        (env_src / rel).unlink(missing_ok=True)

    have = sum(1 for _ in env_src.rglob("*") if _.is_file())
    want = man.get("file_count")
    if want is not None and have != want:
        return f"FAIL: {have} files, manifest says {want}"
    stamp.unlink()
    return f"ok ({have} files)"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("units", nargs="*")
    ap.add_argument("--sweep", help="restore every reclaimed unit in a sweep dir")
    args = ap.parse_args()

    units = [pathlib.Path(u) for u in args.units]
    if args.sweep:
        units += sorted(p.parent for p in pathlib.Path(args.sweep).glob("*/.reclaimed.json"))
    if not units:
        ap.error("nothing to restore")

    cache: dict[str, pathlib.Path] = {}
    bad = 0
    try:
        for u in units:
            res = restore(u, cache)
            bad += res.startswith("FAIL")
            print(f"{res:40s}  {u}")
    finally:
        for tmp in cache.values():
            shutil.rmtree(tmp, ignore_errors=True)
    return 1 if bad else 0


if __name__ == "__main__":
    raise SystemExit(main())
