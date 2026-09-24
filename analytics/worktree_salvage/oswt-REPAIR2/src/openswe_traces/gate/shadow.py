"""Shadow-implementation gate: can a weak implementer rebuild the excised
behaviour from the contract alone?

For each unit we generate a *shadow implementation*: a second, independent
implementation of the excised functions, written by a weak free-tier model
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
import tempfile
import time
from collections.abc import Iterable
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.gate import composer

# The shadow implementer runs on Composer (cursor-agent). The gate's value is
# that a WEAK implementer suffices to prove a contract complete; composer-2.5
# is stronger than the free-tier models this gate was designed around, which
# biases the gate toward false "pass" — acceptable: the repair loop only ever
# repairs per-assertion unstated commitments, so a strong shadow can only
# under-discover contract gaps, never invent repairs. (OpenRouter free-tier
# is forbidden: its shared rate limits stalled the gate for hours while the
# account spent $0.00.)
SHADOW_MODELS = (
    "composer-2.5",
)

MAX_REQUESTS = 300
MAX_ATTEMPTS = 6            # per generation call, across models
MAX_REPAIRS = 2             # compile-repair passes per unit
MAX_DIGEST_CHARS = 60_000   # cap on the package declaration surface
MAX_CONTRACT_CHARS = 40_000
MAX_TOTAL_CHARS = 140_000
MAX_ERROR_CHARS = 12_000    # compiler output fed back for repair
REQUEST_TIMEOUT = 300
TEST_TIMEOUT = 1500         # seconds for one docker verifier run
BUILD_TIMEOUT = 900

GATE_DIR = ROOT / "outputs" / "shadow_gate"
GEN_DIR = GATE_DIR / "gen"
CTX_DIR = GATE_DIR / "contexts"
RESULTS_PATH = GATE_DIR / "results.jsonl"
LOG_PATH = ROOT / "outputs" / "REPAIR2.log"

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
FAIL_TEST_RE = re.compile(r"^--- FAIL:\s+(\w+)", re.MULTILINE)
FAIL_BLOCK_RE = re.compile(r"^--- FAIL:\s+(\S+?)\s*(?:\([\d.]+s\))?\s*$", re.MULTILINE)
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
- You may add unexported helper functions; they go in the same package.
- Do NOT emit a package clause, test code, or commentary.

Implement every function in the work list."""
    prompt = _truncate(prompt, MAX_TOTAL_CHARS)
    return Context(
        prompt=prompt,
        manifest=manifest,
        excised=excised_map,
        context_hash=hashlib.sha256(prompt.encode()).hexdigest()[:16],
    )


# --------------------------------------------------------------------------
# Composer client — cached, budgeted, logged. ``_post`` keeps the old
# OpenRouter-shaped signature so callers need no changes; ``api_key`` is
# unused (cursor-agent authenticates from the eval_tasks env file).
# --------------------------------------------------------------------------


@dataclass
class GenResult:
    text: str | None
    model: str
    requests: int
    tokens: int          # total tokens across calls (prompt+completion)
    error: str | None = None
    cache_hit: bool = False


def load_api_key(env_path: Path | None = None) -> str:
    """Verify the Composer path is usable; returns an opaque marker."""
    composer.check_ready()
    return "composer"


def _post(api_key: str, model: str, prompt: str) -> tuple[str | None, int, str | None]:
    try:
        text, tokens = composer.ask(prompt, model=model or composer.MODEL)
    except (RuntimeError, subprocess.TimeoutExpired, OSError) as e:
        return None, 0, f"{type(e).__name__}: {e}"
    return text, tokens, None


