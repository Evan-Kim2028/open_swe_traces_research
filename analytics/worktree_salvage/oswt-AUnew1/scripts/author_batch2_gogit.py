"""author_batch2_gogit — generate excision/gold patches for the 25 go-git units.

Thin driver over openswe_traces.synth.author_excise. For each UnitSpec it
writes _author/excised/excision.patch, _author/gold.patch and the stubbed
file copies, then compile-checks the excised tree (git apply into a scratch
copy, `go build` + `go test -run NONE` on the touched packages, revert).

Usage:
    uv run python scripts/author_batch2_gogit.py            # all units
    uv run python scripts/author_batch2_gogit.py revparse   # one unit
"""
from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

from openswe_traces.synth.author_excise import UnitSpec, write_unit_artifacts

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "experiments/pipeline/repos2/go-git/src"
OUT = ROOT / "experiments/pipeline/authored_batch2/go-git"
SCRATCH = Path("/tmp/gogit-check")


def tests_in(*dirs: str) -> tuple[str, ...]:
    out: list[str] = []
    for d in dirs:
        out.extend(sorted(p.name for p in []))  # placeholder
    return out


def all_tests(d: str) -> tuple[str, ...]:
    return tuple(
        f"{d}/{p.name}" for p in sorted((SRC / d).glob("*_test.go"))
    )


def _specs() -> list[tuple[UnitSpec, list[str]]]:
    """(spec, build_pkgs) pairs."""
    s: list[tuple[UnitSpec, list[str]]] = []

    def add(name, pkg, files, testdirs, pkgs=None):
        remove = tuple(t for d in testdirs for t in all_tests(d))
        extra_tests = {
            "fsrefs": ("storage/tests/storage_test.go",),
        }.get(name, ())
        s.append((
            UnitSpec(name=name, package=pkg, excise_files=tuple(files),
                     remove_tests=remove + extra_tests),
            pkgs or [pkg],
        ))

    add("revparse", "internal/revision",
        ["internal/revision/parser.go", "internal/revision/scanner.go"],
        ["internal/revision"])
    add("refnames", "plumbing",
        ["plumbing/reference.go"], ["plumbing"])
    add("objectid", "plumbing",
        ["plumbing/objectid.go"], ["plumbing"])
    add("pktline", "plumbing/format/pktline",
        ["plumbing/format/pktline/pktline.go",
         "plumbing/format/pktline/length.go",
         "plumbing/format/pktline/scanner.go"], ["plumbing/format/pktline"])
    add("cfgdecode", "plumbing/format/config",
        ["plumbing/format/config/decoder.go",
         "plumbing/format/config/fold.go"], ["plumbing/format/config"])
    add("cfgencode", "plumbing/format/config",
        ["plumbing/format/config/encoder.go"], ["plumbing/format/config"])
    add("cfgsection", "plumbing/format/config",
        ["plumbing/format/config/section.go",
         "plumbing/format/config/option.go"], ["plumbing/format/config"])
    add("ignorepattern", "plumbing/format/gitignore",
        ["plumbing/format/gitignore/pattern.go",
         "plumbing/format/gitignore/matcher.go"], ["plumbing/format/gitignore"])
    add("ignorescope", "plumbing/format/gitignore",
        ["plumbing/format/gitignore/scope.go",
         "plumbing/format/gitignore/dir.go"], ["plumbing/format/gitignore"])
    add("gitattrs", "plumbing/format/gitattributes",
        ["plumbing/format/gitattributes/attributes.go",
         "plumbing/format/gitattributes/pattern.go",
         "plumbing/format/gitattributes/matcher.go",
         "plumbing/format/gitattributes/dir.go"], ["plumbing/format/gitattributes"])
    add("indexdec", "plumbing/format/index",
        ["plumbing/format/index/decoder.go"], ["plumbing/format/index"])
    add("indexops", "plumbing/format/index",
        ["plumbing/format/index/index.go",
         "plumbing/format/index/match.go"], ["plumbing/format/index"])
    add("packdelta", "plumbing/format/packfile",
        ["plumbing/format/packfile/patch_delta.go"], ["plumbing/format/packfile"])
    add("packscan", "plumbing/format/packfile",
        ["plumbing/format/packfile/scanner.go",
         "plumbing/format/packfile/scanner_reader.go"], ["plumbing/format/packfile"])
    add("idxdecode", "plumbing/format/idxfile",
        ["plumbing/format/idxfile/decoder.go",
         "plumbing/format/idxfile/encoder.go"], ["plumbing/format/idxfile"])
    add("commitobj", "plumbing/object",
        ["plumbing/object/commit.go",
         "plumbing/object/commit_scanner.go"], ["plumbing/object"])
    add("treeobj", "plumbing/object",
        ["plumbing/object/tree.go",
         "plumbing/object/treenoder.go"], ["plumbing/object"])
    add("ulreq", "plumbing/protocol/packp",
        ["plumbing/protocol/packp/ulreq.go",
         "plumbing/protocol/packp/ulreq_decode.go",
         "plumbing/protocol/packp/ulreq_encode.go"], ["plumbing/protocol/packp"])
    add("advrefs", "plumbing/protocol/packp",
        ["plumbing/protocol/packp/advrefs.go",
         "plumbing/protocol/packp/advrefs_decode.go",
         "plumbing/protocol/packp/advrefs_encode.go"], ["plumbing/protocol/packp"])
    add("sideband", "plumbing/protocol/packp/sideband",
        ["plumbing/protocol/packp/sideband/demux.go",
         "plumbing/protocol/packp/sideband/muxer.go"],
        ["plumbing/protocol/packp/sideband"])
    add("endpoints", "internal/url",
        ["internal/url/url.go"], ["internal/url"])
    add("refspec", "config",
        ["config/refspec.go"], ["config"])
    add("cfgurl", "config",
        ["config/url.go"], ["config"])
    add("fsrefs", "storage/filesystem",
        ["storage/filesystem/reference.go", "storage/filesystem/reflog.go",
         "storage/filesystem/shallow.go",
         "storage/filesystem/dotgit/dotgit_setref.go",
         "storage/filesystem/dotgit/dotgit_rewrite_packed_refs.go"],
        ["storage/filesystem", "storage/filesystem/dotgit"],
        pkgs=["storage/filesystem", "storage/filesystem/dotgit"])
    add("merklediff", "utils/merkletrie",
        ["utils/merkletrie/difftree.go", "utils/merkletrie/iter.go",
         "utils/merkletrie/doubleiter.go", "utils/merkletrie/change.go"],
        ["utils/merkletrie"])
    return s


