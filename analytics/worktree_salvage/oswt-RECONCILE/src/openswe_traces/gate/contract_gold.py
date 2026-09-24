"""A13 — the contract must be consistent with gold (static tier).

Motivation: the 2026-09-19 solver-free audit
(``oswt-LIE/outputs/contract_gold_dump/``) found 20 of 50 bank units whose
contract misdescribes gold: 17 coverage rows that are false of
``gold.patch`` (11 units) and 18 hidden-suite assertions no coverage row or
contract sentence states (9 missing-only units, plus 7 of the false units).

Three defect classes:

- ``missing``: a hidden-suite assertion is not stated by any coverage row or
  contract sentence. Deterministic signal: hidden tests absent from the
  suite's own contract->property map, plus exported identifiers the hidden
  suite calls that gold.patch touches but the contract never mentions.
- ``false``: a coverage row's claim is untrue of ``gold.patch``.
  Deterministic signal: quoted/backticked identifiers or literals in the
  row that appear nowhere in the gold patch or the hidden suite.
- ``vague``: a row that names a topic rather than stating a checkable
  claim. Surfaced in evidence; never fails the gate.

The deterministic part runs inside the gate. The LLM-judged part lives in
``openswe_traces.contract_judge`` (driven by
``scripts/contract_consistency_audit.py``) and caches to
``outputs/contract_judge/<key>.json``; this module only reads that cache —
the gate itself never touches the network. A suspect row the judge has not
resolved keeps the unit failing: the check is fail-closed on unproven
suspects, fail-open never.
"""

from __future__ import annotations

import hashlib
import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable

from openswe_traces.data import ROOT
from openswe_traces.gate.context import GateContext
from openswe_traces.gate.core import STATIC, Verdict, _now_iso, register

RULE_ID = "A13"
PROV = "gate/static"

JUDGE_DIR = ROOT / "outputs" / "contract_judge"
PROMPT_VERSION = "v1"

# ---------------------------------------------------------------------------
# parsing
# ---------------------------------------------------------------------------

_ROW_RE = re.compile(r"^\|\s*`([^`]+)`\s*\|\s*(.*?)\s*\|\s*$")
_TEST_RE = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_FATAL_RE = re.compile(
    r"""t\.(?:Fatal|Fatalf|Error|Errorf)\(\s*(?:\"((?:\\.|[^\"])*)\"|`([^`]*)`)"""
)
_MAP_TESTS_RE = re.compile(r"Test[A-Z][A-Za-z0-9_]*")
# A contract-map entry opens with an optional label then a quoted phrase:
#   S4 "phrase"  /  C5 "phrase"  /  "phrase"
_MAP_OPEN_RE = re.compile(r'^(?:[A-Z]{1,2}\d+\s+)?["“]')
_MAP_ENTRY_RE = re.compile(
    r'^\s*(?:[A-Z]{1,2}\d+\s+)?["“](.+?)["”]\s*->\s*(.*)$', re.DOTALL
)
_IMPORT_RE = re.compile(
    r"""^\s*(?:(\w+)\s+)?["`]([^"`]+)["`]\s*$""", re.MULTILINE
)
_SELECTOR_RE = re.compile(r"\b([a-z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\b")
_QUOTED_RE = re.compile(r"`([^`\n]{2,})`|\"([^\"\n]{2,})\"")
_IDENT_TOKEN_RE = re.compile(r"\b[A-Za-z_][A-Za-z0-9_]*\b")

# contract-sentence markers that say "this row asserts behavior"
_CLAIM_WORDS = re.compile(
    r"\b(?:is|are|must|only|never|no |not |error|errors|reject|fail|return|skip|"
    r"drop|preserve|sort|order|overwrite|override|strip|trim|tolerat|produc|"
    r"allow|panic|close|fill|requir|decod|encod|valid|merge|propagat|select|"
    r"pick|cap|bound|truncat|join|append|dedup|replac|isolat|terminal|recover|"
    r"follow|contain|equal|match|round.?trip|omit|empty|truncat|retry|retries|"
    r"in progress|first|before|after|lexical|stable|wholesale)\w*",
    re.IGNORECASE,
)
# topic-list tells: a row that enumerates what is exercised, not what holds
_VAGUE_TELLS = re.compile(
    r"\b(?:etc\.?|various|including|semantics|boundaries|wholesale|"
    r"the same rules|correctness|and friends|e\.g\.|per contract)\b",
    re.IGNORECASE,
)


