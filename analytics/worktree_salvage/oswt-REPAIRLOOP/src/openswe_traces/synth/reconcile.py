"""Derive contract.md coverage rows FROM hidden tests plus gold.

Author writes DETAILS.md (numbered commitments) and bugreport.md (L0).
Verifier writes one property test per commitment (TestDetailNN).
This pass writes contract.md after both exist: prose invariants plus one
coverage row per hidden test, each row stating what that test actually
asserts, in behavioural prose with no symbol, file, or line names (B7).

The reconciler may read gold and the hidden tests. It must not touch
bugreport.md and must not weaken any hidden test. Every coverage row
originates in an assertion, so "missing" is structurally impossible.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import time
from collections.abc import Callable, Iterable
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT

PROMPT_VERSION = "reconcile-v2"
CACHE_DIR = ROOT / "outputs" / "reconcile"
LOG_PATH = ROOT / "outputs" / "RECONCILE.log"
COLD_PRIOR_THRESHOLD = 3
MAX_GOLD_CHARS = 20_000
MAX_BODY_CHARS = 2_400
MAX_TOTAL_CHARS = 80_000
MAX_REQUESTS = 300

_DETAIL_RE = re.compile(r"(?m)^(\d+)\.\s+(.*?)(?=\n\d+\.\s+|\Z)", re.DOTALL)
_TEST_RE = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_DETAIL_IDX_RE = re.compile(r"^TestDetail(\d+)_")
_FATAL_RE = re.compile(
    r"""t\.(?:Fatal|Fatalf|Error|Errorf)\(\s*(?:"((?:\\.|[^"])*)"|`([^`]*)`)"""
)
_STR_LIT_RE = re.compile(r"""(?:"((?:\\.|[^"\\]){2,})"|`([^`]{2,})`)""")
_DIFF_FILE_RE = re.compile(
    r"^(?:\+\+\+\s+b/|diff --git a/\S+ b/)(\S+)", re.MULTILINE
)
_FUNC_RE = re.compile(
    r"func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\b"
)
_TYPE_RE = re.compile(r"type\s+([A-Za-z_][A-Za-z0-9_]*)\b")
_EXCISED_RE = re.compile(r"excised:\s*([A-Za-z_][A-Za-z0-9_]*)")
_IDENT_RE = re.compile(r"\b[A-Za-z_][A-Za-z0-9_]{2,}\b")
_LINE_RE = re.compile(r"\b\S+\.go:\d+\b")

# --- fix 1: ordering must be pairwise, never a class direction --------------
_ORDER_WORD_RE = re.compile(
    r"\b(sort\w*|order\w*|precedence|rank\w*|arranged?|listed|descending|"
    r"ascending|newest|oldest)\b",
    re.IGNORECASE,
)
_DIRECTION_WORD_RE = re.compile(
    r"\b(before|after|below|above|ahead of|behind|earlier than|later than|"
    r"higher than|lower than)\b",
    re.IGNORECASE,
)
_CLASS_NOUN_RE = re.compile(
    r"\b(pre-?releases?|prereleases?|releases?|stable|versions?|entries|"
    r"build\s+metadata|metadata)\b",
    re.IGNORECASE,
)
_VERSION_LIT_RE = re.compile(
    r"\bv?\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z][0-9A-Za-z.\-]*)?"
    r"(?:\+[0-9A-Za-z][0-9A-Za-z.\-]*)?\b"
)
_CLAUSE_END_RE = re.compile(r"[();]|;\s+|:\s+|\.\s+")


def _semver_key(v: str) -> tuple | None:
    m = re.match(
        r"^v?(\d+)\.(\d+)(?:\.(\d+))?(?:-([0-9A-Za-z][0-9A-Za-z.\-]*))?"
        r"(?:\+[0-9A-Za-z][0-9A-Za-z.\-]*)?$",
        v,
    )
    if not m:
        return None
    major, minor, patch = int(m.group(1)), int(m.group(2)), int(m.group(3) or 0)
    pre = m.group(4)
    pre_key: tuple = ()
    if pre:
        parts = []
        for p in pre.split("."):
            parts.append((0, int(p)) if p.isdigit() else (1, p))
        pre_key = tuple(parts)
    return (major, minor, patch, pre is None, pre_key)


def _is_directional_order_clause(clause: str) -> bool:
    """A clause claims a class-level ordering direction instead of a pairwise
    rule. 'prereleases sort before releases' is directional; '`0.0.3-beta.2`
    sorts above `0.0.1`' is pairwise and passes."""
    if not (_ORDER_WORD_RE.search(clause) and _DIRECTION_WORD_RE.search(clause)):
        return False
    if not _CLASS_NOUN_RE.search(clause):
        return False
    return len(set(_VERSION_LIT_RE.findall(clause))) < 2


