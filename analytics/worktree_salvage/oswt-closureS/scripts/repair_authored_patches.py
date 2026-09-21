"""Repair authored excision/gold/cheat patches that cannot apply to the base tree.

Route for context drift that is not naming: hunks whose old range overruns EOF
by exactly a trailing blank line (the authored tree's files ended ``}\\n\\n``,
the on-disk base tree ends ``}\\n``). ``trim_trailing_blank_overruns`` drops the
overrunning context line; the result is identical to diff(base_tree, excised).

Per unit: fixes excision against the base tree, builds the excised tree from the
fixed excision, then fixes gold/cheat against that excised tree (trying the
patch as-is first). Fixed gold/cheat are propagated to every packaged level dir
under tasks/ and tasks_composerver/ so the preflight gate applies the same text
the authored _author/ holds.

Verifies every fix by re-applying with patch(1) ``--fuzz=0``.
"""

from __future__ import annotations

import argparse
import shutil
import subprocess
import tempfile
from pathlib import Path

from openswe_traces.pipeline.config import load_config
from openswe_traces.pipeline.materialize import (
    base_tree_for,
    trim_trailing_blank_overruns,
)


def _apply_fuzz0(tree: Path, patch_text: str) -> tuple[int, str]:
    with tempfile.NamedTemporaryFile("w", suffix=".patch", delete=False) as tmp:
        tmp.write(patch_text)
        tmpname = tmp.name
    try:
        proc = subprocess.run(
            ["patch", "-p1", "--forward", "--batch", "--fuzz=0", "-i", tmpname],
            cwd=tree,
            capture_output=True,
            text=True,
            check=False,
        )
        return proc.returncode, (proc.stderr or proc.stdout)[-600:]
    finally:
        Path(tmpname).unlink(missing_ok=True)


def _fix(patch_text: str, tree: Path, label: str, dry: bool) -> tuple[str, list[str]]:
    fixed, notes = trim_trailing_blank_overruns(patch_text, tree)
    if fixed == patch_text:
        return fixed, notes
    for note in notes:
        print(f"  trim {label}: {note}")
    if dry:
        return fixed, notes
    # verify on a scratch copy — never mutate the reference tree
    scratch = Path(tempfile.mkdtemp(prefix=f"verify-{label}-"))
    try:
        shutil.copytree(tree, scratch / "t")
        rc, err = _apply_fuzz0(scratch / "t", fixed)
    finally:
        shutil.rmtree(scratch, ignore_errors=True)
    if rc != 0:
        raise RuntimeError(f"fixed {label} still fails: {err}")
    return fixed, notes


def repair_unit(
    cfg,
    base_tree: Path,
    repo: str,
    unit: str,
    *,
    dry: bool,
) -> list[str]:
    author = cfg.authored_dir / repo / unit / "_author"
    rows: list[str] = []
    exc_patch = author / "excised" / "excision.patch"
    gold_patch = author / "gold.patch"
    cheat_patch = author / "cheat.patch"

    # 1. excision against the base tree
    exc_text = exc_patch.read_text(encoding="utf-8")
    if dry:
        _fix(exc_text, base_tree, "excision", dry=True)
        return rows
    exc_fixed, exc_notes = _fix(exc_text, base_tree, "excision", dry=False)
    if exc_fixed != exc_text:
        exc_patch.write_text(exc_fixed, encoding="utf-8")
        rows.append(f"{unit}: excision trimmed ({'; '.join(exc_notes)})")
    else:
        rows.append(f"{unit}: excision applies as-is")

    # 2. excised tree from the (fixed) excision
    work = Path(tempfile.mkdtemp(prefix=f"repair-{unit}-"))
    try:
        shutil.copytree(base_tree, work / "excised")
        rc, err = _apply_fuzz0(work / "excised", exc_fixed)
        if rc != 0:
            raise RuntimeError(f"excision apply failed for {unit}: {err}")

        # 3. gold and cheat against a pristine excised tree. A failed apply
        # attempt leaves earlier hunks applied, so every trial runs on a fresh
        # copy of the pristine excised tree.
        pristine = work / "excised_pristine"
        shutil.copytree(work / "excised", pristine)
        for name, patch in (("gold", gold_patch), ("cheat", cheat_patch)):
            if not patch.is_file():
                continue
            text = patch.read_text(encoding="utf-8")
            trial = work / f"trial_{name}"
            shutil.copytree(pristine, trial)
            rc, err = _apply_fuzz0(trial, text)
            if rc == 0:
                rows.append(f"{unit}: {name} applies as-is")
                continue
            fixed, notes = trim_trailing_blank_overruns(text, pristine)
            if fixed == text:
                rows.append(f"{unit}: {name} CANNOT APPLY (no trim route): {err.strip()[:160]}")
                continue
            for note in notes:
                print(f"  trim {name}: {note}")
            shutil.rmtree(trial, ignore_errors=True)
            shutil.copytree(pristine, trial)
            rc2, err2 = _apply_fuzz0(trial, fixed)
            if rc2 != 0:
                raise RuntimeError(f"fixed {name} still fails: {err2}")
            patch.write_text(fixed, encoding="utf-8")
            rows.append(f"{unit}: {name} trimmed ({'; '.join(notes)})")

        # 4. propagate fixed gold/cheat to packaged level dirs
        for root in (cfg.tasks_dir, cfg.tasks_dir.parent / "tasks_composerver"):
            unit_root = root / repo
            if not unit_root.is_dir():
                continue
            for td in sorted(unit_root.iterdir()):
                if not td.is_dir():
                    continue
                base = td.name.removesuffix("-L0").removesuffix("-L2").removesuffix("-L5").removesuffix("-L6")
                if base != unit:
                    continue
                tests_dir = td / "tests"
                if not tests_dir.is_dir():
                    continue
                for name in ("gold.patch", "cheat.patch"):
                    src = author / name
                    if src.is_file() and (tests_dir / name).exists():
                        (tests_dir / name).write_bytes(src.read_bytes())
    finally:
        shutil.rmtree(work, ignore_errors=True)
    return rows


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="experiments/pipeline/config.yaml")
    ap.add_argument("--repos", default="experiments/pipeline/repos.yaml")
    ap.add_argument("--repo", default="client-go")
    ap.add_argument("--units", nargs="*", default=None)
    ap.add_argument("--dry", action="store_true")
    args = ap.parse_args()
    cfg = load_config(args.config, args.repos)
    spec = cfg.repo(args.repo)
    base_tree = base_tree_for(spec, cfg)
    print(f"base tree: {base_tree}")
    units = args.units or sorted(p.name for p in (cfg.authored_dir / args.repo).iterdir() if p.is_dir())
    all_rows: list[str] = []
    for unit in units:
        print(f"== {unit}")
        all_rows.extend(repair_unit(cfg, base_tree, args.repo, unit, dry=args.dry))
    print("\n".join(all_rows))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