@dataclass(frozen=True)
class CoverageRow:
    test: str
    sentence: str


@dataclass
class HiddenTest:
    name: str
    fatals: list[str] = field(default_factory=list)
    called: set[str] = field(default_factory=set)  # exported idents via pkg.Ident
    body: str = ""  # source excerpt; attached only for the judge prompt


@dataclass(frozen=True)
class MapEntry:
    phrase: str
    tests: tuple[str, ...]


@dataclass
class UnitEvidence:
    """Everything A13 needs out of one packaged task dir."""

    repo: str
    unit: str
    level: int | None
    rows: list[CoverageRow]
    prose: str
    hidden: list[HiddenTest]
    hidden_text: str
    map_entries: list[MapEntry]
    gold: str
    contract_key: str


@dataclass(frozen=True)
class Finding:
    cls: str  # missing | false | vague | suspect
    subject: str
    detail: str
    source: str  # det | judge


# ---------------------------------------------------------------------------
# extraction
# ---------------------------------------------------------------------------


def parse_coverage_rows(instruction: str) -> list[CoverageRow]:
    """``| `Test` | claim |`` rows — the first column is the *original* test.

    Some suites key rows by source file (``2pc_test.go committer suite``), not
    by ``Test*`` name; the first column is a label, so any backticked label in
    a coverage row counts.
    """
    rows: list[CoverageRow] = []
    for line in instruction.splitlines():
        m = _ROW_RE.match(line.strip())
        if m:
            rows.append(CoverageRow(test=m.group(1), sentence=m.group(2)))
    return rows


def contract_prose(instruction: str) -> str:
    """Instruction minus the coverage table, reproduce block, and boilerplate."""
    out: list[str] = []
    for line in instruction.splitlines():
        s = line.strip()
        if s.startswith("|"):
            continue
        if s.startswith("## Hidden unit tests"):
            break
        out.append(line)
    text = "\n".join(out)
    for marker in ("Reproduce with:", "IMPORTANT: This repository"):
        i = text.find(marker)
        if i >= 0:
            text = text[:i]
    return text.strip()


def parse_contract_map(hidden_text: str) -> list[MapEntry]:
    """`"phrase" -> TestA / TestB` entries from the header comment block.

    Entries wrap: the quoted phrase may span several comment lines and the
    ``-> Tests`` tail lands on a continuation line. Anything the author noted
    after the test names (``. NOTE: ...``) is kept out of the test list.
    """
    entries: list[MapEntry] = []
    cur = ""

    def flush() -> None:
        nonlocal cur
        if "->" in cur:
            m = _MAP_ENTRY_RE.match(cur)
            if m:
                tests = tuple(_MAP_TESTS_RE.findall(m.group(2)))
                if tests:
                    entries.append(MapEntry(phrase=m.group(1), tests=tests))
        cur = ""

    for raw in hidden_text.splitlines():
        s = raw.strip()
        if not s.startswith("//"):
            flush()
            continue
        body = s[2:].strip()
        if _MAP_OPEN_RE.match(body):
            flush()
            cur = body
        elif cur:
            cur += " " + body
    flush()
    return entries


def _package_aliases(text: str) -> set[str]:
    """Import aliases usable as ``alias.Ident`` selectors in this file."""
    out: set[str] = set()
    in_block = False
    for line in text.splitlines():
        s = line.strip()
        if s.startswith("import ("):
            in_block = True
            continue
        if in_block and s == ")":
            in_block = False
            continue
        if in_block or s.startswith("import "):
            m = _IMPORT_RE.match(s if in_block else s[len("import "):])
            if not m:
                continue
            alias, path = m.group(1), m.group(2)
            if alias and alias not in {"_", "."}:
                out.add(alias)
            elif not alias:
                out.add(path.rstrip("/").split("/")[-1])
    return out


