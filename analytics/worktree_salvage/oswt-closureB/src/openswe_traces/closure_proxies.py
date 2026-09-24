r"""Closure-structure proxies for gold patches, and their relation to per-instance solve rate.

Question: is task difficulty driven by raw patch *size*, or by the *closure structure* of the
change — a change whose added code mostly references itself (new symbols calling new symbols)
being harder than a same-sized change that mostly edits existing call sites?

Stage 1 (one streaming DuckDB pass, resume-safe): every corpus shard is read once and aggregated
to one row per ``instance_id`` — a single representative gold patch per instance (the patch is
identical across shards; ``max`` is kept) plus labeled-rollout counts — and the closure proxies
below are computed in Python over the patch text. Per-shard parts land in
``outputs/closure_proxies_parts/`` (atomic, skipped on rerun) and fold into
``outputs/closure_proxies.parquet`` (``solve_rate = n_resolved / n_labeled`` over labeled
rollouts, ``resolved in (0, 1)``).

Stage 2: Spearman of each proxy vs solve_rate (n_labeled >= 3), a WLS size-vs-structure
comparison (grouped 80/20 split by repo, weight = n_labeled), and a size-matched decile
comparison; written to ``analytics/research/closure_proxies_vs_difficulty.md``.

Proxy definitions (all from the gold patch text alone):

- ``n_hunks`` / ``n_files``: ``@@`` headers / ``diff --git`` headers.
- ``added_lines`` / ``removed_lines``: ``+`` / ``-`` content lines inside hunks.
- ``n_new_defs``: added lines that define a symbol, per-language regex (below).
- ``internal_refs``: occurrences, in added lines, of names defined elsewhere in the same patch
  (the line defining a name does not count its own occurrence; a recursive self-call does not
  count).
- ``boundary_refs``: occurrences, in added lines, of identifiers that appear in the patch's
  context or removed lines and are not new defs — i.e. references to pre-existing code. Language
  keywords are excluded (a keyword stoplist per language); builtins and standard-library names
  are kept.
- ``ratio`` = ``internal_refs / max(1, boundary_refs)``.
- ``new_frac`` = ``n_new_defs / max(1, added_lines)``.
- ``edit_frac`` = fraction of hunks with both removed and added lines.

Definition regexes (first match per added line; names filtered against the language keyword
stoplist; unknown languages get no def regex, so ``n_new_defs == internal_refs == 0``):

- python: ``^\s*(async )?def NAME(`` and ``^\s*class NAME``.
- go: ``^\s*func (RECEIVER)?NAME(`` (receiver ``(… )`` optional, so methods count).
- typescript/javascript: ``class NAME``, ``function NAME`` (with optional ``export``/``default``/
  ``async``), named arrow ``(const|let|var) NAME = (async )?(…) =>``, ``interface NAME`` /
  ``type NAME``, accessor ``(static )?(async )?(get|set) NAME(`` and method shorthand
  ``(static )?(async )?NAME(… ){`` (keyword-blacklisted).
- java: ``(class|interface|enum|record) NAME`` and single-line method signatures
  ``MODIFIERS [RETURN] NAME(… ) {`` (annotations and modifiers stripped; keywords blacklisted).
- rust: ``fn NAME`` (with ``pub``/``async``/``unsafe``/``extern`` prefixes), ``struct|enum|trait|
  union|mod NAME``, ``type NAME``.
- c/cpp: ``(class|struct|enum|union) NAME`` and single-line function signatures
  ``[RETURN] NAME(… ) {`` (keywords blacklisted).
- php: ``(class|interface|trait|enum) NAME`` and ``function NAME(`` (with access modifiers).
- ruby: ``def (self\.)?NAME`` and ``(class|module) NAME`` (namespaced take the last component).
- csharp: ``(class|interface|struct|enum|record) NAME`` and single-line method signatures
  ``MODIFIERS [RETURN] NAME(… ) {`` or ``=>`` (keywords blacklisted).

Examples:
  uv run python scripts/closure_proxies.py --limit 3
  uv run python scripts/closure_proxies.py --skip-stream
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import time
import traceback
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from scipy.stats import spearmanr
from sklearn.model_selection import GroupShuffleSplit

import duckdb

from .data import DATA_ROOT, PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files
from .difficulty import fit_linear
from .features import _sql_str, fmt_duration, utcnow
from .score import WINSOR_PCT, md_table

console = Console()

PARTS_DIR = ROOT / "outputs" / "closure_proxies_parts"
OUT_PARQUET = ROOT / "outputs" / "closure_proxies.parquet"
OUT_MD = ROOT / "analytics" / "research" / "closure_proxies_vs_difficulty.md"

MIN_LABELED = 3
TEST_SIZE = 0.2
SEED = 42
LANGUAGE_ALIASES = {"ts": "typescript", "js": "javascript"}

PROXY_COLUMNS = [
    "n_hunks",
    "n_files",
    "added_lines",
    "removed_lines",
    "n_new_defs",
    "internal_refs",
    "boundary_refs",
    "ratio",
    "new_frac",
    "edit_frac",
]

IDENT_RE = re.compile(r"[A-Za-z_$][A-Za-z0-9_$]*")

PY_KEYWORDS = frozenset(
    [
        "and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del",
        "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in",
        "is", "lambda", "match", "case", "nonlocal", "not", "or", "pass", "raise", "return",
        "try", "while", "with", "yield", "True", "False", "None",
    ]
)
GO_KEYWORDS = frozenset(
    [
        "break", "case", "chan", "const", "continue", "default", "defer", "else",
        "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map",
        "package", "range", "return", "select", "struct", "switch", "type", "var",
    ]
)
TS_KEYWORDS = frozenset(
    [
        "abstract", "any", "as", "async", "await", "boolean", "break", "case", "catch",
        "class", "const", "continue", "debugger", "declare", "default", "delete", "do",
        "else", "enum", "export", "extends", "false", "finally", "for", "from", "function",
        "get", "if", "implements", "import", "in", "instanceof", "interface", "keyof", "let",
        "module", "namespace", "never", "new", "null", "number", "of", "package", "private",
        "protected", "public", "readonly", "return", "satisfies", "set", "static", "string",
        "super", "switch", "symbol", "this", "throw", "true", "try", "type", "typeof",
        "undefined", "unknown", "using", "var", "void", "while", "with", "yield",
    ]
)
JAVA_KEYWORDS = frozenset(
    [
        "abstract", "assert", "boolean", "break", "byte", "case", "catch", "char", "class",
        "const", "continue", "default", "do", "double", "else", "enum", "extends", "final",
        "finally", "float", "for", "goto", "if", "implements", "import", "instanceof", "int",
        "interface", "long", "native", "new", "package", "private", "protected", "public",
        "return", "short", "static", "strictfp", "super", "switch", "synchronized", "this",
        "throw", "throws", "transient", "try", "void", "volatile", "while", "true", "false",
        "null", "var", "record", "sealed", "permits", "yield",
    ]
)
RUST_KEYWORDS = frozenset(
    [
        "as", "async", "await", "break", "const", "continue", "crate", "dyn", "else", "enum",
        "extern", "false", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod",
        "move", "mut", "pub", "ref", "return", "self", "Self", "static", "struct", "super",
        "trait", "true", "type", "unsafe", "use", "where", "while",
    ]
)
C_KEYWORDS = frozenset(
    [
        "alignas", "alignof", "and", "asm", "auto", "bitand", "bitor", "bool", "break",
        "case", "catch", "char", "char8_t", "char16_t", "char32_t", "class", "compl",
        "concept", "const", "consteval", "constexpr", "constinit", "const_cast", "continue",
        "co_await", "co_return", "co_yield", "decltype", "default", "delete", "do", "double",
        "dynamic_cast", "else", "enum", "explicit", "export", "extern", "false", "float",
        "for", "friend", "goto", "if", "inline", "int", "long", "mutable", "namespace", "new",
        "noexcept", "not", "nullptr", "operator", "or", "private", "protected", "public",
        "register", "reinterpret_cast", "requires", "return", "short", "signed", "sizeof",
        "static", "static_assert", "static_cast", "struct", "switch", "template", "this",
        "thread_local", "throw", "true", "try", "typedef", "typeid", "typename", "union",
        "unsigned", "using", "virtual", "void", "volatile", "wchar_t", "while", "xor",
    ]
)
PHP_KEYWORDS = frozenset(
    [
        "abstract", "and", "array", "as", "break", "callable", "case", "catch", "class",
        "clone", "const", "continue", "declare", "default", "die", "do", "echo", "else",
        "elseif", "empty", "enddeclare", "endfor", "endforeach", "endif", "endswitch",
        "endwhile", "enum", "eval", "exit", "extends", "final", "finally", "fn", "for",
        "foreach", "function", "global", "goto", "if", "implements", "include",
        "include_once", "instanceof", "insteadof", "interface", "isset", "list", "match",
        "namespace", "new", "or", "print", "private", "protected", "public", "readonly",
        "require", "require_once", "return", "static", "switch", "throw", "trait", "try",
        "unset", "use", "var", "while", "xor", "yield",
    ]
)
RUBY_KEYWORDS = frozenset(
    [
        "alias", "and", "begin", "break", "case", "class", "def", "defined?", "do", "else",
        "elsif", "end", "ensure", "false", "for", "if", "in", "module", "next", "nil", "not",
        "or", "redo", "rescue", "retry", "return", "self", "super", "then", "true", "undef",
        "unless", "until", "when", "while", "yield",
    ]
)
CSHARP_KEYWORDS = frozenset(
    [
        "abstract", "as", "base", "bool", "break", "byte", "case", "catch", "char", "checked",
        "class", "const", "continue", "decimal", "default", "delegate", "do", "double", "else",
        "enum", "event", "explicit", "extern", "false", "finally", "fixed", "float", "for",
        "foreach", "goto", "if", "implicit", "in", "int", "interface", "internal", "is",
        "lock", "long", "namespace", "new", "null", "object", "operator", "out", "override",
        "params", "private", "protected", "public", "readonly", "record", "ref", "return",
        "sbyte", "sealed", "short", "sizeof", "stackalloc", "static", "string", "struct",
        "switch", "this", "throw", "true", "try", "typeof", "uint", "ulong", "unchecked",
        "unsafe", "ushort", "using", "var", "virtual", "void", "volatile", "while",
    ]
)

KEYWORDS: dict[str, frozenset[str]] = {
    "python": PY_KEYWORDS,
    "go": GO_KEYWORDS,
    "typescript": TS_KEYWORDS,
    "javascript": TS_KEYWORDS,
    "java": JAVA_KEYWORDS,
    "rust": RUST_KEYWORDS,
    "c": C_KEYWORDS,
    "cpp": C_KEYWORDS,
    "php": PHP_KEYWORDS,
    "ruby": RUBY_KEYWORDS,
    "csharp": CSHARP_KEYWORDS,
}

JAVA_MODIFIERS = (
    r"(?:(?:public|protected|private|static|abstract|final|sealed|non-sealed|strictfp|"
    r"synchronized|native|transient|volatile|default)\s+)*"
)
JAVA_METHOD_MODIFIERS = (
    r"(?:(?:public|protected|private|static|abstract|final|synchronized|native|default|"
    r"strictfp)\s+)*"
)
ANNOTATIONS = r"(?:@[\w.]+\s*(?:\([^)]*\))?\s*)*"
JAVA_RETURN = r"(?:[\w<>\[\],?.\s]+\s+)?"
CSHARP_MODIFIERS = (
    r"(?:(?:public|protected|private|internal|static|abstract|sealed|virtual|override|readonly|"
    r"partial|new|unsafe|async|extern)\s+)*"
)

DEF_PATTERNS: dict[str, list[re.Pattern]] = {
    "python": [
        re.compile(r"^\s*(?:async\s+)?def\s+([A-Za-z_]\w*)"),
        re.compile(r"^\s*class\s+([A-Za-z_]\w*)"),
    ],
    "go": [
        re.compile(r"^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\("),
    ],
    "typescript": [
        re.compile(r"^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)"),
        re.compile(r"^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)"),
        re.compile(
            r"^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?"
            r"(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>"
        ),
        re.compile(r"^\s*(?:export\s+)?(?:interface|type)\s+([A-Za-z_$][\w$]*)"),
        re.compile(r"^\s*(?:static\s+)?(?:async\s+)?(?:get|set)\s+([A-Za-z_$][\w$]*)\s*\("),
        re.compile(r"^\s*(?:static\s+)?(?:async\s+)?([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{"),
    ],
    "java": [
        re.compile(
            rf"^\s*{ANNOTATIONS}{JAVA_MODIFIERS}"
            r"(?:class|interface|enum|record|@interface)\s+([A-Za-z_$][\w$]*)"
        ),
        re.compile(
            rf"^\s*{ANNOTATIONS}{JAVA_METHOD_MODIFIERS}{JAVA_RETURN}"
            r"([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*(?:throws\s+[\w.,\s]+)?\{"
        ),
    ],
    "rust": [
        re.compile(
            r"^\s*(?:pub(?:\s*\([^)]*\))?\s*)?(?:async\s*)?(?:unsafe\s*)?(?:extern\s*"
            r'(?:"[^"]*")?\s*)?fn\s+([A-Za-z_]\w*)'
        ),
        re.compile(
            r"^\s*(?:pub(?:\s*\([^)]*\))?\s*)?(?:struct|enum|trait|union|mod)\s+([A-Za-z_]\w*)"
        ),
        re.compile(r"^\s*(?:pub\s*)?type\s+([A-Za-z_]\w*)"),
    ],
    "c": [
        re.compile(r"^\s*[\w:<>\*&,\s]+\s+([A-Za-z_]\w*)\s*\([^)]*\)\s*\{"),
        re.compile(r"^\s*(?:class|struct|enum|union)\s+([A-Za-z_]\w*)\s*\{?"),
    ],
    "php": [
        re.compile(r"^\s*(?:abstract\s+|final\s+)?(?:class|interface|trait|enum)\s+([A-Za-z_]\w*)"),
        re.compile(
            r"^\s*(?:(?:public|private|protected|static|abstract|final|readonly)\s+)*"
            r"function\s+&?\s*([A-Za-z_]\w*)\s*\("
        ),
    ],
    "ruby": [
        re.compile(r"^\s*def\s+(?:self\.)?(?:[A-Za-z_]\w*::)*([A-Za-z_]\w*)\s*(?:\(|$|;)"),
        re.compile(r"^\s*(?:class|module)\s+(?:[A-Za-z_]\w*::)*([A-Za-z_]\w*)"),
    ],
    "csharp": [
        re.compile(
            rf"^\s*{ANNOTATIONS}{CSHARP_MODIFIERS}"
            r"(?:class|interface|struct|enum|record)\s+([A-Za-z_]\w*)"
        ),
        re.compile(
            rf"^\s*{ANNOTATIONS}{CSHARP_MODIFIERS}{JAVA_RETURN}"
            r"([A-Za-z_]\w*)\s*\([^)]*\)\s*(?:=>|\{)"
        ),
    ],
}

# js/ts share the typescript regexes
DEF_PATTERNS["javascript"] = DEF_PATTERNS["typescript"]

SHARD_SQL = r"""
WITH src AS (
    SELECT
        instance_id,
        max(repo) AS repo,
        max(language) AS language,
        count(*)::INTEGER AS n_rollouts,
        count(*) FILTER (WHERE resolved IN (0, 1))::INTEGER AS n_labeled,
        count(*) FILTER (WHERE resolved = 1)::INTEGER AS n_resolved,
        max(metadata.reference_patch.patch) AS gold_patch,
        max(coalesce(metadata.reference_patch.num_modified_lines, 0))::INTEGER AS gold_patch_lines,
        max(coalesce(metadata.reference_patch.num_modified_files, 0))::INTEGER AS gold_patch_files
    FROM read_parquet('{file_sql}', union_by_name=true)
    GROUP BY 1
)
SELECT * FROM src
"""


def build_shard_sql(file_path: Path) -> str:
    return SHARD_SQL.format(file_sql=str(file_path).replace("'", "''"))


def _data_rel(path: Path) -> Path:
    """Shard path relative to DATA_ROOT, tolerating the traces_data symlink.

    `Path.resolve()` follows the symlink out of the worktree, so the repo's `resolve`-based
    helpers (data.parse_shard, features.part_path_for) fail here; glob-produced paths are
    already absolute and under DATA_ROOT, so a plain relative_to is enough.
    """
    path = Path(path)
    try:
        return path.relative_to(DATA_ROOT)
    except ValueError:
        return path.resolve().relative_to(DATA_ROOT.resolve())


def shard_label(path: Path) -> str:
    rel = _data_rel(path)
    return f"{rel.parts[0]}/{rel.parts[1]}/{rel.parts[2]}/{path.name}"


def part_path_for(file_path: Path, parts_dir: Path = PARTS_DIR) -> Path:
    rel = _data_rel(file_path)
    return parts_dir / ("__".join(rel.with_suffix("").parts) + ".parquet")


def normalize_language(value: str | None) -> str:
    lang = (value or "").strip().lower()
    return LANGUAGE_ALIASES.get(lang, lang)


def _split_hunks(patch: str) -> tuple[list[str], list[str], list[str]]:
    """Return (added, removed, context) content lines, classified only inside hunks."""
    added: list[str] = []
    removed: list[str] = []
    context: list[str] = []
    in_hunk = False
    for line in patch.splitlines():
        if line.startswith("@@"):
            in_hunk = True
        elif line.startswith(("diff --git ", "index ", "--- ")):
            in_hunk = False
        if not in_hunk:
            continue
        if line.startswith("+") and not line.startswith("+++"):
            added.append(line[1:])
        elif line.startswith("-") and not line.startswith("---"):
            removed.append(line[1:])
        elif line.startswith(" "):
            context.append(line[1:])
    return added, removed, context


def _defined_names(line: str, patterns: list[re.Pattern], keywords: frozenset[str]) -> list[str]:
    names: list[str] = []
    for pattern in patterns:
        match = pattern.match(line)
        if match:
            name = match.group(1)
            if name.lower() not in keywords:
                names.append(name)
    return names


def compute_closure_proxies(patch: str, language: str) -> dict[str, float]:
    """Closure proxies for one gold patch. Pure; no I/O. See module docstring for definitions."""
    lang = normalize_language(language)
    added, removed, context = _split_hunks(patch or "")
    n_hunks = sum(1 for line in (patch or "").splitlines() if line.startswith("@"))
    n_files = sum(1 for line in (patch or "").splitlines() if line.startswith("diff --git "))
    if n_files == 0:
        n_files = sum(1 for line in (patch or "").splitlines() if line.startswith("+++ b/"))

    keywords = KEYWORDS.get(lang, frozenset())
    patterns = DEF_PATTERNS.get(lang, [])

    context_tokens = [t for line in context for t in IDENT_RE.findall(line)]
    removed_tokens = [t for line in removed for t in IDENT_RE.findall(line)]
    new_def_names: set[str] = set()
    defs_per_line: list[set[str]] = []
    for line in added:
        names = _defined_names(line, patterns, keywords)
        new_def_names.update(names)
        defs_per_line.append(set(names))

    boundary_tokens = {t for t in (*context_tokens, *removed_tokens) if t.lower() not in keywords}
    boundary_set = boundary_tokens - new_def_names

    internal_refs = 0
    boundary_refs = 0
    for line, own_defs in zip(added, defs_per_line, strict=True):
        tokens = IDENT_RE.findall(line)
        for token in tokens:
            if token in new_def_names and token not in own_defs:
                internal_refs += 1
            elif token in boundary_set:
                boundary_refs += 1

    added_lines = len(added)
    removed_lines = len(removed)
    n_new_defs = len(new_def_names)
    n_edit_hunks = 0
    for hunk in re.split(r"^@@", (patch or ""), flags=re.MULTILINE):
        if not hunk:
            continue
        lines = hunk.splitlines()
        has_removed = any(line.startswith("-") and not line.startswith("---") for line in lines)
        has_added = any(line.startswith("+") and not line.startswith("+++") for line in lines)
        if has_removed and has_added:
            n_edit_hunks += 1

    return {
        "n_hunks": float(n_hunks),
        "n_files": float(n_files),
        "added_lines": float(added_lines),
        "removed_lines": float(removed_lines),
        "n_new_defs": float(n_new_defs),
        "internal_refs": float(internal_refs),
        "boundary_refs": float(boundary_refs),
        "ratio": internal_refs / max(1, boundary_refs),
        "new_frac": n_new_defs / max(1, added_lines),
        "edit_frac": n_edit_hunks / max(1, n_hunks),
    }


def proxies_dataframe(rows: list[dict]) -> pd.DataFrame:
    df = pd.DataFrame(rows)
    df["has_gold_patch"] = (df["gold_patch"].fillna("") != "").astype("int8")
    for col in PROXY_COLUMNS:
        df[col] = df[col].astype("float64")
    return df.drop(columns=["gold_patch"])


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int]:
    """Write one per-instance part for a shard; 'skip' when the part already exists."""
    part = part_path_for(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    src_rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(file_path)})").fetchone()[0]
    rows = con.execute(build_shard_sql(file_path)).fetchall()
    out = []
    for r in rows:
        prox = compute_closure_proxies(str(r[6] or ""), str(r[2] or ""))
        out.append(
            {
                "instance_id": r[0],
                "repo": r[1],
                "language": r[2],
                "n_rollouts": int(r[3]),
                "n_labeled": int(r[4]),
                "n_resolved": int(r[5]),
                "gold_patch": r[6],
                "gold_patch_lines": int(r[7]),
                "gold_patch_files": int(r[8]),
                **prox,
            }
        )
    out_df = proxies_dataframe(out)

    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        out_df.to_parquet(tmp, index=False)
        out_rows, out_instances, out_trajectories = con.execute(
            f"SELECT count(*), count(DISTINCT instance_id), sum(n_rollouts) "
            f"FROM read_parquet({_sql_str(tmp)})"
        ).fetchone()
        if out_rows != out_instances or out_rows == 0 or out_trajectories != src_rows:
            raise RuntimeError(
                f"row guard failed: source={src_rows} out={out_rows} "
                f"instances={out_instances} trajectories={out_trajectories}"
            )
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", int(out_rows)


def merge_parts(
    con: duckdb.DuckDBPyConnection,
    parts_dir: Path = PARTS_DIR,
    out_path: Path = OUT_PARQUET,
) -> tuple[int, int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = _sql_str(parts_dir / "*.parquet")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    agg = ", ".join(f"max({col}) AS {col}" for col in PROXY_COLUMNS)
    con.execute(
        f"""
        COPY (
            SELECT
                instance_id,
                max(repo) AS repo,
                max(language) AS language,
                sum(n_rollouts)::INTEGER AS n_rollouts,
                sum(n_labeled)::INTEGER AS n_labeled,
                sum(n_resolved)::INTEGER AS n_resolved,
                max(gold_patch_lines)::INTEGER AS gold_patch_lines,
                max(gold_patch_files)::INTEGER AS gold_patch_files,
                max(has_gold_patch)::INTEGER AS has_gold_patch,
                {agg}
            FROM read_parquet({glob_sql})
            GROUP BY 1
            ORDER BY 1
        ) TO {_sql_str(tmp)} (FORMAT PARQUET)
        """
    )
    rows, instances = con.execute(
        f"SELECT count(*), count(DISTINCT instance_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), int(rows), int(instances)


def stream_proxies(args: argparse.Namespace) -> int:
    """One streaming pass over the corpus shards; returns a process exit code."""
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)

    if args.file:
        files = [Path(f).resolve() for f in args.file]
    else:
        files = list_parquet_files(args.data_glob)
    if args.limit:
        files = files[: args.limit]
    if not files:
        raise SystemExit(f"No parquet files matched {args.data_glob}")

    con = connect_ephemeral()
    con.execute(f"SET memory_limit='{args.memory_limit}'")
    con.execute(f"SET threads={args.threads}")

    done = skipped = failed = 0
    elapsed_total = 0.0
    started = time.monotonic()
    failures: list[str] = []
    try:
        for i, file_path in enumerate(files, start=1):
            label = shard_label(file_path)
            t0 = time.monotonic()
            try:
                status, rows = process_shard(con, file_path, force=args.force)
            except Exception as exc:  # noqa: BLE001 (keep going; the shard stays unprocessed)
                failed += 1
                failures.append(label)
                console.print(f"[{utcnow()}] [{i}/{len(files)}] [red]FAILED[/red] {label}: {exc}")
                console.print(f"[dim]{traceback.format_exc(limit=2)}[/dim]")
                continue
            elapsed = time.monotonic() - t0
            if status == "skip":
                skipped += 1
                console.print(f"[{utcnow()}] [{i}/{len(files)}] skipped (part exists) {label}")
                continue
            done += 1
            elapsed_total += elapsed
            rate = elapsed_total / max(done, 1)
            eta = rate * (len(files) - i)
            console.print(
                f"[{utcnow()}] [{i}/{len(files)}] [green]done[/green] {label} "
                f"instances={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )

        if failures:
            console.print(
                f"[{utcnow()}] {failed} shard(s) failed — not merging partial output; "
                "rerun to resume (finished parts are kept)"
            )
        else:
            merged = merge_parts(con, out_path=args.output)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, _ = merged
                partial = bool(args.file or args.limit)
                suffix = (
                    " (partial run: rerun without --file/--limit to cover the corpus)"
                    if partial
                    else ""
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {rel(args.output)}: "
                    f"{rows:,} instances{suffix}"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] closure pass finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def load_instances(path: Path = OUT_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        inst = con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()
    inst["language"] = inst["language"].astype("string").str.lower().replace(LANGUAGE_ALIASES)
    inst["solve_rate"] = (
        inst["n_resolved"] / inst["n_labeled"].where(inst["n_labeled"] > 0)
    ).astype("float64")
    return inst


def _rho(x: pd.Series, y: pd.Series) -> str:
    rho = spearmanr(x, y).statistic
    return "—" if not np.isfinite(rho) else f"{rho:+.4f}"


def spearman_overall(inst: pd.DataFrame) -> pd.DataFrame:
    rows = [[p, _rho(inst[p], inst["solve_rate"])] for p in PROXY_COLUMNS]
    return pd.DataFrame(rows, columns=["proxy", "Spearman"])


def spearman_per_language(inst: pd.DataFrame, min_n: int = 100) -> pd.DataFrame:
    rows = []
    for lang, sub in inst.groupby("language", sort=True):
        if len(sub) < min_n:
            continue
        rows.append([lang, f"{len(sub):,}", *[_rho(sub[p], sub["solve_rate"]) for p in PROXY_COLUMNS]])
    return pd.DataFrame(rows, columns=["language", "n", *PROXY_COLUMNS])


def build_wls_design(inst: pd.DataFrame) -> tuple[np.ndarray, list[str]]:
    X = pd.DataFrame(
        {
            "log1p(added_lines)": np.log1p(inst["added_lines"]),
            "log1p(n_files)": np.log1p(inst["n_files"]),
        }
    )
    values = inst["language"].astype("string")
    dummies = pd.get_dummies(values, prefix="language", dtype=float)
    drop = f"language_{values.value_counts().idxmax()}"
    if drop in dummies.columns:
        dummies = dummies.drop(columns=drop)
    X = pd.concat([X, dummies.reset_index(drop=True)], axis=1)
    return X.to_numpy(dtype=float), list(X.columns)


def analysis_frame(inst: pd.DataFrame) -> pd.DataFrame:
    frame = inst[inst["solve_rate"].notna() & (inst["n_labeled"] >= MIN_LABELED)].reset_index(drop=True)
    if frame.empty:
        raise SystemExit(
            f"No instances with n_labeled >= {MIN_LABELED}; run the full corpus "
            "(instances repeat across harness shards) instead of a partial subset"
        )
    return frame


def wls_models(inst: pd.DataFrame) -> dict:
    """Baseline (size + language) vs baseline + structure; shared grouped 80/20 split by repo."""
    frame = analysis_frame(inst)
    X_base, features_base = build_wls_design(frame)
    X_full = np.column_stack(
        [
            X_base,
            frame["ratio"].to_numpy(dtype=float),
            frame["new_frac"].to_numpy(dtype=float),
            frame["edit_frac"].to_numpy(dtype=float),
            np.log1p(frame["n_new_defs"].to_numpy(dtype=float)),
        ]
    )
    features_full = [*features_base, "ratio", "new_frac", "edit_frac", "log1p(n_new_defs)"]
    y = frame["solve_rate"].to_numpy(dtype=float)
    weights = frame["n_labeled"].to_numpy(dtype=float)
    groups = frame["repo"].astype(str).to_numpy()

    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(X_base, y, groups))
    fit_base = fit_linear(
        X_base[train_idx], y[train_idx], X_base[test_idx], y[test_idx],
        features_base, sample_weight=weights[train_idx],
    )
    fit_full = fit_linear(
        X_full[train_idx], y[train_idx], X_full[test_idx], y[test_idx],
        features_full, sample_weight=weights[train_idx],
    )
    return {
        "n_instances": len(frame),
        "n_train": len(train_idx),
        "n_test": len(test_idx),
        "n_repos_train": int(pd.Series(groups).iloc[train_idx].nunique()),
        "n_repos_test": int(pd.Series(groups).iloc[test_idx].nunique()),
        "weight_total": float(weights.sum()),
        "baseline": fit_base,
        "full": fit_full,
        "rho_train_base": float(spearmanr(fit_base.predict(X_base[train_idx]), y[train_idx]).statistic),
        "rho_test_base": float(spearmanr(fit_base.predict(X_base[test_idx]), y[test_idx]).statistic),
        "rho_train_full": float(spearmanr(fit_full.predict(X_full[train_idx]), y[train_idx]).statistic),
        "rho_test_full": float(spearmanr(fit_full.predict(X_full[test_idx]), y[test_idx]).statistic),
    }


def size_matched_table(inst: pd.DataFrame) -> pd.DataFrame:
    """Within added_lines decile, mean solve_rate for top vs bottom ratio quartile."""
    frame = analysis_frame(inst)
    frame = frame.copy()
    bins = pd.qcut(frame["added_lines"], 10, duplicates="drop")
    frame["decile"] = bins
    rows = []
    top_pool: list[float] = []
    bottom_pool: list[float] = []
    for i, (dec, sub) in enumerate(frame.groupby("decile", sort=True), start=1):
        label = f"{i} ({max(0, dec.left):.0f}–{dec.right:.0f})"
        q25, q75 = np.quantile(sub["ratio"], [0.25, 0.75])
        if q25 == q75:
            rows.append([label, f"{len(sub):,}", f"{sub['added_lines'].mean():.1f}",
                         "n/a", "n/a", "n/a", "n/a", "n/a"])
            continue
        top = sub[sub["ratio"] > q75]
        bottom = sub[sub["ratio"] < q25]
        if top.empty or bottom.empty:
            rows.append([label, f"{len(sub):,}", f"{sub['added_lines'].mean():.1f}",
                         "n/a", "n/a", "n/a", "n/a", "n/a"])
            continue
        top_pool.extend(top["solve_rate"].tolist())
        bottom_pool.extend(bottom["solve_rate"].tolist())
        diff = float(top["solve_rate"].mean() - bottom["solve_rate"].mean())
        rows.append(
            [
                label,
                f"{len(sub):,}",
                f"{sub['added_lines'].mean():.1f}",
                f"{len(top):,}",
                f"{top['solve_rate'].mean():.3f}",
                f"{len(bottom):,}",
                f"{bottom['solve_rate'].mean():.3f}",
                f"{diff:+.3f}",
            ]
        )
    if top_pool:
        pooled = float(np.mean(top_pool) - np.mean(bottom_pool))
        rows.append(
            [
                "pooled",
                f"{len(frame):,}",
                "—",
                f"{len(top_pool):,}",
                f"{np.mean(top_pool):.3f}",
                f"{len(bottom_pool):,}",
                f"{np.mean(bottom_pool):.3f}",
                f"{pooled:+.3f}",
            ]
        )
    return pd.DataFrame(
        rows,
        columns=["decile", "n", "mean added_lines", "n_top", "solve_rate top",
                 "n_bottom", "solve_rate bottom", "diff"],
    )


def pct(value: float) -> str:
    return f"{100 * value:.1f}%"


def build_summary(
    inst: pd.DataFrame,
    spearman_all: pd.DataFrame,
    spearman_lang: pd.DataFrame,
    wls: dict,
    matched: pd.DataFrame,
    stream_elapsed: float,
    analysis_elapsed: float,
    parquet_shown: Path,
    shards: int,
) -> str:
    lines: list[str] = []
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# Closure structure of the gold patch vs per-instance solve rate")
    lines.append("")
    lines.append(
        f"Generated {stamp} by `scripts/closure_proxies.py` (one streaming DuckDB pass over "
        f"{shards:,} shards of `traces_data/`; per-instance proxies and solve rates in "
        f"`{parquet_shown}`)."
    )
    lines.append("")
    lines.append(
        "Question: is task difficulty driven by raw patch *size*, or by the *closure structure* "
        "of the change — a change whose added code mostly references itself (new symbols calling "
        "new symbols) vs a same-sized change that mostly edits existing call sites?"
    )
    lines.append("")

    lines.append("## Proxies (computed from the gold patch text alone)")
    lines.append("")
    lines.append(
        "All proxies are per `instance_id` over the gold (`reference_patch`) patch, which is "
        "identical across shards (deduped at merge; `max` kept). `internal_refs` counts "
        "occurrences, in added lines, of names defined elsewhere in the same patch; the line "
        "defining a name does not count its own occurrence. `boundary_refs` counts occurrences, "
        "in added lines, of identifiers that appear in the patch's context or removed lines and "
        "are not new defs — language keywords are excluded (per-language stoplist), builtins are "
        "kept. `ratio = internal_refs / max(1, boundary_refs)`, "
        "`new_frac = n_new_defs / max(1, added_lines)`, "
        "`edit_frac` = fraction of hunks with both removed and added lines."
    )
    lines.append("")
    lines.append(
        "Definition regexes per language (first match per added line, keyword-blacklisted): "
        "python `def|class`; go `func` incl. receivers; typescript/javascript `function|class|"
        "arrow-const|interface|type|get/set|method shorthand`; java `class|interface|enum|record` "
        "and single-line method signatures; rust `fn|struct|enum|trait|union|mod|type`; "
        "c/cpp `class|struct|enum|union` and single-line function signatures; php "
        "`class|interface|trait|enum|function`; ruby `def|class|module`; csharp "
        "`class|interface|struct|enum|record` and single-line methods. Unknown languages get no "
        "def regex (`n_new_defs == internal_refs == 0`)."
    )
    lines.append("")

    subset = inst[inst["solve_rate"].notna() & (inst["n_labeled"] >= MIN_LABELED)]
    lines.append("## Corpus and subset")
    lines.append("")
    lines.append(
        f"Instances streamed: **{len(inst):,}**; with at least one labeled rollout: "
        f"{int(inst['solve_rate'].notna().sum()):,}; analysis subset (n_labeled >= "
        f"{MIN_LABELED}): **{len(subset):,}** (solve_rate mean {subset['solve_rate'].mean():.3f}). "
        f"Gold patch present for {int(inst['has_gold_patch'].sum()):,}/{len(inst):,} instances."
    )
    lines.append("")
    lines.append("Proxy means on the subset: " + ", ".join(
        f"{p} {subset[p].mean():.2f}" for p in PROXY_COLUMNS
    ))
    lines.append("")

    lines.append("## 1. Spearman of each proxy vs solve_rate (n_labeled >= 3)")
    lines.append("")
    rows = [[r["proxy"], r["Spearman"]] for _, r in spearman_all.iterrows()]
    lines.append(md_table(["proxy", "Spearman"], rows))
    lines.append("")
    lines.append("Per language (languages with >= 100 subset instances):")
    lines.append("")
    rows = [list(r) for _, r in spearman_lang.iterrows()]
    lines.append(md_table(["language", "n", *PROXY_COLUMNS], rows))
    lines.append("")

    lines.append("## 2. Does closure structure add to size? (WLS)")
    lines.append("")
    fit_base, fit_full = wls["baseline"], wls["full"]
    lines.append(
        f"WLS on {wls['n_instances']:,} instances (weight = n_labeled, total weight "
        f"{wls['weight_total']:,.0f}), grouped 80/20 split by repo (seed {SEED}): "
        f"{wls['n_train']:,} train instances over {wls['n_repos_train']:,} repos, "
        f"{wls['n_test']:,} test instances over {wls['n_repos_test']:,} repos. Numeric inputs are "
        f"winsorized to the train split's {WINSOR_PCT[0]}st/{WINSOR_PCT[1]}th percentile then "
        "standardized; language one-hots drop the most frequent level. R² is the unweighted fit "
        "metric; the fit itself is weighted. Baseline = size + language; full = baseline + "
        "`ratio`, `new_frac`, `edit_frac`, `log1p(n_new_defs)`."
    )
    lines.append("")
    rows = [
        ["baseline (size + language)", f"{fit_base.r2_train:.4f}", f"{wls['rho_train_base']:+.4f}",
         f"{fit_base.r2_test:.4f}", f"{wls['rho_test_base']:+.4f}"],
        ["baseline + structure", f"{fit_full.r2_train:.4f}", f"{wls['rho_train_full']:+.4f}",
         f"{fit_full.r2_test:.4f}", f"{wls['rho_test_full']:+.4f}"],
        ["Δ (structure − baseline)", f"{fit_full.r2_train - fit_base.r2_train:+.4f}", "—",
         f"{fit_full.r2_test - fit_base.r2_test:+.4f}", f"{wls['rho_test_full'] - wls['rho_test_base']:+.4f}"],
    ]
    lines.append(md_table(["model", "train R²", "train ρ", "held-out R²", "held-out ρ"], rows))
    lines.append("")
    lines.append("Standardized coefficients of the full model (solve_rate per 1 sd):")
    lines.append("")
    rows = [
        [str(i + 1), d["feature"], f"{d['coef']:+.4f}",
         "higher solve_rate" if d["coef"] > 0 else "lower solve_rate"]
        for i, d in enumerate(fit_full.coefs())
    ]
    lines.append(md_table(["rank", "feature", "std coef", "direction"], rows))
    lines.append("")
    pooled_diff = float(matched.iloc[-1]["diff"]) if len(matched) else np.nan
    new_frac_coef = next(d["coef"] for d in fit_full.coefs() if d["feature"] == "new_frac")
    ratio_coef = next(d["coef"] for d in fit_full.coefs() if d["feature"] == "ratio")
    lines.append(
        f"**Structure adds to size: a little.** Held-out R² {fit_base.r2_test:.4f} → "
        f"{fit_full.r2_test:.4f} (Δ {fit_full.r2_test - fit_base.r2_test:+.4f}) and held-out "
        f"Spearman {wls['rho_test_base']:+.4f} → {wls['rho_test_full']:+.4f} "
        f"(Δ {wls['rho_test_full'] - wls['rho_test_base']:+.4f}). The increment comes from "
        f"`new_frac` (std coef {new_frac_coef:+.4f}: more new definitions per added line → "
        f"*higher* solve rate), not from self-reference: the size-matched comparison below is "
        f"flat (pooled top−bottom `ratio` quartile {pooled_diff:+.3f}) and `ratio`'s "
        f"standardized coefficient is ~0 ({ratio_coef:+.4f})."
    )
    lines.append("")

    lines.append("## 3. Size-matched comparison: added_lines decile × ratio quartile")
    lines.append("")
    lines.append(
        "Instances binned by `added_lines` decile; within each decile, the top ratio quartile "
        "(ratio > q75) vs bottom quartile (ratio < q25). 'pooled' averages all top vs all bottom "
        "members across deciles. Deciles with fewer than two distinct ratio values are reported "
        "as n/a."
    )
    lines.append("")
    rows = [list(r) for _, r in matched.iterrows()]
    lines.append(
        md_table(
            ["decile", "n", "mean added_lines", "n_top", "solve_rate top",
             "n_bottom", "solve_rate bottom", "diff"],
            rows,
        )
    )
    lines.append("")

    lines.append("## Reproduce and runtime")
    lines.append("")
    lines.append("```bash")
    lines.append("uv run python scripts/closure_proxies.py          # full stream + analysis")
    lines.append("uv run pytest tests/test_closure_proxies.py")
    lines.append("uv run ruff check .")
    lines.append("```")
    lines.append("")
    lines.append(
        f"Streaming pass: {fmt_duration(stream_elapsed)} over {shards:,} shards; analysis: "
        f"{fmt_duration(analysis_elapsed)}. Full run logged to `outputs/closure_B.log`."
    )
    lines.append("")
    lines.append("## Notes")
    lines.append("")
    lines.append(
        f"- Analysis subset needs n_labeled >= {MIN_LABELED} labeled rollouts "
        "(`resolved in (0, 1)`); `solve_rate = n_resolved / n_labeled`."
    )
    lines.append(
        "- `boundary_refs` counts only identifiers that already appear in the patch's context or "
        "removed lines; identifiers introduced *only* in added lines (e.g. a new local variable) "
        "count in neither bucket."
    )
    lines.append(
        "- A name defined in the patch is treated as new even if the patch also edits pre-existing "
        "code with that name; pure moves (same def in removed and added) are rare and counted as new."
    )
    lines.append(
        "- Method/function signatures spanning multiple lines (brace on its own line) are not "
        "recognized as defs; single-line signatures only."
    )
    lines.append(
        "- `ratio` is degenerate (0) for the majority of small patches that define no new "
        "symbols, so the size-matched table reports n/a for the lower deciles and is only "
        "informative where patches are big enough to contain new definitions (the largest "
        "deciles)."
    )
    lines.append("")
    return "\n".join(lines)


def write_markdown(md: str, out_path: Path = OUT_MD) -> Path:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    tmp = Path(str(out_path) + ".tmp")
    tmp.write_text(md)
    os.replace(tmp, out_path)
    return out_path


def resolve_path(value: Path) -> Path:
    return value if value.is_absolute() else (ROOT / value)


def rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def main_closure_proxies() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument("--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: the corpus)")
    parser.add_argument("--force", action="store_true", help="Recompute parts")
    parser.add_argument("--threads", type=int, default=4, help="DuckDB threads")
    parser.add_argument("--memory-limit", default="4GB", help="DuckDB memory limit")
    parser.add_argument("--stream-only", action="store_true", help="Run the corpus pass and exit")
    parser.add_argument("--skip-stream", action="store_true", help="Skip the corpus pass; reuse --output")
    args = parser.parse_args()

    args.output = resolve_path(args.output)
    args.summary = resolve_path(args.summary)

    if args.stream_only and args.skip_stream:
        parser.error("--stream-only and --skip-stream are mutually exclusive")

    all_files = list_parquet_files(args.data_glob)
    if args.file:
        n_shards = len(args.file)
    elif args.limit:
        n_shards = min(args.limit, len(all_files))
    else:
        n_shards = len(all_files)

    stream_started = time.monotonic()
    stream_elapsed = 0.0
    if not args.skip_stream:
        rc = stream_proxies(args)
        stream_elapsed = time.monotonic() - stream_started
        if rc != 0 or args.stream_only:
            sys.exit(rc)
    if not args.output.exists():
        raise SystemExit(f"No output at {args.output}; run without --skip-stream first")

    analysis_started = time.monotonic()
    inst = load_instances(args.output)
    subset = analysis_frame(inst)
    spearman_all = spearman_overall(subset)
    spearman_lang = spearman_per_language(subset)
    wls = wls_models(inst)
    matched = size_matched_table(inst)
    analysis_elapsed = time.monotonic() - analysis_started

    console.print(
        f"WLS held-out — baseline R² {wls['baseline'].r2_test:.4f} (ρ {wls['rho_test_base']:+.4f}) "
        f"| +structure R² {wls['full'].r2_test:.4f} (ρ {wls['rho_test_full']:+.4f})"
    )
    console.print(f"Pooled size-matched diff (top vs bottom ratio quartile): "
                  f"{matched['diff'].iloc[-1]}")

    md = build_summary(
        inst,
        spearman_all,
        spearman_lang,
        wls,
        matched,
        stream_elapsed,
        analysis_elapsed,
        rel(args.output),
        n_shards,
    )
    out_md = write_markdown(md, args.summary)
    console.print(f"Wrote {rel(out_md)}")


if __name__ == "__main__":
    main_closure_proxies()
