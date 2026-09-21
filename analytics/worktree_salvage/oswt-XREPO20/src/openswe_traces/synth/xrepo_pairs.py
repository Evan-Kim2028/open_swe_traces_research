"""Cross-repo closure units: excise inside a vendored dependency, verify via the consumer.

A unit here excises a function family inside library ``L`` (a real third-party
dependency staged in-tree — a nested Go module under ``deps/`` for the gin pair,
a pinned module under ``deps/`` for both pairs) while the hidden tests
drive only consumer ``C``'s exported API. The instruction describes the symptom
as a consumer sees it and never names ``L``'s symbols nor reveals that the fault
is in a dependency.

Author dir layout per family::

    authored_batch2/<pair>/<family>/_author/
        api.md                  exported-API surface the suite drives
        bugreport.md            L0 instruction body (symptom only)
        contract.md             L2 contract prose + coverage table
        closure.md              authoring record: what was excised
        DETAILS.md              numbered commitments, one per line
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

EXPERIMENT = ROOT / "experiments" / "xrepo20"
DEFAULT_DEST = EXPERIMENT / "tasks"
GIN_LADDER_BASE = "ladder-base:gin"
KOPS_LADDER_BASE = "ladder-base:kops"


@dataclass(frozen=True)
class PairSpec:
    """A consumer/dependency pair: the staged tree plus where units author."""

    name: str                     # repo bucket name for authored_batch2
    pair_src: Path                # consumer+dep tree root (pristine)
    dep_rel: str                  # dep root inside pair_src
    ladder_base: str
    author_root: Path             # experiments/pipeline/authored_batch2/<name>


PAIRS: dict[str, PairSpec] = {
    "gin": PairSpec(
        name="gin",
        pair_src=EXPERIMENT / "pair" / "gin" / "src",
        dep_rel="deps/validator",
        ladder_base=GIN_LADDER_BASE,
        author_root=ROOT / "experiments" / "pipeline" / "authored_batch2" / "gin",
    ),
    "kops": PairSpec(
        name="kops",
        pair_src=EXPERIMENT / "pair" / "kops" / "src",
        dep_rel="deps",
        ladder_base=KOPS_LADDER_BASE,
        author_root=ROOT / "experiments" / "pipeline" / "authored_batch2" / "kops",
    ),
}

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
    brace = _body_brace(text, m.end() - 1)
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


def _body_brace(text: str, lparen: int) -> int:
    """Index of the ``{`` opening the func body whose parameter list starts at ``lparen``.

    Signatures may contain braces inside the parens (``interface{}``,
    ``struct{...}``) or in result types; the body brace is the first ``{`` at
    paren depth 0 that ends its line (gofmt always newline-terminates it).
    """
    depth = 0
    i = lparen
    while i < len(text):
        ch = text[i]
        if ch == "(":
            depth += 1
        elif ch == ")":
            depth -= 1
        elif ch == "{" and depth == 0:
            j = i + 1
            while j < len(text) and text[j] in " \t":
                j += 1
            if j < len(text) and text[j] == "\n":
                return i
        i += 1
    return -1


def blank_imports(text: str, imports: list[str]) -> str:
    """Rewrite ``"path"`` import specs as ``_ "path"`` (unused after excision).

    Handles plain (``"path"``) and aliased (``alias "path"``) specs.
    """
    for imp in imports:
        pat = re.compile(r'(?m)^\t(?:[A-Za-z_.]+\s+)?"' + re.escape(imp) + r'"$')
        text, n = pat.subn(f'\t_ "{imp}"', text, count=1)
        if n == 0:
            raise KeyError(f"import {imp} not found for blanking")
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
    pair: str                              # key into PAIRS
    dep_rel: str                           # dep root inside the pair tree
    dep_funcs: dict[str, list[str]]        # dep_rel-relative file -> func names to excise
    blank_imports: dict[str, list[str]]    # dep_rel-relative file -> import paths to blank
    hidden_relpath: str                    # relpath under tests/hidden AND /app
    packages: tuple[str, ...]              # consumer packages for go test
    reproduce_pkgs: str
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    coverage: tuple[tuple[str, str], ...] = field(default_factory=tuple)


_GIN = "gin"
_KOPS = "kops"
_DEP_GIN = "deps/validator"
_AM = "deps/apimachinery"
_SEMVER = "deps/semver"


def _u(
    family: str,
    *,
    pair: str = _GIN,
    dep_rel: str = _DEP_GIN,
    dep_funcs: dict[str, list[str]],
    blank_imports: dict[str, list[str]] | None = None,
    hidden_relpath: str,
    packages: tuple[str, ...] = ("binding",),
    reproduce_pkgs: str = "./binding/",
) -> XRepoUnit:
    symbols = tuple(s for funcs in dep_funcs.values() for s in funcs)
    files = tuple(f"{dep_rel}/{rel}" for rel in sorted(dep_funcs))
    return XRepoUnit(
        family=family,
        pair=pair,
        dep_rel=dep_rel,
        dep_funcs=dep_funcs,
        blank_imports=blank_imports or {},
        hidden_relpath=hidden_relpath,
        packages=packages,
        reproduce_pkgs=reproduce_pkgs,
        changed_symbols=symbols,
        changed_files=files,
    )


_KV = "pkg/apis/kops/validation"

UNITS: tuple[XRepoUnit, ...] = (
    # --- existing gin->validator units (kept for rebuild parity) -------------
    _u(
        "condreq",
        dep_funcs={
            "baked_in.go": [
                "requireCheckFieldKind", "requireCheckFieldValue", "requiredIf",
                "excludedIf", "requiredUnless", "skipUnless", "excludedUnless",
                "excludedWith", "requiredWith", "excludedWithAll", "requiredWithAll",
                "excludedWithout", "requiredWithout", "excludedWithoutAll",
                "requiredWithoutAll",
            ],
        },
        hidden_relpath="binding/xr_condreq_test.go",
    ),
    _u(
        "fieldcmp",
        dep_funcs={
            "baked_in.go": [
                "hasLengthOf", "hasMinOf", "hasMaxOf", "isEq", "isEqIgnoreCase",
                "isNe", "isNeIgnoreCase", "isLt", "isLte", "isGt", "isGte",
            ],
        },
        hidden_relpath="binding/xr_fieldcmp_test.go",
    ),
    _u(
        "oneofuniq",
        dep_funcs={
            "baked_in.go": ["isOneOf", "isOneOfCI", "isNoneOf", "isNoneOfCI", "isUnique"],
        },
        blank_imports={"baked_in.go": ["slices"]},
        hidden_relpath="binding/xr_oneofuniq_test.go",
    ),
    # --- new gin->validator units -------------------------------------------
    _u(
        "mailfmt",
        dep_funcs={"baked_in.go": ["isEmail"]},
        blank_imports={"baked_in.go": ["net/mail"]},
        hidden_relpath="binding/xr_mailfmt_test.go",
    ),
    _u(
        "uuidfmt",
        dep_funcs={
            "baked_in.go": [
                "isUUID", "isUUID3", "isUUID4", "isUUID5",
                "isUUIDRFC4122", "isUUID3RFC4122", "isUUID4RFC4122", "isUUID5RFC4122",
                "isULID",
            ],
        },
        hidden_relpath="binding/xr_uuidfmt_test.go",
    ),
    _u(
        "urifmt",
        dep_funcs={
            "baked_in.go": [
                "isURI", "isURL", "isHttpURL", "isHttpsURL",
                "isUrnRFC2141", "isDataURI",
            ],
        },
        blank_imports={"baked_in.go": ["github.com/leodido/go-urn"]},
        hidden_relpath="binding/xr_urifmt_test.go",
    ),
    _u(
        "ipcidr",
        dep_funcs={
            "baked_in.go": [
                "isIPv4", "isIPv6", "isIP", "isCIDRv4", "isCIDRv6", "isCIDR", "isMAC",
            ],
        },
        hidden_relpath="binding/xr_ipcidr_test.go",
    ),
    _u(
        "hostport",
        dep_funcs={
            "baked_in.go": [
                "isHostnameRFC952", "isHostnameRFC1123", "isFQDN",
                "isDnsRFC1035LabelFormat", "isHostnamePort", "isPort",
            ],
        },
        hidden_relpath="binding/xr_hostport_test.go",
    ),
    _u(
        "contain",
        dep_funcs={
            "baked_in.go": ["containsRune", "containsAny", "contains", "fieldContains"],
        },
        hidden_relpath="binding/xr_contain_test.go",
    ),
    _u(
        "exclude",
        dep_funcs={
            "baked_in.go": ["excludesRune", "excludesAll", "excludes", "fieldExcludes"],
        },
        hidden_relpath="binding/xr_exclude_test.go",
    ),
    _u(
        "affix",
        dep_funcs={
            "baked_in.go": ["startsWith", "endsWith", "startsNotWith", "endsNotWith"],
        },
        hidden_relpath="binding/xr_affix_test.go",
    ),
    _u(
        "crossfld",
        dep_funcs={
            "baked_in.go": [
                "isNeField", "isEqField", "isLtField", "isLteField",
                "isGtField", "isGteField",
            ],
        },
        hidden_relpath="binding/xr_crossfld_test.go",
    ),
    _u(
        "xstruct",
        dep_funcs={
            "baked_in.go": [
                "isNeCrossStructField", "isEqCrossStructField", "isLtCrossStructField",
                "isLteCrossStructField", "isGtCrossStructField", "isGteCrossStructField",
            ],
        },
        hidden_relpath="binding/xr_xstruct_test.go",
    ),
    _u(
        "hashfmt",
        dep_funcs={
            "baked_in.go": [
                "isMD4", "isMD5", "isSHA256", "isSHA384", "isSHA512",
                "isRIPEMD128", "isRIPEMD160", "isTIGER128", "isTIGER160", "isTIGER192",
            ],
        },
        hidden_relpath="binding/xr_hashfmt_test.go",
    ),
    _u(
        "colorfmt",
        dep_funcs={
            "baked_in.go": ["isHEXColor", "isRGB", "isRGBA", "isHSL", "isHSLA", "isCMYK"],
        },
        hidden_relpath="binding/xr_colorfmt_test.go",
    ),
    _u(
        "cryptocoin",
        dep_funcs={
            "baked_in.go": [
                "isEthereumAddress", "isEthereumAddressChecksum",
                "isBitcoinAddress", "isBitcoinBech32Address",
            ],
        },
        blank_imports={"baked_in.go": ["crypto/sha256", "encoding/hex", "golang.org/x/crypto/sha3"]},
        hidden_relpath="binding/xr_cryptocoin_test.go",
    ),
    _u(
        "structlvl",
        dep_funcs={
            "struct_level.go": [
                "wrapStructLevelFunc",
                "ReportError",
                "ReportValidationErrors",
            ],
            "validator_instance.go": [
                "RegisterStructValidation",
                "RegisterStructValidationCtx",
                "RegisterStructValidationMapRules",
            ],
        },
        hidden_relpath="binding/xr_structlvl_test.go",
    ),
    # --- new kops->deps-modules units --------------------------------------
    _u(
        "ignames",
        pair=_KOPS,
        dep_rel=_AM,
        dep_funcs={
            "pkg/api/validation/generic.go": [
                "NameIsDNSSubdomain", "NameIsDNSLabel", "NameIsDNS1035Label",
                "maskTrailingDash",
            ],
        },
        blank_imports={
            "pkg/api/validation/generic.go": [
                "strings", "k8s.io/apimachinery/pkg/util/validation",
            ],
        },
        hidden_relpath=f"{_KV}/xr_ignames_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
    _u(
        "clusternames",
        pair=_KOPS,
        dep_rel=_AM,
        dep_funcs={
            "pkg/util/validation/validation.go": [
                "IsDNS1123Subdomain", "IsDNS1123Label", "IsDNS1035Label",
                "IsWildcardDNS1123Subdomain", "IsDNS1123SubdomainWithUnderscore",
            ],
        },
        hidden_relpath=f"{_KV}/xr_clusternames_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
    _u(
        "labelvals",
        pair=_KOPS,
        dep_rel=_AM,
        dep_funcs={
            "pkg/api/validate/content/kube.go": [
                "IsLabelValue", "IsLabelKey", "IsPrefixedLabelKey", "prefixEach",
            ],
        },
        blank_imports={"pkg/api/validate/content/kube.go": ["strings"]},
        hidden_relpath=f"{_KV}/xr_labelvals_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
    _u(
        "portrange",
        pair=_KOPS,
        dep_rel=_AM,
        dep_funcs={
            "pkg/util/net/port_range.go": [
                "Set", "ParsePortRange", "ParsePortRangeOrDie", "Contains", "String", "Type",
            ],
        },
        blank_imports={"pkg/util/net/port_range.go": ["fmt", "strconv", "strings"]},
        hidden_relpath=f"{_KV}/xr_portrange_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
    _u(
        "intstrpct",
        pair=_KOPS,
        dep_rel=_AM,
        dep_funcs={
            "pkg/util/intstr/intstr.go": [
                "GetScaledValueFromIntOrPercent", "GetValueFromIntOrPercent",
                "getIntOrPercentValue", "getIntOrPercentValueSafely",
            ],
        },
        blank_imports={"pkg/util/intstr/intstr.go": ["errors", "strings"]},
        hidden_relpath=f"{_KV}/xr_intstrpct_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
    _u(
        "semververs",
        pair=_KOPS,
        dep_rel=_SEMVER,
        dep_funcs={
            "semver.go": ["Parse", "ParseTolerant"],
            "range.go": ["ParseRange"],
        },
        hidden_relpath=f"{_KV}/xr_semververs_test.go",
        packages=(_KV,),
        reproduce_pkgs=f"./{_KV}/",
    ),
)


def pair_for(unit: XRepoUnit) -> PairSpec:
    return PAIRS[unit.pair]


def unit_for(family: str) -> XRepoUnit:
    for u in UNITS:
        if u.family == family:
            return u
    raise KeyError(f"unknown family {family}; known: {[u.family for u in UNITS]}")


def write_patches(unit: XRepoUnit, author: Path, pair_src: Path | None = None) -> tuple[Path, Path]:
    """Derive excision.patch (pristine->stubbed) and gold.patch (stubbed->pristine)."""
    if pair_src is None:
        pair_src = pair_for(unit).pair_src
    excised_dir = author / "excised"
    excised_dir.mkdir(parents=True, exist_ok=True)
    excision_parts: list[str] = []
    gold_parts: list[str] = []
    for rel, funcs in sorted(unit.dep_funcs.items()):
        pristine = (pair_src / unit.dep_rel / rel).read_text(encoding="utf-8")
        buggy = build_buggy_file(pristine, funcs, unit.blank_imports.get(rel, []))
        dep_rel = f"{unit.dep_rel}/{rel}"
        excision_parts.append(unified_diff(pristine, buggy, dep_rel))
        gold_parts.append(unified_diff(buggy, pristine, dep_rel))
    excision = excised_dir / "excision.patch"
    excision.write_text("".join(excision_parts), encoding="utf-8")
    gold = author / "gold.patch"
    gold.write_text("".join(gold_parts), encoding="utf-8")
    return excision, gold


def materialize_buggy_tree(unit: XRepoUnit, author: Path, dest: Path, pair_src: Path | None = None) -> None:
    """Pair tree (consumer + dep) with the family's dep closure stubbed out."""
    if pair_src is None:
        pair_src = pair_for(unit).pair_src
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


def construct_unit(unit: XRepoUnit, dest_root: Path, *, author_root: Path | None = None) -> dict[int, Path]:
    pair = pair_for(unit)
    if author_root is None:
        author_root = pair.author_root
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
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(pair.ladder_base), encoding="utf-8")
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
        dockerfile_from=pair.ladder_base,
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
    ap.add_argument("--pair", choices=sorted(PAIRS), help="limit to one pair")
    ap.add_argument("--dest", type=Path, default=DEFAULT_DEST)
    ap.add_argument("--author-root", type=Path, default=None)
    ap.add_argument("--pair-src", type=Path, default=None)
    ap.add_argument("--dry-run", action="store_true", help="generate patches only; no tree copy")
    args = ap.parse_args(argv)
    units = [unit_for(f) for f in args.family] if args.family else list(UNITS)
    if args.pair:
        units = [u for u in units if u.pair == args.pair]
    for unit in units:
        pair = pair_for(unit)
        author = (args.author_root or pair.author_root) / unit.family / "_author"
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
