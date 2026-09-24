"""Refuse duplicate closures in authored_batch3: no shared file+symbol with the
existing cohort or between new units. Adapted from check_unit_overlap.py.

Usage: check_unit_overlap_batch3.py <new_units_dir> <existing_units_dir> [<existing2>...]
"""
from __future__ import annotations
import re, sys
from pathlib import Path

HDR_RE = re.compile(r"^\+\+\+ b/(\S+)")
EXCISED_RE = re.compile(r"excised:\s*([A-Za-z_][A-Za-z0-9_]*)")
FUNC_DECL_RE = re.compile(
    r"^[+-]\s*func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(", re.M
)
CLOSURE_RE = re.compile(r"[\w./-]+\.(?:go|py|ts|js|java|rs)")


def patch_symbols(text: str) -> dict[str, set[str]]:
    out: dict[str, set[str]] = {}
    cur: str | None = None
    for line in text.splitlines():
        m = HDR_RE.match(line)
        if m:
            cur = m.group(1)
            continue
        if cur is None or line.startswith(("+++", "---")):
            continue
        if not line.startswith(("+", "-")):
            continue
        syms = out.setdefault(cur, set())
        syms.update(EXCISED_RE.findall(line))
        fm = FUNC_DECL_RE.match(line)
        if fm:
            syms.add(fm.group(1))
    return out


def files_of(unit_dir: Path) -> dict[str, set[str]]:
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


def units(base: Path) -> dict[str, dict[str, set[str]]]:
    res: dict[str, dict[str, set[str]]] = {}
    if not base.is_dir():
        return res
    for unit in sorted(p for p in base.iterdir() if p.is_dir() and (p / "_author").is_dir()):
        res[unit.name] = files_of(unit)
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
    new = units(new_dir)
    bad = 0
    for ed in existing_dirs:
        old = units(ed)
        for unit, files in new.items():
            for oname, ofiles in old.items():
                shared, disjoint = shared_overlap(files, ofiles)
                if shared:
                    print(f"OVERLAP {unit} (new) vs {ed.parent.name}/{oname} (existing): {shared}")
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
    print(f"checked {len(new)} new units vs {sum(len(units(d)) for d in existing_dirs)} existing"
          f" — {'CLEAN' if not bad else 'OVERLAP'}")
    print("overlaps:", bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
