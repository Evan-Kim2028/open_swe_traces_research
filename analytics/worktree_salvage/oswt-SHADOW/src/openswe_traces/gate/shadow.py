"""Shadow-implementation gate: can a weak implementer rebuild the excised
behaviour from the contract alone?

For each unit we generate a *shadow implementation*: a second, independent
implementation of the excised functions, written by a deliberately WEAK model
whose ONLY context is the unit's contract (``contract.md`` if present, else
``instruction.md``) plus the declaration surface of the affected package —
every top-level declaration with the bodies of non-excised functions elided.
No gold patch, no hidden tests, no function bodies, no other files: that is
the whole point, and :func:`build_context` asserts it.

The shadow is spliced into a copy of the excised tree, baked into the unit's
own image (``environment/Dockerfile``), and scored by the unit's own verifier
(``tests/test.sh``), exactly as ``scripts/ops/devin_solve.sh`` scores a solver
patch. Per-assertion results are parsed from the ``go test`` output.

    shadow passes  -> the contract carries enough information to reconstruct
                      the behaviour (the property trials were buying).
    shadow fails   -> either the contract is incomplete/wrong, or the suite
                      demands something only gold can produce (e.g. a literal
                      digest). Both defect classes fall out of one check,
                      and the failing assertion names the defect.

Generation is cached under ``outputs/shadow_gate/`` keyed on the prompt hash,
results append to ``outputs/shadow_gate/results.jsonl``, and a unit already
in that file is skipped — the job is resume-safe.
"""

from __future__ import annotations

import hashlib
import json
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import time
from collections.abc import Iterable
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT

# Deliberately WEAK models, served through cursor-agent (scripts/ops/
# ask_composer.py) — NOT OpenRouter, whose shared free-tier rate limits made
# four earlier gate jobs queue for hours and produce nothing. The gate's
# value is that a weak implementer suffices: if a weak model can rebuild the
# behaviour from the contract, the contract is complete. The first model is
# the weak pick of record; later entries are flake fallbacks only, and each
# unit records the model it actually used.
SHADOW_MODELS = (
    "gemini-3.6-flash-minimal",  # the weak pick of record: gpt-5.4-nano-low
                               # could not emit compilable Go even with the
                               # full signature surface (wrong arity, unused
                               # imports, through all repair passes); flash-
                               # minimal is the weakest model that compiles
    "gpt-5.4-mini-low",          # flake fallback only
)

MAX_REQUESTS = 300
MAX_ATTEMPTS = 4            # per generation call, across models
MAX_REPAIRS = 3             # compile-repair passes per unit
MAX_DIGEST_CHARS = 160_000  # cap on the package declaration surface — a
                            # smaller cap silently cuts signatures mid-file
                            # (it hid registry.Client.Pull once and the model
                            # hallucinated DownloadChart); keep it above what
                            # the largest dep surface needs
MAX_CONTRACT_CHARS = 40_000
MAX_TOTAL_CHARS = 240_000
MAX_ERROR_CHARS = 12_000    # compiler output fed back for repair
REQUEST_TIMEOUT = 300
TEST_TIMEOUT = 1500         # seconds for one docker verifier run
BUILD_TIMEOUT = 900

GATE_DIR = ROOT / "outputs" / "shadow_gate"
GEN_DIR = GATE_DIR / "gen"
CTX_DIR = GATE_DIR / "contexts"
RESULTS_PATH = GATE_DIR / "results.jsonl"
LOG_PATH = ROOT / "outputs" / "SHADOW.log"

EXCISED_RE = re.compile(r'panic\("excised:[^"]*"\)')
TESTFILE_RE = re.compile(r"_test\.go$")
# Sources that must never reach the generator's context. Asserted in code.
# Scoped to verifier/packaging artifacts: a src package can legitimately be
# named `tests` or `hidden` (kops ships util/pkg/vfs/tests) and `gold` is a
# substring of real library names (goldmark), so those match only as
# verifier-shaped path components.
FORBIDDEN_RE = re.compile(
    r"_test\.go$|(^|/)(gold|cheat)(/|\.|_)|\.patch$|"
    r"validation\.json|affordance\.json|task\.toml|Dockerfile",
    re.IGNORECASE,
)

FENCE_RE = re.compile(r"```(?:go|golang)?\s*\n(.*?)```", re.DOTALL)
IMPORT_BLOCK_RE = re.compile(r"import\s*\(\s*([^)]*?)\s*\)", re.DOTALL)
IMPORT_SINGLE_RE = re.compile(r'(?m)^import\s+"([^"]+)"')
IMPORT_LINE_RE = re.compile(r'"([^"]+)"')
IMPORT_ALIAS_RE = re.compile(r'(?m)^\s*(?:(\w+|\.|_)\s+)?"([^"]+)"')
FAIL_TEST_RE = re.compile(r"^--- FAIL:\s+(\w+)", re.MULTILINE)
PASS_TEST_RE = re.compile(r"^--- PASS:\s+(\w+)", re.MULTILINE)
BUILD_FAIL_RE = re.compile(r"^FAIL\s+\S+\s+\[build failed\]", re.MULTILINE)
COMPILE_ERR_RE = re.compile(r"^(?:\./)?[\w./-]+\.go:\d+:\d+:.*$", re.MULTILINE)
FATAL_MSG_RE = re.compile(r"^\s*([\w./-]+\.go:\d+:.*)$", re.MULTILINE)
DECL_RE = re.compile(r"(?m)^(func|type|var|const|import)\b")
WORD_CHARS = set("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")


def log(msg: str) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    line = f"{time.strftime('%H:%M:%S')} {msg}"
    print(line, flush=True)
    with LOG_PATH.open("a") as f:
        f.write(line + "\n")


# --------------------------------------------------------------------------
# Go source scanning: top-level decls, brace matching, excised-stub detection
# --------------------------------------------------------------------------


