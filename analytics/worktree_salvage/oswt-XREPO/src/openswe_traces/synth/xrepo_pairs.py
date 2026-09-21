"""Cross-repo closure units: excise inside a vendored dependency, verify via the consumer.

A unit here excises a function family inside library ``L`` (a real third-party
dependency staged as a nested Go module under ``deps/``) while the hidden tests
drive only consumer ``C``'s exported API. The instruction describes the symptom
as a consumer sees it and never names ``L``'s symbols nor reveals that the fault
is in a dependency.

Author dir layout per family::

    authored/<family>/_author/
        bugreport.md            L0 instruction body (symptom only)
        contract.md             L2 contract prose + coverage table
        closure.md              authoring record: what was excised
        difficulty.md           why it is hard / commitment list
        cheat.patch             C-level hardcode that must fail
        hidden/<relpath>        black-box *_test.go driving C's API

The excision and gold patches are derived deterministically from the family's
function list (see ``UNITS``); the cheat patch is hand-authored.
"""

from __future__ import annotations

import argparse
import difflib
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    build_affordance_levels,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.bigl0 import (
    _copy_patches_unread,
    _write_executable,
    l2_instruction,
)
from openswe_traces.synth.composerver_gin_batch import load_hidden  # noqa: F401  (kept for parity)

PAIR_SRC = ROOT / "experiments" / "xrepo" / "pair" / "src"
AUTHOR_ROOT = ROOT / "experiments" / "xrepo" / "authored"
DEFAULT_DEST = ROOT / "experiments" / "xrepo" / "tasks"
GIN_LADDER_BASE = "ladder-base:gin"
CONSUMER_MODULE = "example.internal/httprouter"
DEP_MODULE = "github.com/go-playground/validator/v10"
DEP_REL = "deps/validator"

_FUNC_HDR_RE = r"^func\s+(?:\([^)]*\)\s+)?%s\s*\("
_TEST_FUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)


def _test_names(content: str) -> list[str]:
    return _TEST_FUNC_RE.findall(content)


def _parse_contract_coverage(path: Path) -> list[tuple[str, str]]:
    """(property, contract-sentence) rows from a contract.md table."""
    rows: list[tuple[str, str]] = []
    if not path.is_file():
        return rows
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line.startswith("|") or line.startswith("|-") or "---" in line:
            continue
        cells = [c.strip().strip("`") for c in line.strip("|").split("|")]
        if len(cells) < 2 or not cells[0]:
            continue
        first = cells[0]
        if first.lower() in {"property", "test", "contract sentence"}:
            continue
        if first.startswith("Test"):
            rows.append((first, cells[-1]))
    return rows