def parse_hidden_tests(hidden_files: dict[str, str]) -> list[HiddenTest]:
    out: list[HiddenTest] = []
    for rel in sorted(hidden_files):
        text = hidden_files[rel]
        aliases = _package_aliases(text)
        funcs = list(_TEST_RE.finditer(text))
        for i, m in enumerate(funcs):
            end = funcs[i + 1].start() if i + 1 < len(funcs) else len(text)
            body = text[m.start() : end]
            fatals = [
                (fm.group(1) if fm.group(1) is not None else fm.group(2))[:200]
                for fm in _FATAL_RE.finditer(body)
            ]
            called = {
                ident
                for alias, ident in _SELECTOR_RE.findall(body)
                if alias in aliases
            }
            out.append(HiddenTest(name=m.group(1), fatals=fatals[:12], called=called))
    return out


def _gold_text(task_dir: Path) -> str:
    for cand in (
        task_dir / "tests" / "gold.patch",
        task_dir / "patches" / "gold.patch",
    ):
        if cand.is_file():
            return cand.read_text(encoding="utf-8", errors="replace")
    return ""


def _unit_repo(task_dir: Path) -> tuple[str, str]:
    name = task_dir.name
    unit = re.sub(r"-L\d+$", "", name)
    return task_dir.parent.name, unit


def contract_key(repo: str, unit: str, prose: str, rows: list[CoverageRow], gold: str, hidden: Iterable[HiddenTest]) -> str:
    h = hashlib.sha256()
    h.update(repo.encode() + b"\0" + unit.encode() + b"\0")
    h.update(prose.encode())
    for r in rows:
        h.update(b"\0" + r.test.encode() + b"\0" + r.sentence.encode())
    h.update(b"\0gold:" + hashlib.sha256(gold.encode()).digest())
    for t in hidden:
        h.update(b"\0t:" + t.name.encode())
    h.update(b"\0" + PROMPT_VERSION.encode())
    return h.hexdigest()


def gather_evidence(ctx: GateContext) -> UnitEvidence:
    r = ctx.rules
    repo, unit = _unit_repo(ctx.task_dir)
    instruction = r.instruction or ""
    rows = parse_coverage_rows(instruction)
    prose = contract_prose(instruction)
    hidden_files = r.hidden_files or {}
    hidden = parse_hidden_tests(hidden_files)
    hidden_text = "\n".join(hidden_files.values())
    maps: list[MapEntry] = []
    for text in hidden_files.values():
        maps.extend(parse_contract_map(text))
    gold = _gold_text(ctx.task_dir)
    key = contract_key(repo, unit, prose, rows, gold, hidden)
    return UnitEvidence(
        repo=repo,
        unit=unit,
        level=ctx.level,
        rows=rows,
        prose=prose,
        hidden=hidden,
        hidden_text=hidden_text,
        map_entries=maps,
        gold=gold,
        contract_key=key,
    )


# ---------------------------------------------------------------------------
# deterministic findings
# ---------------------------------------------------------------------------


def _contract_text(ev: UnitEvidence) -> str:
    return ev.prose + "\n" + "\n".join(r.sentence for r in ev.rows)


def _row_tokens(row: CoverageRow) -> set[str]:
    """Quoted/backticked identifiers and literals asserted by a row."""
    out: set[str] = set()
    for m in _QUOTED_RE.finditer(row.sentence):
        out.add((m.group(1) or m.group(2) or "").strip())
    out.update(
        t for t in _IDENT_TOKEN_RE.findall(row.sentence)
        if any(c.isupper() for c in t) and len(t) >= 3
    )
    return {t for t in out if t and not t.isspace()}


def is_vague(row: CoverageRow) -> bool:
    """Syntactic tells only — a hedge marker or a comma-enumerated topic list
    with no behavioral claim. Semantic vagueness is the judge's call; this
    intentionally under-reports (vague never fails the gate)."""
    s = row.sentence
    if _VAGUE_TELLS.search(s):
        return True
    if s.count(",") >= 2 and not _CLAIM_WORDS.search(s):
        return True
    return False