def _strip_strings_comments(src: str) -> str:
    """Replace string/char/comment content with spaces, preserving offsets.

    What remains is safe for brace matching and decl-boundary searches:
    keywords, punctuation, and structural whitespace are untouched.
    """
    out = list(src)
    i, n = 0, len(src)
    while i < n:
        c = src[i]
        if c == "/" and i + 1 < n and src[i + 1] == "/":
            j = i
            while j < n and src[j] != "\n":
                out[j] = " "
                j += 1
            i = j
        elif c == "/" and i + 1 < n and src[i + 1] == "*":
            j = i
            while j + 1 < n and not (src[j] == "*" and src[j + 1] == "/"):
                if src[j] != "\n":
                    out[j] = " "
                j += 1
            for k in range(j, min(j + 2, n)):
                if src[k] != "\n":
                    out[k] = " "
            i = j + 2
        elif c in ('"', "'", "`"):
            quote = c
            out[i] = " "
            i += 1
            while i < n:
                if src[i] == "\\" and quote != "`":
                    out[i] = " "
                    if i + 1 < n and src[i + 1] != "\n":
                        out[i + 1] = " "
                    i += 2
                    continue
                if src[i] == quote:
                    out[i] = " "
                    i += 1
                    break
                if src[i] != "\n":
                    out[i] = " "
                i += 1
        else:
            i += 1
    return "".join(out)


def _word_before(mask: str, pos: int, lo: int) -> str:
    """The identifier immediately before ``pos`` (skipping spaces), or ""."""
    j = pos - 1
    while j >= lo and mask[j] in " \t":
        j -= 1
    end = j + 1
    while j >= lo and mask[j] in WORD_CHARS:
        j -= 1
    return mask[j + 1 : end]


def _skip_balanced(mask: str, pos: int, limit: int) -> int:
    """pos is at '{'; return offset just past its match (or limit)."""
    depth, i = 1, pos + 1
    while i < limit and depth:
        if mask[i] == "{":
            depth += 1
        elif mask[i] == "}":
            depth -= 1
        i += 1
    return i


@dataclass
class FuncSpan:
    """A top-level ``func`` declaration: offsets into the file text."""

    name: str
    recv: str          # receiver type name, "" for plain funcs
    start: int         # offset of the 'func' keyword
    sig_end: int       # offset just past the '{' opening the body
    end: int           # offset just past the matching '}'
    excised: bool      # body is a panic("excised: ...") stub

    @property
    def key(self) -> tuple[str, str]:
        return (self.recv, self.name)


_FUNC_SIG_RE = re.compile(
    r"func\s*(?:\(\s*[\w*]+\s+\*?([\w.]+)\s*\)\s*)?(\w+)\s*(?:\[[^\]]*\])?\s*\(")


def go_func_spans(src: str) -> list[FuncSpan]:
    """Locate every top-level func decl (col-0 `func`) with brace matching.

    The body-open brace is the first '{' that is not the brace of an
    ``interface{...}`` / ``struct{...}`` type inside the signature — those
    are skipped as balanced pairs, so multi-line interface types and
    signature braces cannot corrupt the span.
    """
    mask = _strip_strings_comments(src)
    decls = [m for m in DECL_RE.finditer(mask) if mask.startswith("func", m.start())]
    bounds = [m.start() for m in DECL_RE.finditer(mask)]
    spans: list[FuncSpan] = []
    for m in decls:
        start = m.start()
        limit = min((b for b in bounds if b > start), default=len(mask))
        # find the body-open brace: first '{' not owned by interface{/struct{
        i, sig_end = m.end(), -1
        while i < limit:
            if mask[i] == "{":
                if _word_before(mask, i, start) in ("interface", "struct"):
                    i = _skip_balanced(mask, i, limit)
                    continue
                sig_end = i + 1
                break
            i += 1
        if sig_end < 0:
            continue
        end = _skip_balanced(mask, sig_end - 1, len(mask))
        head = src[start:sig_end]
        body = src[sig_end:end]
        hm = _FUNC_SIG_RE.match(head)
        if not hm:
            continue
        recv, name = hm.group(1) or "", hm.group(2)
        spans.append(
            FuncSpan(name=name, recv=recv, start=start, sig_end=sig_end,
                     end=end, excised=bool(EXCISED_RE.search(body))))
    return spans


def signature_digest(src: str, exported_only: bool = False) -> str:
    """The declaration surface of one file: every decl verbatim except that
    bodies of non-excised functions are replaced by ``{ /* body omitted */ }``.
    Excised panic stubs are kept whole — they are the work list.

    ``exported_only`` drops unexported top-level funcs entirely (used for
    imported dependency packages, whose unexported decls are not callable)."""
    spans = go_func_spans(src)
    if not spans:
        return src
    out: list[str] = []
    prev = 0
    for sp in spans:
        if exported_only and not sp.name[:1].isupper():
            out.append(src[prev:sp.start])
            prev = sp.end
            continue
        out.append(src[prev:sp.start])
        if sp.excised:
            out.append(src[sp.start:sp.end])
        else:
            out.append(src[sp.start:sp.sig_end] + " /* body omitted */ }")
        prev = sp.end
    out.append(src[prev:])
    return "".join(out)


def find_excised_files(src_root: Path) -> dict[Path, str]:
    """All non-test .go files under src_root containing an excised stub."""
    out: dict[Path, str] = {}
    for f in sorted(src_root.rglob("*.go")):
        if TESTFILE_RE.search(f.name):
            continue
        text = f.read_text(errors="replace")
        if EXCISED_RE.search(text):
            out[f.relative_to(src_root)] = text
    return out


# --------------------------------------------------------------------------
# Context: contract + package declaration surface, nothing else
# --------------------------------------------------------------------------


@dataclass
class Context:
    prompt: str
    manifest: list[dict]      # every source that fed the prompt
    excised: dict[str, list[tuple[str, str]]]  # file -> [(recv, name)]
    context_hash: str


def _contract_text(unit_dir: Path) -> tuple[str, str]:
    for name in ("contract.md", "instruction.md"):
        p = unit_dir / name
        if p.is_file():
            return name, p.read_text(errors="replace")
    raise FileNotFoundError(f"{unit_dir}: no contract.md or instruction.md")


def _truncate(text: str, cap: int) -> str:
    if len(text) <= cap:
        return text
    head = cap * 3 // 4
    return text[:head] + f"\n...[{len(text) - cap} chars elided]...\n" + text[-(cap - head):]


def _module_path(src_root: Path) -> str:
    gomod = src_root / "go.mod"
    if gomod.is_file():
        m = re.search(r"(?m)^module\s+(\S+)", gomod.read_text(errors="replace"))
        if m:
            return m.group(1)
    return ""


