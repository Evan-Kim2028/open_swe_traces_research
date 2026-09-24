"""Refuse duplicate closures in authored_batch4/bbolt: no shared file+symbol with the
existing bbolt bank or between new units. Adapted from check_unit_overlap_batch3.py.

The bbolt bank lives in dose_response task dirs (patches/gold.patch + environment/src
with panic("excised: X") stubs), not _author layout — so existing units are scanned
for excised markers directly, keyed by the file each marker appears in.

Usage: check_unit_overlap_bbolt.py <new_units_dir> <existing_task_dirs...>
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

HDR_RE = re.compile(r"^\+\+\+ b/(\S+)")
EXCISED_RE = re.compile(r"excised:\s*([A-Za-z_][A-Za-z0-9_.]*)")
FUNC_DECL_RE = re.compile(
    r"^[+-]\s*func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(", re.MULTILINE
)
CLOSURE_RE = re.compile(r"[\w./-]+\.(?:go|py|ts|js|java|rs)")


def patch_symbols(text: str) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    cur: str | None = None
    pending: str | None = None
    for line in text.splitlines():
        if line.startswith("--- "):
            m = re.match(r"^--- a/(\S+)", line)
            pending = m.group(1) if m else None
            continue
        if line.startswith("+++ "):
            m = HDR_RE.match(line)
            cur = m.group(1) if m else pending
            continue
        if cur is None:
            continue
        if not line.startswith(("+", "-")):
            continue
        syms = out.setdefault(cur, set())
        syms.update(s.split(".")[-1] for s in EXCISED_RE.findall(line))
        fm = FUNC_DECL_RE.match(line)
        if fm:
            syms.add(fm.group(1))
    return out


def files_of_author(unit_dir: Path) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    author = unit_dir / "_author"
    for name in ("gold.patch", "excised/excision.patch", "cheat.patch"):
        p = author / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    if not out:
        c = author / "closure.md"
        if c.is_file():
            for f in CLOSURE_RE.findall(c.read_text(errors="replace")):
                out.setdefault(f, set())
    return {f: s for f, s in out.items() if not f.endswith(("_test.go", "/dev/null"))}


def files_of_task(unit_dir: Path) -> dict[str, set[str]]:
    """dose_response layout: excised markers live in environment/src/*.go."""
    out: dict[str, set[str]] = {}
    src = unit_dir / "environment" / "src"
    if src.is_dir():
        for go in src.rglob("*.go"):
            if go.name.endswith("_test.go"):
                continue
            txt = go.read_text(errors="replace")
            syms = {s.split(".")[-1] for s in EXCISED_RE.findall(txt)}
            if syms:
                out[str(go.relative_to(src))] = syms
    for name in ("patches/gold.patch", "patches/cheat.patch"):
        p = unit_dir / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    return {f: s for f, s in out.items() if not f.endswith(("_test.go", "/dev/null"))}


def units_author(base: Path) -> dict[str, dict[str, set[str]]]:
    res: dict[str, dict[str, set[str]]] = {}
    if not base.is_dir():
        return res
    for unit in sorted(p for p in base.iterdir() if p.is_dir() and (p / "_author").is_dir()):
        res[unit.name] = files_of_author(unit)
    return res


def units_task(base: Path) -> dict[str, dict[str, set[str]]]:
    res: dict[str, dict[str, set[str]]] = {}
    if not base.is_dir():
        return res
    for unit in sorted(p for p in base.iterdir() if p.is_dir()):
        if (unit / "environment").is_dir():
            res[unit.name] = files_of_task(unit)
    return res


def shared_overlap(a, b):
    overlap, disjoint = [], []
    for f in sorted(set(a) & set(b)):
        sa, sb = a[f], b[f]
        if sa and sb and sa.isdisjoint(sb):
            disjoint.append(f)
        else:
            overlap.append(f)
    return overlap, disjoint


def main() -> int:
    new_dir = Path(sys.argv[1])
    existing_dirs = [Path(p) for p in sys.argv[2:]]
    new = units_author(new_dir)
    bad = 0
    for ed in existing_dirs:
        old = units_task(ed)
        for unit, files in new.items():
            for oname, ofiles in old.items():
                shared, disjoint = shared_overlap(files, ofiles)
                if shared:
                    print(f"OVERLAP {unit} (new) vs {ed.name}/{oname} (existing): {shared}")
                    bad += 1
                for f in disjoint:
                    print(f"INFO {unit} vs {oname}: share {f}, disjoint symbols")
    names = sorted(new)
    for i, u in enumerate(names):
        for o in names[i + 1:]:
            shared, disjoint = shared_overlap(new[u], new[o])
            if shared:
                print(f"OVERLAP {u} vs {o} (both new): {shared}")
                bad += 1
            for f in disjoint:
                print(f"INFO {u} vs {o}: share {f}, disjoint symbols")
    print(f"checked {len(new)} new units vs {sum(len(units_task(d)) for d in existing_dirs)} existing"
          f" — {'CLEAN' if not bad else 'OVERLAP'}")
    print("overlaps:", bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
