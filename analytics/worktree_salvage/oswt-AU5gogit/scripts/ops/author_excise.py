#!/usr/bin/env python3
"""Build the mechanical artifacts for one authored unit.

spec.json:
{
  "pristine": "/abs/path/to/gold/tree",
  "unit": "updreq",
  "out": "/abs/path/to/<unit>/_author",
  "excise": {"relpath.go": ["Func", "Recv.Method", ...]},
  "delete_tests_in": ["pkg/dir", ...]        # rm all *_test.go under these dirs
}

Produces in <out>:
  excised/excision.patch   gold tree -> excised tree (stubs + test deletion)
  gold.patch               excised tree -> gold tree, source files only (A12)
  excised/tree/            the excised tree (for cheat.patch staging)

Also prints a compile check (go build of the touched packages, GOPROXY=off).
"""
from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
EXCISER = REPO / "scripts" / "excise_funcs.py"


def sh(cmd: list[str], **kw) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, capture_output=True, text=True, **kw)


def diff_dirs(a: Path, b: Path) -> str:
    """git diff --no-index a b  ->  'a/<rel>' 'b/<rel>' style output."""
    with tempfile.TemporaryDirectory() as td:
        tda, tdb = Path(td) / "a", Path(td) / "b"
        subprocess.run(["cp", "-al", str(a) + "/.", str(tda)], check=True)
        subprocess.run(["cp", "-al", str(b) + "/.", str(tdb)], check=True)
        # hardlinks share inodes with the source; diff only reads, safe
        p = subprocess.run(["git", "diff", "--no-index", "a", "b"],
                           cwd=td, capture_output=True, text=True)
        # git --no-index writes "<prefix>/<argname>/<rel>" paths (and mirrors the
        # a-side arg path on the b-side for add/delete) -> collapse to <prefix>/<rel>
        def norm(line: str) -> str:
            parts = line.split(" ")
            for i, tok in enumerate(parts):
                t = tok.rstrip("\n")
                for pre in ("a/", "b/"):
                    if t.startswith(pre):
                        rest = t[len(pre):]
                        if rest.startswith(("a/", "b/")):
                            parts[i] = pre + rest[2:] + ("\n" if tok.endswith("\n") else "")
                        break
            return " ".join(parts)

        out = []
        for line in p.stdout.splitlines(keepends=True):
            if line.startswith(("diff --git ", "rename from ", "rename to ",
                                "copy from ", "copy to ", "--- ", "+++ ")):
                line = norm(line)
            out.append(line)
        return "".join(out)


def strip_test_files(patch: str) -> str:
    """Drop file-blocks whose path ends in _test.go (A12 for gold.patch)."""
    out, keep = [], True
    for line in patch.splitlines(keepends=True):
        if line.startswith("diff --git "):
            keep = "_test.go" not in line
        if keep:
            out.append(line)
    return "".join(out)


def main() -> int:
    spec = json.loads(Path(sys.argv[1]).read_text())
    pristine = Path(spec["pristine"]).resolve()
    out = Path(spec["out"]).resolve()
    unit = spec["unit"]
    excise: dict[str, list[str]] = spec["excise"]
    del_dirs: list[str] = spec.get("delete_tests_in", [])

    work = out / "excised" / "tree"
    if work.exists():
        shutil.rmtree(work)
    work.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(pristine, work, symlinks=True)

    exc_spec = {unit: excise}
    spec_file = out / "excised" / "excise_spec.json"
    spec_file.write_text(json.dumps(exc_spec, indent=1))
    p = sh(["uv", "run", "python", str(EXCISER), str(work), str(spec_file)], cwd=REPO)
    print(p.stdout, p.stderr, file=sys.stderr)

    # The exciser's import usage check counts doc comments; blank genuinely-unused
    # imports reported by the compiler instead.
    env0 = {"GOFLAGS": "-mod=mod", "GOPROXY": "off", "GOPATH": "/home/evan/go",
            "HOME": "/home/evan", "PATH": "/home/evan/go/bin:/usr/bin:/bin"}
    pkgs = sorted({f"./{str(Path(f).parent)}" for f in excise})
    for _ in range(4):
        b = sh(["go", "build", *pkgs], cwd=work, env=env0)
        unused = {}
        for line in b.stderr.splitlines():
            m = __import__("re").search(
                r"^(\S+\.go):\d+:\d+: \"?([\w./]+)\"? imported(?: as (\w+))? and not used", line)
            if m:
                unused.setdefault(str(work / m.group(1).lstrip("./")), []).append(m.group(2))
        if not unused:
            break
        import re as _re
        for f, paths in unused.items():
            lines = Path(f).read_text().splitlines(keepends=True)
            for i, l in enumerate(lines):
                if l.lstrip().startswith("import"):
                    m2 = _re.match(r'(\s*)import\s+"([^"]+)"\s*$', l)
                    if m2 and m2.group(2) in paths:
                        lines[i] = f'{m2.group(1)}import _ "{m2.group(2)}"\n'
                    continue
                lm = _re.match(r'(\s*)(?:(\w+|\.|_)\s+)?"([^"]+)"\s*$', l)
                if lm and lm.group(3) in paths:
                    lines[i] = f'{lm.group(1)}_ "{lm.group(3)}"\n'
            Path(f).write_text("".join(lines))

    deleted = []
    for d in del_dirs:
        for t in sorted((work / d).glob("*_test.go")):
            deleted.append(str(t.relative_to(work)))
            t.unlink()
    for t in spec.get("delete_tests", []):
        f = work / t
        if f.exists():
            deleted.append(t)
            f.unlink()

    (out / "excised").mkdir(parents=True, exist_ok=True)
    (out / "excised" / "excision.patch").write_text(diff_dirs(pristine, work))
    (out / "gold.patch").write_text(strip_test_files(diff_dirs(work, pristine)))

    touched_pkgs = sorted({str(Path(f).parent) for f in excise})
    env = {"GOFLAGS": "-mod=mod", "GOPROXY": "off", "GOPATH": "/home/evan/go",
           "HOME": "/home/evan", "PATH": "/home/evan/go/bin:/usr/bin:/bin"}
    build = sh(["go", "build", *[f"./{d}" for d in touched_pkgs]], cwd=work, env=env)
    vet = sh(["gofmt", "-l", *[str(work / f) for f in excise]], env=env)
    print(f"deleted_tests: {len(deleted)}")
    print(f"build rc={build.returncode} {build.stderr[:400]}")
    print(f"gofmt leftovers: {vet.stdout.strip() or 'none'}")
    if build.returncode != 0 or vet.stdout.strip():
        return 1

    # prune the excised tree to the touched files only (batch2 convention)
    for f in sorted(work.rglob("*")):
        if f.is_file() or f.is_symlink():
            rel = str(f.relative_to(work))
            if rel not in excise:
                f.unlink()
    for d in sorted((p for p in work.rglob("*") if p.is_dir()), reverse=True):
        if not any(d.iterdir()):
            d.rmdir()
    return 0


if __name__ == "__main__":
    sys.exit(main())
