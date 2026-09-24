"""Symptom-to-cause distance: where the issue text points vs where the gold patch acts.

Per instance, from the issue text (``rung_features.task_text`` / ``instance.task_text``)
and the gold patch (``metadata.reference_patch.patch``):

- ``mentioned_locs``: file paths, function/class names, and stack-frame locations named
  in the issue text (regex per language: paths with extensions, ``module.func``,
  ``func(``, ``File "x.py", line N``, ``at pkg.Class.method(``, Go ``pkg/file.go:N``).
- ``gold_locs``: files and enclosing function names touched by gold *src* hunks
  (``@@ ... @@ ctx`` header context; file classes as in ``patch_split``).
- ``dist_class`` in {no_mention, same_file_same_func, same_file_other_func, other_file};
  ``distance0`` = any mentioned file/func in gold_locs; ``n_gold_funcs`` /
  ``n_gold_files`` / ``n_src_hunks`` = site counts.

No model calls; everything is deterministic regex + unified-diff parsing, unit-tested
in ``tests/test_symptom_distance.py``.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

# --- file classification (ported from oswt-closureG patch_split.py) --------------

TEST_DIR_COMPONENTS = frozenset({"test", "tests", "testing", "__tests__", "spec", "specs", "e2e"})
TEST_BASENAME_RES = [
    re.compile(r"^test_.+\.", re.IGNORECASE),
    re.compile(r".+_tests?\.", re.IGNORECASE),
    re.compile(r"^Test.+\.java$"),
    re.compile(r".+(?:Test|Tests|IT|ITCase)\.java$"),
    re.compile(r".+\.(?:test|spec)\.", re.IGNORECASE),
    re.compile(r".+_spec\.", re.IGNORECASE),
    re.compile(r"^conftest\.py$"),
]
DOC_EXTENSIONS = frozenset({".md", ".markdown", ".rst", ".txt", ".adoc"})
DOC_BASENAMES = frozenset(
    {
        "changelog", "changes", "history", "news", "license", "licence", "copying",
        "notice", "authors", "contributors", "readme", "install", "todo",
        "codeowners", "contributing",
    }
)
DOC_DIR_COMPONENTS = frozenset({"docs", "doc", "documentation"})
CONFIG_EXTENSIONS = frozenset(
    {
        ".yaml", ".yml", ".toml", ".json", ".jsonl", ".lock", ".ini", ".cfg",
        ".conf", ".properties", ".env", ".xml", ".gradle",
    }
)
CONFIG_BASENAME_RES = [
    re.compile(r"^requirements.*\.txt$"),
    re.compile(
        r"^(?:dockerfile|makefile|rakefile|gemfile|podfile|vagrantfile|brewfile|"
        r"cmakelists\.txt|manifest\.in|setup\.cfg|\.gitignore|\.gitattributes|"
        r"\.editorconfig|\.dockerignore|\.pre-commit-config\.yaml|\.babelrc|"
        r"\.eslintrc.*|\.prettierrc.*|\.stylelintrc.*|package\.json|"
        r"package-lock\.json|yarn\.lock|pnpm-lock\.yaml|poetry\.lock|pipfile.*|"
        r"cargo\.lock|go\.mod|go\.sum|composer\.(?:json|lock)|gemfile\.lock)$",
        re.IGNORECASE,
    ),
]


def classify_file(path: str) -> str:
    """Map a diff path to one of {test, config, doc, src}; first match wins."""
    parts = [p for p in path.strip().split("/") if p and p not in (".", "..")]
    if not parts:
        return "src"
    basename = parts[-1]
    stem, dot, ext = basename.rpartition(".")
    if dot:
        ext = "." + ext.lower()
        stem_lower = stem.lower()
    else:
        ext = ""
        stem_lower = basename.lower()
    dir_parts = {p.lower() for p in parts[:-1]}
    if dir_parts & TEST_DIR_COMPONENTS or any(r.match(basename) for r in TEST_BASENAME_RES):
        return "test"
    if ext in CONFIG_EXTENSIONS or any(r.match(basename) for r in CONFIG_BASENAME_RES):
        return "config"
    if ext in DOC_EXTENSIONS or stem_lower in DOC_BASENAMES or dir_parts & DOC_DIR_COMPONENTS:
        return "doc"
    return "src"


# --- unified-diff parsing --------------------------------------------------------

DIFF_GIT_RE = re.compile(r"^diff --git a/\S+ b/(\S+)")
HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@\s?(.*)$")
IDENT_RE = re.compile(r"\b[A-Za-z_]\w*\b")

CTX_STOPLIST = frozenset(
    {
        "def", "class", "function", "func", "fn", "sub", "if", "else", "elif", "for",
        "while", "return", "switch", "case", "static", "public", "private", "protected",
        "void", "int", "long", "char", "float", "double", "bool", "var", "let", "const",
        "async", "await", "export", "import", "package", "using", "namespace", "struct",
        "enum", "interface", "impl", "type", "new", "try", "catch", "throw", "throws",
        "extends", "implements", "final", "abstract", "override", "virtual", "template",
        "typename", "self", "this", "none", "null", "nil", "true", "false",
    }
)


@dataclass(frozen=True)
class Hunk:
    path: str
    func_ctx: str  # text after the second @@ of the hunk header (enclosing func ctx)
    start: int  # first new-side line number
    end: int  # last new-side line number (inclusive)
    n_add: int
    n_del: int

    @property
    def func_names(self) -> frozenset[str]:
        """Identifier candidates inside the hunk-header context."""
        return frozenset(
            t for t in IDENT_RE.findall(self.func_ctx) if t.lower() not in CTX_STOPLIST
        )


def parse_patch(patch: str | None) -> list[Hunk]:
    """Parse a unified diff into hunks with new-side line ranges + header context."""
    if not patch:
        return []
    parsed: list[tuple[str, str, list[int]]] = []
    path = ""
    cur: list[int] | None = None  # [start, end, n_add, n_del]
    ctx = ""
    for line in patch.splitlines():
        m = DIFF_GIT_RE.match(line)
        if m:
            path = m.group(1)
            cur = None
            continue
        if line.startswith("+++"):
            p = line[4:].strip()
            path = p.removeprefix("b/")
            continue
        if line.startswith("--- "):
            cur = None
            continue
        m = HUNK_RE.match(line)
        if m:
            start = int(m.group(1))
            span = int(m.group(2)) if m.group(2) else 1
            cur = [start, start + max(span - 1, 0), 0, 0]
            ctx = m.group(3) or ""
            parsed.append((path, ctx, cur))
            continue
        if cur is not None:
            if line.startswith("+"):
                cur[2] += 1
            elif line.startswith("-"):
                cur[3] += 1
    return [Hunk(p, c.strip(), v[0], v[1], v[2], v[3]) for p, c, v in parsed]


# --- mention extraction ----------------------------------------------------------

CODE_EXTENSIONS = (
    "py|pyi|js|jsx|ts|tsx|mts|cts|mjs|cjs|go|rs|java|kt|kts|scala|c|h|cc|cpp|cxx|"
    "hpp|hh|cs|rb|php|pl|pm|swift|sh|bash|zsh|sql|lua|r|jl|ex|exs|erl|hrl|clj|cljs|"
    "hs|ml|mli|fs|fsx|vb|dart|groovy|proto|thrift|vue|svelte"
)
FILE_RE = re.compile(
    rf"(?<![\w./@-])(?:[\w.~-]+/)*[\w.~-]+\.(?:{CODE_EXTENSIONS})\b(?::\d+)?"
)
PY_FRAME_RE = re.compile(r'File "([^"]+)", line (\d+)(?:, in (\S+))?')
JAVA_FRAME_RE = re.compile(r"\bat\s+([\w$]+(?:\.[\w$]+)+)\s*\(")
DOTTED_CALL_RE = re.compile(r"\b((?:[A-Za-z_]\w*\.)+[A-Za-z_]\w*)\s*\(")
CALL_RE = re.compile(r"\b([A-Za-z_]\w{2,})\s*\(")
URL_RE = re.compile(r"https?://\S+")

CALL_STOPLIST = frozenset(
    {
        "if", "for", "while", "switch", "catch", "return", "sizeof", "elif", "def",
        "class", "function", "func", "fn", "new", "delete", "print", "echo", "in",
        "e.g", "i.e", "etc", "http", "https", "www", "com", "org", "net",
        # non-location words that appear before '(' or in tracebacks
        "Traceback", "Exception", "Error", "ValueError", "TypeError", "KeyError",
        "IndexError", "AttributeError", "RuntimeError", "AssertionError",
        "ImportError", "IOError", "OSError", "FileNotFoundError", "StopIteration",
        "NotImplementedError", "NameError", "SyntaxError", "Warning",
    }
)


@dataclass
class MentionedLocs:
    """Locations named in issue text. ``pairs`` are (file, func) from stack frames."""

    files: set[str] = field(default_factory=set)
    funcs: set[str] = field(default_factory=set)
    pairs: set[tuple[str, str]] = field(default_factory=set)

    @property
    def empty(self) -> bool:
        return not self.files and not self.funcs and not self.pairs


def _norm_path(p: str) -> str:
    p = p.strip().strip("`'\".,;:()[]{}<>")
    p = p.replace("\\", "/")  # windows-style paths in tracebacks
    p = re.sub(r":\d+$", "", p)  # go-style file.go:123
    for pre in ("./", "a/", "b/"):
        p = p.removeprefix(pre)
    return p.lstrip("/")


def extract_mentions(text: str | None) -> MentionedLocs:
    """Pull file paths, function names, and stack-frame (file, func) pairs from text."""
    out = MentionedLocs()
    if not text:
        return out
    t = URL_RE.sub(" ", text)
    for m in PY_FRAME_RE.finditer(t):
        path = _norm_path(m.group(1))
        out.files.add(path)
        if m.group(3) and m.group(3) not in ("<module>", "<lambda>"):
            out.funcs.add(m.group(3))
            out.pairs.add((path, m.group(3)))
    for m in JAVA_FRAME_RE.finditer(t):
        dotted = m.group(1)
        out.funcs.add(dotted.rsplit(".", 1)[-1])
    for m in DOTTED_CALL_RE.finditer(t):
        out.funcs.add(m.group(1).rsplit(".", 1)[-1])
    for m in FILE_RE.finditer(t):
        p = _norm_path(m.group(0))
        if p and "." in p.rsplit("/", 1)[-1]:
            out.files.add(p)
    for m in CALL_RE.finditer(t):
        name = m.group(1)
        if name.lower() not in CALL_STOPLIST:
            out.funcs.add(name)
    return out


# --- distance classification -----------------------------------------------------

DIST_CLASSES = ("no_mention", "same_file_same_func", "same_file_other_func", "other_file")


def paths_match(mentioned: str, gold: str) -> bool:
    """True if a mentioned path plausibly denotes the gold path (suffix/basename)."""
    if mentioned == gold:
        return True
    if gold.endswith("/" + mentioned) or mentioned.endswith("/" + gold):
        return True
    return "/" not in mentioned and mentioned == gold.rsplit("/", 1)[-1]


def distance_features(mentions: MentionedLocs, gold_hunks: list[Hunk]) -> dict:
    """Distance class + site counts for one instance.

    ``gold_hunks`` should already be restricted to src-class hunks.
    """
    gold_files = {h.path for h in gold_hunks}
    gold_funcs_by_file: dict[str, set[str]] = {}
    for h in gold_hunks:
        gold_funcs_by_file.setdefault(h.path, set()).update(h.func_names)
    gold_funcs_all = set().union(*gold_funcs_by_file.values()) if gold_funcs_by_file else set()

    hit_files = {
        f for f in mentions.files if any(paths_match(f, g) for g in gold_files)
    }
    pair_hits = {
        (f, fn)
        for f, fn in mentions.pairs
        if fn in gold_funcs_all and any(paths_match(f, g) for g in gold_files)
    }
    func_hit = bool(mentions.funcs & gold_funcs_all) or bool(pair_hits)

    if mentions.empty:
        dist_class = "no_mention"
    elif hit_files:
        same_func = any(
            (mentions.funcs & gold_funcs_by_file.get(f, set()))
            or any(fn in gold_funcs_by_file.get(f, set()) for mf, fn in pair_hits if paths_match(mf, f))
            for f in gold_files
            if any(paths_match(mf, f) for mf in hit_files)
        )
        dist_class = "same_file_same_func" if same_func else "same_file_other_func"
    else:
        dist_class = "other_file"

    return {
        "dist_class": dist_class,
        "distance0": bool(hit_files) or func_hit,
        "n_mentioned_files": len(mentions.files),
        "n_mentioned_funcs": len(mentions.funcs),
        "n_gold_files": len(gold_files),
        "n_gold_funcs": len({(p, h.func_ctx) for h in gold_hunks for p in [h.path] if h.func_ctx}),
        "n_src_hunks": len(gold_hunks),
        "mention_file_hit": bool(hit_files),
        "mention_func_hit": func_hit,
    }


def gold_locs(patch: str | None) -> list[Hunk]:
    """Gold src-hunk locations (file + enclosing-func context) for one instance."""
    return [h for h in parse_patch(patch) if classify_file(h.path) == "src"]