def _clause_spans(text: str) -> list[tuple[int, int]]:
    spans: list[tuple[int, int]] = []
    start = 0
    for m in _CLAUSE_END_RE.finditer(text):
        if m.start() > start:
            spans.append((start, m.start()))
        start = m.end()
    if start < len(text):
        spans.append((start, len(text)))
    return spans


def _pairwise_order_clause(hidden_source: str) -> str:
    """Build the pairwise ordering rule, grounded in suite version literals."""
    vers: dict[str, tuple] = {}
    for lit in _VERSION_LIT_RE.findall(hidden_source):
        key = _semver_key(lit)
        if key is not None:
            vers.setdefault(lit, key)
    pres = [v for v, k in vers.items() if not k[3]]
    rels = [v for v, k in vers.items() if k[3]]
    low: str | None = None
    same: tuple[str, str] | None = None
    for p in pres:
        core = ".".join(str(x) for x in vers[p][:3])
        for r in rels:
            rcore = ".".join(str(x) for x in vers[r][:3])
            if rcore == core and same is None:
                same = (r, p)
            if vers[r][:3] < vers[p][:3] and low is None:
                low = r
        if same is None and f"{p.split('-')[0]}" in hidden_source:
            same = (p.split("-")[0].split("+")[0], p)
    pieces = []
    if same:
        pieces.append(
            f"`{same[0]}` comes before `{same[1]}` — a pre-release follows only "
            "the release it belongs to"
        )
    else:
        pieces.append("a pre-release follows only the release it belongs to")
    if pres and low:
        pieces.append(
            f"`{pres[0]}` comes before `{low}` — a pre-release still precedes "
            "every version with a lower numeric core"
        )
    elif pres:
        pieces.append("a pre-release still precedes every lower numeric core")
    pieces.append("build metadata does not participate in the comparison")
    return "semantic-version precedence is pairwise: " + "; ".join(pieces)


def rewrite_directional_ordering(
    sentence: str, hidden_source: str
) -> tuple[str, bool]:
    """Replace class-direction ordering claims with the pairwise rule."""
    spans = _clause_spans(sentence)
    bad = [s for s in spans if _is_directional_order_clause(sentence[s[0] : s[1]])]
    if not bad:
        return sentence, False
    replacement = _pairwise_order_clause(hidden_source)
    out = sentence
    for s in reversed(bad):
        span_text = sentence[s[0] : s[1]]
        bullet = re.match(r"\s*(-\s+)", span_text)
        rep = (bullet.group(1) if bullet else "") + replacement
        out = out[: s[0]] + rep + out[s[1] :]
    out = re.sub(r"\(\s*\)", "", out)
    out = re.sub(r"\)\s*\(", ") (", out)
    out = re.sub(r"\s{2,}", " ", out)
    out = re.sub(r"\s+([.,;)])", r"\1", out)
    return out, True


# --- fix 2: worked examples must be grounded in the hidden suite ------------
_QUOTED_LIT_RE = re.compile(r"`([^`\n]{2,120})`|\"([^\"\n]{2,120})\"")
_BARE_LIT_RE = re.compile(
    r"(?:[~^><=!]+\s*)?v?\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z.\-]+)?"
    r"(?:\+[0-9A-Za-z.\-]+)?(?:\s+-\s+v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.\-]+)?)?"
    r"|https?://[^\s,;'\"\)\]`]+"
    r"|[\w.\-/]+\.(?:tgz|yaml|yml|json|tar\.gz|zip|go)\b"
    r"|sha256:[0-9a-fA-F]{4,}"
)
_EXAMPLE_MARK_RE = re.compile(
    r"(?i)\b(e\.g\.?|for example|worked examples?|example:|i\.e\.?)\b|→|->|"
    r"\byields?\b|\breturns?\b"
)
_PAREN_RE = re.compile(r"\([^()]*\)")


def example_literals(text: str) -> list[str]:
    """Literal-looking tokens a worked example claims are concrete data."""
    lits: list[str] = []
    for m in _QUOTED_LIT_RE.finditer(text):
        lit = m.group(1) if m.group(1) is not None else m.group(2)
        if not lit:
            continue
        # Reject prose spans a quote-pair can sweep up (e.g. `") with versions ["`
        # or ` remains); ... = `): literals do not contain sentence punctuation,
        # and whitespace is allowed only alongside a digit ("1.0.0 - 3.0.0").
        # `<...>` spans are metavalue placeholders, not literals.
        if re.search(r"[();'\"=,→<>]", lit):
            continue
        if re.search(r"\s", lit) and not re.search(r"\d", lit):
            continue
        if not lit.isidentifier() or re.search(r"[\d./:+\-_]", lit):
            lits.append(lit)
    for m in _BARE_LIT_RE.finditer(text):
        lit = m.group(0).strip()
        if len(lit) >= 2 and re.search(r"\d", lit):
            lits.append(lit)
    return list(dict.fromkeys(lits))


