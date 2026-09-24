#!/usr/bin/env python3
"""Free-symbol inventory for bbolt: every buildable top-level func in the pristine
tree, minus every symbol the bank already excises.

Bank sources (all scanned for `excised: NAME` markers and patched func decls):
  - dose_response task dirs (environment/src stubs + patches/*.patch)
  - authored_* dirs (_author/{gold,cheat}.patch + _author/excised/excision.patch)

Usage: uv run python scripts/free_symbols_bbolt.py
"""
from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
GOLD = REPO / "outputs/scratch/bbolt_gold"

MAIN = Path("/home/evan/Documents/open_swe_traces_research")
AU4 = Path("/home/evan/Documents/oswt-AU4bbolt2")

TASK_DIRS = [
    MAIN / "experiments/dose_response/sweep_bbolt",
    MAIN / "experiments/dose_response/sweep_bbolt_L2",
    MAIN / "experiments/dose_response/sweep_loop",
]
AUTHOR_DIRS = [
    AU4 / "experiments/pipeline/authored_batch4/bbolt",
    REPO / "experiments/pipeline/authored_au5bbolt/bbolt",
]

HDR_RE = re.compile(r"^\+\+\+ b/(\S+)")
EXCISED_RE = re.compile(r"excised:\s*([A-Za-z_][A-Za-z0-9_.]*)")
FUNC_DECL_RE = re.compile(
    r"^[+-]\s*func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(", re.M
)
FUNC_RE = re.compile(r"^func\s+(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(", re.M)


def patch_symbols(text: str) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    cur = None
    pending = None
    for line in text.splitlines():
        if line.startswith("--- "):
            m = re.match(r"^--- a/(\S+)", line)
            pending = m.group(1) if m else None
            continue
        if line.startswith("+++ "):
            m = HDR_RE.match(line)
            cur = m.group(1) if m else pending
            continue
        if cur is None or not line.startswith(("+", "-")):
            continue
        syms = out.setdefault(cur, set())
        syms.update(s.split(".")[-1] for s in EXCISED_RE.findall(line))
        fm = FUNC_DECL_RE.match(line)
        if fm:
            syms.add(fm.group(1))
    return out


def bank_from_task(unit: Path) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    src = unit / "environment" / "src"
    if src.is_dir():
        for go in src.rglob("*.go"):
            if go.name.endswith("_test.go"):
                continue
            syms = {s.split(".")[-1] for s in EXCISED_RE.findall(go.read_text(errors="replace"))}
            if syms:
                out.setdefault(str(go.relative_to(src)), set()).update(syms)
    for name in ("patches/gold.patch", "patches/cheat.patch"):
        p = unit / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    return out


def bank_from_author(unit: Path) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    au = unit / "_author"
    for name in ("gold.patch", "excised/excision.patch", "cheat.patch"):
        p = au / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    return out


def all_funcs() -> dict[str, list[str]]:
    """relpath -> [qualified func names] for the linux/amd64 build set."""
    env = {"GOFLAGS": "-mod=mod", "GOPROXY": "off", "HOME": "/home/evan",
           "PATH": "/home/evan/go/bin:/usr/bin:/bin", "GOPATH": "/home/evan/go"}
    p = subprocess.run(
        ["go", "list", "-f", "{{.ImportPath}} {{.Dir}}: {{join .GoFiles \" \"}}", "./..."],
        cwd=GOLD, env=env, capture_output=True, text=True)
    if p.returncode != 0:
        print(p.stderr, file=sys.stderr)
        sys.exit(1)
    out: dict[str, list[str]] = {}
    for line in p.stdout.splitlines():
        head, _, files = line.partition(": ")
        pkgdir = head.rsplit(" ", 1)[-1]
        for fn in files.split():
            rel = str((Path(pkgdir) / fn).relative_to(GOLD))
            txt = (GOLD / rel).read_text(errors="replace")
            for m in FUNC_RE.finditer(txt):
                recv, name = m.group(1), m.group(2)
                qname = f"{recv.strip('() ').split()[-1].lstrip('*')}.{name}" if recv else name
                out.setdefault(rel, []).append(qname)
    return out


def main() -> int:
    bank: dict[str, set[str]] = {}
    provenance: dict[tuple[str, str], str] = {}
    for d in TASK_DIRS:
        if not d.is_dir():
            continue
        for unit in sorted(p for p in d.iterdir() if p.is_dir()):
            if "bbolt" not in unit.name and not (unit / "environment/src").is_dir():
                continue
            for f, syms in bank_from_task(unit).items():
                bank.setdefault(f, set()).update(syms)
                for s in syms:
                    provenance[(f, s)] = f"{d.name}/{unit.name}"
    for d in AUTHOR_DIRS:
        if not d.is_dir():
            continue
        for unit in sorted(p for p in d.iterdir() if (p / "_author").is_dir()):
            for f, syms in bank_from_author(unit).items():
                bank.setdefault(f, set()).update(syms)
                for s in syms:
                    provenance[(f, s)] = f"{d.parent.name}/{unit.name}"

    funcs = all_funcs()
    free: dict[str, list[str]] = {}
    taken: dict[str, list[str]] = {}
    for f, names in sorted(funcs.items()):
        banked = bank.get(f, set())
        for q in names:
            bare = q.split(".")[-1]
            if bare in banked or q in banked:
                taken.setdefault(f, []).append(q)
            else:
                free.setdefault(f, []).append(q)

    n_free = sum(len(v) for v in free.values())
    n_taken = sum(len(v) for v in taken.values())
    print(f"# buildable funcs: {n_free + n_taken}  banked: {n_taken}  FREE: {n_free}\n")
    for f, names in sorted(free.items()):
        print(f"{f}  ({len(names)} free / {len(funcs[f])} total)")
        for q in names:
            print(f"    {q}")
    Path("/tmp/bbolt_free_symbols.json").write_text(json.dumps(
        {"free": free, "taken": taken}, indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main())
