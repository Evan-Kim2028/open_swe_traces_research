#!/usr/bin/env python3
"""Retire spent ``oswt-*`` authoring worktrees, salvaging what only exists there.

103 worktrees hold 153GB - more than the main checkout. Almost all of it is
regenerable: ``environment/src`` staging trees, harvested authored units, and the
git-tracked ``experiments/harbor_nex`` baseline that every worktree replicates.
But 88 of them also carry untracked one-off scripts and per-batch research notes
that were never copied back to main - 445 files, 7.9MB. Those are the reason this
is not ``rm -rf``.

Order of operations, and why each step is where it is:

1. SALVAGE first. Every untracked file outside ``experiments/`` and ``outputs/``
   is copied to ``analytics/worktree_salvage/<worktree>/`` unless main already
   holds a byte-identical copy. Artifact directories are excluded because
   ``harvest.py`` owns them and has already been run.
2. IDLE check. A worktree with a file written inside ``--idle-hours``, or that is
   some process's working directory, is left alone. Authoring sessions run for
   hours and write sporadically; mtime alone is not enough, so both must hold.
3. HARVEST check. ``harvest.py`` must report 0 recoverable before anything is
   removed - the ``authored_batch*`` glob bug hid finished units three separate
   times, so a worktree that looks spent may not be.
4. REMOVE the checkout, KEEP the branch. ``git worktree remove`` deletes the
   working directory; the branch ref and every commit stay in the main repo's
   object store. Recovery is ``git worktree add <path> <branch>``, so the only
   thing this destroys permanently is untracked files - which step 1 just saved.

Usage::

    worktree_gc.py                     # dry run
    worktree_gc.py --salvage-only      # do step 1, stop
    worktree_gc.py --apply
    worktree_gc.py --apply --keep 10   # retain the 10 most recent worktrees
"""

from __future__ import annotations

import argparse
import hashlib
import pathlib
import shutil
import subprocess
import sys
import time

REPO = pathlib.Path(__file__).resolve().parents[2]
SALVAGE = REPO / "analytics" / "worktree_salvage"
# harvest.py owns these; they are artifacts, not one-off source.
ARTIFACT_PREFIXES = ("experiments/", "outputs/", ".guard_stamps")
IDLE_HOURS_DEFAULT = 6.0


def worktrees() -> list[pathlib.Path]:
    return sorted((p for p in REPO.parent.glob("oswt-*") if p.is_dir()),
                  key=lambda p: p.stat().st_mtime)


def git(wt: pathlib.Path, *args: str) -> str:
    return subprocess.run(["git", "-C", str(wt), *args],
                          capture_output=True, text=True).stdout


def untracked(wt: pathlib.Path) -> list[str]:
    out = git(wt, "status", "--porcelain")
    rels = [ln[3:] for ln in out.splitlines() if ln.startswith("?? ")]
    return [r for r in rels if not r.startswith(ARTIFACT_PREFIXES)]


def sha(p: pathlib.Path) -> str:
    h = hashlib.sha1()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def salvage(wt: pathlib.Path, apply: bool) -> int:
    """Copy untracked one-off files to main. Returns how many were new."""
    new = 0
    for rel in untracked(wt):
        src = wt / rel
        for f in ([src] if src.is_file() else
                  [q for q in src.rglob("*") if q.is_file()] if src.is_dir() else []):
            r = f.relative_to(wt)
            # already in main, byte-identical? then the worktree adds nothing.
            twin = REPO / r
            try:
                if twin.is_file() and sha(twin) == sha(f):
                    continue
            except OSError:
                pass
            dest = SALVAGE / wt.name / r
            if dest.is_file():
                continue
            new += 1
            if apply:
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(f, dest)
    return new


def busy(wt: pathlib.Path, idle_hours: float) -> str | None:
    cutoff = time.time() - idle_hours * 3600
    r = subprocess.run(
        ["find", str(wt), "-xdev", "-type", "f", "-not", "-path", "*/.git/*",
         "-newermt", f"@{int(cutoff)}"],
        capture_output=True, text=True, timeout=600).stdout
    if r.strip():
        return f"written within {idle_hours}h"
    for proc in pathlib.Path("/proc").iterdir():
        if not proc.name.isdigit():
            continue
        try:
            if (proc / "cwd").resolve() == wt or str(wt) in str((proc / "cwd").resolve()):
                return f"pid {proc.name} is cwd'd here"
        except OSError:
            continue
    return None


def harvest_clean() -> bool:
    out = subprocess.run(
        ["uv", "run", "python", "scripts/ops/harvest.py"],
        cwd=REPO, capture_output=True, text=True, timeout=1800).stdout
    return "0 recoverable" in out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--apply", action="store_true")
    ap.add_argument("--salvage-only", action="store_true")
    ap.add_argument("--keep", type=int, default=0,
                    help="retain the N most recently modified worktrees")
    ap.add_argument("--idle-hours", type=float, default=IDLE_HOURS_DEFAULT)
    args = ap.parse_args()

    wts = worktrees()
    if args.keep:
        wts = wts[:-args.keep] if args.keep < len(wts) else []
    print(f"candidates: {len(wts)} worktree(s)")

    total_new = sum(salvage(w, args.apply or args.salvage_only) for w in wts)
    print(f"salvage: {total_new} file(s) exist only in a worktree"
          f"{' — copied to ' + str(SALVAGE.relative_to(REPO)) if (args.apply or args.salvage_only) else ''}")
    if args.salvage_only:
        return 0

    if args.apply and not harvest_clean():
        print("REFUSING: harvest.py reports recoverable work — run it with --apply first")
        return 1

    removed = skipped = 0
    for w in wts:
        why = busy(w, args.idle_hours)
        if why:
            print(f"  keep   {w.name}  ({why})")
            skipped += 1
            continue
        removed += 1
        if args.apply:
            r = subprocess.run(["git", "-C", str(REPO), "worktree", "remove",
                                "--force", str(w)], capture_output=True, text=True)
            if r.returncode != 0:
                shutil.rmtree(w, ignore_errors=True)
                subprocess.run(["git", "-C", str(REPO), "worktree", "prune"],
                               capture_output=True)
    verb = "removed" if args.apply else "would remove"
    print(f"{verb}: {removed}   kept busy: {skipped}   (branches are preserved)")
    if not args.apply:
        print("\n(dry run — rerun with --apply)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