def stub_go_func(text: str, name: str) -> str:
    """Replace the body of top-level func ``name`` with ``panic("excised: name")``."""
    pat = re.compile(_FUNC_HDR_RE % re.escape(name), re.MULTILINE)
    m = pat.search(text)
    if not m:
        raise KeyError(f"func {name} not found")
    brace = text.find("{", m.start())
    if brace < 0:
        raise ValueError(f"func {name} has no body")
    depth = 0
    i = brace
    while i < len(text):
        ch = text[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return text[:brace] + f'{{\n\tpanic("excised: {name}")\n}}' + text[i + 1 :]
        i += 1
    raise ValueError(f"func {name} unbalanced braces")


def blank_imports(text: str, imports: list[str]) -> str:
    """Rewrite ``"path"`` import specs as ``_ "path"`` (unused after excision)."""
    for imp in imports:
        text = text.replace(f'\t"{imp}"', f'\t_ "{imp}"', 1)
    return text


def build_buggy_file(pristine: str, funcs: list[str], blanks: list[str]) -> str:
    out = pristine
    for name in funcs:
        out = stub_go_func(out, name)
    return blank_imports(out, blanks)


def unified_diff(old: str, new: str, rel: str) -> str:
    """``patch -p1``-compatible diff a/<rel> -> b/<rel>."""
    return "".join(
        difflib.unified_diff(
            old.splitlines(keepends=True),
            new.splitlines(keepends=True),
            fromfile=f"a/{rel}",
            tofile=f"b/{rel}",
        )
    )


@dataclass(frozen=True)
class XRepoUnit:
    family: str
    dep_funcs: dict[str, list[str]]          # dep-relative file -> func names to excise
    blank_imports: dict[str, list[str]]      # dep-relative file -> import paths to blank
    hidden_relpath: str                      # relpath under tests/hidden AND /app
    packages: tuple[str, ...]                # consumer packages for go test
    reproduce_pkgs: str
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    coverage: tuple[tuple[str, str], ...] = field(default_factory=tuple)


UNITS: tuple[XRepoUnit, ...] = (
    XRepoUnit(
        family="condreq",
        dep_funcs={
            "baked_in.go": [
                "requireCheckFieldKind",
                "requireCheckFieldValue",
                "requiredIf",
                "excludedIf",
                "requiredUnless",
                "skipUnless",
                "excludedUnless",
                "excludedWith",
                "requiredWith",
                "excludedWithAll",
                "requiredWithAll",
                "excludedWithout",
                "requiredWithout",
                "excludedWithoutAll",
                "requiredWithoutAll",
            ],
        },
        blank_imports={},
        hidden_relpath="binding/xr_condreq_test.go",
        packages=("binding",),
        reproduce_pkgs="./binding/",
        changed_symbols=(
            "requiredIf", "requiredUnless", "requiredWith", "requiredWithAll",
            "requiredWithout", "requiredWithoutAll", "excludedIf", "excludedUnless",
            "excludedWith", "excludedWithAll", "excludedWithout", "excludedWithoutAll",
            "skipUnless", "requireCheckFieldKind", "requireCheckFieldValue",
        ),
        changed_files=(f"{DEP_REL}/baked_in.go",),
    ),
    XRepoUnit(
        family="fieldcmp",
        dep_funcs={
            "baked_in.go": [
                "hasLengthOf",
                "hasMinOf",
                "hasMaxOf",
                "isEq",
                "isEqIgnoreCase",
                "isNe",
                "isNeIgnoreCase",
                "isLt",
                "isLte",
                "isGt",
                "isGte",
            ],
        },
        blank_imports={},
        hidden_relpath="binding/xr_fieldcmp_test.go",
        packages=("binding",),
        reproduce_pkgs="./binding/",
        changed_symbols=(
            "hasLengthOf", "hasMinOf", "hasMaxOf", "isEq", "isEqIgnoreCase",
            "isNe", "isNeIgnoreCase", "isLt", "isLte", "isGt", "isGte",
        ),
        changed_files=(f"{DEP_REL}/baked_in.go",),
    ),
    XRepoUnit(
        family="oneofuniq",
        dep_funcs={
            "baked_in.go": [
                "isOneOf",
                "isOneOfCI",
                "isNoneOf",
                "isNoneOfCI",
                "isUnique",
            ],
        },
        blank_imports={"baked_in.go": ["slices"]},
        hidden_relpath="binding/xr_oneofuniq_test.go",
        packages=("binding",),
        reproduce_pkgs="./binding/",
        changed_symbols=(
            "isOneOf", "isOneOfCI", "isNoneOf", "isNoneOfCI", "isUnique",
        ),
        changed_files=(f"{DEP_REL}/baked_in.go",),
    ),
)


def unit_for(family: str) -> XRepoUnit:
    for u in UNITS:
        if u.family == family:
            return u
    raise KeyError(f"unknown family {family}; known: {[u.family for u in UNITS]}")


def write_patches(unit: XRepoUnit, author: Path, pair_src: Path = PAIR_SRC) -> tuple[Path, Path]:
    """Derive excision.patch (pristine->stubbed) and gold.patch (stubbed->pristine)."""
    excised_dir = author / "excised"
    excised_dir.mkdir(parents=True, exist_ok=True)
    excision_parts: list[str] = []
    gold_parts: list[str] = []
    for rel, funcs in sorted(unit.dep_funcs.items()):
        pristine = (pair_src / DEP_REL / rel).read_text(encoding="utf-8")
        buggy = build_buggy_file(pristine, funcs, unit.blank_imports.get(rel, []))
        dep_rel = f"{DEP_REL}/{rel}"
        excision_parts.append(unified_diff(pristine, buggy, dep_rel))
        gold_parts.append(unified_diff(buggy, pristine, dep_rel))
    excision = excised_dir / "excision.patch"
    excision.write_text("".join(excision_parts), encoding="utf-8")
    gold = author / "gold.patch"
    gold.write_text("".join(gold_parts), encoding="utf-8")
    return excision, gold


def materialize_buggy_tree(unit: XRepoUnit, author: Path, dest: Path, pair_src: Path = PAIR_SRC) -> None:
    """Pair tree (consumer + dep) with the family's dep closure stubbed out."""
    if dest.exists():
        shutil.rmtree(dest)
    _copytree(pair_src, dest)
    patch = author / "excised" / "excision.patch"
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch)],
        cwd=dest,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"excision.patch failed for {unit.family}: {proc.stderr or proc.stdout}")


