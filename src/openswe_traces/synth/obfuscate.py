"""Identity-obfuscate a Harbor Go task so upstream lookup cannot solve it."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import tempfile
from collections.abc import Iterable, Sequence
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT

OLD_MODULE = "github.com/tikv/client-go/v2"
NEW_MODULE = "example.internal/kvstore/v2"

DIR_RENAMES: tuple[tuple[str, str], ...] = (
    ("tikvrpc", "wirerpc"),
    ("mocktikv", "mockkv"),
    ("tikv", "kvclient"),
)

# Comments / docs / unprotected strings only. Never applied to import paths.
_BRAND_PAIRS: tuple[tuple[str, str], ...] = (
    ("PingCAP", "Acme"),
    ("pingcap", "acme"),
    ("TiKV", "KVStore"),
    ("tikv", "kvstore"),
    ("TiDB", "SQLEngine"),
    ("tidb", "sqlengine"),
)

NO_WEB_CLAUSE = (
    "IMPORTANT: This repository is fully self-contained. Do NOT use web search, "
    "web fetch, or any tool that accesses the internet, and do not attempt to "
    "download or consult upstream sources; any such use disqualifies the attempt. "
    "Work only from the files in the repository and the test output."
)

_SKIP_RENAME = {
    "Error",
    "String",
    "Close",
    "Unwrap",
    "Is",
    "As",
    "GoString",
    "Format",
}

_TEXT_SUFFIXES = {
    ".go",
    ".md",
    ".txt",
    ".mod",
    ".sum",
    ".toml",
    ".yml",
    ".yaml",
    ".json",
    ".sh",
    ".proto",
    ".cfg",
    ".ini",
    ".xml",
    ".html",
    ".rst",
    ".adoc",
}

_DOC_NAMES = {
    "readme",
    "changelog",
    "contributing",
    "authors",
    "notice",
    "code_of_conduct",
    "security",
    "maintainers",
}

_ADJECTIVES = (
    "Amber",
    "Brine",
    "Cedar",
    "Dusk",
    "Ember",
    "Flint",
    "Grove",
    "Haze",
    "Ivory",
    "Jade",
    "Kelp",
    "Lumen",
    "Mist",
    "Nimbus",
    "Ochre",
    "Pebble",
    "Quartz",
    "Ridge",
    "Sable",
    "Thorn",
    "Umber",
    "Vellum",
    "Willow",
    "Yarrow",
    "Zest",
)
_NOUNS = (
    "Ref",
    "Slot",
    "Node",
    "Pipe",
    "Unit",
    "Span",
    "Gate",
    "Wire",
    "Pack",
    "Fold",
    "Clip",
    "Join",
    "Seal",
    "Heap",
    "Ring",
    "Path",
    "Bolt",
    "Core",
    "Link",
    "Port",
)

_HUNK_RE = re.compile(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@")
_DIFF_FILE_RE = re.compile(r"^diff --git a/(.+) b/(.+)$")
_CHECKSUM_LINE_RE = re.compile(
    r'^echo "([0-9a-f]{64})  /app/([^"]+)" \| sha256sum -c',
    re.MULTILINE,
)
_AGG_HASH_RE = re.compile(r"^expected='([0-9a-f]{64})'", re.MULTILINE)
_STRING_LIT_RE = re.compile(r'"(?:\\.|[^"\\])*"|`[^`]*`')
_IDENT_RE = re.compile(r"\b([A-Z][A-Za-z0-9]{3,})\b")

IDENT_HELPER = Path(__file__).with_name("obfuscate_idents.go")


@dataclass
class Ident:
    name: str
    recv: str
    file: str
    line: int
    col: int
    exported: bool
    kind: str
    hunk: bool
    role: str


@dataclass
class TaskSpec:
    name: str
    src_rel: str
    bug_patch: Path
    alt_patch: Path
    cheat_patch: Path
    alt_expected_pass: bool = True
    notes: str = ""


@dataclass
class ObfuscateResult:
    dest: Path
    mapping: dict[str, str]
    mapping_size: int
    validation: dict[str, object]
    searchability: list[dict[str, object]]
    tests_adjusted: list[str]
    skipped_renames: list[str] = field(default_factory=list)


def default_task_specs() -> list[TaskSpec]:
    bugs = ROOT / "experiments/codegraph_bugs/bugs/client-go"
    two = ROOT / "experiments/codegraph_bugs/bugs/client-go-two-repo"
    nex = ROOT / "experiments/harbor_nex"
    return [
        TaskSpec(
            "client-go-keyspacecodec",
            str(nex / "tasks_iter9/client-go-keyspacecodec"),
            bugs / "KeyspaceCodec.patch",
            bugs / "KeyspaceCodec.alt.patch",
            bugs / "KeyspaceCodec.cheat.patch",
            alt_expected_pass=False,
            notes="one-function alt cannot pass f2p at this size",
        ),
        TaskSpec(
            "client-go-memget",
            str(nex / "tasks_iter9/client-go-memget"),
            bugs / "MemGet.patch",
            bugs / "MemGet.alt.patch",
            bugs / "MemGet.cheat.patch",
            alt_expected_pass=False,
            notes="naive linear alt fails the perf gate",
        ),
        TaskSpec(
            "client-go-batchcmds",
            str(nex / "tasks_iter8/client-go-batchcmds"),
            bugs / "BatchCmds.patch",
            bugs / "BatchCmds.alt.patch",
            bugs / "BatchCmds.cheat.patch",
        ),
        TaskSpec(
            "client-go-interceptor",
            str(nex / "tasks_iter8/client-go-interceptor"),
            bugs / "Interceptor.patch",
            bugs / "Interceptor.alt.patch",
            bugs / "Interceptor.cheat.patch",
        ),
        TaskSpec(
            "client-go-onepc-scope",
            str(nex / "tasks_two_repo/client-go-onepc-scope"),
            two / "checkOnePC.patch",
            two / "checkOnePC.alt.patch",
            two / "checkOnePC.cheat.patch",
        ),
        TaskSpec(
            "client-go-memsetvalue",
            str(nex / "tasks_iter7/client-go-memsetvalue"),
            bugs / "MemSetValue.patch",
            bugs / "MemSetValue.alt.patch",
            bugs / "MemSetValue.cheat.patch",
        ),
    ]


def _env() -> dict[str, str]:
    env = os.environ.copy()
    gopath = env.get("GOPATH") or str(Path.home() / "go")
    extras = [
        "/usr/local/go/bin",
        str(Path(gopath) / "bin"),
        str(Path.home() / "go" / "bin"),
    ]
    env["PATH"] = ":".join([*extras, env.get("PATH", "")])
    env.setdefault("GOTOOLCHAIN", "local")
    return env


def _run(
    args: Sequence[str],
    *,
    cwd: Path | None = None,
    timeout: int = 120,
    input_text: str | None = None,
    extra_env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    env = _env()
    if extra_env:
        env.update(extra_env)
    return subprocess.run(
        list(args),
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
        input=input_text,
    )


def ensure_gopls() -> Path:
    proc = _run(["gopls", "version"])
    if proc.returncode == 0:
        path = shutil.which("gopls", path=_env()["PATH"])
        if path:
            return Path(path)
    install = _run(["go", "install", "golang.org/x/tools/gopls@v0.16.2"], timeout=180)
    if install.returncode != 0:
        raise RuntimeError(f"go install gopls failed: {install.stderr}")
    path = shutil.which("gopls", path=_env()["PATH"])
    if not path:
        raise RuntimeError("gopls installed but not on PATH")
    return Path(path)


def parse_patch_files(patch_text: str) -> list[str]:
    files: list[str] = []
    for ln in patch_text.splitlines():
        m = _DIFF_FILE_RE.match(ln)
        if m:
            files.append(m.group(1))
    return files


def parse_patch_hunk_lines(patch_text: str) -> dict[str, list[int]]:
    """Gold-file line numbers touched by a bug patch (minus side)."""
    current = ""
    out: dict[str, list[int]] = {}
    for ln in patch_text.splitlines():
        fm = _DIFF_FILE_RE.match(ln)
        if fm:
            current = fm.group(1)
            out.setdefault(current, [])
            continue
        hm = _HUNK_RE.match(ln)
        if hm and current:
            start = int(hm.group(1))
            count = int(hm.group(2) or "1")
            out.setdefault(current, []).extend(range(start, start + max(count, 1)))
    return out


def remap_relpath(rel: str) -> str:
    parts = Path(rel).parts
    mapped = [dict(DIR_RENAMES).get(p, p) for p in parts]
    return "/".join(mapped)


def rewrite_module_path(tree: Path, old: str = OLD_MODULE, new: str = NEW_MODULE) -> int:
    n = 0
    for path in _iter_text_files(tree):
        text = path.read_text(encoding="utf-8", errors="replace")
        if old not in text:
            continue
        path.write_text(text.replace(old, new), encoding="utf-8")
        n += 1
    return n


def rename_identity_dirs(tree: Path) -> dict[str, str]:
    """Rename brand directory names; rewrite local import path segments."""
    moved: dict[str, str] = {}
    for old, new in sorted(DIR_RENAMES, key=lambda p: len(p[0]), reverse=True):
        for dirpath in sorted(tree.rglob(old), key=lambda p: len(p.parts), reverse=True):
            if not dirpath.is_dir() or dirpath.name != old:
                continue
            dest = dirpath.with_name(new)
            if dest.exists():
                continue
            dirpath.rename(dest)
            rel_old = dirpath.relative_to(tree).as_posix()
            rel_new = dest.relative_to(tree).as_posix()
            moved[rel_old] = rel_new
    for path in _iter_text_files(tree):
        text = path.read_text(encoding="utf-8", errors="replace")
        new = _remap_local_path_segments(text)
        if new != text:
            path.write_text(new, encoding="utf-8")
    return moved


def _remap_local_path_segments(text: str) -> str:
    """Rewrite DIR_RENAMES segments only inside NEW_MODULE paths, not github.com/tikv/*."""
    mapping = dict(DIR_RENAMES)
    chunks = text.split(NEW_MODULE)
    out = [chunks[0]]
    for chunk in chunks[1:]:
        i = 0
        while i < len(chunk) and (chunk[i].isalnum() or chunk[i] in "/._-"):
            i += 1
        path, rest = chunk[:i], chunk[i:]
        parts = [_dir_rename_seg(p, mapping) for p in path.split("/")]
        out.append(NEW_MODULE + "/".join(parts) + rest)
    return "".join(out)


def _dir_rename_seg(seg: str, mapping: dict[str, str]) -> str:
    return mapping.get(seg, seg)


def collect_protected_literals(tree: Path) -> set[str]:
    """String literals that tests compare against — never rewrite these."""
    protected: set[str] = set()
    for path in tree.rglob("*_test.go"):
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        for m in _STRING_LIT_RE.finditer(text):
            raw = m.group(0)
            protected.add(_unquote(raw))
            protected.add(raw)
    return protected


def _unquote(lit: str) -> str:
    if len(lit) >= 2 and lit[0] == "`" and lit[-1] == "`":
        return lit[1:-1]
    if len(lit) >= 2 and lit[0] == '"' and lit[-1] == '"':
        try:
            return bytes(lit[1:-1], "utf-8").decode("unicode_escape")
        except (UnicodeDecodeError, ValueError):
            return lit[1:-1]
    return lit


def _brand_replace(text: str) -> str:
    out = text
    for old, new in _BRAND_PAIRS:
        out = re.sub(r"\b" + re.escape(old) + r"\b", new, out)
    out = re.sub(r"\bPD\b", "Meta", out)
    out = re.sub(r"\bpd\b", "meta", out)
    return out


def strip_identity_text(tree: Path, protected: set[str]) -> dict[str, int]:
    docs = 0
    comments = 0
    for path in _iter_text_files(tree):
        rel = path.relative_to(tree).as_posix()
        if path.suffix == ".go":
            new, n = _strip_go_file(path.read_text(encoding="utf-8", errors="replace"), protected)
            if n:
                path.write_text(new, encoding="utf-8")
                comments += n
            continue
        if path.name in {"go.mod", "go.sum"}:
            continue
        stem = path.stem.lower()
        if path.suffix.lower() in {".md", ".txt"} or stem in _DOC_NAMES or "license" in stem:
            orig = path.read_text(encoding="utf-8", errors="replace")
            if stem.startswith(("readme", "changelog", "contributing")):
                path.write_text(_neutral_doc(path.name, orig), encoding="utf-8")
                docs += 1
                continue
            if "license" in stem:
                path.write_text(_brand_replace(orig), encoding="utf-8")
                docs += 1
                continue
            if rel.startswith("docs/") or "/docs/" in f"/{rel}":
                path.write_text(_brand_replace(orig), encoding="utf-8")
                docs += 1
    return {"docs": docs, "go_comment_hits": comments}


def _neutral_doc(name: str, orig: str) -> str:
    return (
        f"# KV client library\n\n"
        f"Internal storage client. Original {name} identity text removed.\n"
    )


def _strip_go_file(text: str, protected: set[str]) -> tuple[str, int]:
    """Rewrite comments and unprotected string literals; leave code/imports intact."""
    out: list[str] = []
    i = 0
    n = len(text)
    hits = 0

    def take_until(end: str, kind: str, start: int) -> None:
        nonlocal i, hits
        j = text.find(end, start)
        if j < 0:
            chunk = text[i:]
            i = n
        else:
            chunk = text[i : j + len(end)]
            i = j + len(end)
        body = chunk
        if kind == "comment":
            new = _brand_replace(body)
            if new != body:
                hits += 1
            out.append(new)
            return
        inner = _unquote(body) if body[:1] in "\"`" else body
        if inner in protected or body in protected:
            out.append(body)
            return
        if kind == "tag":
            out.append(body)
            return
        new = _brand_replace(body)
        if new != body:
            hits += 1
        out.append(new)

    in_import_paren = False
    while i < n:
        if text.startswith("import (", i):
            in_import_paren = True
            out.append("import (")
            i += len("import (")
            continue
        if in_import_paren:
            if text[i] == ")":
                in_import_paren = False
            out.append(text[i])
            i += 1
            continue
        if text.startswith('import "', i) or text.startswith("import `", i):
            nl = text.find("\n", i)
            if nl < 0:
                out.append(text[i:])
                break
            out.append(text[i : nl + 1])
            i = nl + 1
            continue
        if text.startswith("//", i):
            take_until("\n", "comment", i + 2)
            continue
        if text.startswith("/*", i):
            take_until("*/", "comment", i + 2)
            continue
        ch = text[i]
        if ch == '"':
            take_until_escaped = i + 1
            while take_until_escaped < n:
                if text[take_until_escaped] == "\\":
                    take_until_escaped += 2
                    continue
                if text[take_until_escaped] == '"':
                    take_until_escaped += 1
                    break
                take_until_escaped += 1
            chunk = text[i:take_until_escaped]
            inner = _unquote(chunk) if len(chunk) >= 2 else chunk
            if inner in protected or chunk in protected:
                out.append(chunk)
            else:
                new = _brand_replace(chunk)
                if new != chunk:
                    hits += 1
                out.append(new)
            i = take_until_escaped
            continue
        if ch == "`":
            j = text.find("`", i + 1)
            if j < 0:
                out.append(text[i:])
                break
            chunk = text[i : j + 1]
            inner = chunk[1:-1]
            # struct tags stay; protected literals stay
            if inner in protected or chunk in protected or _looks_like_struct_tag(inner):
                out.append(chunk)
            else:
                new = "`" + _brand_replace(inner) + "`"
                if new != chunk:
                    hits += 1
                out.append(new)
            i = j + 1
            continue
        if ch == "'":
            j = i + 1
            if j < n and text[j] == "\\":
                j += 2
            elif j < n:
                j += 1
            if j < n and text[j] == "'":
                j += 1
            out.append(text[i:j])
            i = j
            continue
        out.append(ch)
        i += 1
    return "".join(out), hits


