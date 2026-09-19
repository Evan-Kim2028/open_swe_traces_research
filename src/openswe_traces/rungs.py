"""Heuristic affordance-ladder (L0–L6) mapping for Open-SWE-Traces task text.

Maps the PR/issue description the agent receives onto the Harbor ladder defined in
``analytics/research/verifier_rules.md`` (L0 symptom-only … L6 all tests in tree).
Transparent regex/heuristic features only — no model calls.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

# --- harness boilerplate -------------------------------------------------------

_PR_OPEN_RE = re.compile(
    r"<pr_description>\s*(?:Consider the following PR description:\s*)?",
    re.IGNORECASE,
)
_PR_CLOSE_RE = re.compile(r"</pr_description>", re.IGNORECASE)
_INSTR_OPEN_RE = re.compile(r"<instructions>", re.IGNORECASE)
_UPLOAD_OPEN_RE = re.compile(r"<uploaded_files>", re.IGNORECASE)
_ISSUE_OPEN_RE = re.compile(r"<issue_description>", re.IGNORECASE)
_ISSUE_CLOSE_RE = re.compile(r"</issue_description>", re.IGNORECASE)
_HARNESS_SIG_RE = re.compile(
    r"\n+New interfaces introduced:\s*.*$",
    re.IGNORECASE | re.DOTALL,
)

# --- feature patterns ----------------------------------------------------------

_REPRO_RE = re.compile(
    r"(?:"
    r"steps?\s+to\s+reproduce|how\s+to\s+reproduce|reproduction\s+steps|"
    r"reproduce\s+with|to\s+reproduce|repro(?:duce|duction)\b|"
    r"run\s+(?:the\s+)?(?:following\s+)?(?:command|test)s?"
    r")",
    re.IGNORECASE,
)
_REPRO_CMD_RE = re.compile(
    r"(?:```[^\n]*\n[^`]*(?:pytest|go\s+test|cargo\s+test|npm\s+(?:run\s+)?test|"
    r"mvn\s+test|make\s+test|ctest|bundle\s+exec\s+rspec|phpunit|jest)[^`]*```|"
    r"(?:^|\n)\s*(?:pytest|go\s+test|cargo\s+test|npm\s+(?:run\s+)?test|"
    r"mvn\s+test|make\s+test)\b)",
    re.IGNORECASE | re.MULTILINE,
)
_EXPECTED_ACTUAL_RE = re.compile(
    r"(?:"
    r"expected\s+(?:behavior|result|output|value)|observed\s+behavior|"
    r"actual\s+(?:behavior|result|output|value)|"
    r"should\s+(?:be|have|return|not)|instead\s+of|"
    r"expected\b.{0,40}\bactual\b|\bactual\b.{0,40}\bexpected\b|"
    r"works?\s+correctly|does\s+not\s+work|fails?\s+with|"
    r"error\s+message|incorrect(?:ly)?|wrong\s+(?:value|result|output)"
    r")",
    re.IGNORECASE,
)
_STACK_TRACE_RE = re.compile(
    r"(?:"
    r"traceback|stack\s*trace|exception\s+trace|"
    r"(?:Error|Exception|FAILED|AssertionError|TypeError|ValueError|"
    r"AttributeError|RuntimeError|KeyError|IndexError)\s*:|"
    r"at\s+[\w./]+\.(?:py|go|rs|java|js|ts|rb|php):\d+|"
    r"File\s+\"[^\"]+\",\s+line\s+\d+"
    r")",
    re.IGNORECASE,
)
_GO_TEST_NAME_RE = re.compile(r"\bTest[A-Z][A-Za-z0-9_]*\b")
_PY_TEST_NAME_RE = re.compile(r"\btest_[a-z][a-z0-9_]*\b")
_OTHER_TEST_NAME_RE = re.compile(
    r"\b(?:it|describe|test)\s*\(\s*['\"][^'\"]+['\"]",
    re.IGNORECASE,
)
_SIGNATURE_RE = re.compile(
    r"(?:"
    r"\b(?:func|def|fn|pub\s+fn|function|interface|type|struct|class|enum)\s+\w+|"
    r"\b(?:public|private|protected|export|exported)\s+(?:func|def|function|interface|class)|"
    r"\bsignature\b|\bapi\s+(?:surface|contract)\b|"
    r"```(?:go|python|typescript|javascript|rust|java)\s*\n\s*(?:func|def|interface|type|class)\b"
    r")",
    re.IGNORECASE,
)
_TEST_CODE_RE = re.compile(
    r"(?:"
    r"func\s+Test[A-Z]\w*\s*\(|"
    r"def\s+test_[a-z]\w*\s*\(|"
    r"@Test\b|@pytest\.mark|"
    r"\b(?:assert|expect)\s*[\(\.]|"
    r"\bdescribe\s*\(|"
    r"it\s*\(\s*['\"]"
    r")",
    re.IGNORECASE,
)
_TEST_FILE_RE = re.compile(
    r"[\w./\\-]+(?:_test\.go|\.test\.(?:js|ts|tsx)|_test\.py|Test\.java|_spec\.(?:js|ts|rb))",
    re.IGNORECASE,
)
_DIFF_FILE_RE = re.compile(r"diff --git a/([^\s]+) b/([^\s]+)", re.MULTILINE)
_PLUS_FILE_RE = re.compile(r"\+\+\+ b/([^\s]+)", re.MULTILINE)
_HUNK_FUNC_RE = re.compile(
    r"^[+-]?(?:func|def|class|interface|type|struct)\s+([A-Za-z_][\w]*)",
    re.MULTILINE,
)
_IDENTIFIER_RE = re.compile(r"\b[A-Za-z_][A-Za-z0-9_]{2,}\b")

_TEST_PATH_MARKERS = (
    "_test.go",
    "_test.py",
    ".test.",
    "_spec.",
    "test/",
    "tests/",
    "/test_",
)


@dataclass(frozen=True)
class RungFeatures:
    word_count: int
    has_repro: bool
    has_expected_actual: bool
    has_stack_trace: bool
    has_test_names: bool
    has_signature: bool
    has_leakage: bool
    leakage_count: int
    has_test_code: bool
    n_test_funcs_in_text: int
    n_test_files_in_text: int
    patch_has_tests: bool


@dataclass(frozen=True)
class RungResult:
    rung: int
    features: RungFeatures
    task_text: str
    leakage_symbols: tuple[str, ...] = field(default_factory=tuple)


def strip_task_text(raw: str | None) -> str:
    """Extract the PR/issue body; drop harness ``<instructions>`` boilerplate."""
    if not raw:
        return ""
    text = raw.strip()
    upload = _UPLOAD_OPEN_RE.search(text)
    if upload:
        text = text[upload.end() :]
        issue = _ISSUE_OPEN_RE.search(text)
        if issue:
            text = text[issue.end() :]
            close = _ISSUE_CLOSE_RE.search(text)
            if close:
                text = text[: close.start()]
    m = _PR_OPEN_RE.search(text)
    if m:
        text = text[m.end() :]
        close = _PR_CLOSE_RE.search(text)
        if close:
            text = text[: close.start()]
    else:
        instr = _INSTR_OPEN_RE.search(text)
        if instr:
            text = text[: instr.start()]
    text = _PR_CLOSE_RE.sub("", text)
    text = _HARNESS_SIG_RE.sub("", text)
    text = re.sub(
        r"^Consider the following PR description:\s*",
        "",
        text,
        count=1,
        flags=re.IGNORECASE,
    )
    text = re.sub(
        r"^I've uploaded a \w+ code repository in the directory [^\n]+\.\s*"
        r"Consider the following issue description:\s*",
        "",
        text,
        count=1,
        flags=re.IGNORECASE,
    )
    return text.strip()


def _word_count(text: str) -> int:
    return len(re.findall(r"\b\w+\b", text))


def _test_names_in_text(text: str) -> list[str]:
    names: list[str] = []
    seen: set[str] = set()
    for m in _GO_TEST_NAME_RE.finditer(text):
        name = m.group(0)
        if name not in seen:
            seen.add(name)
            names.append(name)
    for m in _PY_TEST_NAME_RE.finditer(text):
        name = m.group(0)
        # Skip file-path fragments like tests/test_foo.py (not a named check).
        start = m.start()
        if start > 0 and text[start - 1] in "/\\":
            continue
        end = m.end()
        if end < len(text) and text[end] == ".":
            continue
        if name not in seen:
            seen.add(name)
            names.append(name)
    return names


def _test_funcs_in_text(text: str) -> int:
    count = len(re.findall(r"func\s+Test[A-Z]\w*\s*\(", text))
    count += len(re.findall(r"def\s+test_[a-z]\w*\s*\(", text, flags=re.IGNORECASE))
    return count


def _has_substantial_test_body(text: str) -> bool:
    """True when the text includes a multi-assertion test body (L5/L6), not a repro snippet."""
    if not _TEST_CODE_RE.search(text):
        return False
    asserts = len(re.findall(r"\b(?:assert|require\.|expect\()", text, flags=re.IGNORECASE))
    funcs = _test_funcs_in_text(text)
    lines = [ln for ln in text.splitlines() if ln.strip()]
    return (funcs >= 2 and asserts >= 2) or (funcs >= 1 and asserts >= 3 and len(lines) >= 15)


def _test_files_in_text(text: str) -> int:
    return len(set(_TEST_FILE_RE.findall(text)))


def extract_patch_symbols(reference_patch: str | None) -> set[str]:
    """File paths and changed symbol names from a unified diff."""
    if not reference_patch:
        return set()
    symbols: set[str] = set()
    for m in _DIFF_FILE_RE.finditer(reference_patch):
        for path in (m.group(1), m.group(2)):
            symbols.add(path)
            symbols.add(path.split("/")[-1])
            stem = re.sub(r"\.[^.]+$", "", path.split("/")[-1])
            if len(stem) >= 3:
                symbols.add(stem)
    for m in _PLUS_FILE_RE.finditer(reference_patch):
        path = m.group(1)
        symbols.add(path)
        symbols.add(path.split("/")[-1])
    for m in _HUNK_FUNC_RE.finditer(reference_patch):
        symbols.add(m.group(1))
    # Drop very common words that happen to match identifiers.
    stop = {
        "the",
        "and",
        "for",
        "with",
        "from",
        "this",
        "that",
        "test",
        "tests",
        "file",
        "path",
        "error",
        "true",
        "false",
        "null",
        "none",
        "int",
        "str",
        "string",
        "bool",
        "void",
        "new",
        "get",
        "set",
        "add",
        "run",
    }
    return {s for s in symbols if len(s) >= 3 and s.lower() not in stop}


def patch_has_test_files(reference_patch: str | None) -> bool:
    """Whether the gold/reference patch touches test files (PR included test changes)."""
    if not reference_patch:
        return False
    paths = [m.group(1) for m in _DIFF_FILE_RE.finditer(reference_patch)]
    paths += [m.group(1) for m in _PLUS_FILE_RE.finditer(reference_patch)]
    return any(any(marker in p.lower() for marker in _TEST_PATH_MARKERS) for p in paths)


def detect_leakage(task_text: str, reference_patch: str | None) -> tuple[bool, tuple[str, ...]]:
    """Symbols from ``reference_patch`` that appear in the task text (B7 analogue)."""
    symbols = extract_patch_symbols(reference_patch)
    if not symbols or not task_text:
        return False, ()
    text_lower = task_text.lower()
    leaked: list[str] = []
    for sym in sorted(symbols, key=len, reverse=True):
        if len(sym) < 4:
            continue
        # File paths: require path-like mention.
        if "/" in sym or "." in sym and sym.endswith((".py", ".go", ".rs", ".js", ".ts", ".java")):
            if sym.lower() in text_lower:
                leaked.append(sym)
            continue
        # Function/type names: word boundary match.
        if re.search(rf"\b{re.escape(sym)}\b", task_text):
            leaked.append(sym)
    return bool(leaked), tuple(leaked[:20])


def compute_features(task_text: str, reference_patch: str | None = None) -> RungFeatures:
    text = strip_task_text(task_text)
    has_repro = bool(_REPRO_RE.search(text) or _REPRO_CMD_RE.search(text))
    has_expected = bool(_EXPECTED_ACTUAL_RE.search(text))
    has_stack = bool(_STACK_TRACE_RE.search(text))
    test_names = _test_names_in_text(text)
    has_test_names = bool(test_names) or bool(_OTHER_TEST_NAME_RE.search(text))
    has_signature = bool(_SIGNATURE_RE.search(text))
    has_leakage, _ = detect_leakage(text, reference_patch)
    _, leaked = detect_leakage(text, reference_patch)
    n_test_funcs = _test_funcs_in_text(text)
    n_test_files = _test_files_in_text(text)
    has_test_code = _has_substantial_test_body(text)
    return RungFeatures(
        word_count=_word_count(text),
        has_repro=has_repro,
        has_expected_actual=has_expected,
        has_stack_trace=has_stack,
        has_test_names=has_test_names,
        has_signature=has_signature,
        has_leakage=has_leakage,
        leakage_count=len(leaked),
        has_test_code=has_test_code,
        n_test_funcs_in_text=n_test_funcs,
        n_test_files_in_text=n_test_files,
        patch_has_tests=patch_has_test_files(reference_patch),
    )


def assign_rung(features: RungFeatures) -> int:
    """Map feature vector to heuristic ladder rung L0–L6 (highest matching level)."""
    # L6: multiple test files or many test functions with bodies in the text.
    if features.has_test_code and (
        features.n_test_files_in_text >= 2 or features.n_test_funcs_in_text >= 3
    ):
        return 6
    # L5: one hidden test file worth of test code in the instruction.
    if features.has_test_code and features.n_test_funcs_in_text >= 1:
        return 5
    # L4: exported signatures / API stubs without full test bodies (not bare repro commands).
    if (
        features.has_signature
        and not features.has_test_code
        and not (features.has_repro and features.has_expected_actual)
    ):
        return 4
    # L3: named checks/tests without full test bodies.
    if features.has_test_names and not features.has_test_code:
        return 3
    # L2: full behavioral contract — repro plus expected/observed behavior.
    if features.has_repro and features.has_expected_actual:
        return 2
    # L1: symptom plus partial requirements (repro OR expected/actual OR stack trace).
    if features.has_repro or features.has_expected_actual or features.has_stack_trace:
        return 1
    # L0: symptom only (title/description with no structured requirements).
    return 0


def classify_task(task_text: str, reference_patch: str | None = None) -> RungResult:
    """Strip harness text, compute features, and assign a heuristic rung."""
    stripped = strip_task_text(task_text)
    features = compute_features(task_text, reference_patch)
    has_leakage, leaked = detect_leakage(stripped, reference_patch)
    features = RungFeatures(
        **{**features.__dict__, "has_leakage": has_leakage, "leakage_count": len(leaked)}
    )
    return RungResult(
        rung=assign_rung(features),
        features=features,
        task_text=stripped,
        leakage_symbols=leaked,
    )


def features_as_dict(features: RungFeatures) -> dict[str, int | bool]:
    return {
        "word_count": features.word_count,
        "has_repro": int(features.has_repro),
        "has_expected_actual": int(features.has_expected_actual),
        "has_stack_trace": int(features.has_stack_trace),
        "has_test_names": int(features.has_test_names),
        "has_signature": int(features.has_signature),
        "has_leakage": int(features.has_leakage),
        "leakage_count": features.leakage_count,
        "has_test_code": int(features.has_test_code),
        "n_test_funcs_in_text": features.n_test_funcs_in_text,
        "n_test_files_in_text": features.n_test_files_in_text,
        "patch_has_tests": int(features.patch_has_tests),
    }