def _strip_tree_leaks(src: Path) -> None:
    for path in src.rglob("*"):
        if path.is_file() and path.name.endswith("_bb_prop_test.go"):
            path.unlink()


def load_hidden_file(author: Path) -> tuple[str, str]:
    hidden_dir = author / "hidden"
    files = sorted(hidden_dir.rglob("*.go"))
    if len(files) != 1:
        raise FileNotFoundError(f"{hidden_dir} must hold exactly one .go file, got {files}")
    return files[0].read_text(encoding="utf-8"), files[0].relative_to(author / "hidden").as_posix()


def construct_unit(unit: XRepoUnit, dest_root: Path, *, author_root: Path = AUTHOR_ROOT) -> dict[int, Path]:
    author = author_root / unit.family / "_author"
    content, rel = load_hidden_file(author)
    names = _test_names(content)
    hidden = (
        HiddenTest(
            relpath=unit.hidden_relpath,
            content=content,
            one_liner=names[0] if names else Path(rel).stem,
            test_names=tuple(names),
        ),
    )
    skeleton = dest_root / f"_{unit.family}_skel"
    env = skeleton / "environment"
    env.mkdir(parents=True, exist_ok=True)
    write_patches(unit, author)
    materialize_buggy_tree(unit, author, env / "src")
    _strip_tree_leaks(env / "src")
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(GIN_LADDER_BASE), encoding="utf-8")
    (skeleton / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    contract = (author / "contract.md").read_text(encoding="utf-8")
    # The coverage table names hidden tests; locality-2 instructions must not.
    contract_instr = contract.split("\n## Coverage", 1)[0].rstrip() + "\n"
    l2 = l2_instruction(contract_instr, expected_actual="", reproduce_pkgs=unit.reproduce_pkgs)
    (skeleton / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skeleton / "tests", hidden)
    results = build_affordance_levels(
        skeleton,
        hidden,
        levels=(-2, 0),
        dest_root=dest_root,
        family=unit.family,
        instruction_a0=l2,
        packages=unit.packages,
        changed_symbols=unit.changed_symbols,
        changed_files=unit.changed_files,
        name_scheme="L",
        dockerfile_from=GIN_LADDER_BASE,
        instructions={-2: bugreport, 0: l2},
        representative=unit.hidden_relpath,
    )
    for dest in results.values():
        _copy_patches_unread(author, dest)
        _write_executable(
            dest / "tests" / "measure_gold.sh",
            "#!/bin/bash\nset -euo pipefail\necho \"no timing gate on this unit\"\nexit 0\n",
        )
    return results


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="Build cross-repo closure units (dep excision, consumer-driven tests)")
    ap.add_argument("--family", action="append", help="limit to families (default: all)")
    ap.add_argument("--dest", type=Path, default=DEFAULT_DEST)
    ap.add_argument("--author-root", type=Path, default=AUTHOR_ROOT)
    ap.add_argument("--pair-src", type=Path, default=PAIR_SRC)
    ap.add_argument("--dry-run", action="store_true", help="generate patches only; no tree copy")
    args = ap.parse_args(argv)
    units = [unit_for(f) for f in args.family] if args.family else list(UNITS)
    for unit in units:
        author = args.author_root / unit.family / "_author"
        if args.dry_run:
            e, g = write_patches(unit, author, args.pair_src)
            print(f"{unit.family}: wrote {e} + {g}")
            continue
        results = construct_unit(unit, args.dest, author_root=args.author_root)
        for level, path in sorted(results.items()):
            print(f"{unit.family} L{level + 2} -> {path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