def _import_paths(src: str) -> list[str]:
    out: list[str] = []
    for im in IMPORT_BLOCK_RE.finditer(src):
        out += IMPORT_LINE_RE.findall(im.group(1))
    out += IMPORT_SINGLE_RE.findall(src)
    return out


def build_context(unit_dir: Path) -> Context:
    """Assemble the generator's ONLY context and prove it is clean.

    Sources: the contract text; the signature digest of every non-test .go
    file in each package holding an excised stub; and the exported-decl digest
    of each module-internal package those files import (a re-implementer must
    know the field and function shapes it writes against — but never a body).
    The manifest is logged and asserted: no test file, nothing under tests/,
    no gold/cheat patch, no packaging metadata may contribute a byte.
    """
    src_root = unit_dir / "environment" / "src"
    excised_files = find_excised_files(src_root)
    if not excised_files:
        raise FileNotFoundError(f"{unit_dir}: no excised stubs under {src_root}")

    pkg_dirs = sorted({p.parent for p in excised_files})
    manifest: list[dict] = []
    excised_map: dict[str, list[tuple[str, str]]] = {}

    contract_name, contract = _contract_text(unit_dir)
    manifest.append({"source": contract_name,
                     "sha256": hashlib.sha256(contract.encode()).hexdigest()[:16]})

    module = _module_path(src_root)
    excised_set = {str(p) for p in excised_files}
    digest_parts: list[str] = []
    dep_dirs: set[Path] = set()
    for pkg in pkg_dirs:
        for f in sorted((src_root / pkg).glob("*.go")):
            rel = f.name if str(pkg) == "." else f"{pkg}/{f.name}"
            if TESTFILE_RE.search(f.name):
                continue  # test files never enter context
            raw = f.read_text(errors="replace")
            for imp in _import_paths(raw):
                if module and imp.startswith(module + "/"):
                    dep_dirs.add(Path(imp[len(module) + 1:]))
            digest = signature_digest(raw)
            manifest.append({
                "source": f"environment/src/{rel}",
                "sha256": hashlib.sha256(digest.encode()).hexdigest()[:16],
                "form": "signature digest (non-excised bodies omitted)",
            })
            if rel in excised_set:
                spans = go_func_spans(raw)
                excised_map[rel] = [sp.key for sp in spans if sp.excised]
            digest_parts.append(f"### FILE: {rel}\n{digest}\n")

    # dependency packages, exported decls only — no bodies anywhere
    dep_dirs -= set(pkg_dirs)
    for dep in sorted(dep_dirs):
        ddir = src_root / dep
        if not ddir.is_dir():
            continue
        for f in sorted(ddir.glob("*.go")):
            if TESTFILE_RE.search(f.name):
                continue
            raw = f.read_text(errors="replace")
            digest = signature_digest(raw, exported_only=True)
            manifest.append({
                "source": f"environment/src/{dep}/{f.name}",
                "sha256": hashlib.sha256(digest.encode()).hexdigest()[:16],
                "form": "exported signatures only (dependency package)",
            })
            digest_parts.append(f"### DEPENDENCY FILE: {dep}/{f.name}\n{digest}\n")

    for entry in manifest:
        src = entry["source"]
        assert not FORBIDDEN_RE.search(src), f"forbidden source in context: {src}"
    assert excised_map, "no excised symbols found in digested files"

    digest_block = _truncate("\n".join(digest_parts), MAX_DIGEST_CHARS)
    work_list = "\n".join(
        f"  {rel}: " + ", ".join(f"{r + '.' if r else ''}{n}" for r, n in keys)
        for rel, keys in excised_map.items()
    )
    prompt = f"""You are re-implementing excised functions in a Go package. You are
given (1) the task CONTRACT and (2) the package's declaration surface — every
top-level declaration with the bodies of non-excised functions omitted. That
is ALL you see: there is no test file, no reference implementation, and no
other file. Implement only from what is written here.

CONTRACT:
{_truncate(contract, MAX_CONTRACT_CHARS)}

PACKAGE DECLARATION SURFACE (bodies of non-excised functions omitted):
{digest_block}

FUNCTIONS TO IMPLEMENT (every body currently reads panic("excised: ...")):
{work_list}

RULES:
- Output ONE ```go fenced block and nothing else.
- In it, write one complete `func` declaration per excised function. Copy the
  signature EXACTLY as shown above (receiver, name, parameters, results).
- If your code needs imports the file does not already have, put an
  `import ( ... )` block before the funcs listing ONLY the extra imports.
- You may add unexported helper functions and helper type declarations;
  they go in the same package.
- Do NOT emit a package clause, test code, or commentary.
- Implement what the CONTRACT says, not what you remember of the upstream
  codebase. Where your prior knowledge of this library disagrees with the
  contract, the contract wins.

Implement every function in the work list."""
    prompt = _truncate(prompt, MAX_TOTAL_CHARS)
    return Context(
        prompt=prompt,
        manifest=manifest,
        excised=excised_map,
        context_hash=hashlib.sha256(prompt.encode()).hexdigest()[:16],
    )


# --------------------------------------------------------------------------
# Composer client (cursor-agent via scripts/ops/ask_composer.py) — cached,
# budgeted, logged. Token counts are len(prompt+text)//4 estimates: the CLI
# does not report usage, so treat ``tokens`` as a scale, not a bill.
# --------------------------------------------------------------------------


@dataclass
class GenResult:
    text: str | None
    model: str
    requests: int
    tokens: int          # estimated: (prompt+completion chars)//4
    error: str | None = None
    cache_hit: bool = False


def _composer(prompt: str, model: str, cache_key: str) -> str:
    """One cursor-agent one-shot through the repo helper. ``scripts/ops`` is
    not a package, so it is added to sys.path lazily. The helper handles the
    eval_tasks/.env credentials (which must beat an inherited CURSOR_API_KEY)
    and runs the call in an empty temp cwd so the agent cannot wander into
    the repo."""
    ops = str(ROOT / "scripts" / "ops")
    if ops not in sys.path:
        sys.path.insert(0, ops)
    from ask_composer import ask
    return ask(prompt, cache_key=cache_key, model=model, timeout=REQUEST_TIMEOUT)