def ungrounded_literals(fragment: str, hidden_source: str) -> list[str]:
    return [l for l in example_literals(fragment) if l not in hidden_source]


def ground_examples(
    examples: list[str], hidden_source: str
) -> tuple[list[str], list[str]]:
    kept, dropped = [], []
    for ex in examples:
        (dropped if ungrounded_literals(ex, hidden_source) else kept).append(ex)
    return kept, dropped


def _paren_spans(text: str) -> list[tuple[int, int]]:
    """Spans of the outermost balanced (...) groups (nested parens included)."""
    spans, stack = [], []
    for i, ch in enumerate(text):
        if ch == "(":
            stack.append(i)
        elif ch == ")" and stack:
            start = stack.pop()
            if not stack:
                spans.append((start, i + 1))
    return spans


def strip_ungrounded_example_clauses(
    sentence: str, hidden_source: str
) -> tuple[str, list[str]]:
    """Remove parenthesised/trailing example clauses whose literals are not in
    the hidden suite. Clauses that cannot be excised cleanly are returned as
    survivors for the self-check to fail on."""
    survivors: list[str] = []
    out = sentence
    for start, end in reversed(_paren_spans(out)):
        frag = out[start:end]
        if _EXAMPLE_MARK_RE.search(frag) and ungrounded_literals(frag, hidden_source):
            out = out[:start] + out[end:]
    tail = re.search(
        r"(?i)(?:^|[.;(]\s*)(?:e\.g\.?,?|for example,?|for instance,?|"
        r"worked examples?:?|i\.e\.?,?)\s+(.+?)\.?\s*$",
        out,
    )
    if tail and ungrounded_literals(tail.group(1), hidden_source):
        out = out[: tail.start()].rstrip(" .;(") + "."
    out = re.sub(r"\s{2,}", " ", out)
    out = re.sub(r"\s+([.,;)])", r"\1", out)
    for m in _EXAMPLE_MARK_RE.finditer(out):
        frag = out[m.start() :]
        if ungrounded_literals(frag, hidden_source):
            survivors.append(frag[:120])
            break
    return out, survivors


def contract_grounding_offenders(contract: str, hidden_source: str) -> list[str]:
    """Self-check: any surviving worked-example clause whose literals do not
    appear in the hidden suite. A unit that emits one fails."""
    offenders: list[str] = []
    for line in contract.splitlines():
        for m in _EXAMPLE_MARK_RE.finditer(line):
            frag = line[m.start() :]
            bad = ungrounded_literals(frag, hidden_source)
            if bad:
                offenders.append(f"{frag[:100]} :: {bad[:3]}")
                break
    return offenders


# --- fix 3: descriptive noun phrases for scrubbed symbols -------------------
_AUX_WORDS = {"must", "new", "do", "try", "make", "init", "all", "any"}
_GERUND = {
    "merge": "merging", "add": "adding", "sort": "sorting", "index": "indexing",
    "load": "loading", "write": "writing", "parse": "parsing", "scan": "scanning",
    "validate": "validating", "marshal": "marshalling", "unmarshal": "unmarshalling",
    "encode": "encoding", "decode": "decoding", "read": "reading",
    "delete": "deleting", "update": "updating", "create": "creating",
    "check": "checking", "compute": "computing", "resolve": "resolving",
    "list": "listing", "escape": "escaping", "split": "splitting",
    "join": "joining", "format": "formatting", "apply": "applying",
    "build": "building", "register": "registering", "reset": "resetting",
    "compare": "comparing", "render": "rendering", "copy": "copying",
    "set": "setting", "get": "getting", "put": "putting", "run": "running",
    "walk": "walking", "find": "finding", "dedup": "deduplicating",
    "insert": "inserting", "append": "appending", "remove": "removing",
    "report": "reporting", "record": "recording", "emit": "emitting",
    "wrap": "wrapping", "hash": "hashing", "dup": "duplicating",
    "freeze": "freezing", "sample": "sampling", "dispatch": "dispatching",
}
_SOLO_PHRASE = {
    "has": "presence-testing", "get": "the lookup", "exists": "the existence check",
    "contains": "the containment check", "is": "the predicate", "new": "the constructor",
}
_PHRASE_BY_WORDS = {
    ("index", "directory"): "the directory scan",
    ("scan", "directory"): "the directory scan",
    ("load", "index", "file"): "loading the index file",
    ("index", "file"): "the index file",
}
_WORD_SPLIT_RE = re.compile(r"[A-Z]+(?![a-z])|[A-Za-z][a-z0-9]*|\d+")


