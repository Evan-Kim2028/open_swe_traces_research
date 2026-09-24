"""L0 fairness self-audit for fabricated units (verifier_rules.md B4-B7, C1 class (c)).

The L0 information set of a unit dir is ``excised_tree/`` (including the
in-tree smoke suite and ``params.go``), ``_author/bugreport.md`` and the output
of ``./repro.sh`` — which runs the smoke suite, so every literal it can print
is already present in the smoke-suite source.

Checks per unit:

- ``params_intact``: ``excised_tree/params.go`` exists, is identical to
  ``module/params.go``, contains ``param*`` constants and no excision stubs.
- ``hidden_literals``: every hex literal > 0xFF in ``hidden/*.go``, every
  literal inside ``var|const ref*`` declarations, and the 2^64-1 sentinel
  (when the suite asserts on ``^uint64(0)``/``math.MaxUint64``) appear in the
  L0 corpus.  This is the class-(c) tripwire: withholding a constant is a
  feasibility gate, not difficulty.
- ``gold_patch_literals``: ``gold.patch`` introduces no numeric literal
  (>255 dec / >0xFF hex) absent from the L0 corpus — constants are never
  excised.
- ``coverage_table``: every ``func TestX`` in the hidden suite has a
  ``| TestX |`` row in ``_author/contract.md``.
- ``bugreport_floor`` (B6): symptom, expected-vs-got, reproduction command.
- ``bugreport_ceiling`` (B7): no excised symbol names, file names, diffs or
  line numbers.
- ``examples_tied``: every quoted string and number in the bugreport's
  "Worked examples" bullets appears in the in-tree smoke suite (``param*``
  names resolved to their declared values), so repro output prints them.
- ``black_box`` (B4-light): the hidden suite never calls unexported members
  via the package qualifier.
- ``details_manifest``: the manifest's ``dial``/``details`` blocks are
  consistent — every drawn detail has an id, description, prose stated in the
  bug report, hidden tests that actually exist in the hidden suite, and
  ``checkable_from_repro`` matching the unit's v level.
- ``b3_not_complete_spec``: for every drawn detail the hidden per-detail test
  checks inputs beyond the detail's L0 worked examples (a seeded-random loop
  or strictly more Go literals than the in-tree smoke example), so the L0
  information set never pins the detail's property — a complete example suite
  would be a spec (verifier_rules.md B3).
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

HEX_RE = re.compile(r"0x[0-9a-fA-F]+")
NUM_RE = re.compile(r"\b\d+\b")
REF_DECL_RE = re.compile(r"(?:var|const)\s+(ref\w+)\s*(?:[\w\[\].]+\s*)?=\s*(.+)")
TEST_FN_RE = re.compile(r"^func (Test\w+)\s*\(", re.MULTILINE)
COVERAGE_ROW_RE = re.compile(r"^\|\s*(Test\w+)\s*\|", re.MULTILINE)
QUOTED_RE = re.compile(r'"([^"\n]*)"')
RESERVED = (1 << 64) - 1
RESERVED_MARKERS = ("paramReservedKey", "MaxUint64", "18446744073709551615")


@dataclass
class Check:
    name: str
    passed: bool
    detail: str = ""


@dataclass
class UnitAudit:
    unit_dir: Path
    checks: list[Check] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return all(c.passed for c in self.checks)

    def add(self, name: str, passed: bool, detail: str = "") -> None:
        self.checks.append(Check(name, passed, detail))


def _literals(text: str) -> set[int]:
    vals = {int(m, 16) for m in HEX_RE.findall(text)}
    vals |= {int(m) for m in NUM_RE.findall(text)}
    if any(m in text for m in RESERVED_MARKERS) or re.search(r"1\s*<<\s*64\s*-\s*1|2\^64", text):
        vals.add(RESERVED)
    return vals


def _l0_corpus(unit_dir: Path) -> str:
    parts: list[str] = []
    tree = unit_dir / "excised_tree"
    for p in sorted(tree.rglob("*")):
        if p.is_file():
            parts.append(p.read_text(encoding="utf-8", errors="replace"))
    for rel in ("_author/bugreport.md", "repro.sh"):
        f = unit_dir / rel
        if f.is_file():
            parts.append(f.read_text(encoding="utf-8", errors="replace"))
    return "\n".join(parts)


def _param_values(params_text: str) -> dict[str, int]:
    """``paramXxx``/``nbuckets`` const declarations -> integer value."""
    vals: dict[str, int] = {}
    for m in re.finditer(r"(?:const|var)\s+(\w+)\s*(?:\w+\s*)?=\s*([^\n]+)", params_text):
        name, rhs = m.group(1), m.group(2)
        nums = [int(x, 16) for x in HEX_RE.findall(rhs)] + [int(x) for x in NUM_RE.findall(rhs)]
        if "MaxUint64" in rhs or re.search(r"1\s*<<\s*64\s*-\s*1", rhs):
            nums.append(RESERVED)
        if nums:
            vals[name] = nums[0]
    return vals


def audit_unit(unit_dir: Path) -> UnitAudit:
    unit_dir = Path(unit_dir)
    audit = UnitAudit(unit_dir)
    manifest_path = unit_dir / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8")) if manifest_path.is_file() else {}

    hidden_dir = unit_dir / "hidden"
    hidden_files = sorted(hidden_dir.glob("*_test.go")) if hidden_dir.is_dir() else []
    hidden_text = "\n".join(f.read_text(encoding="utf-8") for f in hidden_files)
    bugreport = (unit_dir / "_author" / "bugreport.md")
    bugreport_text = bugreport.read_text(encoding="utf-8") if bugreport.is_file() else ""
    contract_path = unit_dir / "_author" / "contract.md"
    contract_text = contract_path.read_text(encoding="utf-8") if contract_path.is_file() else ""
    corpus = _l0_corpus(unit_dir)
    corpus_vals = _literals(corpus)

    # --- params_intact -------------------------------------------------------
    excised_params = unit_dir / "excised_tree" / "params.go"
    module_params = unit_dir / "module" / "params.go"
    if not excised_params.is_file():
        audit.add("params_intact", False, "excised_tree/params.go missing")
    elif not module_params.is_file() or excised_params.read_text() != module_params.read_text():
        audit.add("params_intact", False, "params.go differs between excised_tree and module")
    else:
        text = excised_params.read_text(encoding="utf-8")
        stubs = 'panic("excised' in text
        n_params = len(re.findall(r"\bparam\w+\s*=", text))
        audit.add(
            "params_intact",
            not stubs and n_params >= 3,
            f"param consts: {n_params}; stubs: {stubs}",
        )

    # --- hidden_literals -----------------------------------------------------
    need: set[int] = {int(m, 16) for m in HEX_RE.findall(hidden_text) if int(m, 16) > 0xFF}
    for m in REF_DECL_RE.finditer(hidden_text):
        need |= {int(x, 16) for x in HEX_RE.findall(m.group(2))}
        need |= {int(x) for x in NUM_RE.findall(m.group(2))}
    if re.search(r"\^uint64\(0\)|MaxUint64|18446744073709551615", hidden_text):
        need.add(RESERVED)
    missing = sorted(v for v in need if v not in corpus_vals)
    audit.add(
        "hidden_literals",
        not missing,
        "all hidden literals present in L0 corpus"
        if not missing
        else f"missing from L0 corpus: {[hex(v) for v in missing]}",
    )

    # --- gold_patch_literals --------------------------------------------------
    gold = unit_dir / "_author" / "gold.patch"
    new_consts: set[int] = set()
    if gold.is_file():
        for line in gold.read_text(encoding="utf-8").splitlines():
            if not line.startswith("+") or line.startswith("+++"):
                continue
            new_consts |= {int(x, 16) for x in HEX_RE.findall(line) if int(x, 16) > 0xFF}
            new_consts |= {int(x) for x in NUM_RE.findall(line) if int(x) > 255}
    leaked = sorted(v for v in new_consts if v not in corpus_vals)
    audit.add(
        "gold_patch_literals",
        not leaked,
        "gold.patch introduces no constants absent from L0"
        if not leaked
        else f"excised-body literals not in L0 corpus: {[hex(v) for v in leaked]}",
    )

    # --- coverage_table -------------------------------------------------------
    hidden_tests = TEST_FN_RE.findall(hidden_text)
    covered = set(COVERAGE_ROW_RE.findall(contract_text))
    uncovered = [t for t in hidden_tests if t not in covered]
    audit.add(
        "coverage_table",
        bool(hidden_tests) and not uncovered,
        f"{len(hidden_tests)} hidden tests, {len(covered)} coverage rows"
        if not uncovered
        else f"no coverage row for: {uncovered}",
    )

    # --- bugreport_floor (B6) --------------------------------------------------
    has_symptom = bool(re.search(r"(?i)expected.*got|got:|panic", bugreport_text))
    has_repro = "./repro.sh" in bugreport_text or "go test" in bugreport_text
    audit.add(
        "bugreport_floor",
        has_symptom and has_repro,
        f"symptom={has_symptom} repro_cmd={has_repro}",
    )

    # --- bugreport_ceiling (B7) ------------------------------------------------
    leaks = [
        s
        for s in manifest.get("excision", {}).get("S", [])
        if re.search(rf"\b{re.escape(s)}\b", bugreport_text)
    ]
    file_leaks = re.findall(r"\w+\.go\b", bugreport_text)
    line_leaks = bool(re.search(r"\.go:\d+|diff --git|^@@", bugreport_text, re.MULTILINE))
    audit.add(
        "bugreport_ceiling",
        not leaks and not file_leaks and not line_leaks,
        "no symbol/file/diff leaks"
        if not leaks and not file_leaks and not line_leaks
        else f"symbols={leaks} files={file_leaks} line_nos_or_diff={line_leaks}",
    )

    # --- examples_tied ---------------------------------------------------------
    smoke_files = sorted((unit_dir / "excised_tree").glob("*_test.go"))
    smoke_text = "\n".join(f.read_text(encoding="utf-8") for f in smoke_files)
    params_text = excised_params.read_text(encoding="utf-8") if excised_params.is_file() else ""
    pmap = _param_values(params_text)
    smoke_vals = _literals(smoke_text)
    smoke_vals |= {v for name, v in pmap.items() if re.search(rf"\b{re.escape(name)}\b", smoke_text)}
    smoke_vals |= {v + 1 for name, v in pmap.items() if re.search(rf"\b{re.escape(name)}\s*\+\s*1\b", smoke_text)}
    smoke_vals |= {v - 1 for name, v in pmap.items() if re.search(rf"\b{re.escape(name)}\s*-\s*1\b", smoke_text)}
    smoke_vals |= {v * 10 for name, v in pmap.items() if re.search(rf"\b{re.escape(name)}\s*\*\s*10\b", smoke_text)}
    in_examples = False
    missing_items: list[str] = []
    for line in bugreport_text.splitlines():
        if "worked examples" in line.lower():
            in_examples = True
            continue
        if in_examples and not line.strip().startswith("-"):
            in_examples = False
            continue
        if not in_examples:
            continue
        for s in QUOTED_RE.findall(line):
            if s and s not in smoke_text:
                missing_items.append(f'"{s}"')
        for n in NUM_RE.findall(line):
            v = int(n)
            if v == RESERVED:
                if not any(m in smoke_text for m in RESERVED_MARKERS):
                    missing_items.append(n)
            elif v not in smoke_vals:
                missing_items.append(n)
    audit.add(
        "examples_tied",
        bool(smoke_files) and not missing_items,
        "worked-example values present in smoke suite"
        if not missing_items
        else f"not asserted by smoke suite: {missing_items}",
    )

    # --- black_box (B4-light) ---------------------------------------------------
    pkg = manifest.get("domain") or ""
    bb_calls = re.findall(rf"\b{re.escape(pkg)}\.([a-z][A-Za-z0-9_]*)\(", hidden_text) if pkg else []
    audit.add(
        "black_box",
        not bb_calls,
        "hidden suite calls only exported API" if not bb_calls else f"unexported calls: {bb_calls}",
    )

    # --- details_manifest -------------------------------------------------------
    dial = manifest.get("dial") or {}
    details = manifest.get("details") or []
    if not dial or not details:
        audit.add("details_manifest", False, "manifest has no dial/details block")
    else:
        v = str(dial.get("v"))
        problems: list[str] = []
        smoke_files = sorted((unit_dir / "excised_tree").glob("*_test.go"))
        smoke_text = "\n".join(f.read_text(encoding="utf-8") for f in smoke_files)
        for d in details:
            did = str(d.get("id"))
            desc = str(d.get("description") or "")
            prose = str(d.get("prose") or "")
            stated = bool(d.get("stated_in_bugreport"))
            checkable = bool(d.get("checkable_from_repro"))
            covering = d.get("hidden_tests_covering") or []
            if not desc:
                problems.append(f"{did}: no description")
            if not prose or prose not in bugreport_text:
                problems.append(f"{did}: prose not stated in bugreport")
            if not stated:
                problems.append(f"{did}: stated_in_bugreport false")
            if checkable != (v == "high"):
                problems.append(f"{did}: checkable_from_repro={checkable} but v={v}")
            for t in covering:
                if not re.search(rf"^func {re.escape(str(t))}\s*\(", hidden_text, re.MULTILINE):
                    problems.append(f"{did}: hidden test {t} missing")
        audit.add(
            "details_manifest",
            not problems,
            "details block consistent" if not problems else "; ".join(problems[:5]),
        )

    # --- b3_not_complete_spec ---------------------------------------------------
    problems = []
    for d in details:
        did = str(d.get("id"))
        hidden_name = d.get("hidden_tests_covering") or []
        if not hidden_name:
            problems.append(f"{did}: no hidden test")
            continue
        m = re.search(rf"^func ({re.escape(str(hidden_name[0]))})\s*\(", hidden_text, re.MULTILINE)
        if not m:
            problems.append(f"{did}: hidden test not found")
            continue
        start = m.start()
        depth = 0
        end = len(hidden_text)
        for i in range(start, len(hidden_text)):
            if hidden_text[i] == "{":
                depth += 1
            elif hidden_text[i] == "}":
                depth -= 1
                if depth == 0:
                    end = i + 1
                    break
        hidden_body = hidden_text[start:end]
        smoke_name = f"TestSmoke_{did}"
        sm = re.search(rf"^func ({re.escape(smoke_name)})\s*\(", smoke_text, re.MULTILINE)
        shown_lits: set[str] = set()
        if sm:
            sstart = sm.start()
            depth = 0
            send = len(smoke_text)
            for i in range(sstart, len(smoke_text)):
                if smoke_text[i] == "{":
                    depth += 1
                elif smoke_text[i] == "}":
                    depth -= 1
                    if depth == 0:
                        send = i + 1
                        break
            shown_lits = _go_literals(smoke_text[sstart:send])
        hidden_lits = _go_literals(hidden_body)
        rng = "rand.New" in hidden_body
        extra = hidden_lits - shown_lits
        if not rng and not extra:
            problems.append(f"{did}: hidden test checks no input beyond the L0 examples")
    audit.add(
        "b3_not_complete_spec",
        not problems,
        "hidden tests check inputs beyond every L0 example"
        if not problems
        else "; ".join(problems[:5]),
    )
    return audit


def _go_literals(text: str) -> set[str]:
    """Numbers and quoted strings appearing in a Go source fragment."""
    out: set[str] = set()
    out |= set(QUOTED_RE.findall(text))
    for m in re.finditer(r"\b\d+\b", text):
        out.add(f"num:{m.group(0)}")
    for m in re.finditer(r"\bparam[A-Za-z0-9_]+", text):
        out.add(f"sym:{m.group(0)}")
    for m in re.finditer(r"\brefCap\b|\brefLimit\b|\brefBuckets\b", text):
        out.add(f"sym:{m.group(0)}")
    return out


def audit_units(paths: list[Path]) -> list[UnitAudit]:
    return [audit_unit(Path(p)) for p in paths]


def main_cli(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("units", nargs="+", type=Path, help="fabricated unit dirs (with excised_tree/, hidden/, _author/)")
    p.add_argument("--json", action="store_true", help="emit JSONL verdicts instead of a table")
    args = p.parse_args(argv)

    audits = audit_units(args.units)
    for a in audits:
        if args.json:
            print(json.dumps({"unit": a.unit_dir.name, "ok": a.ok,
                              "checks": {c.name: (c.passed, c.detail) for c in a.checks}}))
            continue
        print(f"{a.unit_dir.name}: {'PASS' if a.ok else 'FAIL'}")
        for c in a.checks:
            mark = "ok " if c.passed else "BAD"
            print(f"  [{mark}] {c.name}: {c.detail}")
    return 0 if all(a.ok for a in audits) else 1


if __name__ == "__main__":
    sys.exit(main_cli())