def _composer_cached(cache_key: str) -> bool:
    ops = str(ROOT / "scripts" / "ops")
    if ops not in sys.path:
        sys.path.insert(0, ops)
    import ask_composer
    return (ask_composer.CACHE / f"{ask_composer._key('', cache_key)}.json").is_file()


def generate(
    prompt: str,
    tag: str,
    models: Iterable[str] = SHADOW_MODELS,
    budget: int = MAX_REQUESTS,
) -> GenResult:
    """One generation, cached on the prompt hash. ``tag`` scopes the cache
    (initial generation vs. repair passes). A request counts only when the
    composer cache misses — a hit costs no cursor-agent call."""
    GEN_DIR.mkdir(parents=True, exist_ok=True)
    requests, tokens = 0, 0
    last_err = "no models configured"
    models = list(models)
    for attempt in range(MAX_ATTEMPTS):
        model = models[min(attempt, len(models) - 1)]
        # cache key is per-model: a --models override must not replay another
        # model's generation
        key = hashlib.sha256((tag + "\n" + model + "\n" + prompt).encode()).hexdigest()[:24]
        path = GEN_DIR / f"{key}.json"
        if path.is_file():
            try:
                data = json.loads(path.read_text())
                return GenResult(data["text"], data.get("model", "cached"),
                                 int(data.get("requests", 0)),
                                 int(data.get("tokens", 0)), cache_hit=True)
            except (OSError, json.JSONDecodeError, KeyError):
                pass
        if requests >= budget:
            return GenResult(None, model, requests, tokens, "request budget exhausted")
        cache_key = f"shadow-gate/{key}/{model}"
        if not _composer_cached(cache_key):
            requests += 1
        try:
            text = _composer(prompt, model, cache_key).strip()
        except Exception as e:  # noqa: BLE001 — CLI failure is data, retried
            last_err = f"{type(e).__name__}: {e}"[-300:]
            time.sleep(10 * (attempt + 1))
            continue
        tokens += (len(prompt) + len(text)) // 4
        if text:
            path.write_text(json.dumps(
                {"model": model, "tokens": tokens, "requests": requests,
                 "tag": tag, "prompt_sha": key, "text": text}, indent=1) + "\n")
            return GenResult(text, model, requests, tokens)
        last_err = "empty completion"
    return GenResult(None, models[-1], requests, tokens, last_err)


# --------------------------------------------------------------------------
# Response parsing + splice into the excised tree
# --------------------------------------------------------------------------


@dataclass
class ShadowCode:
    imports: list[str]                     # extra import paths
    funcs: dict[tuple[str, str], str]      # (recv, name) -> full func decl text
    aliases: dict[str, str] = field(default_factory=dict)  # path -> alias
    helpers: list[str] = field(default_factory=list)       # type/var/const decls


def _top_level_decls(code: str) -> list[str]:
    """Non-func, non-import top-level decls (``type``/``var``/``const``),
    captured verbatim. Weak models invent helper types (``rawIndexFile``)
    the methods they emit need; discarding them makes a valid design look
    like a compile error."""
    mask = _strip_strings_comments(code)
    out: list[str] = []
    for m in DECL_RE.finditer(mask):
        kw = mask[m.start():m.end()]
        if kw in ("func", "import"):
            continue
        i = m.end()
        while i < len(mask) and mask[i] in " \t":
            i += 1
        if i < len(mask) and mask[i] == "(":       # grouped decl: type ( ... )
            depth, j = 1, i + 1
            while j < len(mask) and depth:
                if mask[j] == "(":
                    depth += 1
                elif mask[j] == ")":
                    depth -= 1
                j += 1
            out.append(code[m.start():j])
            continue
        # single decl: find '{' if it opens a composite type, else the line
        depth, j, brace = 0, i, -1
        while j < len(mask) and mask[j] != "\n":
            if mask[j] == "{":
                brace = j
                break
            j += 1
        if brace >= 0:
            end = _skip_balanced(mask, brace, len(mask))
            out.append(code[m.start():end])
        else:
            out.append(code[m.start():j])
    return out


def parse_response(text: str) -> ShadowCode:
    m = FENCE_RE.search(text)
    code = m.group(1) if m else text
    imports: list[str] = []
    aliases: dict[str, str] = {}
    for im in IMPORT_BLOCK_RE.finditer(code):
        for a in IMPORT_ALIAS_RE.finditer(im.group(1)):
            imports.append(a.group(2))
            if a.group(1):
                aliases[a.group(2)] = a.group(1)
    imports += IMPORT_SINGLE_RE.findall(code)
    funcs: dict[tuple[str, str], str] = {}
    for sp in go_func_spans(code):
        funcs.setdefault(sp.key, code[sp.start:sp.end])
    return ShadowCode(imports=imports, funcs=funcs, aliases=aliases,
                      helpers=_top_level_decls(code))


def _pkg_name_for(path: str) -> str:
    base = path.rsplit("/", 1)[-1]
    if base.startswith("v") and base[1:].isdigit() and "/" in path:
        base = path.rsplit("/", 2)[-2]
    return re.sub(r"\W", "", base)


def _pkg_name_candidates(path: str) -> set[str]:
    """Plausible package identifiers for an import path. The declared package
    name is not derivable from the path alone (``k8s.io/api/core/v1`` is
    ``package v1``; ``gopkg.in/yaml.v3`` is ``package yaml``), so return every
    candidate a splice might reference."""
    out = {_pkg_name_for(path)}
    last = path.rsplit("/", 1)[-1]
    if last.startswith("v") and last[1:].isdigit() and "/" in path:
        out.add(last)                                  # package vN (k8s)
    m = re.match(r"^(.*?)\.v\d+$", last)
    if m:                                              # yaml.v3 -> yaml
        out.add(re.sub(r"\W", "", m.group(1)))
    out.discard("")
    return out


_IMPORT_SPEC_RE = re.compile(r'(?m)^[ \t]*(?:(\w+|\.|_)[ \t]+)?"([^"]+)"')
_IMPORT_SINGLE_SPEC_RE = re.compile(
    r'(?m)^import[ \t]+(?:(\w+|\.|_)[ \t]+)?"([^"]+)"')