def generate(
    prompt: str,
    api_key: str,
    tag: str,
    models: Iterable[str] = SHADOW_MODELS,
    budget: int = MAX_REQUESTS,
    gen_dir: Path | None = None,
) -> GenResult:
    """One generation, cached on the prompt hash. ``tag`` scopes the cache
    (initial generation vs. repair passes)."""
    gdir = gen_dir or GEN_DIR
    gdir.mkdir(parents=True, exist_ok=True)
    key = hashlib.sha256((tag + "\n" + prompt).encode()).hexdigest()[:24]
    path = gdir / f"{key}.json"
    if path.is_file():
        try:
            data = json.loads(path.read_text())
            return GenResult(data["text"], data.get("model", "cached"),
                             int(data.get("requests", 0)),
                             int(data.get("tokens", 0)), cache_hit=True)
        except (OSError, json.JSONDecodeError, KeyError):
            pass

    requests, tokens = 0, 0
    last_err = "no models configured"
    models = list(models)
    for attempt in range(MAX_ATTEMPTS):
        model = models[min(attempt, len(models) - 1)]
        if requests >= budget:
            return GenResult(None, model, requests, tokens, "request budget exhausted")
        requests += 1
        text, tk, err = _post(api_key, model, prompt)
        tokens += tk
        if text is not None:
            path.write_text(json.dumps(
                {"model": model, "tokens": tokens, "requests": requests,
                 "tag": tag, "prompt_sha": key, "text": text}, indent=1) + "\n")
            return GenResult(text, model, requests, tokens)
        last_err = err
        if err and ("404" in err or "No endpoints" in err or "400" in err):
            continue
        if err and "429" in err:
            time.sleep(45 * (attempt + 1))
            continue
        time.sleep(5)
    return GenResult(None, models[-1], requests, tokens, last_err)


# --------------------------------------------------------------------------
# Response parsing + splice into the excised tree
# --------------------------------------------------------------------------


@dataclass
class ShadowCode:
    imports: list[str]                     # extra import paths
    funcs: dict[tuple[str, str], str]      # (recv, name) -> full func decl text


def parse_response(text: str) -> ShadowCode:
    m = FENCE_RE.search(text)
    code = m.group(1) if m else text
    imports = [p for im in IMPORT_BLOCK_RE.finditer(code)
               for p in IMPORT_LINE_RE.findall(im.group(1))]
    imports += IMPORT_SINGLE_RE.findall(code)
    funcs: dict[tuple[str, str], str] = {}
    for sp in go_func_spans(code):
        funcs.setdefault(sp.key, code[sp.start:sp.end])
    return ShadowCode(imports=imports, funcs=funcs)


def _pkg_name_for(path: str) -> str:
    base = path.rsplit("/", 1)[-1]
    if base.startswith("v") and base[1:].isdigit() and "/" in path:
        base = path.rsplit("/", 2)[-2]
    return re.sub(r"\W", "", base)


def _unblank_imports(file_text: str, used_code: str) -> str:
    """Turn ``_ "path"`` into ``"path"`` when the shadow code uses the
    package — a blank import cannot be referenced by name."""
    def repl(m: re.Match) -> str:
        indent, path = m.group(1), m.group(2)
        pkg = _pkg_name_for(path)
        if pkg and re.search(rf"\b{re.escape(pkg)}\s*\.", used_code):
            return f'{indent}"{path}"'
        return m.group(0)
    return re.sub(r'(?m)^(\s*)_\s*"([^"]+)"', repl, file_text)


