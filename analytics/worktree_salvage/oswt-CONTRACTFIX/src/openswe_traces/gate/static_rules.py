"""Static-tier rules: packaging facts derivable from the task dir alone.

- ``coverage``: every hidden test is wired into ``tests/test.sh`` and, when the
  package declares a coverage table (validation.json ``coverage`` entries or
  ``| TestX |`` rows in the instruction), every hidden test has a row.
- ``nesting``: across ``<unit>-L<n>`` siblings, hidden tests are identical
  (checksum) and the instruction's information set is monotone non-decreasing
  with level — L0's instruction ⊆ L2's.
"""

from __future__ import annotations

import hashlib
import re
from pathlib import Path
from typing import Any

from openswe_traces.gate.context import GateContext, level_dirs
from openswe_traces.gate.core import STATIC, Verdict, _now_iso, register

_FUNC_RE = re.compile(
    r"(?:^|\s)func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?(Test[A-Za-z0-9_]*)"
    r"|(?:^|\s)def\s+(test_[A-Za-z0-9_]*)"
    r"|(?:it|test)\(\s*['\"]([^'\"]+)['\"]",
    re.MULTILINE,
)
_RUN_RE = re.compile(r"-run[=\s]+['\"]?([A-Za-z0-9_^$|/()'\"]+)")
_TEST_NAME_RE = re.compile(r"\b(Test[A-Z][A-Za-z0-9_]*|test_[a-z][A-Za-z0-9_]*)\b")
_CODE_TOKEN_RE = re.compile(r"`([^`\n]+)`|([A-Za-z_][A-Za-z0-9_]*(?:[./][A-Za-z0-9_./-]+)+)")


def _hidden_test_names(hidden_files: dict[str, str]) -> set[str]:
    names: set[str] = set()
    for text in hidden_files.values():
        for m in _FUNC_RE.finditer(text):
            names.add(next(g for g in m.groups() if g))
    return names


def _wired_names(test_sh: str) -> set[str] | None:
    """Names the suite runs explicitly, or None when it runs whole packages."""
    names: set[str] = set()
    for m in _RUN_RE.finditer(test_sh):
        pat = m.group(1).strip("'\"")
        names.update(t for t in re.split(r"[|^$()/]", pat) if t and t[:1].isalpha())
    names.update(re.findall(r"\b(?:Test[A-Z][A-Za-z0-9_]*|test_[a-z][A-Za-z0-9_]*)\b", test_sh))
    return names or None


def _declared_coverage(task_dir: Path, instruction: str, validation: dict[str, Any]) -> set[str]:
    """Test names declared in a coverage table row or validation.json coverage."""
    declared: set[str] = set()
    for line in instruction.splitlines():
        if line.strip().startswith("|"):
            declared.update(_TEST_NAME_RE.findall(line))
    for row in validation.get("coverage") or []:
        if isinstance(row, dict):
            val = row.get("property") or row.get("test") or ""
        else:
            val = str(row)
        if val:
            declared.update(_TEST_NAME_RE.findall(str(val)) or [str(val)])
    return declared


def check_coverage(ctx: GateContext) -> Verdict:
    r = ctx.rules
    hidden = _hidden_test_names(r.hidden_files)
    if not hidden:
        return Verdict(
            "coverage", True, True, STATIC,
            "no hidden tests under tests/hidden", _now_iso(), "gate/static",
        )
    problems: list[str] = []
    wired = _wired_names(r.test_sh or "")
    if wired is not None:
        missing = sorted(hidden - wired)
        if missing:
            problems.append(f"hidden tests not wired into test.sh: {', '.join(missing)}")
    declared = _declared_coverage(ctx.task_dir, r.instruction, r.validation)
    if declared:
        uncovered = sorted(hidden - declared)
        if uncovered:
            problems.append(f"coverage table lacks rows for: {', '.join(uncovered)}")
        extra = sorted(d for d in declared if d not in hidden and d.startswith(("Test", "test_")))
        if extra:
            problems.append(f"coverage rows with no hidden test: {', '.join(extra)}")
    passed = not problems
    evidence = (
        "ok: " + "; ".join(sorted(hidden))[:400]
        if passed
        else "; ".join(problems)[:1500]
    )
    return Verdict("coverage", passed, False, STATIC, evidence, _now_iso(), "gate/static")


def _hidden_digest(task_dir: Path) -> str:
    h = hashlib.sha256()
    for root in (task_dir / "tests" / "hidden", task_dir / "tests" / "test.sh"):
        if root.is_file():
            h.update(root.name.encode() + b"\0" + hashlib.sha256(root.read_bytes()).digest())
        elif root.is_dir():
            for p in sorted(root.rglob("*")):
                if p.is_file():
                    h.update(p.relative_to(root).as_posix().encode())
                    h.update(b"\0")
                    h.update(hashlib.sha256(p.read_bytes()).digest())
    return h.hexdigest()


def _info_tokens(instruction: str) -> set[str]:
    """Code-ish tokens an instruction reveals: backtick spans + paths/idents."""
    out: set[str] = set()
    for m in _CODE_TOKEN_RE.finditer(instruction):
        tok = (m.group(1) or m.group(2) or "").strip()
        for piece in re.split(r"[\s,;:()\[\]{}'\"]+", tok):
            piece = piece.strip().strip("`").removeprefix("./").rstrip("/")
            if len(piece) >= 3 and (
                "_" in piece or "/" in piece or "." in piece or any(c.isupper() for c in piece)
            ):
                out.add(piece)
    return out