def _import_specs(file_text: str) -> list[tuple[str, str, int, int]]:
    """(bound_name, path, start, end) for each import spec, where bound_name
    is the explicit alias/blank/dot or the inferred package name. Spans cover
    the ``name "path"`` portion so a bound name can be rewritten in place."""
    specs: list[tuple[str, str, int, int]] = []
    for b in IMPORT_BLOCK_RE.finditer(file_text):
        base = b.start(1)
        for m in _IMPORT_SPEC_RE.finditer(b.group(1)):
            lead = len(m.group(0)) - len(m.group(0).lstrip())
            specs.append((m.group(1) or _pkg_name_for(m.group(2)),
                          m.group(2), base + m.start() + lead, base + m.end()))
    for m in _IMPORT_SINGLE_SPEC_RE.finditer(file_text):
        specs.append((m.group(1) or _pkg_name_for(m.group(2)),
                      m.group(2), m.start() + len("import "), m.end()))
    return specs


def _resolve_imports(file_text: str, extra: list[str],
                     allowed: set[str] | None = None,
                     aliases: dict[str, str] | None = None) -> str:
    """Make the file's imports serve the spliced code.

    - upgrade ``_``/``.`` specs to a name the code references (a blank import
      cannot be referenced; deepest path wins a contested name);
    - merge ``extra`` imports whose package is referenced and resolvable
      offline, honouring aliases;
    - rebind an existing spec when the code needs the same path under a
      different name and the current name is unused;
    - never bind a second path to a name already taken (``redeclared``)."""
    aliases = aliases or {}
    specs = _import_specs(file_text)
    by_path: dict[str, list] = {p: [n, s, e] for n, p, s, e in specs}
    by_name = {n: p for n, p, _, _ in specs if n not in "_."}
    rewrites: list[tuple[int, int, str]] = []
    additions: list[tuple[str | None, str]] = []

    def referenced(name: str) -> bool:
        return bool(re.search(rf"\b{re.escape(name)}\s*\.", file_text))

    def rebind(path: str, pick: str) -> None:
        name, s, e = by_path[path]
        spec = f'{pick} "{path}"' if pick != _pkg_name_for(path) else f'"{path}"'
        rewrites.append((s, e, spec))
        by_name.pop(name, None)
        by_name[pick] = path
        by_path[path][0] = pick

    # blank/dot specs, deepest path first so a module root never beats the
    # API package that actually exports the referenced members
    for name, path, s, e in sorted(specs, key=lambda t: -len(t[1])):
        if name not in "_.":
            continue
        alias = aliases.get(path)
        cands = ([alias] if alias not in (None, "_", ".") else []) + \
            sorted(_pkg_name_candidates(path))
        pick = next((c for c in cands if referenced(c) and c not in by_name), None)
        if pick:
            rebind(path, pick)
    for p in extra:
        want = aliases.get(p)
        want = want if want not in (None, "_", ".") else None
        cands = [want] if want else sorted(_pkg_name_candidates(p))
        refs = [c for c in cands if referenced(c)]
        if p in by_path:
            name = by_path[p][0]
            if refs and name not in refs and (name in "_." or not referenced(name)):
                pick = next((c for c in refs
                             if c not in by_name or by_name[c] == p), None)
                if pick:
                    rebind(p, pick)
            continue
        if not _import_resolvable(p, allowed, file_text, want):
            continue
        bound = want or cands[0]
        if bound in by_name and by_name[bound] != p:
            continue
        additions.append((want, p))
        by_name[bound] = p
        by_path[p] = [bound, -1, -1]

    for s, e, txt in sorted(rewrites, reverse=True):
        file_text = file_text[:s] + txt + file_text[e:]
    if additions:
        lines = "".join(f'\t{a + " " if a else ""}"{p}"\n' for a, p in additions)
        m = re.search(r"(?m)^import\s*\(", file_text)
        if m:
            close = file_text.index(")", m.end())
            return file_text[:close] + lines + file_text[close:]
        m2 = re.search(r'(?m)^import\s+"[^"]+"\s*$', file_text)
        if m2:
            old = IMPORT_LINE_RE.search(m2.group(0))
            block = f'import (\n\t"{old.group(1) if old else ""}"\n{lines})'
            return file_text[:m2.start()] + block + file_text[m2.end():]
        pm = re.search(r"(?m)^package\s+\w+\s*$", file_text)
        if pm:
            return file_text[:pm.end()] + f"\n\nimport (\n{lines})\n" + file_text[pm.end():]
    return file_text


def _importable_paths(src_root: Path) -> set[str] | None:
    """Import paths the build can resolve without a network: stdlib-shaped
    paths are NOT included here — they pass the no-dot first-segment test in
    ``_resolve_imports``. Returns module-internal dirs plus every module in
    go.mod's require/replace. ``None`` means no go.mod — allow everything."""
    gomod = src_root / "go.mod"
    if not gomod.is_file():
        return None
    out: set[str] = set()
    mod = _module_path(src_root)
    for d in src_root.rglob(""):
        if d.is_dir() and any(c.suffix == ".go" for c in d.iterdir()):
            rel = d.relative_to(src_root).as_posix()
            out.add(f"{mod}/{rel}" if mod and rel != "." else rel)
    for line in gomod.read_text(errors="replace").splitlines():
        line = line.strip()
        if not line or line.startswith("//"):
            continue
        m = re.match(r"(?:require\s+)?([^\s]+)\s+v[\w.+-]+", line)
        if m:
            out.add(m.group(1))
        m = re.match(r"replace\s+(?:[^\s]+\s*=>\s*)?([^\s]+)", line)
        if m:
            out.add(m.group(1))
    return out


# Stdlib packages a weak model routinely *references* without declaring —
# every gate fail_build on `undefined: sha256`/`hex`/`json` is a mechanical
# miss the repair loop wastes passes on. Auto-added only when the identifier
# is referenced, not already imported, and not shadowed by a local decl.
STDLIB_IMPORTS = {
    "sha256": "crypto/sha256", "sha1": "crypto/sha1", "md5": "crypto/md5",
    "hex": "encoding/hex", "json": "encoding/json",
    "base64": "encoding/base64", "csv": "encoding/csv", "xml": "encoding/xml",
    "fmt": "fmt", "errors": "errors", "sort": "sort", "strings": "strings",
    "bytes": "bytes", "io": "io", "os": "os", "context": "context",
    "time": "time", "strconv": "strconv", "regexp": "regexp", "sync": "sync",
    "filepath": "path/filepath", "http": "net/http", "url": "net/url",
    "slices": "slices", "maps": "maps", "math": "math", "rand": "math/rand",
    "fs": "io/fs", "exec": "os/exec", "signal": "os/signal",
    "template": "text/template", "log": "log", "slog": "log/slog",
    "atomic": "sync/atomic", "reflect": "reflect", "hash": "hash",
    "cipher": "crypto/cipher", "tls": "crypto/tls", "x509": "crypto/x509",
    "pem": "encoding/pem", "binary": "encoding/binary",
}


