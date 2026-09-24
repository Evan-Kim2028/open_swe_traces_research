#!/usr/bin/env python3
"""Refuse duplicate closures: a new unit must not excise symbols an existing unit already excises.

Usage: check_unit_overlap.py <new_units_dir> <existing_dirs...>

New units are author-layout (<unit>/_author/{gold,cheat}.patch, excised/excision.patch).
Existing dirs are auto-detected per unit subdir:
  - author layout  (_author/ present)        -> patches scanned
  - task layout    (environment/src present) -> excised markers in src + patches/*.patch

Granularity: a shared file is overlap only when changed-symbol sets intersect, or when
either side yields no symbols for that file (unverifiable -> refuse).
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


def files_of_author(unit_dir: Path) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    au = unit_dir / "_author"
    for name in ("gold.patch", "excised/excision.patch", "cheat.patch"):
        p = au / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    if not out:
        c = au / "closure.md"
        if c.is_file():
            for f in CLOSURE_RE.findall(c.read_text(errors="replace")):
                out.setdefault(f, set())
    return {f: s for f, s in out.items() if not f.endswith(("_test.go", "/dev/null"))}


def files_of_task(unit_dir: Path) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    src = unit_dir / "environment" / "src"
    if src.is_dir():
        for go in src.rglob("*.go"):
            if go.name.endswith("_test.go"):
                continue
            syms = {s.split(".")[-1] for s in EXCISED_RE.findall(go.read_text(errors="replace"))}
            if syms:
                out.setdefault(str(go.relative_to(src)), set()).update(syms)
    for name in ("patches/gold.patch", "patches/cheat.patch"):
        p = unit_dir / name
        if p.is_file():
            for f, syms in patch_symbols(p.read_text(errors="replace")).items():
                out.setdefault(f, set()).update(syms)
    return {f: s for f, s in out.items() if not f.endswith(("_test.go", "/dev/null"))}


def units_in(base: Path) -> dict[str, dict[str, set[str]]]:
    """Auto-detect per subdir: author layout or task layout."""
    res: dict[str, dict[str, set[str]]] = {}
    if not base.is_dir():
        return res
    for unit in sorted(p for p in base.iterdir() if p.is_dir()):
        if (unit / "_author").is_dir():
            res[f"{base.name}/{unit.name}"] = files_of_author(unit)
        elif (unit / "environment").is_dir():
            res[f"{base.name}/{unit.name}"] = files_of_task(unit)
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
    if len(sys.argv) < 3:
        print(__doc__)
        return 2
    new_dir = Path(sys.argv[1])
    existing_dirs = [Path(p) for p in sys.argv[2:]]
    new = units_in(new_dir)
    bad = 0
    n_old = 0
    for ed in existing_dirs:
        old = units_in(ed)
        n_old += len(old)
        for unit, files in new.items():
            for oname, ofiles in old.items():
                shared, disjoint = shared_overlap(files, ofiles)
                if shared:
                    print(f"OVERLAP {unit} (new) vs {oname} (existing): {shared}")
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
    print(f"checked {len(new)} new units vs {n_old} existing"
          f" — {'CLEAN' if not bad else 'OVERLAP'}")
    print("overlaps:", bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