def check_nesting(ctx: GateContext) -> Verdict:
    siblings = level_dirs(ctx.task_dir)
    if len(siblings) < 2:
        return Verdict(
            "nesting", True, True, STATIC,
            "single packaged level — nothing to nest", _now_iso(), "gate/static",
        )
    problems: list[str] = []
    digests: dict[int, str] = {}
    infos: dict[int, set[str]] = {}
    for level, d in siblings:
        digests[level] = _hidden_digest(d)
        infos[level] = _info_tokens(
            (d / "instruction.md").read_text(encoding="utf-8", errors="replace")
            if (d / "instruction.md").is_file()
            else ""
        )
    levels = sorted(digests)
    base = digests[levels[0]]
    for lv in levels[1:]:
        if digests[lv] != base:
            problems.append(
                f"hidden tests differ across levels: L{levels[0]} sha {base[:12]} "
                f"!= L{lv} sha {digests[lv][:12]}"
            )
    for i in range(len(levels) - 1):
        lo, hi = levels[i], levels[i + 1]
        hi_exact = {t.lower() for t in infos[hi]}
        hi_pieces = {
            p.lower()
            for t in infos[hi]
            for p in re.split(r"[\s,;:()\[\]{}'\"/]+", t)
            if len(p) >= 3
        }
        lost = sorted(
            t
            for t in infos[lo]
            if t.lower() not in hi_exact
            and not all(
                p.lower() in hi_pieces
                for p in t.split("/")
                if len(p) >= 3
            )
        )
        if lost:
            problems.append(
                f"L{hi} instruction drops information L{lo} revealed: {', '.join(lost[:12])}"
            )
    passed = not problems
    evidence = (
        f"monotone across levels {levels}; hidden sha {base[:12]}"
        if passed
        else "; ".join(problems)[:1500]
    )
    return Verdict("nesting", passed, False, STATIC, evidence, _now_iso(), "gate/static")


_LINE_REF_RE = re.compile(r"([A-Za-z0-9_./-]+\.(?:go|py|rs|ts|java|tsx|js)):(\d+)")
_SYMBOL_SHAPED_RE = re.compile(r"\.|[a-z]_[a-z]|[a-z][A-Z]|[A-Z][a-z]+[A-Z]")


def check_b7(ctx: GateContext) -> Verdict:
    """Instruction ceiling: no symbol, file, line-number, or diff leaks.

    Contract-level instructions (a coverage table is present) sanction API
    naming — the contract is the information channel — so only the declared
    ``changed_*`` fields count as leaks there. Bug-report instructions get the
    full excision-derived check: any mention of an excised symbol, a touched
    file, a ``file:N`` line reference, or a diff fails.
    """
    from openswe_traces.gate.contract_gold import parse_coverage_rows

    r = ctx.rules
    instruction = r.instruction or ""
    if not instruction:
        return Verdict(
            "B7", True, True, STATIC, "no instruction.md", _now_iso(), "gate/static"
        )
    leaks: list[str] = []
    if parse_coverage_rows(instruction):
        symbols = {
            str(s)
            for s in (
                r.validation.get("changed_symbols")
                or ctx.extra.get("changed_symbols")
                or []
            )
            if s
        }
        files = {
            str(f)
            for f in (
                r.validation.get("changed_files")
                or ctx.extra.get("changed_files")
                or []
            )
            if f
        }
    else:
        info = ctx.excised
        # Unqualified excised symbols count only when symbol-shaped (qualified
        # ``T.Fn``, snake_case, or interior-cased ``SecretFunc``); single-case
        # words like ``Do``/``GET``/``Create`` are ordinary English at L0.
        symbols = {s for s in info.symbols if _SYMBOL_SHAPED_RE.search(s)}
        symbols.update(
            str(s)
            for s in (
                r.validation.get("changed_symbols")
                or ctx.extra.get("changed_symbols")
                or []
            )
            if s
        )
        files = set(info.files)
        removed = {(Path(f).name, n) for f, n in info.removed_lines}
        for m in _LINE_REF_RE.finditer(instruction):
            base = Path(m.group(1)).name
            if (base, int(m.group(2))) in removed:
                leaks.append(f"line:{m.group(1)}:{m.group(2)}")
        if re.search(r"\.go:\d+", instruction) and not any(
            x.startswith("line:") for x in leaks
        ):
            leaks.append("line-nos")
    for s in sorted(symbols):
        if re.search(rf"\b{re.escape(s)}\b", instruction):
            leaks.append(f"symbol:{s}")
    for f in sorted(files):
        base = Path(f).name
        if base and base.lower() in instruction.lower():
            leaks.append(f"file:{f}")
    if "diff --git" in instruction or instruction.strip().startswith("@@"):
        leaks.append("diff")
    passed = not leaks
    evidence = (
        "no symbol/file/line/diff leaks"
        if passed
        else "; ".join(dict.fromkeys(leaks))[:1500]
    )
    return Verdict("B7", passed, False, STATIC, evidence, _now_iso(), "gate/static")


register("coverage", STATIC, provenance="gate/static", description="every hidden test wired + covered")(check_coverage)
register("nesting", STATIC, provenance="gate/static", description="level info monotone; hidden tests identical")(check_nesting)
register("B7", STATIC, provenance="gate/static", description="no symbol/file/line/diff leaks in the instruction")(check_b7)
