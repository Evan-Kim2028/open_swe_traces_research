"""Mechanical excision for authored_batch2 units.

Given a unit spec (source files to stub, test files to delete) this module:

* stubs every top-level function body to ``panic("excised: <name>")`` while
  keeping declarations, signatures and doc comments;
* blanks imports that are no longer referenced (``_ "path"``);
* emits ``excised/excision.patch`` (base -> excised, including the deleted
  ``*_test.go`` files) and ``gold.patch`` (excised -> base, non-test only);
* writes the stubbed file copies under ``_author/excised/<relpath>``.

Patch direction matches the batch builders: ``patch -p1`` applies
``excision.patch`` to the pristine tree; ``gold.patch`` applies to the excised
tree. ``gold.patch`` never touches ``*_test.go`` (rule A12).
"""

from __future__ import annotations

import difflib
import json
import re
import subprocess
from dataclasses import dataclass, field
from pathlib import Path

GOSTUB = "/tmp/gostub/gostub"


@dataclass(frozen=True)
class UnitSpec:
    """One authored unit: which files to stub and which tests to delete."""

    name: str
    package: str  # go package import path relative to module, e.g. "plumbing/format/pktline"
    excise_files: tuple[str, ...]  # repo-relative non-test .go files to stub
    remove_tests: tuple[str, ...]  # repo-relative *_test.go files to delete
    excise_keep: tuple[str, ...] = ()  # function names to NOT stub (e.g. init, var-initializers)


@dataclass
class UnitResult:
    name: str
    excision_patch: str
    gold_patch: str
    excised_files: dict[str, str] = field(default_factory=dict)  # relpath -> stubbed text
    warnings: list[str] = field(default_factory=list)


def _gostub(path: Path) -> dict:
    proc = subprocess.run([GOSTUB, str(path)], capture_output=True, text=True, check=False)
    return json.loads(proc.stdout)


def _import_ref_name(path: str, explicit: str) -> str:
    """Best-effort package identifier for an import path."""
    if explicit:
        return explicit
    elems = [e for e in path.split("/") if e]
    if not elems:
        return ""
    last = elems[-1]
    if re.fullmatch(r"v\d+", last) and len(elems) >= 2:
        last = elems[-2]
    m = re.match(r"^(?P<n>[\w]+?)(?:\.v\d+)?$", last)
    name = m.group("n") if m else last
    return name.replace("-", "_")


def stub_source(path: Path, keep: tuple[str, ...]) -> tuple[str, list[str]]:
    """Return stubbed text + warnings for one Go source file."""
    src = path.read_text(encoding="utf-8")
    info = _gostub(path)
    if info.get("err"):
        raise RuntimeError(f"parse failed {path}: {info['err']}")
    warnings: list[str] = []
    funcs = [f for f in info.get("funcs") or [] if f["name"] not in keep]
    skipped = [f["name"] for f in info.get("funcs") or [] if f["name"] in keep]
    for fn in skipped:
        warnings.append(f"kept body: {fn}")
    # package-level var initializers that call functions we are stubbing panic at init
    stubbed_names = {f["name"].split(".")[-1] for f in funcs}
    for call in info.get("var_calls") or []:
        if call.split(".")[-1] in stubbed_names:
            warnings.append(f"INIT-RISK: package-level var calls stubbed func {call}")
    # splice stub bodies, last first so offsets stay valid.
    # gostub spans are BYTE offsets — splice on the encoded form.
    raw = src.encode("utf-8")
    for f in sorted(funcs, key=lambda f: f["start"], reverse=True):
        stub = '{\n\tpanic("excised: %s")\n}' % f["name"]
        raw = raw[: f["start"]] + stub.encode("utf-8") + raw[f["end"] :]
    text = raw.decode("utf-8")
    # blank unused imports: work on stubbed text
    tmp = path.with_suffix(".stubtmp.go")
    tmp.write_text(text, encoding="utf-8")
    try:
        info2 = _gostub(tmp)
    finally:
        tmp.unlink()
    if info2.get("err"):
        raise RuntimeError(f"stubbed parse failed {path}: {info2['err']}")
    for imp in sorted(info2.get("imports") or [], key=lambda i: i["start"], reverse=True):
        name = imp["name"]
        if name in ("_", "."):
            continue
        # AST-reported use (comments/strings cannot fake a reference)
        if not imp.get("used"):
            raw = text.encode("utf-8")
            raw = raw[: imp["start"]] + ('_ "%s"' % imp["path"]).encode() + raw[imp["end"] :]
            text = raw.decode("utf-8")
    return text, warnings


def _udiff(a_lines: list[str], b_lines: list[str], a_name: str, b_name: str) -> str:
    diff = difflib.unified_diff(a_lines, b_lines, a_name, b_name, lineterm="\n")
    out = []
    for i, line in enumerate(diff):
        if i < 2:
            out.append(line.rstrip("\n"))
        else:
            out.append(line.rstrip("\n"))
    return "\n".join(out) + "\n"


def _delete_patch(relpath: str, text: str) -> str:
    lines = text.splitlines(keepends=True)
    lines = [l if l.endswith("\n") else l + "\n" for l in lines]
    n = len(lines)
    body = "".join("-" + l for l in lines)
    return (
        f"diff --git a/{relpath} b/{relpath}\n"
        f"deleted file mode 100644\n"
        f"--- a/{relpath}\n"
        f"+++ /dev/null\n"
        f"@@ -1,{n} +0,0 @@\n"
        f"{body}"
    )


def build_unit(spec: UnitSpec, src_root: Path) -> UnitResult:
    """Generate excision.patch + gold.patch + stubbed file copies for a unit."""
    res = UnitResult(name=spec.name, excision_patch="", gold_patch="")
    exc_parts: list[str] = []
    gold_parts: list[str] = []
    for rel in spec.excise_files:
        orig = (src_root / rel).read_text(encoding="utf-8")
        stubbed, warnings = stub_source(src_root / rel, spec.excise_keep)
        res.warnings.extend(f"{rel}: {w}" for w in warnings)
        res.excised_files[rel] = stubbed
        if stubbed == orig:
            res.warnings.append(f"{rel}: no functions stubbed")
            continue
        exc_parts.append(
            _udiff(orig.splitlines(keepends=True), stubbed.splitlines(keepends=True),
                   f"a/{rel}", f"b/{rel}")
        )
        gold_parts.append(
            _udiff(stubbed.splitlines(keepends=True), orig.splitlines(keepends=True),
                   f"a/{rel}", f"b/{rel}")
        )
    for rel in spec.remove_tests:
        text = (src_root / rel).read_text(encoding="utf-8")
        exc_parts.append(_delete_patch(rel, text))
    res.excision_patch = "".join(exc_parts)
    res.gold_patch = "".join(gold_parts)
    return res


def write_unit_artifacts(spec: UnitSpec, src_root: Path, author_dir: Path) -> UnitResult:
    """Build a unit and write excised/excision.patch, gold.patch, stubbed copies."""
    res = build_unit(spec, src_root)
    excised_dir = author_dir / "excised"
    excised_dir.mkdir(parents=True, exist_ok=True)
    (excised_dir / "excision.patch").write_text(res.excision_patch, encoding="utf-8")
    (author_dir / "gold.patch").write_text(res.gold_patch, encoding="utf-8")
    for rel, text in res.excised_files.items():
        dest = excised_dir / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(text, encoding="utf-8")
    return res