def symbol_noun_phrase(name: str) -> str:
    """Descriptive noun phrase for a scrubbed symbol (B7 still forbids the
    symbol itself): Merge -> 'merging', IndexDirectory -> 'the directory
    scan', Has -> 'presence-testing'."""
    words = [w.lower() for w in _WORD_SPLIT_RE.findall(name)]
    words = [w for w in words if w not in _AUX_WORDS]
    if not words:
        return "the scrubbed operation"
    key = tuple(words)
    if key in _PHRASE_BY_WORDS:
        return _PHRASE_BY_WORDS[key]
    verb_i = next(
        (i for i, w in enumerate(words) if w in _GERUND or w in _SOLO_PHRASE), None
    )
    if verb_i is None:
        return "the " + " ".join(words)
    verb = words[verb_i]
    objects = [w for j, w in enumerate(words) if j != verb_i]
    if objects:
        if verb in _SOLO_PHRASE and verb not in _GERUND:
            return f"the {' '.join(objects)} presence check"
        return f"{_GERUND.get(verb, verb + 'ing')} the {' '.join(objects)}"
    return _SOLO_PHRASE.get(verb) or _GERUND.get(verb, "the operation")

# Commitments prose describes badly: JSON field order, omitempty, null-vs-absent.
_ENCODING_SHAPE_RE = re.compile(
    r"omitempty|null-vs-absent|null vs absent|field order|fixed[- ]field|"
    r"json:\"-\"|key presence|absent,? not null|emits? \"[a-z_]+\":null|"
    r"struct's fixed|declaration order",
    re.IGNORECASE,
)

_COMMON_IDENTS = {
    "func", "return", "error", "string", "int", "bool", "nil", "true", "false",
    "json", "JSON", "Error", "String", "New", "Get", "Set", "Add", "List",
    "Test", "Detail", "Marshal", "Unmarshal", "Parse", "Format", "Encode",
    "Decode", "Write", "Read", "Type", "Value", "Name", "Body", "Header",
    "Request", "Response", "Context", "Client", "Server", "HTTP", "URL",
    "SHA", "ID", "OK", "TODO", "FIXME", "package", "import", "const", "var",
    "type", "struct", "interface", "map", "chan", "go", "defer", "range",
    "if", "else", "for", "switch", "case", "default", "break", "continue",
    "make", "append", "len", "cap", "copy", "delete", "panic", "recover",
    "int64", "int32", "uint64", "byte", "rune", "float64", "any",
}


@dataclass(frozen=True)
class Detail:
    index: int
    text: str


@dataclass
class HiddenFunc:
    name: str
    index: int | None
    comment: str
    body: str
    fatals: tuple[str, ...]
    literals: tuple[str, ...]


@dataclass
class CoverageRow:
    test: str
    sentence: str


@dataclass
class ReconcileResult:
    unit: str
    contract: str
    rows: list[CoverageRow]
    encoding_shape: bool
    unreconcilable: bool
    unreconcilable_reason: str
    model: str
    requests: int
    cache_hit: bool
    b7_leaks: list[str] = field(default_factory=list)
    applied_fixes: list[str] = field(default_factory=list)


class A13ColdGateError(RuntimeError):
    """Cold repo's L2 contract failed A13; refuse to screen the batch."""


def log(msg: str) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    line = f"[{time.strftime('%H:%M:%S')}] {msg}"
    print(line, flush=True)
    with LOG_PATH.open("a", encoding="utf-8") as fh:
        fh.write(line + "\n")


def parse_details(text: str) -> list[Detail]:
    out: list[Detail] = []
    for m in _DETAIL_RE.finditer(text):
        body = re.sub(r"\s+", " ", m.group(2)).strip()
        # Drop "Covered by `Test…`" provenance — that's author notes, not a commitment.
        body = re.sub(r"\s*Covered by\b.*", "", body, flags=re.IGNORECASE).strip()
        out.append(Detail(index=int(m.group(1)), text=body))
    return out


def parse_hidden_funcs(hidden_files: dict[str, str]) -> list[HiddenFunc]:
    out: list[HiddenFunc] = []
    for rel in sorted(hidden_files):
        text = hidden_files[rel]
        funcs = list(_TEST_RE.finditer(text))
        for i, m in enumerate(funcs):
            start = m.start()
            end = funcs[i + 1].start() if i + 1 < len(funcs) else len(text)
            # Comment immediately above the func.
            pre = text[:start].rstrip()
            comment_lines: list[str] = []
            for ln in reversed(pre.splitlines()):
                s = ln.strip()
                if s.startswith("//"):
                    comment_lines.append(s[2:].strip())
                elif s == "":
                    if comment_lines:
                        break
                else:
                    break
            comment = " ".join(reversed(comment_lines)).strip()
            body = text[start:end]
            name = m.group(1)
            idx_m = _DETAIL_IDX_RE.match(name)
            idx = int(idx_m.group(1)) if idx_m else None
            fatals = tuple(
                (fm.group(1) if fm.group(1) is not None else fm.group(2) or "")[:200]
                for fm in _FATAL_RE.finditer(body)
            )
            lits = []
            for sm in _STR_LIT_RE.finditer(body):
                lit = sm.group(1) if sm.group(1) is not None else sm.group(2) or ""
                if "\\" in lit:
                    try:
                        lit = bytes(lit, "utf-8").decode("unicode_escape")
                    except (UnicodeDecodeError, ValueError):
                        pass
                if 2 <= len(lit) <= 80 and not lit.startswith("github.com"):
                    lits.append(lit)
            out.append(
                HiddenFunc(
                    name=name,
                    index=idx,
                    comment=comment,
                    body=body,
                    fatals=fatals[:8],
                    literals=tuple(dict.fromkeys(lits)),
                )
            )
    return out


