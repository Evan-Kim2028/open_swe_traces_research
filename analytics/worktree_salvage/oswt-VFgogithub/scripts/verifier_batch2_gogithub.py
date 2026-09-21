"""Verifier-author packaging for go-github batch-2 units.

For each unit under the AU worktree's authored_batch2/go-github/<unit>/_author/:

- ``materialize``: rebuild the excised tree (pristine orig + excision.patch)
  into ``work/go-github/units/<unit>/tree`` for reading/compile checks.
- ``package``: assemble the A0-shaped skeleton (environment/src, Dockerfile,
  task.toml, instruction.md, patches) plus the hidden suite from
  ``work/go-github/units/<unit>/_verifier/tests/hidden`` and emit
  ``experiments/pipeline/tasks_batch2/go-github/<unit>-L{0,2}`` via
  ``synth.affordance.build_affordance_levels``.

Usage:
    uv run python scripts/verifier_batch2_gogithub.py materialize [unit ...]
    uv run python scripts/verifier_batch2_gogithub.py package [unit ...]
"""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    build_affordance_levels,
    coerce_hidden_test,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)

AU_AUTHOR = Path("/home/evan/Documents/oswt-AUgogithub/experiments/pipeline/authored_batch2/go-github")
ORIG = ROOT / "experiments" / "pipeline" / "work" / "go-github" / "orig"
UNITS = ROOT / "experiments" / "pipeline" / "work" / "go-github" / "units"
DEST_ROOT = ROOT / "experiments" / "pipeline" / "tasks_batch2" / "go-github"
BASE_IMAGE = "ladder-base:go-github"
REQUIRED = ("api.md", "contract.md", "bugreport.md", "gold.patch", "cheat.patch")

IGNORE = shutil.ignore_patterns(".git", "__pycache__")


def apply_patch(tree: Path, patch: Path) -> None:
    """plain `patch -p1`: git apply silently no-ops inside an outer worktree."""
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch)],
        cwd=tree,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"{patch} failed in {tree}: {(proc.stderr or proc.stdout)[-800:]}")


def units(arg: list[str]) -> list[str]:
    if arg:
        return arg
    return sorted(p.name for p in AU_AUTHOR.iterdir() if (p / "_author").is_dir())


def author_dir(unit: str) -> Path:
    d = AU_AUTHOR / unit / "_author"
    missing = [n for n in REQUIRED if not (d / n).is_file()]
    if missing:
        raise FileNotFoundError(f"{unit}: missing author artifacts {missing}")
    return d


def materialize(unit: str) -> Path:
    """orig + excision.patch -> work/go-github/units/<unit>/tree."""
    author = author_dir(unit)
    patch = author / "excised" / "excision.patch"
    if not patch.is_file():
        raise FileNotFoundError(f"{unit}: no excision.patch")
    dest = UNITS / unit / "tree"
    if dest.exists():
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(ORIG, dest, symlinks=True, ignore=IGNORE)
    apply_patch(dest, patch)
    return dest


def hidden_for(unit: str) -> list:
    root = UNITS / unit / "_verifier" / "tests" / "hidden"
    out = []
    for path in sorted(root.rglob("*_test.go")):
        rel = path.relative_to(root).as_posix()
        out.append(coerce_hidden_test({"relpath": rel, "content": path.read_text(encoding="utf-8")}))
    return out


def skeleton(unit: str, hidden: list) -> Path:
    """A0-shaped task dir the affordance builder clones per level."""
    author = author_dir(unit)
    skel = UNITS / unit / "_skel"
    if skel.exists():
        shutil.rmtree(skel)
    env = skel / "environment"
    src = env / "src"
    src.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(ORIG, src, symlinks=True, ignore=IGNORE)
    apply_patch(src, author / "excised" / "excision.patch")
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(BASE_IMAGE), encoding="utf-8")
    (skel / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    (skel / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skel / "tests", hidden)
    for name in ("gold.patch", "cheat.patch"):
        p = author / name
        if p.is_file():
            (skel / "patches").mkdir(exist_ok=True)
            shutil.copy2(p, skel / "patches" / name)
            shutil.copy2(p, skel / "tests" / name)
    return skel


def _l2_instruction(contract: str, bugreport: str) -> str:
    text = contract.rstrip() + "\n"
    if "expected" not in text.lower() and "actual" not in text.lower():
        text += "\n" + bugreport + "\n"
    if "Reproduce with:" not in text:
        text += (
            "\nReproduce with:\n\n```\ntests/test.sh\n```\n\n"
            "That script installs the hidden suite and runs it. Do not skip, delete, "
            "or weaken the tests. Work in `/app`.\n"
        )
    return with_no_web(text)


def package(unit: str) -> dict[int, Path]:
    author = author_dir(unit)
    hidden = hidden_for(unit)
    if not hidden:
        raise FileNotFoundError(f"{unit}: no hidden tests under {UNITS}/{unit}/_verifier/tests/hidden")
    skel = skeleton(unit, hidden)
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    contract = (author / "contract.md").read_text(encoding="utf-8")
    pkgs = sorted({str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden})
    DEST_ROOT.mkdir(parents=True, exist_ok=True)
    built = build_affordance_levels(
        skel,
        hidden,
        levels=(-2, 0),
        dest_root=DEST_ROOT,
        family=unit,
        instruction_a0=_l2_instruction(contract, bugreport),
        packages=pkgs,
        name_scheme="L",
        dockerfile_from=BASE_IMAGE,
        instructions={-2: bugreport, 0: _l2_instruction(contract, bugreport)},
    )
    return built


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=("materialize", "package"))
    ap.add_argument("units", nargs="*")
    args = ap.parse_args()
    rc = 0
    for unit in units(args.units):
        try:
            if args.cmd == "materialize":
                dest = materialize(unit)
                print(f"{unit}: materialized -> {dest}")
            else:
                built = package(unit)
                for lv, path in sorted(built.items()):
                    print(f"{unit}: L{lv + 2} -> {path}")
        except Exception as exc:  # noqa: BLE001 - per-unit isolation
            rc = 1
            print(f"{unit}: FAILED {exc}", file=sys.stderr)
    return rc


if __name__ == "__main__":
    sys.exit(main())
