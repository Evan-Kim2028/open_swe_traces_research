#!/usr/bin/env python3
"""Reclaim regenerable disk: staged ``environment/src`` trees of SETTLED units.

Why this exists
---------------
A staged Harbor unit is ~99.9% ``environment/src`` - a full excised copy of the
base repo (kops is 279MB, helm 100MB, go-github 11MB). The part that took tokens
to make is the other 0.1%: ``instruction.md``, ``tests/``, ``patches/``,
``affordance.json``, ``validation.json``. 11,115 of these trees exist across the
main checkout and 103 worktrees, holding 142GB - more than half of everything
this project has on disk.

``environment/src`` is DERIVED, never authored: ``stage_units.py`` builds it by
applying ``_author/excised/excision.patch`` forward onto ``ladder-base:<repo>``'s
``/app``. So once a unit's rung has a settled verdict, its staged tree is pure
cache. Deleting it loses nothing that a re-stage cannot rebuild bit-for-bit.

What counts as safe
-------------------
Four independent guards, all of which must pass. Each one is here because the
cheap version of this script would have been wrong:

1. SETTLED - the ledger holds a non-errored verdict for this exact ``(base,
   rung)``. A unit staged but never trialled is the ~90-deep queue behind the
   Devin cap; deleting it would silently un-stage pending work.
2. NOT LIVE - the unit is not named by a running container. Harbor copies the
   tree in at job start, but a mid-flight re-read would fault.
3. SWEEP QUIET - no job directory for the unit's sweep has been touched in
   ``--quiet-hours``. Covers the window between "container exited" and "verdict
   written", where guard 1 says pending and guard 2 says idle.
4. NEVER the protected paths - ``experiments/pipeline/repos/*/src`` are the
   pristine upstream checkouts that every excision is cut from, and they are not
   regenerable from anything on this disk.

Each reclaimed unit gets a ``.reclaimed.json`` manifest that makes the removal
exactly reversible. ``regen_env_src.sh`` alone is not enough: it rebuilds the
excised tree by reverse-applying ``gold.patch`` to the base image's pristine
``/app``, but 18 of 20 batch-3 kops excisions DELETE in-tree ``*_test.go`` files
that gold never restores, so gold-reverse leaves those files present and the
rebuilt tree is not the tree the verifier proved the suite against. The manifest
closes that gap by recording, per unit, which pristine paths the excision removed
- so restore is ``gold-reverse, then delete these``. A unit whose excision ADDED
a file is kept untouched, because a name list cannot carry that file's content.

``restore_env_src.py`` is the inverse operation.

Usage::

    reclaim_disk.py                 # dry run: what would go, and how much
    reclaim_disk.py --apply
    reclaim_disk.py --roots all     # include the oswt-* worktrees
"""

from __future__ import annotations

import argparse
import collections
import json
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))

import trial_ledger as TL  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
UNIT_RE = re.compile(r"^(.*)-L(\d+)$")
# environment/src under these is the pristine upstream tree, not a staged excision.
PROTECTED = ("experiments/pipeline/repos/",)
QUIET_HOURS_DEFAULT = 3.0


PRISTINE_INDEX = REPO / "outputs" / "supervisor" / "pristine_index"


def base_image(unit: pathlib.Path) -> str | None:
    """The ``ladder-base:<repo>`` tag a staged unit is cut from, per its Dockerfile."""
    df = unit / "environment" / "Dockerfile"
    try:
        m = re.search(r"ladder-base:([A-Za-z0-9._-]+)", df.read_text())
    except OSError:
        return None
    return m.group(1) if m else None


def pristine_files(repo: str) -> set[str] | None:
    """Cached file list of ``/app`` inside ``ladder-base:<repo>``.

    Built once per repo and kept on disk: the index is what makes a deletion
    reversible, so it must outlive the process that wrote it.
    """
    cache = PRISTINE_INDEX / f"{repo}.txt"
    if not cache.is_file() or cache.stat().st_size == 0:
        PRISTINE_INDEX.mkdir(parents=True, exist_ok=True)
        try:
            out = subprocess.run(
                ["docker", "run", "--rm", "--entrypoint", "/bin/sh",
                 f"ladder-base:{repo}", "-c",
                 r'cd /app && find . -type f | sed "s|^\./||" | LC_ALL=C sort'],
                capture_output=True, text=True, timeout=600,
            ).stdout
        except Exception:
            return None
        if not out.strip():
            return None
        cache.write_text(out)
    return set(cache.read_text().split("\n")) - {""}