def _stdlib_fixups(file_text: str, have: set[str], bound: set[str]) -> list[str]:
    """Import paths for stdlib packages referenced in ``file_text`` but not
    yet imported. ``pkg.`` references that are actually local variables,
    parameters or named decls are excluded — adding a real import for those
    would produce "imported and not used". ``bound`` names (already taken by
    another path, e.g. a forked ``text/template``) are skipped too."""
    out: list[str] = []
    for name, path in STDLIB_IMPORTS.items():
        if (path in have or name in bound
                or not re.search(rf"\b{name}\s*\.", file_text)):
            continue
        if re.search(rf"(?m)(var\s+{name}\b|\b{name}\s*:=|\(\s*\*?\w*\s+{name}\b"
                     rf"|func\s+{name}\b|type\s+{name}\b|\b{name}\s+\w+\s*[,)=])",
                     file_text):
            continue
        out.append(path)
    return out


def _import_resolvable(path: str, allowed: set[str] | None, used_code: str,
                       alias: str | None = None) -> bool:
    """Merge an import only if the spliced code references its package AND
    the build can resolve it offline. A hallucinated path (wrong module,
    invented dep) is dropped so the compiler error names a missing symbol —
    the repair loop then fixes the reference instead of fetching a phantom
    module under --network none."""
    if allowed is None:
        return True
    if alias and alias not in "_.":
        cands = {alias}
    else:
        cands = _pkg_name_candidates(path)
    if not any(re.search(rf"\b{re.escape(c)}\s*\.", used_code) for c in cands):
        return False                      # merged into a file that cannot use it
    if "." not in path.split("/", 1)[0]:
        return True                       # stdlib-shaped
    return any(path == a or path.startswith(a + "/") for a in allowed)


def _fix_imports(file_text: str, shadow: ShadowCode,
                 allowed: set[str] | None) -> str:
    """Model-declared imports, then stdlib fixups for referenced packages."""
    file_text = _resolve_imports(file_text, shadow.imports, allowed,
                                 shadow.aliases)
    specs = _import_specs(file_text)
    return _resolve_imports(
        file_text,
        _stdlib_fixups(file_text,
                       {p for _, p, _, _ in specs},
                       {n for n, _, _, _ in specs if n not in "_."}),
        allowed)


@dataclass
class ApplyReport:
    ok: bool
    implemented: list[str] = field(default_factory=list)
    missing: list[str] = field(default_factory=list)
    note: str = ""


def apply_shadow(src_dir: Path, excised_files: dict[Path, str],
                 shadow: ShadowCode) -> ApplyReport:
    """Splice shadow func decls over their stubs inside ``src_dir`` (already a
    writable copy). Every stub must be implemented or the apply fails.
    Unmatched funcs are appended to the first excised file as helpers."""
    report = ApplyReport(ok=True)
    missing: list[str] = []
    used: set[tuple[str, str]] = set()
    allowed = _importable_paths(src_dir)
    for rel_path, text in excised_files.items():
        rel = str(rel_path)
        spans = go_func_spans(text)
        out: list[str] = []
        prev = 0
        for sp in spans:
            if not sp.excised:
                continue
            impl = shadow.funcs.get(sp.key)
            out.append(text[prev:sp.start])
            if impl is not None:
                out.append(impl)
                used.add(sp.key)
                report.implemented.append(
                    f"{rel}:{sp.recv + '.' if sp.recv else ''}{sp.name}")
            else:
                out.append(text[sp.start:sp.end])
                missing.append(f"{rel}:{sp.recv + '.' if sp.recv else ''}{sp.name}")
            prev = sp.end
        out.append(text[prev:])
        new_text = "".join(out)
        new_text = _fix_imports(new_text, shadow, allowed)
        (src_dir / rel_path).write_text(new_text)
    # helper decls the model emitted that match no stub: append them unless
    # the name collides with an existing decl in the package (a redeclared
    # helper is a compile error, not a semantic result)
    existing = {sp.name for t in excised_files.values()
                for sp in go_func_spans(t)}
    for t in excised_files.values():
        existing |= set(re.findall(r"(?m)^type\s+(\w+)", t))
        existing |= set(re.findall(r"(?m)^(?:var|const)\s+(\w+)", t))
    extras = [d for k, d in shadow.funcs.items()
              if k not in used and k[1] not in existing]
    helper_decls = [h for h in shadow.helpers
                    if not (m := re.match(r"(?:type|var|const)\s*\(?\s*(\w+)", h))
                    or m.group(1) not in existing] + extras
    if helper_decls:
        first = src_dir / next(iter(excised_files))
        # helpers may reference packages nothing else did — resolve imports
        # once more after they land
        t = first.read_text() + "\n// --- shadow helpers ---\n\n" + "\n\n".join(helper_decls)
        first.write_text(_fix_imports(t, shadow, allowed))
    if missing:
        report.ok = False
        report.missing = missing
        report.note = f"{len(missing)} stub(s) not implemented"
    return report


# --------------------------------------------------------------------------
# Docker: bake the shadowed tree into the unit's image, run its verifier
# --------------------------------------------------------------------------


@dataclass
class ScoreResult:
    reward: int | None          # 1, 0, or None (verifier could not run)
    failing_tests: list[str]
    passing_tests: list[str]
    build_failed: bool
    compile_errors: list[str]
    fatal_lines: list[str]      # first fatal messages, naming the defect
    log_tail: str
    seconds: float
    note: str = ""


def _image_name(unit_dir: Path) -> str:
    base = re.sub(r"[^a-z0-9-]", "", unit_dir.name.lower().replace("_", "-"))
    return f"shadow-{base}"[:60]