def _merge_imports(file_text: str, extra: list[str]) -> str:
    if not extra:
        return file_text
    have = set(IMPORT_LINE_RE.findall(file_text))
    new: list[str] = []
    for p in extra:
        if p in have:
            continue
        # Only merge an import the spliced code actually references: the
        # model's import block is not proof of use, and an unused import is
        # a compile error (helm-repindex r0 died on exactly this).
        pkg = _pkg_name_for(p)
        if not pkg or not re.search(rf"\b{re.escape(pkg)}\s*\.", file_text):
            continue
        new.append(p)
    if not new:
        return file_text
    lines = "".join(f'\t"{p}"\n' for p in new)
    m = re.search(r"(?m)^import\s*\(", file_text)
    if m:
        close = file_text.index(")", m.end())
        return file_text[:close] + lines + file_text[close:]
    m2 = re.search(r'(?m)^import\s+"[^"]+"\s*$', file_text)
    if m2:
        old = IMPORT_LINE_RE.search(m2.group(0))
        old_path = old.group(1) if old else ""
        block = f'import (\n\t"{old_path}"\n{lines})'
        return file_text[:m2.start()] + block + file_text[m2.end():]
    pm = re.search(r"(?m)^package\s+\w+\s*$", file_text)
    if pm:
        return file_text[:pm.end()] + f"\n\nimport (\n{lines})\n" + file_text[pm.end():]
    return file_text


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
        new_text = _unblank_imports(new_text, "\n".join(shadow.funcs.values()))
        new_text = _merge_imports(new_text, shadow.imports)
        (src_dir / rel_path).write_text(new_text)
    # helper funcs the model emitted that match no stub: append them unless
    # the name collides with an existing decl in the package (a redeclared
    # helper is a compile error, not a semantic result)
    existing = {sp.name for t in excised_files.values()
                for sp in go_func_spans(t)}
    extras = [d for k, d in shadow.funcs.items()
              if k not in used and k[1] not in existing]
    if extras:
        first = src_dir / next(iter(excised_files))
        first.write_text(first.read_text() +
                         "\n// --- shadow helpers ---\n\n" + "\n\n".join(extras))
    if missing:
        report.ok = False
        report.missing = missing
        report.note = f"{len(missing)} stub(s) not implemented"
    return report


# --------------------------------------------------------------------------
# Docker: bake the shadowed tree into the unit's image, run its verifier
# --------------------------------------------------------------------------


def parse_fail_blocks(test_log: str) -> dict[str, list[str]]:
    """Map each failing test to its assertion messages.

    Non-verbose ``go test`` output is ``--- FAIL: TestX (t)`` followed by the
    test's indented log lines (``file.go:NN: message``); a ``t.Fatal*`` aborts,
    so most blocks hold exactly one assertion — the one that fired. Subtests
    appear as ``TestX/sub`` and are kept under their own key.
    """
    blocks: dict[str, list[str]] = {}
    cur: str | None = None
    for line in test_log.splitlines():
        m = FAIL_BLOCK_RE.match(line)
        if m:
            cur = m.group(1)
            blocks.setdefault(cur, [])
            continue
        if cur is None:
            continue
        if re.match(r"^(--- |ok \t|FAIL\t|FAIL$|PASS$|\? )", line):
            cur = None
            continue
        s = line.strip()
        if s and (line.startswith((" ", "\t")) or s.startswith(("panic", "goroutine"))):
            blocks[cur].append(s)
    return blocks


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
    fail_blocks: dict[str, list[str]] = field(default_factory=dict)
    test_log: str = ""


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
            seconds=time.time() - started,
            fail_blocks=parse_fail_blocks(test_log), test_log=test_log)
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
    api_key: str,
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
        gen = generate(prompt, api_key, tag, models,
                       max(0, budget - spent_requests))
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
        shadow.imports = sorted(set(shadow.imports) | set(new.imports))
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
) -> list[UnitResult]:
    api_key = load_api_key()
    GATE_DIR.mkdir(parents=True, exist_ok=True)
    done = done_units()
    spent = 0
    results: list[UnitResult] = []
    for u in unit_dirs:
        if u.name in done:
            log(f"{u.name}: already gated, skipping")
            continue
        if spent >= max_requests:
            log(f"request budget {max_requests} exhausted")
            break
        log(f"gating {u.name} (spent {spent}/{max_requests})")
        r = gate_unit(u, api_key, models, max_requests - spent,
                      out_tree_dir=GATE_DIR / "trees" if keep_trees else None)
        spent += r.requests
        results.append(r)
        with RESULTS_PATH.open("a") as f:
            f.write(_result_to_json(r) + "\n")
        log(f"  {u.name}: {r.outcome} reward={r.reward} model={r.model} "
            f"req={r.requests} tok={r.tokens} {r.seconds:.0f}s "
            f"failing={r.failing_tests[:4]}")
    return results