def unit_files(env_src: pathlib.Path) -> set[str]:
    out = subprocess.run(["find", ".", "-type", "f"], cwd=env_src,
                         capture_output=True, text=True, timeout=600).stdout
    return {line[2:] for line in out.split("\n") if line.startswith("./")}


def live_units() -> set[str]:
    """Unit names (lowercased, e.g. ``vfsimpl-l0``) of every running trial container."""
    try:
        out = subprocess.run(
            ["docker", "ps", "--format", "{{.Names}}"],
            capture_output=True, text=True, timeout=30,
        ).stdout
    except Exception:
        # No docker answer means no evidence of safety. Treat everything as live.
        return {"*"}
    return {n.split("__", 1)[0] for n in out.split() if "__env-" in n}


def busy_sweeps(root: pathlib.Path, quiet_hours: float) -> set[str]:
    """Sweeps whose job dirs were touched recently - verdicts may still be landing."""
    jobs = root / "experiments" / "dose_response" / "jobs"
    if not jobs.is_dir():
        return set()
    cutoff = time.time() - quiet_hours * 3600
    busy = set()
    for d in jobs.iterdir():
        if not d.is_dir():
            continue
        newest = d.stat().st_mtime
        for child in list(d.iterdir())[:400]:
            newest = max(newest, child.stat().st_mtime)
            if newest >= cutoff:
                break
        if newest >= cutoff:
            # job dirs are "<sweep>_r<n>[_<stamp>]"; strip the run suffix
            busy.add(re.sub(r"_r\d+(_\d+)?$", "", d.name))
    return busy


def staged_trees(root: pathlib.Path):
    """Yield ``(unit_dir, env_src)`` for every staged unit under a checkout root."""
    for base in ("dose_response", "pipeline", "harbor_nex"):
        top = root / "experiments" / base
        if not top.is_dir():
            continue
        # two layouts: <cohort>/<unit>/ and <cohort>/<repo>/<unit>/. Missing the second
        # left 12GB in tasks_composerver untouched while the report said "done".
        for env_src in [*top.glob("*/*/environment/src"),
                        *top.glob("*/*/*/environment/src")]:
            rel = env_src.relative_to(root).as_posix()
            if any(p in rel for p in PROTECTED):
                continue
            if env_src.is_dir():
                yield env_src.parents[1], env_src


def tree_mb(p: pathlib.Path) -> int:
    try:
        out = subprocess.run(["du", "-sm", str(p)], capture_output=True,
                             text=True, timeout=120).stdout
        return int(out.split()[0])
    except Exception:
        return 0


def classify(root: pathlib.Path, ledger, live, busy):
    """-> list of (unit_dir, env_src, verdict) where verdict is 'reclaim' or a reason."""
    rows = []
    for unit, env_src in staged_trees(root):
        m = UNIT_RE.match(unit.name)
        base, rung = (m.group(1), m.group(2)) if m else (unit.name, None)
        sweep = unit.parent.name
        if rung is None or not ledger.get(base, {}).get(rung):
            rows.append((unit, env_src, "pending: no verdict for this rung"))
        elif unit.name.lower() in live or "*" in live:
            rows.append((unit, env_src, "live: container running"))
        elif sweep in busy:
            rows.append((unit, env_src, "busy: sweep job dir touched recently"))
        else:
            rows.append((unit, env_src, "reclaim"))
    return rows


_PRISTINE_TREES: dict[str, pathlib.Path | None] = {}


def pristine_tree(repo: str) -> pathlib.Path | None:
    """One extracted copy of ``ladder-base:<repo>``'s /app, cached for the whole run."""
    if repo in _PRISTINE_TREES:
        return _PRISTINE_TREES[repo]
    tmp = pathlib.Path(tempfile.mkdtemp(prefix=f"pristine-{repo}-"))
    cid = subprocess.run(["docker", "create", f"ladder-base:{repo}"],
                         capture_output=True, text=True).stdout.strip()
    got = None
    if cid:
        try:
            if subprocess.run(["docker", "cp", f"{cid}:/app/.", str(tmp)],
                              capture_output=True).returncode == 0:
                got = tmp
        finally:
            subprocess.run(["docker", "rm", cid], capture_output=True)
    if got is None:
        shutil.rmtree(tmp, ignore_errors=True)
    _PRISTINE_TREES[repo] = got
    return got


