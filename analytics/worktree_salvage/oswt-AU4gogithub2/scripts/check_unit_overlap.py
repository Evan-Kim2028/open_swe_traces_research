"""Refuse duplicate closures: a new unit must not excise symbols an existing unit already excises.

Compares every unit under experiments/pipeline/authored_batch4/<repo>/ against the union of
experiments/pipeline/authored/<repo>/ and experiments/pipeline/authored_batch3/<repo>/ (plus any
extra roots passed with --extra). Exits non-zero on overlap.

Granularity: a shared *file* between two units is overlap only when their changed-symbol sets
intersect — or when either side yields no symbols for that file (unverifiable -> refuse).
Symbols are extracted per patched file from ``excised: NAME`` markers (excision/gold patches)
and added/removed ``func`` declarations (cheat patches).
"""
from __future__ import annotations
import argparse, re, sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PATCH_RE = re.compile(r"^(?:\+\+\+|---)\s+[ab]/(\S+)", re.M)
CLOSURE_RE = re.compile(r"[\w./-]+\.(?:go|py|ts|js|java|rs)")
HDR_RE = re.compile(r"^\+\+\+ b/(\S+)")
EXCISED_RE = re.compile(r"excised:\s*([A-Za-z_][A-Za-z0-9_.]*)")
FUNC_DECL_RE = re.compile(
    r"^[+-]\s*func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(", re.M
)


def patch_symbols(text: str) -> dict[str, set[str]]:
    """Per-file symbol sets touched by one patch: excised markers + func decls."""
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
    """file -> changed-symbol set, unioned across the unit's patches."""
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


def units(base: Path) -> dict[str, dict[str, dict[str, set[str]]]]:
    res: dict[str, dict[str, dict[str, set[str]]]] = {}
    if not base.is_dir():
        return res
    for repo in sorted(p for p in base.iterdir() if p.is_dir()):
        for unit in sorted(p for p in repo.iterdir() if p.is_dir() and (p / "_author").is_dir()):
            res.setdefault(repo.name, {})[unit.name] = files_of(unit)
    return res


def shared_overlap(a: dict[str, set[str]], b: dict[str, set[str]]) -> tuple[list[str], list[str]]:
    """(overlapping files, file-shared/symbol-disjoint files)."""
    overlap, disjoint = [], []
    for f in sorted(set(a) & set(b)):
        sa, sb = a[f], b[f]
        if sa and sb and sa.isdisjoint(sb):
            disjoint.append(f)
        else:
            overlap.append(f)
    return overlap, disjoint


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--new", default="experiments/pipeline/authored_batch4")
    ap.add_argument("--extra", nargs="*", default=[])
    a = ap.parse_args()
    existing: dict[str, dict[str, dict[str, set[str]]]] = {}
    for prior in ("experiments/pipeline/authored", "experiments/pipeline/authored_batch3"):
        for repo, us in units(ROOT / prior).items():
            existing.setdefault(repo, {}).update(us)
    roots = [ROOT] + [Path(p) for p in a.extra]
    bad = 0
    for root in roots:
        new = units(root / a.new)
        for repo, us in new.items():
            old = existing.get(repo, {})
            for unit, files in us.items():
                for oname, ofiles in old.items():
                    shared, disjoint = shared_overlap(files, ofiles)
                    if shared:
                        print(f"OVERLAP {repo}/{unit} (new) vs {repo}/{oname} (existing): {shared}")
                        bad += 1
                    for f in disjoint:
                        print(f"INFO {repo}/{unit} vs {repo}/{oname}: share {f}, disjoint symbols")
                for oname, ofiles in us.items():
                    if oname >= unit:
                        continue
                    shared, disjoint = shared_overlap(files, ofiles)
                    if shared:
                        print(f"OVERLAP {repo}/{unit} vs {repo}/{oname} (both new): {shared}")
                        bad += 1
                    for f in disjoint:
                        print(f"INFO {repo}/{unit} vs {repo}/{oname}: share {f}, disjoint symbols")
            print(f"checked {repo}: {len(us)} new units vs {len(old)} existing — {'CLEAN' if not bad else 'OVERLAP'}")
    print("overlaps:", bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