def gold_denylist(gold: str) -> set[str]:
    """Symbol and file names the contract must not mention (B7)."""
    names: set[str] = set()
    funcs: set[str] = set()
    for m in _DIFF_FILE_RE.finditer(gold):
        path = m.group(1)
        names.add(path)
        names.add(Path(path).name)
        names.add(Path(path).stem)
    for m in _FUNC_RE.finditer(gold):
        funcs.add(m.group(1))
        names.add(m.group(1))
    for m in _TYPE_RE.finditer(gold):
        names.add(m.group(1))
    for m in _EXCISED_RE.finditer(gold):
        funcs.add(m.group(1))
        names.add(m.group(1))
    out: set[str] = set()
    for n in names:
        if not n or n in _COMMON_IDENTS or len(n) < 3:
            continue
        # Drop lowercase English that isn't a real func/excised name.
        if n.islower() and n not in funcs and "_" not in n and "." not in n:
            continue
        out.add(n)
    return out


def shared_literals(func: HiddenFunc, gold: str) -> list[str]:
    return [lit for lit in func.literals if lit in gold][:6]


def is_encoding_shape(details: list[Detail], funcs: list[HiddenFunc]) -> bool:
    blob = " ".join(d.text for d in details) + " " + " ".join(
        f.comment for f in funcs
    )
    return bool(_ENCODING_SHAPE_RE.search(blob))


def scrub_b7(text: str, denylist: Iterable[str]) -> tuple[str, list[str]]:
    """Strip file/symbol/line leaks. Returns (cleaned, leaks_found)."""
    leaks: list[str] = []
    out = text
    for name in sorted(denylist, key=len, reverse=True):
        if re.search(rf"\b{re.escape(name)}\b", out):
            leaks.append(name)
            phrase = symbol_noun_phrase(name)
            out = re.sub(
                rf"\b(?:a|an|the)\s+{re.escape(name)}\b|\b{re.escape(name)}\b",
                phrase,
                out,
            )
            out = re.sub(
                rf"\b{re.escape(phrase)}\s+{re.escape(phrase)}\b", phrase, out
            )
    for m in list(_LINE_RE.finditer(out)):
        leaks.append(m.group(0))
        out = out.replace(m.group(0), "the call site")
    return out, leaks


def _truncate(text: str, cap: int) -> str:
    if len(text) <= cap:
        return text
    head = cap * 3 // 4
    return text[:head] + f"\n...[{len(text) - cap} chars elided]...\n" + text[-(cap - head) :]


def cache_key(details: str, hidden: dict[str, str], gold: str) -> str:
    h = hashlib.sha256()
    h.update(PROMPT_VERSION.encode())
    h.update(b"\0")
    h.update(details.encode())
    for rel in sorted(hidden):
        h.update(b"\0")
        h.update(rel.encode())
        h.update(hidden[rel].encode())
    h.update(b"\0gold:")
    h.update(hashlib.sha256(gold.encode()).digest())
    return h.hexdigest()