def reversible(unit: pathlib.Path, repo: str) -> bool:
    """Does this unit's gold.patch actually reverse onto pristine?

    The manifest promised a rebuild; it did not prove one. 86 of the first 574 units
    reclaimed cannot be rebuilt, because a contract repair regenerated gold.patch after
    the tree was staged, so gold no longer describes the distance from that tree to
    pristine. ``git apply --check`` writes nothing and answers in milliseconds - there is
    no excuse for deleting a tree without asking first.
    """
    gold = unit / "patches" / "gold.patch"
    tree = pristine_tree(repo)
    if not gold.is_file() or tree is None:
        return False
    return subprocess.run(
        ["git", "apply", "-R", "--check", "--whitespace=nowarn", str(gold.resolve())],
        cwd=tree, capture_output=True).returncode == 0


def manifest(unit: pathlib.Path, env_src: pathlib.Path):
    """-> (manifest dict, reason-to-keep or None).

    The manifest is the whole safety story: without the ``excision_deleted`` list a
    restore silently leaves test files that the excision had removed.
    """
    repo = base_image(unit)
    if not repo:
        return None, "unmappable: no ladder-base in Dockerfile"
    pristine = pristine_files(repo)
    if not pristine:
        return None, f"unindexable: no pristine index for {repo}"
    have = unit_files(env_src)
    added = sorted(have - pristine)
    if added:
        return None, "irreversible: excision added files"
    if not (unit / "patches" / "gold.patch").is_file():
        return None, "irreversible: no gold.patch to reverse"
    if not reversible(unit, repo):
        return None, "irreversible: gold.patch does not reverse onto pristine"
    return {
        "removed": "environment/src",
        "at": time.strftime("%Y-%m-%dT%H:%M:%S"),
        "base_image": f"ladder-base:{repo}",
        "file_count": len(have),
        "excision_deleted": sorted(pristine - have),
        "why": "settled verdict in ledger; the tree is derived, not authored",
        "restore": f"uv run python scripts/ops/restore_env_src.py {unit}",
    }, None


def stamp(unit: pathlib.Path, man: dict, mb: int) -> None:
    man["megabytes"] = mb
    (unit / ".reclaimed.json").write_text(json.dumps(man, indent=2) + "\n")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--apply", action="store_true")
    ap.add_argument("--roots", choices=("main", "all"), default="main")
    ap.add_argument("--quiet-hours", type=float, default=QUIET_HOURS_DEFAULT)
    args = ap.parse_args()

    roots = [REPO]
    if args.roots == "all":
        roots += sorted(p for p in REPO.parent.glob("oswt-*") if p.is_dir())

    ledger = TL.ledger()
    live = live_units()
    busy = busy_sweeps(REPO, args.quiet_hours)
    print(f"ledger bases={len(ledger)}  live={len(live)}  busy sweeps={len(busy)}")

    reclaim, kept = [], collections.Counter()
    for root in roots:
        for unit, env_src, verdict in classify(root, ledger, live, busy):
            if verdict == "reclaim":
                reclaim.append((unit, env_src))
            else:
                kept[verdict.split(":")[0]] += 1

    freed = 0
    done = 0
    for unit, env_src in reclaim:
        man, keep = manifest(unit, env_src)
        if keep:
            kept[keep.split(":")[0]] += 1
            continue
        mb = tree_mb(env_src)
        freed += mb
        done += 1
        if args.apply:
            shutil.rmtree(env_src, ignore_errors=True)
            stamp(unit, man, mb)
    reclaim = reclaim[:done]

    for tmp in _PRISTINE_TREES.values():
        if tmp is not None:
            shutil.rmtree(tmp, ignore_errors=True)

    verb = "freed" if args.apply else "would free"
    print(f"reclaimable units: {len(reclaim)}   {verb}: {freed/1024:.1f} GB")
    for reason, n in kept.most_common():
        print(f"  kept {n:5d}  {reason}")
    if not args.apply and reclaim:
        print("\n(dry run — rerun with --apply)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
