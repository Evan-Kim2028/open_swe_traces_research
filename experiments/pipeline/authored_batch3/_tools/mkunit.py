#!/usr/bin/env python3
"""mkunit.py — materialize excision.patch + gold.patch for one unit.

Usage: mkunit.py <unit>
Reads _tools/spec/<unit>.json:

  {
    "impl":     {"github/x.go": "stub" | {"Func": "PANIC"|"<body>"}},
    "deltests": {"github/x_test.go": ["TestA", ...]},
    "unimport": {"github/x.go": ["net/url"], ...},          # optional
    "testre":   "TestX|TestY"                                # used by verifyunit
  }

impl maps a file to a dict name->body; body "PANIC" means the panic stub,
anything else is a literal stub body. A func is matched by Name or Recv.Name.

Flow (scratch = git repo seeded from orig):
  restore listed files from index -> stub/delfunc/unimport -> go build ->
  git diff = excision.patch -> snapshot stubs -> restore impl files ->
  mkcheat.py stubbed-vs-orig = gold.patch -> restore test files.
"""
from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

TOOLS = Path(__file__).resolve().parent
SRC = TOOLS.parents[1] / "work" / "go-github" / "scratch"
ROOT = TOOLS.parents[0] / "go-github"
EXC = TOOLS / "excise"


def run(cmd, **kw):
    return subprocess.run(cmd, capture_output=True, text=True, **kw)


def restore(paths: list[str]) -> None:
    for rel in paths:
        p = run(["git", "-C", str(SRC), "show", f":{rel}"])
        if p.returncode != 0:
            sys.exit(f"restore {rel}: {p.stderr}")
        (SRC / rel).write_text(p.stdout)


def main() -> int:
    unit = sys.argv[1]
    spec = json.loads((TOOLS / "spec" / f"{unit}.json").read_text())
    udir = ROOT / unit / "_author"
    (udir / "excised").mkdir(parents=True, exist_ok=True)
    excdir = Path(f"/tmp/exc3/{unit}")
    shutil.rmtree(excdir, ignore_errors=True)

    impl_files = list(spec["impl"].keys())
    test_files = list(spec.get("deltests", {}).keys())
    restore(impl_files + test_files)

    # 1. stub + delete + unimport
    for rel, bodies in spec["impl"].items():
        panic = [n for n, b in bodies.items() if b == "PANIC"]
        custom = {n: b for n, b in bodies.items() if b != "PANIC"}
        if panic:
            p = run([str(EXC), "stub", str(SRC / rel), *panic])
            if p.returncode != 0:
                sys.exit(p.stderr)
        if custom:
            jpath = f"/tmp/exc3-{unit}-bodies.json"
            Path(jpath).write_text(json.dumps(custom))
            p = run([str(EXC), "stubx", str(SRC / rel), jpath])
            if p.returncode != 0:
                sys.exit(p.stderr)
        if not panic and not custom:
            sys.exit(f"{rel}: empty impl mapping")
    for rel, funcs in spec.get("deltests", {}).items():
        if funcs:
            p = run([str(EXC), "delfunc", str(SRC / rel), *funcs])
            if p.returncode != 0:
                sys.exit(p.stderr)
    for rel, pkgs in spec.get("unimport", {}).items():
        if pkgs:
            p = run([str(EXC), "unimport", str(SRC / rel), *pkgs])
            if p.returncode != 0:
                sys.exit(p.stderr)

    # 2. build check
    env = dict(os.environ, GOPROXY="off")
    p = run(["go", "build", "./github"], cwd=SRC, env=env)
    if p.returncode != 0:
        sys.exit(f"BUILD_FAIL {unit}:\n{p.stdout}{p.stderr}")
    print(f"BUILD_OK {unit}")

    # 3. snapshot stubbed impl files
    for rel in impl_files:
        (excdir / rel).parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(SRC / rel, excdir / rel)

    # 4. excision.patch = worktree diff over impl + test files
    p = run(["git", "-C", str(SRC), "diff", "--", *impl_files, *test_files])
    if p.returncode != 0:
        sys.exit(p.stderr)
    (udir / "excised" / "excision.patch").write_text(p.stdout)

    # 5. gold.patch: restore impl files, diff stubbed-vs-orig
    restore(impl_files)
    env2 = dict(os.environ, EXCDIR=str(excdir))
    p = run(
        [sys.executable, str(TOOLS / "mkcheat.py"), str(SRC), str(udir / "gold.patch"), *impl_files],
        env=env2,
    )
    if p.returncode != 0:
        sys.exit(p.stderr or p.stdout)
    print(p.stdout.strip())

    # 6. restore test files
    restore(test_files)
    left = run(["git", "-C", str(SRC), "status", "--porcelain"]).stdout.strip()
    if left:
        sys.exit(f"scratch not clean after {unit}:\n{left}")
    print(f"DONE {unit} -> {udir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