def score_tree(unit_dir: Path, tree: Path, keep_image: bool = False) -> ScoreResult:
    """Build ``tree`` into the unit's image and run ``tests/test.sh``.

    The verifier writes /logs/verifier as root inside the container, so the
    mounted log dir ends up root-owned: it is emptied by a helper container
    before the image is dropped and the workdir removed.
    """
    started = time.time()
    dockerfile = unit_dir / "environment" / "Dockerfile"
    tests_dir = unit_dir / "tests"
    img = _image_name(unit_dir)
    w = Path(tempfile.mkdtemp(prefix="shadowgate-"))
    test_log = ""
    try:
        shutil.copytree(tree, w / "src", symlinks=True)
        shutil.copy2(dockerfile, w / "Dockerfile")
        build = subprocess.run(
            ["docker", "build", "-q", "-t", img, str(w)],
            capture_output=True, text=True, timeout=BUILD_TIMEOUT, check=False)
        if build.returncode != 0:
            return ScoreResult(None, [], [], False, [], [],
                               build.stderr[-2000:], time.time() - started,
                               "docker build failed: " + build.stderr[-300:])
        logs = w / "logs"
        logs.mkdir(exist_ok=True)
        try:
            run = subprocess.run(
                ["docker", "run", "--rm", "--network", "none",
                 "-v", f"{tests_dir.resolve()}:/tests:ro",
                 "-v", f"{logs}:/logs",
                 img, "bash", "/tests/test.sh"],
                capture_output=True, text=True, timeout=TEST_TIMEOUT, check=False)
            test_log = run.stdout + "\n" + run.stderr
        except subprocess.TimeoutExpired:
            return ScoreResult(None, [], [], False, [], [],
                               "", time.time() - started, "verifier timeout")
        finally:
            # read the reward before the root-owned files are cleaned out
            reward_file = logs / "verifier" / "reward.txt"
            reward = None
            if reward_file.is_file():
                try:
                    reward = int(reward_file.read_text().strip())
                except ValueError:
                    reward = None
            subprocess.run(
                ["docker", "run", "--rm", "-v", f"{logs}:/logs",
                 img, "sh", "-c", "rm -rf /logs/*"],
                capture_output=True, timeout=120, check=False)
            if not keep_image:
                subprocess.run(["docker", "rmi", "-f", img],
                               capture_output=True, timeout=120, check=False)
        failing = FAIL_TEST_RE.findall(test_log)
        passing = PASS_TEST_RE.findall(test_log)
        build_failed = bool(BUILD_FAIL_RE.search(test_log))
        compile_errors = COMPILE_ERR_RE.findall(test_log)[:20]
        fatals = [ln.strip() for ln in FATAL_MSG_RE.findall(test_log)]
        return ScoreResult(
            reward=reward, failing_tests=failing, passing_tests=passing,
            build_failed=build_failed, compile_errors=compile_errors,
            fatal_lines=fatals[:10], log_tail=test_log[-4000:],
            seconds=time.time() - started)
    finally:
        shutil.rmtree(w, ignore_errors=True)


# --------------------------------------------------------------------------
# The gate
# --------------------------------------------------------------------------


@dataclass
class UnitResult:
    unit: str
    outcome: str          # pass | fail_assert | fail_build | fail_apply | error
    reward: int | None
    model: str
    requests: int
    tokens: int
    seconds: float
    context_hash: str
    manifest: list[dict]
    excised: dict[str, list[tuple[str, str]]]
    failing_tests: list[str]
    fatal_lines: list[str]
    compile_errors: list[str]
    missing_stubs: list[str]
    note: str = ""


def _annotate_compile_errors(tree: Path, errors: list[str]) -> list[str]:
    """Add ``(inside func X)`` to ``file.go:NN:CC: msg`` errors, resolved
    against the spliced tree. The generator only sees its own emitted funcs,
    so a bare line number cannot be mapped back to the offending function —
    without the annotation repair passes guess at the wrong decl."""
    out: list[str] = []
    for ce in errors:
        m = re.match(r"(?:\./)?([\w./-]+\.go):(\d+):\d+:", ce)
        f = tree / m.group(1) if m else None
        if not (m and f and f.is_file()):
            out.append(ce)
            continue
        src = f.read_text(errors="replace")
        off = 0
        for i, ln in enumerate(src.splitlines(keepends=True), 1):
            if i == int(m.group(2)):
                break
            off += len(ln)
        enclosing = next(
            (f"{sp.recv + '.' if sp.recv else ''}{sp.name}"
             for sp in go_func_spans(src) if sp.start <= off < sp.end), "")
        out.append(ce + (f"   (inside func {enclosing})" if enclosing else ""))
    return out


def _repair_prompt(ctx: Context, prev_code: str, problems: list[str]) -> str:
    prob_block = "\n".join(problems)[:MAX_ERROR_CHARS]
    return ctx.prompt + f"""

-----
Your previous implementation had problems:

{prob_block}

Your previous code:
```go
{prev_code[:40_000]}
```

Return the corrected complete implementation (every function in the work
list), same format as before."""