# Exported idents too generic to prove a coverage gap on their own —
# `storage.Set` failing on nil is a real assertion, but the *name* tells you
# nothing the contract would say.
_GENERIC_IDENTS = {
    "New", "Get", "Set", "Add", "List", "Query", "Delete", "Update", "Create",
    "Close", "String", "Error", "Equal", "Contains", "HasPrefix", "Buffer",
    "Marshal", "Unmarshal", "Render", "Write", "Read", "Load", "Save", "Parse",
    "Sort", "Run", "Open", "Reset", "Copy", "Len", "Keys", "Values", "Next",
    "Seek", "Sum", "Format", "Encode", "Decode", "Validate", "Check", "Lock",
    "Unlock", "Wait", "Done", "Stringer", "Round", "Int", "Bool", "Map",
}
_MIN_IDENT_LEN = 7


def deterministic_findings(ev: UnitEvidence, excised: Iterable[str] = ()) -> list[Finding]:
    findings: list[Finding] = []
    contract = _contract_text(ev)
    gold = ev.gold
    mapped_tests = {t for e in ev.map_entries for t in e.tests}

    # Generic fuzz/property loops (``*Random*``, ``*Simple*``, ``*Fuzz*``)
    # stress the whole surface; they are often deliberately unmapped. A mapped
    # suite leaving a *specific* test unmapped is only a coverage *suspect* —
    # the contract prose may still cover it (bindingdispatch proved this).
    fuzz_name = re.compile(r"(?:Unseen|Random|Simple|Smoke|Fuzz)", re.IGNORECASE)
    if ev.map_entries:
        for t in ev.hidden:
            if t.name in mapped_tests or fuzz_name.search(t.name):
                continue
            findings.append(
                Finding(
                    "suspect", t.name,
                    "hidden test has no entry in the suite's contract->property map",
                    "det",
                )
            )
    # excised symbols the hidden suite calls that the contract never names:
    # the suite exercises behavior the contract does not describe.
    excised_names = set(excised)
    contract_idents = set(_IDENT_TOKEN_RE.findall(contract))
    for t in ev.hidden:
        for ident in sorted(t.called & excised_names - contract_idents):
            if len(ident) < _MIN_IDENT_LEN or ident in _GENERIC_IDENTS:
                continue
            findings.append(
                Finding(
                    "suspect", f"{t.name}:{ident}",
                    f"excised `{ident}` exercised by {t.name} "
                    "but never named in the contract",
                    "det",
                )
            )
    # row tokens that appear nowhere in gold or the hidden suite
    for row in ev.rows:
        toks = _row_tokens(row)
        if not toks:
            continue
        missing_toks = sorted(
            tok for tok in toks
            if tok not in gold and tok not in ev.hidden_text and tok != row.test
        )
        if missing_toks:
            findings.append(
                Finding(
                    "suspect", row.test,
                    f"row asserts {', '.join(missing_toks[:4])} — absent from "
                    "gold.patch and the hidden suite",
                    "det",
                )
            )
    for row in ev.rows:
        if is_vague(row):
            findings.append(Finding("vague", row.test, row.sentence[:160], "det"))
    return findings


# ---------------------------------------------------------------------------
# judge cache (read-only here)
# ---------------------------------------------------------------------------


def judge_cache_path(key: str, cache_dir: Path | None = None) -> Path:
    return (cache_dir or JUDGE_DIR) / f"{key}.json"


def load_judgement(ev: UnitEvidence, cache_dir: Path | None = None) -> dict[str, Any] | None:
    path = judge_cache_path(ev.contract_key, cache_dir)
    if not path.is_file():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    return data if isinstance(data, dict) else None