def _looks_like_struct_tag(inner: str) -> bool:
    return bool(re.match(r'^\s*\w+:"', inner)) or bool(re.search(r'\b(json|yaml|xml|toml|db|bson):"', inner))


def _iter_text_files(tree: Path) -> Iterable[Path]:
    for path in tree.rglob("*"):
        if not path.is_file():
            continue
        if (
            path.suffix.lower() in _TEXT_SUFFIXES
            or path.name in {"go.mod", "go.sum", "LICENSE"}
            or path.stem.lower() in _DOC_NAMES
        ):
            yield path


def go_mod_tidy_offline(tree: Path) -> None:
    extra = {
        "GOPROXY": "off",
        "GOSUMDB": "off",
        "GOFLAGS": "-mod=mod",
        "GOTOOLCHAIN": "local",
    }
    mods = [p.parent for p in tree.rglob("go.mod")]
    mods.sort(key=lambda p: len(p.parts))
    for mod in mods:
        tidy = _run(["go", "mod", "tidy"], cwd=mod, timeout=180, extra_env=extra)
        listed = _run(["go", "list", "./..."], cwd=mod, timeout=180, extra_env=extra)
        if listed.returncode != 0:
            err = (tidy.stderr or "") + (listed.stderr or "")
            raise RuntimeError(f"offline go list failed in {mod}: {err[-2500:]}")
        # tidy may try to resolve *dependency* test imports (ginkgo, helloworld)
        # that were never in go.sum. The module itself must still list offline.