def gate_unit(
    unit_dir: Path,
    models: Iterable[str] = SHADOW_MODELS,
    budget: int = MAX_REQUESTS,
    out_tree_dir: Path | None = None,
) -> UnitResult:
    """Run the gate on one unit dir. Never raises for model/build failures —
    they are data, returned in the result."""
    t0 = time.time()
    name = unit_dir.name
    unit_dir = unit_dir.resolve()
    try:
        ctx = build_context(unit_dir)
    except (OSError, AssertionError) as e:  # packaging defect — no stubs/contract
        return UnitResult(name, "error", None, "", 0, 0, 0, "", [], {},
                          [], [], [], [], f"{type(e).__name__}: {e}")
    CTX_DIR.mkdir(parents=True, exist_ok=True)
    (CTX_DIR / f"{name}.prompt.txt").write_text(ctx.prompt)
    (CTX_DIR / f"{name}.manifest.json").write_text(json.dumps(ctx.manifest, indent=1))

    src_root = unit_dir / "environment" / "src"
    excised_files = find_excised_files(src_root)

    spent_requests = 0
    spent_tokens = 0
    model_used = ""
    shadow = ShadowCode(imports=[], funcs={})
    prev_code = ""
    problems: list[str] = []
    repair = 0
    while True:
        if repair == 0:
            prompt, tag = ctx.prompt, f"{name}#r0"
        else:
            prompt, tag = _repair_prompt(ctx, prev_code, problems), f"{name}#r{repair}"
        gen = generate(prompt, tag, models, max(0, budget - spent_requests))
        spent_requests += gen.requests
        spent_tokens += gen.tokens
        model_used = gen.model
        if gen.text is None:
            return UnitResult(name, "error", None, model_used, spent_requests,
                              spent_tokens, time.time() - t0, ctx.context_hash,
                              ctx.manifest, ctx.excised, [], [], [], [],
                              gen.error or "generation failed")
        new = parse_response(gen.text)
        shadow.funcs.update(new.funcs)
        # imports come from the LATEST pass — a repair returns the complete
        # implementation, and unioning would pin a phantom import from an
        # earlier attempt into every later apply (chartkit/v4 did exactly that)
        shadow.imports = new.imports
        shadow.aliases = new.aliases
        shadow.helpers = new.helpers
        prev_code = "\n\n".join(shadow.funcs.values())
        with tempfile.TemporaryDirectory(prefix="shadowtree-") as td:
            tree = Path(td) / "src"
            shutil.copytree(src_root, tree, symlinks=True)
            rep = apply_shadow(tree, excised_files, shadow)
            score: ScoreResult | None = None
            if rep.ok:
                if out_tree_dir:
                    out_tree_dir.mkdir(parents=True, exist_ok=True)
                    dest = out_tree_dir / name
                    if dest.exists():
                        shutil.rmtree(dest)
                    shutil.copytree(tree, dest, symlinks=True)
                score = score_tree(unit_dir, tree)
                if score.compile_errors:
                    score.compile_errors = _annotate_compile_errors(
                        tree, score.compile_errors)
        if score is not None and score.reward == 1:
            return UnitResult(name, "pass", 1, model_used, spent_requests,
                              spent_tokens, time.time() - t0, ctx.context_hash,
                              ctx.manifest, ctx.excised, score.failing_tests,
                              score.fatal_lines, [], [], score.note)
        problems: list[str] = []
        if not rep.ok:
            problems = [f"unimplemented stubs (still panic): {m}"
                        for m in rep.missing]
        elif score is not None:
            is_compile = score.build_failed or bool(score.compile_errors)
            problems = score.compile_errors[:] if is_compile else []
        if problems and repair < MAX_REPAIRS and spent_requests < budget:
            repair += 1
            log(f"  {name}: {'missing stubs' if not rep.ok else 'compile failed'}, repair pass {repair}")
            continue
        if not rep.ok:
            return UnitResult(name, "fail_apply", None, model_used,
                              spent_requests, spent_tokens, time.time() - t0,
                              ctx.context_hash, ctx.manifest, ctx.excised,
                              [], [], [], rep.missing, rep.note)
        assert score is not None
        is_compile = score.build_failed or bool(score.compile_errors)
        if is_compile:
            outcome = "fail_build"
        elif score.reward == 0:
            outcome = "fail_assert"
        else:
            outcome = "error"
        return UnitResult(name, outcome, score.reward, model_used,
                          spent_requests, spent_tokens, time.time() - t0,
                          ctx.context_hash, ctx.manifest, ctx.excised,
                          score.failing_tests, score.fatal_lines,
                          score.compile_errors, [], score.note)


# --------------------------------------------------------------------------
# Batch driver — resume-safe over results.jsonl
# --------------------------------------------------------------------------


def _result_to_json(r: UnitResult) -> str:
    d = {
        "unit": r.unit, "outcome": r.outcome, "reward": r.reward,
        "model": r.model, "requests": r.requests, "tokens": r.tokens,
        "seconds": round(r.seconds, 1), "context_hash": r.context_hash,
        "manifest": r.manifest,
        "excised": {k: [f"{a}.{b}" if a else b for a, b in v]
                    for k, v in r.excised.items()},
        "failing_tests": r.failing_tests, "fatal_lines": r.fatal_lines,
        "compile_errors": r.compile_errors[:10],
        "missing_stubs": r.missing_stubs, "note": r.note,
    }
    return json.dumps(d)


def done_units(path: Path = RESULTS_PATH) -> set[str]:
    """Units with a real gate verdict. ``error`` outcomes are transient
    (rate limit, docker hiccup, generation failure) and are retried on the
    next run."""
    if not path.is_file():
        return set()
    out = set()
    for line in path.read_text().splitlines():
        try:
            d = json.loads(line)
            if d["outcome"] != "error":
                out.add(d["unit"])
        except (json.JSONDecodeError, KeyError):
            continue
    return out


def gate_batch(
    unit_dirs: list[Path],
    models: Iterable[str] = SHADOW_MODELS,
    max_requests: int = MAX_REQUESTS,
    keep_trees: bool = False,
    workers: int = 1,
) -> list[UnitResult]:
    GATE_DIR.mkdir(parents=True, exist_ok=True)
    done = done_units()
    lock = threading.Lock()
    spent = [0]
    results: list[UnitResult] = []

    def one(u: Path) -> UnitResult | None:
        with lock:
            if u.name in done:
                log(f"{u.name}: already gated, skipping")
                return None
            if spent[0] >= max_requests:
                log(f"request budget {max_requests} exhausted; skipping {u.name}")
                return None
            log(f"gating {u.name} (spent {spent[0]}/{max_requests})")
        r = gate_unit(u, models, max_requests,
                      out_tree_dir=GATE_DIR / "trees" if keep_trees else None)
        with lock:
            spent[0] += r.requests
            results.append(r)
            with RESULTS_PATH.open("a") as f:
                f.write(_result_to_json(r) + "\n")
            done.add(u.name)
        log(f"  {u.name}: {r.outcome} reward={r.reward} model={r.model} "
            f"req={r.requests} tok={r.tokens} {r.seconds:.0f}s "
            f"failing={r.failing_tests[:4]}")
        return r

    if workers <= 1:
        for u in unit_dirs:
            one(u)
    else:
        with ThreadPoolExecutor(max_workers=workers) as ex:
            list(ex.map(one, unit_dirs))
    return results