def run(cmd: list[str], cwd: Path) -> tuple[int, str]:
    p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    return p.returncode, (p.stdout + p.stderr).strip()


def write_cheat_patch(unit: str) -> str | None:
    """Diff _author/excised/<rel> vs _author/cheat/<rel> into cheat.patch."""
    author = OUT / unit / "_author"
    cheat_dir = author / "cheat"
    if not cheat_dir.is_dir():
        return None
    parts: list[str] = []
    for cf in sorted(cheat_dir.rglob("*.go")):
        rel = cf.relative_to(cheat_dir).as_posix()
        ex = author / "excised" / rel
        if not ex.is_file():
            raise RuntimeError(f"{unit}: cheat file {rel} has no excised counterpart")
        a = ex.read_text(encoding="utf-8").splitlines(keepends=True)
        b = cf.read_text(encoding="utf-8").splitlines(keepends=True)
        parts.append(_udiff(a, b, f"a/{rel}", f"b/{rel}"))
    patch = "".join(parts)
    (author / "cheat.patch").write_text(patch, encoding="utf-8")
    return patch


def _udiff(a_lines, b_lines, a_name, b_name):
    import difflib
    return "\n".join(
        line.rstrip("\n")
        for line in difflib.unified_diff(a_lines, b_lines, a_name, b_name, lineterm="\n")
    ) + "\n"


def check_unit(spec: UnitSpec, pkgs: list[str]) -> list[str]:
    """Apply excision in the scratch tree, compile, revert. Return problems."""
    problems: list[str] = []
    patch = OUT / spec.name / "_author/excised/excision.patch"
    if not SCRATCH.is_dir():
        problems.append("scratch missing")
        return problems
    rc, out = run(["git", "apply", "--check", str(patch)], SCRATCH)
    if rc:
        problems.append(f"excision.patch does not apply: {out[:300]}")
        return problems
    run(["git", "apply", str(patch)], SCRATCH)
    try:
        for pkg in pkgs:
            rc, out = run(["go", "build", f"./{pkg}/"], SCRATCH)
            if rc:
                problems.append(f"go build ./{pkg}/ FAILED: {out[:400]}")
            rc, out = run(["go", "test", "-count=1", "-run", "NONE", f"./{pkg}/"], SCRATCH)
            if rc:
                problems.append(f"test-compile ./{pkg}/ FAILED: {out[:400]}")
    finally:
        rc, out = run(["git", "apply", "-R", str(patch)], SCRATCH)
        if rc:
            problems.append(f"REVERT FAILED: {out[:300]} — scratch tree dirty")
    return problems


def main() -> int:
    only = set(sys.argv[1:])
    specs = _specs()
    bad = 0
    for spec, pkgs in specs:
        if only and spec.name not in only:
            continue
        res = write_unit_artifacts(spec, SRC, OUT / spec.name / "_author")
        n_excised = len(re.findall(r"excised:", res.excision_patch))
        print(f"== {spec.name}: {len(spec.excise_files)} files, "
              f"{len(spec.remove_tests)} tests removed, {n_excised} stubs")
        for w in res.warnings:
            print(f"   WARN {w}")
        for p in check_unit(spec, pkgs):
            print(f"   FAIL {p}")
            bad += 1
        # optional cheat.patch (cheat/ dir of stub-replacement files)
        try:
            cp = write_cheat_patch(spec.name)
        except RuntimeError as e:
            print(f"   FAIL cheat: {e}")
            bad += 1
            continue
        if cp is not None:
            # cheat must apply on the excised tree and compile
            run(["git", "apply", str(OUT / spec.name / "_author/excised/excision.patch")], SCRATCH)
            try:
                rc, out = run(["git", "apply", str(OUT / spec.name / "_author/cheat.patch")], SCRATCH)
                if rc:
                    print(f"   FAIL cheat.patch does not apply on excised tree: {out[:200]}")
                    bad += 1
                else:
                    for pkg in pkgs:
                        rc, out = run(["go", "build", f"./{pkg}/"], SCRATCH)
                        if rc:
                            print(f"   FAIL cheat build ./{pkg}/: {out[:300]}")
                            bad += 1
            finally:
                run(["git", "apply", "-R", str(OUT / spec.name / "_author/cheat.patch")], SCRATCH)
                run(["git", "apply", "-R", str(OUT / spec.name / "_author/excised/excision.patch")], SCRATCH)
    print("problems:", bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