def list_idents(tree: Path, files: Sequence[tuple[str, list[int]]]) -> list[Ident]:
    payload = {
        "root": str(tree.resolve()),
        "files": [{"path": p, "hunk_lines": lines} for p, lines in files],
    }
    helper_src = IDENT_HELPER.read_text(encoding="utf-8")
    with tempfile.TemporaryDirectory(prefix="obf_idents_") as td:
        tdir = Path(td)
        (tdir / "go.mod").write_text("module obfidents\n\ngo 1.21\n")
        (tdir / "main.go").write_text(helper_src)
        proc = _run(
            ["go", "run", "."],
            cwd=tdir,
            timeout=120,
            input_text=json.dumps(payload),
        )
    if proc.returncode != 0:
        raise RuntimeError(f"ident helper failed: {proc.stderr[-2000:]}\n{proc.stdout[-500:]}")
    raw = json.loads(proc.stdout or "[]")
    return [Ident(**row) for row in raw]


def _neutral_name(recv: str, name: str, taken: set[str]) -> str:
    seed = f"{recv}.{name}"
    h = int(hashlib.sha256(seed.encode()).hexdigest(), 16)
    for k in range(len(_ADJECTIVES) * len(_NOUNS)):
        adj = _ADJECTIVES[(h + k) % len(_ADJECTIVES)]
        noun = _NOUNS[(h // len(_ADJECTIVES) + k) % len(_NOUNS)]
        cand = adj + noun
        if cand not in taken and cand != name:
            taken.add(cand)
            return cand
    raise RuntimeError(f"exhausted name pool for {seed}")


def assign_mapping(idents: Sequence[Ident], instruction: str) -> dict[str, str]:
    """Map recv.name -> new name. Skip instruction tokens unless we will rewrite it."""
    taken = {i.name for i in idents}
    instr_tokens = set(_IDENT_RE.findall(instruction))
    mapping: dict[str, str] = {}
    # Prefer hunk defs, then exported defs, then callees.
    ordered = sorted(
        idents,
        key=lambda i: (0 if i.hunk else 1 if i.role == "def" else 2, i.kind != "func", i.name, i.recv),
    )
    seen_keys: set[str] = set()
    for ident in ordered:
        if ident.name in _SKIP_RENAME and not ident.hunk:
            continue
        key = _ident_key(ident)
        if key in seen_keys:
            continue
        seen_keys.add(key)
        if ident.name in instr_tokens and not ident.hunk:
            # Will rewrite instruction to the new name instead of skipping.
            pass
        mapping[key] = _neutral_name(ident.recv, ident.name, taken)
    return mapping


def _ident_key(ident: Ident) -> str:
    return f"{ident.recv}.{ident.name}" if ident.recv else ident.name


_FUNC_DECL_RE = re.compile(
    r"^func\s+(?:\((?P<recv>[^)]+)\)\s+)?(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*\("
)
_TYPE_DECL_RE = re.compile(r"^type\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s+")


def _recv_type(recv_field: str) -> str:
    return recv_field.replace("*", " ").split()[-1]


def find_decl_pos(tree: Path, recv: str, name: str, files: Sequence[str]) -> tuple[Path, int, int] | None:
    paths: list[Path] = []
    for rel in files:
        p = tree / rel
        if p.is_file():
            paths.append(p)
    if not paths:
        paths = [p for p in tree.rglob("*.go") if "/vendor/" not in p.as_posix()]
    for path in paths:
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        for i, line in enumerate(text.splitlines(), 1):
            stripped = line.lstrip()
            fm = _FUNC_DECL_RE.match(stripped)
            if fm and fm.group("name") == name:
                got_recv = _recv_type(fm.group("recv")) if fm.group("recv") else ""
                if recv and got_recv != recv:
                    continue
                col = line.find(name) + 1
                return path, i, col
            # interface method: Name(args) inside a type block
            if recv and re.match(r"^\s+" + re.escape(name) + r"\s*\(", line):
                col = line.find(name) + 1
                return path, i, col
            tm = _TYPE_DECL_RE.match(stripped)
            if tm and tm.group("name") == name and not recv:
                col = line.find(name) + 1
                return path, i, col
    return None


def apply_gopls_renames(
    tree: Path,
    idents: Sequence[Ident],
    mapping: dict[str, str],
) -> tuple[dict[str, str], list[str]]:
    """Rename by (recv, old_name). Re-find positions after each rename."""
    ensure_gopls()
    applied: dict[str, str] = {}
    skipped: list[str] = []
    pending_hunk: list[tuple[str, str, str]] = []
    pending_iface: list[tuple[str, str, str]] = []
    pending_methods: list[tuple[str, str, str]] = []
    pending_funcs: list[tuple[str, str, str]] = []
    pending_types: list[tuple[str, str, str]] = []
    seen: set[str] = set()
    gold_files = sorted({i.file for i in idents if i.file})
    hunk_keys = {_ident_key(i) for i in idents if i.hunk}
    hunk_names = {i.name for i in idents if i.hunk}
    for ident in idents:
        key = _ident_key(ident)
        new = mapping.get(key)
        if not new or new == ident.name:
            continue
        if key in seen:
            continue
        seen.add(key)
        item = (ident.recv, ident.name, new)
        if key in hunk_keys:
            pending_hunk.append(item)
        elif ident.kind == "type":
            pending_types.append(item)
        elif ident.kind == "iface":
            pending_iface.append(item)
        elif ident.recv:
            pending_methods.append(item)
        else:
            pending_funcs.append(item)
    pending = pending_hunk + pending_iface + pending_methods + pending_funcs + pending_types
    extra = {
        "GOPROXY": "off",
        "GOSUMDB": "off",
        "GOFLAGS": "-mod=mod",
        "GOTOOLCHAIN": "local",
    }
    recv_alias: dict[str, str] = {}
    for recv, current, new in pending:
        lookup_recv = recv_alias.get(recv, recv)
        pos = find_decl_pos(tree, lookup_recv, current, gold_files)
        if pos is None and lookup_recv != recv:
            pos = find_decl_pos(tree, recv, current, gold_files)
        if pos is None:
            skipped.append(f"{recv}.{current}" if recv else current)
            continue
        path, line, col = pos
        target = f"{path.resolve()}:{line}:{col}"
        proc = _run(
            ["gopls", "rename", "-w", target, new],
            cwd=tree,
            timeout=180,
            extra_env=extra,
        )
        if proc.returncode != 0:
            if current in hunk_names and len(current) >= 8:
                n = _textual_ident_rename(tree, current, new)
                if n:
                    applied[key] = new
                    continue
            skipped.append(f"{recv}.{current}->{new}: {(proc.stderr or proc.stdout)[-400:]}")
            continue
        key = f"{recv}.{current}" if recv else current
        applied[key] = new
        if not recv:
            recv_alias[current] = new
    return applied, skipped


def _textual_ident_rename(tree: Path, old: str, new: str) -> int:
    """Last-resort whole-identifier rewrite when gopls cannot load a broken package."""
    n = 0
    pat = re.compile(r"\b" + re.escape(old) + r"\b")
    for path in tree.rglob("*.go"):
        if "/vendor/" in path.as_posix():
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        updated, count = pat.subn(new, text)
        if count:
            path.write_text(updated, encoding="utf-8")
            n += count
    return n


def prepare_tree(tree: Path) -> set[str]:
    protected = collect_protected_literals(tree)
    rewrite_module_path(tree)
    rename_identity_dirs(tree)
    strip_identity_text(tree, protected)
    go_mod_tidy_offline(tree)
    return protected


def restore_gold_files(buggy: Path, upstream: Path, rels: Sequence[str]) -> None:
    for rel in rels:
        src = upstream / rel
        dest = buggy / rel
        if not src.is_file():
            continue
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)


def apply_patch(tree: Path, patch: Path) -> None:
    proc = _run(
        [
            "patch",
            "-p1",
            "--forward",
            "--batch",
            "-i",
            str(patch.resolve()),
            "-d",
            str(tree),
        ],
        timeout=60,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"apply {patch} failed: {proc.stdout}\n{proc.stderr}")


def diff_trees(old: Path, new: Path) -> str:
    proc = _run(["diff", "-ruN", str(old), str(new)], timeout=120)
    # diff returns 1 when files differ
    text = proc.stdout or ""
    old_s = str(old)
    new_s = str(new)
    lines: list[str] = []
    for ln in text.splitlines():
        if ln.startswith("diff -ruN "):
            continue
        if ln.startswith("--- "):
            rest = ln[4:]
            rel = rest.split("\t")[0].replace(old_s + "/", "").replace(old_s, "")
            rel = rel.lstrip("/")
            lines.append(f"--- a/{rel}")
            continue
        if ln.startswith("+++ "):
            rest = ln[4:]
            rel = rest.split("\t")[0].replace(new_s + "/", "").replace(new_s, "")
            rel = rel.lstrip("/")
            lines.append(f"+++ b/{rel}")
            continue
        lines.append(ln)
    body = "\n".join(lines)
    return (body + "\n") if body else ""


def rewrite_instruction(text: str, mapping: dict[str, str]) -> str:
    out = text.replace(OLD_MODULE, NEW_MODULE)
    for old, new in sorted(DIR_RENAMES, key=lambda p: len(p[0]), reverse=True):
        out = out.replace(f"/{old}/", f"/{new}/")
        out = out.replace(f"/{old}", f"/{new}")
        out = out.replace(f"./{old}/", f"./{new}/")
        out = out.replace(f"./{old}", f"./{new}")
    # Apply symbol mapping longest-first (values may also need original names).
    items = sorted(mapping.items(), key=lambda kv: len(kv[0].split(".")[-1]), reverse=True)
    for key, new in items:
        old = key.split(".")[-1]
        if old == new or len(old) < 3:
            continue
        out = re.sub(r"\b" + re.escape(old) + r"\b", new, out)
    out = _brand_replace(out)
    if NO_WEB_CLAUSE not in out:
        out = out.rstrip() + "\n\n" + NO_WEB_CLAUSE + "\n"
    return out


def refresh_checksums(test_sh: Path, src: Path) -> None:
    text = test_sh.read_text(encoding="utf-8")

    def repl_line(m: re.Match[str]) -> str:
        rel = remap_relpath(m.group(2))
        path = src / rel
        if not path.is_file():
            # original relative path after dir rename
            path = src / m.group(2)
            rel = m.group(2)
            if not path.is_file():
                return m.group(0)
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        return f'echo "{digest}  /app/{rel}" | sha256sum -c'

    text = _CHECKSUM_LINE_RE.sub(repl_line, text)
    for old, new in sorted(DIR_RENAMES, key=lambda p: len(p[0]), reverse=True):
        text = text.replace(f"/{old}/", f"/{new}/")
        text = text.replace(f"./{old}/", f"./{new}/")
        text = text.replace(f"./{old}", f"./{new}")
    text = text.replace(OLD_MODULE, NEW_MODULE)

    def repl_agg(m: re.Match[str]) -> str:
        # Recalculate from integration_tests *_test.go if that dir exists.
        tests_root = src / "integration_tests"
        if not tests_root.is_dir():
            return m.group(0)
        proc = _run(
            [
                "bash",
                "-lc",
                "find . -name '*_test.go' | LC_ALL=C sort | xargs -r sha256sum | sha256sum | awk '{print $1}'",
            ],
            cwd=tests_root,
            timeout=60,
        )
        digest = (proc.stdout or "").strip()
        if re.fullmatch(r"[0-9a-f]{64}", digest):
            return f"expected='{digest}'"
        return m.group(0)

    text = _AGG_HASH_RE.sub(repl_agg, text)
    test_sh.write_text(text, encoding="utf-8")


def distinctive_identifiers(tree: Path, mapping: dict[str, str]) -> list[str]:
    names = [v for v in mapping.values() if len(v) >= 6]
    names = sorted(set(names), key=lambda s: (-len(s), s))
    if len(names) >= 5:
        return names[:5]
    extra: list[str] = []
    blob = []
    for p in list(tree.rglob("*.go"))[:80]:
        try:
            blob.append(p.read_text(encoding="utf-8", errors="replace"))
        except OSError:
            continue
    text = "\n".join(blob)
    for tok in _IDENT_RE.findall(text):
        if tok in names or tok in extra:
            continue
        if tok in _SKIP_RENAME:
            continue
        extra.append(tok)
        if len(names) + len(extra) >= 5:
            break
    return (names + extra)[:5]


def searchability_check(idents: Sequence[str], upstream: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for ident in idents:
        proc = _run(
            ["rg", "-F", "-w", "--glob", "!**/.git/**", ident, str(upstream)],
            timeout=60,
        )
        matches = [ln for ln in (proc.stdout or "").splitlines() if ln.strip()]
        rows.append({"ident": ident, "matches": len(matches), "ok": len(matches) == 0})
    return rows


def file_sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _copytree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest, symlinks=True, ignore=shutil.ignore_patterns(".git", "__pycache__"))


def obfuscate_task(
    task_dir: Path | str,
    dest_dir: Path | str,
    *,
    bug_patch: Path | str,
    alt_patch: Path | str,
    cheat_patch: Path | str,
    upstream: Path | str,
    skip_docker: bool = False,
    image_tag: str = "",
    alt_expected_pass: bool = True,
) -> ObfuscateResult:
    """Copy a Harbor task and obfuscate identity. Gold/alt/cheat patches are regenerated."""
    task_dir = Path(task_dir)
    dest_dir = Path(dest_dir)
    bug_patch = Path(bug_patch)
    alt_patch = Path(alt_patch)
    cheat_patch = Path(cheat_patch)
    upstream = Path(upstream)
    src_in = task_dir / "environment" / "src"
    if not src_in.is_dir():
        raise FileNotFoundError(f"missing {src_in}")
    if not (task_dir / "instruction.md").is_file():
        raise FileNotFoundError("instruction.md")

    _copytree(task_dir, dest_dir)
    dest_src = dest_dir / "environment" / "src"
    instruction = (dest_dir / "instruction.md").read_text(encoding="utf-8")
    bug_text = bug_patch.read_text(encoding="utf-8", errors="replace")
    gold_rels = parse_patch_files(bug_text)
    hunks = parse_patch_hunk_lines(bug_text)

    with tempfile.TemporaryDirectory(prefix="obf_trees_") as tmp:
        tmp_path = Path(tmp)
        gold = tmp_path / "gold"
        buggy = tmp_path / "buggy"
        alt = tmp_path / "alt"
        cheat = tmp_path / "cheat"
        _copytree(src_in, gold)
        restore_gold_files(gold, upstream, gold_rels)
        _copytree(src_in, buggy)
        _copytree(src_in, alt)
        apply_patch(alt, alt_patch)
        _copytree(src_in, cheat)
        apply_patch(cheat, cheat_patch)

        prepare_tree(gold)
        prepare_tree(buggy)
        prepare_tree(alt)
        prepare_tree(cheat)

        gold_files = [
            (remap_relpath(rel), hunks.get(rel, []))
            for rel in gold_rels
            if (gold / remap_relpath(rel)).is_file()
        ]
        idents = list_idents(gold, gold_files)
        mapping = assign_mapping(idents, instruction)
        print(f"obfuscate: idents={len(idents)} mapping={len(mapping)}", flush=True)

        def _rename_one(label_tree: tuple[str, Path]) -> tuple[str, dict[str, str], list[str]]:
            label, tree = label_tree
            print(f"obfuscate: gopls rename on {label}...", flush=True)
            app, skip = apply_gopls_renames(tree, idents, mapping)
            print(f"obfuscate: {label} applied={len(app)} skipped={len(skip)}", flush=True)
            return label, app, skip

        applied: dict[str, str] = {}
        skipped: list[str] = []
        with ThreadPoolExecutor(max_workers=2) as pool:
            for _label, app, skip in pool.map(
                _rename_one, (("gold", gold), ("buggy", buggy), ("alt", alt), ("cheat", cheat))
            ):
                applied.update(app)
                skipped.extend(skip)

        gold_patch_text = diff_trees(buggy, gold)
        alt_patch_text = diff_trees(buggy, alt)
        cheat_patch_text = diff_trees(buggy, cheat)

        if dest_src.exists():
            shutil.rmtree(dest_src)
        _copytree(buggy, dest_src)

    patches_dir = dest_dir / "patches"
    patches_dir.mkdir(parents=True, exist_ok=True)
    (patches_dir / "gold.patch").write_text(gold_patch_text)
    (patches_dir / "alt.patch").write_text(alt_patch_text)
    (patches_dir / "cheat.patch").write_text(cheat_patch_text)
    (dest_dir / "mapping.json").write_text(json.dumps(mapping, indent=2, sort_keys=True) + "\n")

    new_instruction = rewrite_instruction(instruction, mapping)
    (dest_dir / "instruction.md").write_text(new_instruction)
    refresh_checksums(dest_dir / "tests" / "test.sh", dest_src)

    toml = dest_dir / "task.toml"
    # Keep existing network settings; do not clobber.
    if not toml.is_file():
        raise FileNotFoundError(toml)

    tests_adjusted: list[str] = []
    tag = image_tag or f"harbor-obf-{dest_dir.name}"
    validation: dict[str, object]
    if skip_docker:
        validation = {"skipped": True}
    else:
        validation = validate_with_docker(
            dest_dir,
            tag,
            alt_expected_pass=alt_expected_pass,
        )

    search_idents = distinctive_identifiers(dest_src, mapping)
    search_rows = searchability_check(search_idents, upstream)

    result = ObfuscateResult(
        dest=dest_dir,
        mapping=mapping,
        mapping_size=len(mapping),
        validation=validation,
        searchability=search_rows,
        tests_adjusted=tests_adjusted,
        skipped_renames=skipped,
    )
    (dest_dir / "obfuscate_result.json").write_text(
        json.dumps(
            {
                "mapping_size": result.mapping_size,
                "mapping": mapping,
                "validation": validation,
                "searchability": search_rows,
                "tests_adjusted": tests_adjusted,
                "skipped_renames": skipped,
            },
            indent=2,
        )
        + "\n"
    )
    return result


def validate_with_docker(
    task_dir: Path,
    image: str,
    *,
    alt_expected_pass: bool = True,
) -> dict[str, object]:
    env_dir = task_dir / "environment"
    test_sh = task_dir / "tests" / "test.sh"
    patches = task_dir / "patches"
    build = _run(
        ["docker", "build", "-t", image, "-f", "Dockerfile", "."],
        cwd=env_dir,
        timeout=1800,
    )
    if build.returncode != 0:
        return {
            "builds": False,
            "error": (build.stderr or build.stdout)[-2000:],
        }
    rows: dict[str, object] = {"builds": True, "image": image}

    def reward(patch: Path | None) -> dict[str, object]:
        vols = [
            "-v",
            f"{test_sh.resolve()}:/tmp/test.sh:ro",
        ]
        cmd = "mkdir -p /logs/verifier; bash /tmp/test.sh; echo REWARD=$(cat /logs/verifier/reward.txt 2>/dev/null || echo missing)"
        if patch is not None:
            vols.extend(["-v", f"{patch.resolve()}:/tmp/apply.patch:ro"])
            cmd = (
                "mkdir -p /logs/verifier; cd /app && patch -p1 --forward --batch -i /tmp/apply.patch "
                "&& bash /tmp/test.sh; echo REWARD=$(cat /logs/verifier/reward.txt 2>/dev/null || echo missing)"
            )
        proc = _run(
            ["docker", "run", "--rm", *vols, image, "bash", "-lc", cmd],
            timeout=1800,
        )
        out = (proc.stdout or "") + (proc.stderr or "")
        m = re.search(r"REWARD=(\S+)", out)
        val = m.group(1) if m else "missing"
        return {"reward": val, "rc": proc.returncode, "tail": out[-1500:]}

    rows["buggy"] = reward(None)
    rows["gold"] = reward(patches / "gold.patch")
    rows["alt"] = reward(patches / "alt.patch")
    rows["cheat"] = reward(patches / "cheat.patch")
    rows["ok"] = (
        str(rows["buggy"]["reward"]) == "0"
        and str(rows["gold"]["reward"]) == "1"
        and str(rows["cheat"]["reward"]) == "0"
        and (
            (alt_expected_pass and str(rows["alt"]["reward"]) == "1")
            or (not alt_expected_pass and str(rows["alt"]["reward"]) != "1")
        )
    )
    return rows


def render_obfuscation_md(results: Sequence[tuple[TaskSpec, ObfuscateResult]]) -> str:
    lines = [
        "# harbor_nex obfuscation",
        "",
        "Date: 2026-09-18. Identity obfuscation of six client-go Harbor tasks so",
        "server-side web search/fetch of `tikv/client-go` cannot recall the fix.",
        "Module path `github.com/tikv/client-go/v2` → `example.internal/kvstore/v2`.",
        "Brand text stripped; subsystem symbols renamed via `gopls rename`.",
        "Did not launch a Harbor job.",
        "",
        "Reproduce:",
        "",
        "```",
        "uv run python scripts/obfuscate_task.py --batch",
        "```",
        "",
        "Output: `experiments/harbor_nex/tasks_obf/<name>-obf/`.",
        "",
    ]
    for spec, res in results:
        lines.append(f"## {spec.name}-obf")
        lines.append("")
        lines.append(f"- mapping size: **{res.mapping_size}**")
        if spec.notes:
            lines.append(f"- notes: {spec.notes}")
        val = res.validation
        lines.append("")
        lines.append("| check | result |")
        lines.append("|---|---|")
        if val.get("skipped"):
            lines.append("| docker | skipped |")
        else:
            lines.append(f"| builds | {'yes' if val.get('builds') else 'no'} |")
            for key in ("buggy", "gold", "alt", "cheat"):
                row = val.get(key) or {}
                lines.append(f"| {key} reward | {row.get('reward', '?')} |")
            lines.append(f"| table ok | {val.get('ok')} |")
        lines.append("| tests adjusted (semantic) | none |")
        lines.append("")
        lines.append("Searchability (new identifiers vs upstream checkout):")
        lines.append("")
        lines.append("| ident | upstream matches | ok |")
        lines.append("|---|---:|---|")
        for row in res.searchability:
            lines.append(f"| `{row['ident']}` | {row['matches']} | {row['ok']} |")
        if res.tests_adjusted:
            lines.append("")
            lines.append("Tests adjusted: " + ", ".join(res.tests_adjusted))
        else:
            lines.append("")
            lines.append("No test semantics changed (checksums refreshed after rename/import rewrite).")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def obfuscate_batch(
    *,
    skip_docker: bool = False,
    out_root: Path | None = None,
) -> list[tuple[TaskSpec, ObfuscateResult]]:
    out_root = out_root or (ROOT / "experiments/harbor_nex/tasks_obf")
    out_root.mkdir(parents=True, exist_ok=True)
    upstream = ROOT / "experiments/codegraph_bugs/repos/client-go"
    results: list[tuple[TaskSpec, ObfuscateResult]] = []
    for spec in default_task_specs():
        dest = out_root / f"{spec.name}-obf"
        print(f"=== obfuscate {spec.name} -> {dest} ===", flush=True)
        res = obfuscate_task(
            Path(spec.src_rel),
            dest,
            bug_patch=spec.bug_patch,
            alt_patch=spec.alt_patch,
            cheat_patch=spec.cheat_patch,
            upstream=upstream,
            skip_docker=skip_docker,
            image_tag=f"harbor-obf-{spec.name}",
            alt_expected_pass=spec.alt_expected_pass,
        )
        results.append((spec, res))
    md = render_obfuscation_md(results)
    (ROOT / "experiments/harbor_nex/OBFUSCATION.md").write_text(md, encoding="utf-8")
    return results


def _docker_only() -> int:
    """Rebuild images and re-run buggy/gold/alt/cheat on existing tasks_obf dirs."""
    out_root = ROOT / "experiments/harbor_nex/tasks_obf"
    upstream = ROOT / "experiments/codegraph_bugs/repos/client-go"
    results: list[tuple[TaskSpec, ObfuscateResult]] = []
    for spec in default_task_specs():
        dest = out_root / f"{spec.name}-obf"
        mapping_path = dest / "mapping.json"
        result_path = dest / "obfuscate_result.json"
        if not mapping_path.is_file():
            print(f"skip {spec.name}: no mapping.json", flush=True)
            continue
        mapping = json.loads(mapping_path.read_text())
        print(f"=== docker validate {spec.name} ===", flush=True)
        validation = validate_with_docker(
            dest, f"harbor-obf-{spec.name}", alt_expected_pass=spec.alt_expected_pass
        )
        search_idents = distinctive_identifiers(dest / "environment" / "src", mapping)
        search_rows = searchability_check(search_idents, upstream)
        res = ObfuscateResult(
            dest=dest,
            mapping=mapping,
            mapping_size=len(mapping),
            validation=validation,
            searchability=search_rows,
            tests_adjusted=[],
        )
        if result_path.is_file():
            prev = json.loads(result_path.read_text())
            prev["validation"] = validation
            prev["searchability"] = search_rows
            result_path.write_text(json.dumps(prev, indent=2) + "\n")
        results.append((spec, res))
    md = render_obfuscation_md(results)
    (ROOT / "experiments/harbor_nex/OBFUSCATION.md").write_text(md, encoding="utf-8")
    print(md)
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Obfuscate a Harbor Go task's identity")
    parser.add_argument("task_dir", nargs="?")
    parser.add_argument("out_dir", nargs="?")
    parser.add_argument("--gold-bug-patch", dest="bug_patch")
    parser.add_argument("--alt", dest="alt_patch")
    parser.add_argument("--cheat", dest="cheat_patch")
    parser.add_argument("--upstream", default=str(ROOT / "experiments/codegraph_bugs/repos/client-go"))
    parser.add_argument("--batch", action="store_true")
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--docker-only", action="store_true")
    parser.add_argument("--alt-expected-fail", action="store_true")
    args = parser.parse_args(argv)
    if args.docker_only:
        return _docker_only()
    if args.batch:
        obfuscate_batch(skip_docker=args.skip_docker)
        return 0
    if not args.task_dir or not args.out_dir:
        parser.error("task_dir and out_dir required unless --batch")
    if not args.bug_patch or not args.alt_patch or not args.cheat_patch:
        parser.error("--gold-bug-patch, --alt, and --cheat are required")
    obfuscate_task(
        args.task_dir,
        args.out_dir,
        bug_patch=args.bug_patch,
        alt_patch=args.alt_patch,
        cheat_patch=args.cheat_patch,
        upstream=args.upstream,
        skip_docker=args.skip_docker,
        alt_expected_pass=not args.alt_expected_fail,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