def judge_findings(judgement: dict[str, Any]) -> list[Finding]:
    out: list[Finding] = []
    for row in judgement.get("rows") or []:
        if not isinstance(row, dict):
            continue
        verdict = str(row.get("verdict") or "")
        subject = str(row.get("test") or row.get("row") or "")[:80]
        reason = str(row.get("reason") or "")[:300]
        if verdict == "false":
            out.append(Finding("false", subject, f"judge: {reason}", "judge"))
        elif verdict == "vague":
            out.append(Finding("vague", subject, f"judge: {reason}", "judge"))
    for m in judgement.get("missing") or []:
        if isinstance(m, dict):
            subject = str(m.get("test") or "")
            detail = str(m.get("assertion") or "")
        else:
            subject, detail = "", str(m)
        out.append(Finding("missing", subject or "hidden suite", f"judge: {detail[:300]}", "judge"))
    return out


# ---------------------------------------------------------------------------
# the gate impl
# ---------------------------------------------------------------------------


def check_contract_gold(ctx: GateContext) -> Verdict:
    ev = gather_evidence(ctx)
    if not ev.rows and not ev.hidden:
        return Verdict(
            RULE_ID, True, True, STATIC,
            "no contract coverage table and no hidden suite — nothing to check",
            _now_iso(), PROV,
        )
    if not ev.rows:
        # A contract-less level (L0) still ships hidden tests; with no contract
        # on disk there is nothing to be consistent with.
        return Verdict(
            RULE_ID, True, True, STATIC,
            "no coverage table at this level (bug-report instruction)",
            _now_iso(), PROV,
        )

    det = deterministic_findings(ev, ctx.excised.bare_names)
    judgement = load_judgement(ev)
    jfinds = judge_findings(judgement) if judgement else []

    judged_rows = {
        str(r.get("test") or ""): str(r.get("verdict") or "")
        for r in (judgement or {}).get("rows") or []
        if isinstance(r, dict)
    }
    judged_missing_tests = {
        str(m.get("test") or "")
        for m in (judgement or {}).get("missing") or []
        if isinstance(m, dict)
    }

    missing = [f for f in det if f.cls == "missing"] + [
        f for f in jfinds if f.cls == "missing"
    ]
    false = [f for f in jfinds if f.cls == "false"]
    vague = [f for f in det + jfinds if f.cls == "vague"]

    # A det suspect fails-closed unless the judge ran and resolved it:
    #   - ``TestX:Ident`` (uncovered excised symbol) and unmapped ``TestX``
    #     resolve when the judge's missing list does not name the test;
    #   - a row-token suspect (subject is the row's original test) resolves
    #     when the judge returned any verdict for that row — a "false"
    #     verdict already lands in `false`.
    hidden_names = {t.name for t in ev.hidden}
    suspects: list[Finding] = []
    for f in det:
        if f.cls != "suspect":
            continue
        if judgement is not None:
            base = f.subject.split(":", 1)[0]
            if ":" in f.subject or base in hidden_names:
                if base not in judged_missing_tests:
                    continue
            elif base in judged_rows:
                continue
        suspects.append(f)

    problems: list[str] = []
    problems += [f"missing {f.subject}: {f.detail}" for f in missing]
    problems += [f"false {f.subject}: {f.detail}" for f in false]
    problems += [f"unresolved {f.subject}: {f.detail}" for f in suspects]

    vague_note = ""
    if vague:
        vague_note = " | vague rows surfaced: " + "; ".join(
            f"{f.subject} ({f.detail[:60]})" for f in vague[:6]
        )
    judge_note = "judge: cached verdicts applied" if judgement else "judge: no cache entry"
    passed = not problems
    if passed:
        evidence = (
            f"ok: {len(ev.rows)} rows, {len(ev.hidden)} hidden tests, "
            f"{len(ev.map_entries)} map entries; {judge_note}{vague_note}"
        )
    else:
        evidence = "; ".join(problems)[:1400] + f" | {judge_note}{vague_note}"
    return Verdict(RULE_ID, passed, False, STATIC, evidence[:2000], _now_iso(), PROV)


register(
    RULE_ID,
    STATIC,
    provenance=PROV,
    description="contract coverage rows true of gold; hidden assertions covered",
)(check_contract_gold)