def load_hidden_dir(hidden_dir: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    if not hidden_dir.is_dir():
        return out
    for f in sorted(hidden_dir.rglob("*_test.go")):
        out[f.relative_to(hidden_dir).as_posix()] = f.read_text(
            encoding="utf-8", errors="replace"
        )
    return out


def _fallback_rows(
    details: list[Detail],
    funcs: list[HiddenFunc],
    gold: str,
    denylist: set[str],
) -> tuple[str, list[CoverageRow], list[str]]:
    by_idx = {d.index: d for d in details}
    leaks: list[str] = []
    rows: list[CoverageRow] = []
    for fn in funcs:
        detail = by_idx.get(fn.index) if fn.index is not None else None
        raw = fn.comment or (detail.text if detail else f"hidden test {fn.name}")
        examples = shared_literals(fn, gold) or list(fn.literals[:3])
        sentence = raw
        if examples:
            shown = ", ".join(f"`{e}`" for e in examples[:3])
            if shown not in sentence:
                sentence = sentence.rstrip(".") + f". Worked example: {shown}."
        cleaned, found = scrub_b7(sentence, denylist)
        leaks.extend(found)
        rows.append(CoverageRow(test=fn.name, sentence=cleaned))
    # Unique invariant paragraphs, one per detail index order then leftovers.
    seen: set[str] = set()
    inv_lines: list[str] = []
    for d in details:
        for row in rows:
            if _DETAIL_IDX_RE.match(row.test) and int(_DETAIL_IDX_RE.match(row.test).group(1)) == d.index:
                if row.sentence not in seen:
                    inv_lines.append(f"- {row.sentence}")
                    seen.add(row.sentence)
                break
    for row in rows:
        if row.sentence not in seen:
            inv_lines.append(f"- {row.sentence}")
            seen.add(row.sentence)
    invariants = "\n".join(inv_lines) if inv_lines else "\n".join(f"- {d.text}" for d in details)
    return invariants, rows, leaks


def _build_prompt(
    unit: str,
    details: list[Detail],
    funcs: list[HiddenFunc],
    gold: str,
    denylist: set[str],
) -> str:
    det_block = "\n".join(f"{d.index}. {d.text}" for d in details) or "(none)"
    tests_block = []
    for fn in funcs:
        tests_block.append(
            f"### {fn.name}\n"
            f"comment: {fn.comment or '(none)'}\n"
            f"assertion messages: {'; '.join(fn.fatals) if fn.fatals else '(none)'}\n"
            f"literals: {', '.join(repr(x) for x in fn.literals[:8])}\n"
            f"```go\n{_truncate(fn.body, MAX_BODY_CHARS)}\n```"
        )
    deny = ", ".join(sorted(denylist)[:40]) or "(none)"
    return f"""You write the L2 contract for a synthetic SWE task. The coverage
table must be derived FROM the hidden tests, not from a reading of gold.
Each row states what that test actually asserts.

Rules:
- Behavioural prose only. Do NOT name functions, types, files, packages, or
  line numbers. Forbidden names: {deny}
- Wire-format strings, JSON keys, URL pieces, and concrete input/output
  examples ARE allowed and expected (they are the behaviour).
- One coverage row per hidden test below. First column is the test name
  exactly. Second column is the assertion in prose, with a worked example
  taken from the test's own literals when possible.
- Never state an ordering as a direction ("X sorts before/after/below Y"
  as a class rule). State the pairwise rule instead: which of two specific
  values comes first, and whether build metadata participates in the
  comparison.
- Every literal in a worked example must be one the hidden tests actually
  use. Never invent versions, URLs, digests, or inputs; an example that no
  assertion supports must be omitted.
- Invariants summarise the same commitments, still without symbol names.
- If a commitment cannot be stated without Go encoding internals (struct
  field declaration order that the tests do not pin with wire strings;
  omitempty as a tag rather than "key absent"; null-vs-absent that is not
  observable as "key missing" vs "key present with JSON null"), set
  unreconcilable=true and still fill every row you can.

UNIT: {unit}

NUMBERED COMMITMENTS (DETAILS.md):
{det_block}

GOLD PATCH (for truth, not for naming):
```diff
{_truncate(gold, MAX_GOLD_CHARS)}
```

HIDDEN TESTS:
{chr(10).join(tests_block)}

Answer with JSON only:
{{"invariants": "<markdown bullet list>",
  "rows": [{{"test": "<exact Test name>", "sentence": "<assertion prose>"}}],
  "examples": ["<worked example>", "..."],
  "unreconcilable": false,
  "unreconcilable_reason": ""}}
Include every hidden test exactly once in rows."""


def _llm_complete(prompt: str, api_key: str, budget: int) -> tuple[dict | None, str, int]:
    from openswe_traces.contract_judge import MAX_ATTEMPTS, MODELS, _post

    prompt = _truncate(prompt, MAX_TOTAL_CHARS)
    last_err = "no models"
    requests = 0
    for attempt in range(MAX_ATTEMPTS):
        model = MODELS[min(attempt, len(MODELS) - 1)]
        if requests >= budget:
            return None, model, requests
        requests += 1
        verdict, err = _post(api_key, model, prompt)
        if verdict is not None:
            return verdict, model, requests
        last_err = err or last_err
        if err and ("404" in err or "No endpoints" in err):
            continue
        if err and "429" in err:
            time.sleep(20 * (attempt + 1))
            continue
        time.sleep(5)
    log(f"llm failed after {requests} requests: {last_err}")
    return None, MODELS[-1], requests


def render_contract(
    unit: str,
    invariants: str,
    rows: list[CoverageRow],
    examples: list[str] | None = None,
) -> str:
    parts = [f"# Contract (L2) — {unit}", "", invariants.strip(), ""]
    if examples:
        parts.append("Worked examples:")
        parts.append("")
        for ex in examples:
            parts.append(f"- {ex}")
        parts.append("")
    parts.append("## Coverage of hidden tests")
    parts.append("")
    parts.append("| hidden test | contract sentence |")
    parts.append("|---|---|")
    for row in rows:
        sent = row.sentence.replace("|", "\\|").strip()
        parts.append(f"| `{row.test}` | {sent} |")
    parts.append("")
    return "\n".join(parts)


def reconcile_unit(
    *,
    unit: str,
    details_text: str,
    hidden_files: dict[str, str],
    gold: str,
    llm: bool = True,
    llm_complete: Callable[[str], dict | None] | None = None,
    cache_dir: Path | None = None,
    budget: int = MAX_REQUESTS,
) -> ReconcileResult:
    """Emit a contract from (DETAILS, hidden tests, gold). Never writes bugreport."""
    details = parse_details(details_text)
    funcs = parse_hidden_funcs(hidden_files)
    denylist = gold_denylist(gold)
    encoding = is_encoding_shape(details, funcs)
    key = cache_key(details_text, hidden_files, gold)
    cache = cache_dir or CACHE_DIR
    cache.mkdir(parents=True, exist_ok=True)
    path = cache / f"{key}.json"

    cached = None
    if path.is_file():
        try:
            cached = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            cached = None

    model = "deterministic"
    requests = 0
    cache_hit = False
    payload: dict | None = None
    if cached and isinstance(cached.get("payload"), dict):
        payload = cached["payload"]
        model = str(cached.get("model") or "cached")
        cache_hit = True

    if payload is None and llm:
        prompt = _build_prompt(unit, details, funcs, gold, denylist)
        if llm_complete is not None:
            payload = llm_complete(prompt)
            model = "injected"
        else:
            from openswe_traces.contract_judge import load_api_key

            payload, model, requests = _llm_complete(prompt, load_api_key(), budget)
        if payload is not None:
            path.write_text(
                json.dumps(
                    {
                        "key": key,
                        "prompt_version": PROMPT_VERSION,
                        "unit": unit,
                        "model": model,
                        "payload": payload,
                    },
                    indent=1,
                )
                + "\n",
                encoding="utf-8",
            )

    hidden_source = "\n".join(hidden_files.values())
    fixes: list[str] = []
    survivors: list[str] = []
    leaks: list[str] = []

    def polish(text: str) -> str:
        nonlocal leaks, survivors, fixes
        text, ord_fixed = rewrite_directional_ordering(text, hidden_source)
        if ord_fixed:
            fixes.append("directional-ordering rewritten to pairwise rule")
        text, surv = strip_ungrounded_example_clauses(text, hidden_source)
        survivors.extend(surv)
        text, found = scrub_b7(text, denylist)
        leaks.extend(found)
        return text

    def finish(
        invariants: str,
        rows: list[CoverageRow],
        examples: list[str],
        unrec: bool,
        reason: str,
    ) -> ReconcileResult:
        contract = render_contract(unit, invariants, rows, examples)
        offenders = contract_grounding_offenders(contract, hidden_source)
        if survivors or offenders:
            unrec = True
            bad = (survivors + offenders)[:4]
            reason = (reason + "; " if reason else "") + (
                "ungrounded worked-example literals survived grounding: "
                + " | ".join(bad)
            )
            fixes.append("grounding self-check FAILED")
        if encoding and not reason:
            reason = (
                "encoding-shape commitments (JSON field order / omitempty / "
                "null-vs-absent); rows still derived from tests"
            )
        return ReconcileResult(
            unit=unit,
            contract=contract,
            rows=rows,
            encoding_shape=encoding,
            unreconcilable=unrec,
            unreconcilable_reason=reason,
            model=model,
            requests=requests,
            cache_hit=cache_hit,
            b7_leaks=sorted(set(leaks)),
            applied_fixes=sorted(set(fixes)),
        )

    if payload and isinstance(payload.get("rows"), list) and payload["rows"]:
        rows: list[CoverageRow] = []
        seen: set[str] = set()
        for raw in payload["rows"]:
            if not isinstance(raw, dict):
                continue
            name = str(raw.get("test") or "")
            sent = str(raw.get("sentence") or "").strip()
            if not name or not sent:
                continue
            rows.append(CoverageRow(test=name, sentence=polish(sent)))
            seen.add(name)
        # Structurally: one row per hidden test. Fill any the model dropped.
        inv_fb, fb_rows, fb_leaks = _fallback_rows(details, funcs, gold, denylist)
        leaks.extend(fb_leaks)
        for fn in funcs:
            if fn.name not in seen:
                match = next((r for r in fb_rows if r.test == fn.name), None)
                if match:
                    rows.append(
                        CoverageRow(test=match.test, sentence=polish(match.sentence))
                    )
        # Keep hidden-test order.
        order = {fn.name: i for i, fn in enumerate(funcs)}
        rows.sort(key=lambda r: order.get(r.test, 10_000))
        raw_inv = payload.get("invariants") or inv_fb
        if isinstance(raw_inv, list):
            invariants = "\n".join(
                (x if str(x).startswith("-") else f"- {x}") for x in raw_inv if x
            )
        else:
            invariants = str(raw_inv)
        invariants = polish(invariants)
        raw_ex = [str(x) for x in (payload.get("examples") or []) if x]
        examples, dropped = ground_examples(raw_ex, hidden_source)
        if dropped:
            fixes.append(f"dropped {len(dropped)} ungrounded worked example(s)")
        kept_examples: list[str] = []
        for e in examples:
            cleaned, found = scrub_b7(e, denylist)
            leaks.extend(found)
            kept_examples.append(cleaned)
        examples = kept_examples
        unrec = bool(payload.get("unreconcilable"))
        reason = str(payload.get("unreconcilable_reason") or "")
        return finish(invariants, rows, examples, unrec, reason)

    invariants, rows, leaks = _fallback_rows(details, funcs, gold, denylist)
    invariants = polish(invariants)
    rows = [CoverageRow(test=r.test, sentence=polish(r.sentence)) for r in rows]
    return finish(invariants, rows, [], False, "")


def reconcile_paths(
    *,
    unit: str,
    details_path: Path,
    hidden_dir: Path,
    gold_path: Path,
    out_path: Path | None = None,
    llm: bool = True,
    budget: int = MAX_REQUESTS,
) -> ReconcileResult:
    details_text = details_path.read_text(encoding="utf-8")
    gold = gold_path.read_text(encoding="utf-8", errors="replace")
    hidden = load_hidden_dir(hidden_dir)
    if not hidden:
        raise FileNotFoundError(f"no hidden tests under {hidden_dir}")
    result = reconcile_unit(
        unit=unit,
        details_text=details_text,
        hidden_files=hidden,
        gold=gold,
        llm=llm,
        budget=budget,
    )
    if out_path is not None:
        out_path.parent.mkdir(parents=True, exist_ok=True)
        out_path.write_text(result.contract, encoding="utf-8")
    return result


def splice_contract(instruction: str, contract: str) -> str:
    """Replace the contract prefix; keep bugreport + reproduce + no-web."""
    markers = ("\n# Bug report", "\nReproduce with:")
    idx = -1
    for marker in markers:
        i = instruction.find(marker)
        if i >= 0:
            idx = i
            break
    if idx < 0:
        return contract.rstrip() + "\n"
    return contract.rstrip() + "\n" + instruction[idx + 1 :]


def prior_bank_unit_count(repo: str, *, root: Path | None = None) -> int:
    """Units already in the screened bank (not the batch currently authored)."""
    base = root or ROOT
    names: set[str] = set()
    for folder in (
        base / "experiments" / "pipeline" / "tasks_composerver" / repo,
        base / "experiments" / "pipeline" / "tasks" / repo,
    ):
        if not folder.is_dir():
            continue
        for child in folder.iterdir():
            if not child.is_dir():
                continue
            name = child.name
            if "-L" in name and not name.startswith("_"):
                names.add(name.rsplit("-L", 1)[0])
    return len(names)


def is_cold_repo(repo: str, *, root: Path | None = None, threshold: int = COLD_PRIOR_THRESHOLD) -> bool:
    return prior_bank_unit_count(repo, root=root) < threshold


def enforce_cold_a13(
    task_dir: Path,
    *,
    repo: str | None = None,
    root: Path | None = None,
    skip: bool | None = None,
) -> None:
    """Mandatory A13 for cold repos. Fail-closed on false/missing/unresolved.

    `skip` overrides; default skip only when OSWT_SKIP_A13 is set (tests).
    """
    if skip is None:
        skip = os.environ.get("OSWT_SKIP_A13", "") in {"1", "true", "yes"}
    if skip:
        return
    name = repo or task_dir.parent.name
    if not is_cold_repo(name, root=root):
        return
    from openswe_traces.contract_judge import (
        attach_bodies,
        judge_unit,
        load_api_key,
        load_hidden_files,
    )
    from openswe_traces.gate.context import GateContext
    from openswe_traces.gate.contract_gold import check_contract_gold, gather_evidence

    ctx = GateContext(task_dir=task_dir)
    ev = gather_evidence(ctx)
    if not ev.rows:
        raise A13ColdGateError(f"cold repo {name}: {task_dir} has no coverage table")
    attach_bodies(ev, load_hidden_files(task_dir))
    outcome = judge_unit(ev, load_api_key())
    if outcome.verdict is None:
        raise A13ColdGateError(
            f"cold repo {name}: A13 judge failed for {task_dir}: {outcome.error}"
        )
    verdict = check_contract_gold(ctx)
    if not verdict.passed:
        raise A13ColdGateError(
            f"cold repo {name}: A13 failed for {task_dir}: {verdict.evidence}"
        )
    log(f"A13 cold-gate pass {name}/{task_dir.name}")
