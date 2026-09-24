"""What the excision removed: symbol names, file paths, line numbers.

Sources, strongest first:

1. ``environment/src`` — the excised tree itself carries ``panic("excised: X")``
   stubs (real and fabricated units alike); the stub's own line number is the
   excised site.
2. ``tests/gold.patch`` / ``patches/gold.patch`` — its ``-`` lines remove the
   same stubs; ``+++ b/`` headers name touched files; hunk ``-`` positions give
   the stub (≈ excised) line numbers.
3. ``tests/excision.patch`` / ``patches/excision.patch`` — the raw excision
   (``+`` lines carry the stubs, ``-`` lines the removed implementation).
4. ``validation.json`` ``changed_symbols`` / ``changed_files`` — documentary
   fallback only.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path

EXCISED_RE = re.compile(r'panic\("excised:\s*([A-Za-z0-9_.]+)"')
FUNC_RE = re.compile(r"^func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)")
DIFF_FILE_RE = re.compile(r"^\+\+\+\s+b/(\S+)", re.MULTILINE)
DIFF_FILE_RE2 = re.compile(r"^diff --git a/\S+ b/(\S+)", re.MULTILINE)
_HUNK_RE = re.compile(r"^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@")


@dataclass(frozen=True)
class ExcisedInfo:
    symbols: frozenset[str]
    files: frozenset[str]  # repo-relative paths the excision touched
    source: str  # how the set was derived
    # (file basename, line number) of stub/removal sites — for B7 line leaks
    removed_lines: tuple[tuple[str, int], ...] = field(default=())

    @property
    def bare_names(self) -> frozenset[str]:
        """``Store.Get`` -> ``Get``; bare receiver-stripped names too."""
        out: set[str] = set()
        for s in self.symbols:
            out.add(s)
            out.add(s.rsplit(".", 1)[-1])
        return frozenset(out)


def _read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def _from_tree(task_dir: Path) -> ExcisedInfo:
    src = task_dir / "environment" / "src"
    symbols: set[str] = set()
    files: set[str] = set()
    lines: list[tuple[str, int]] = []
    if src.is_dir():
        for path in sorted(src.rglob("*")):
            if not path.is_file() or path.suffix not in {".go", ".py", ".rs", ".ts", ".java"}:
                continue
            text = _read(path)
            if "excised:" not in text:
                continue
            rel = path.relative_to(src).as_posix()
            for i, ln in enumerate(text.splitlines(), 1):
                for m in EXCISED_RE.finditer(ln):
                    symbols.add(m.group(1))
                    files.add(rel)
                    lines.append((path.name, i))
    return ExcisedInfo(frozenset(symbols), frozenset(files), "environment/src stubs", tuple(lines))


def _patch_minus_lines(text: str) -> tuple[set[str], set[str], list[tuple[str, int]]]:
    """(symbols-in-minus-lines, files, (basename, old-lineno) of '-' payload)."""
    symbols: set[str] = set()
    files = set(DIFF_FILE_RE.findall(text)) | set(DIFF_FILE_RE2.findall(text))
    lines: list[tuple[str, int]] = []
    cur_file = ""
    old_ln = 0
    in_hunk = False
    for raw in text.splitlines():
        if raw.startswith("+++"):
            cur_file = raw[4:].strip().removeprefix("b/")
            continue
        m = _HUNK_RE.match(raw)
        if m:
            old_ln = int(m.group(1))
            in_hunk = True
            continue
        if not in_hunk:
            continue
        if raw.startswith("-") and not raw.startswith("---"):
            for sm in EXCISED_RE.finditer(raw):
                symbols.add(sm.group(1))
            lines.append((Path(cur_file).name, old_ln))
            old_ln += 1
        elif raw.startswith("+"):
            pass
        else:
            old_ln += 1
    return symbols, files, lines


def _patch_plus_stub_lines(text: str) -> tuple[set[str], set[str], list[tuple[str, int]]]:
    """Excision patch: '+' side carries the panic stubs (new-side line numbers)."""
    symbols: set[str] = set()
    files = set(DIFF_FILE_RE.findall(text)) | set(DIFF_FILE_RE2.findall(text))
    lines: list[tuple[str, int]] = []
    cur_file = ""
    new_ln = 0
    in_hunk = False
    for raw in text.splitlines():
        if raw.startswith("+++"):
            cur_file = raw[4:].strip().removeprefix("b/")
            continue
        m = _HUNK_RE.match(raw)
        if m:
            new_ln = int(m.group(2))
            in_hunk = True
            continue
        if not in_hunk:
            continue
        if raw.startswith("+"):
            for sm in EXCISED_RE.finditer(raw):
                symbols.add(sm.group(1))
                lines.append((Path(cur_file).name, new_ln))
            new_ln += 1
        elif raw.startswith("-"):
            pass
        else:
            new_ln += 1
    return symbols, files, lines


def _from_gold(task_dir: Path) -> ExcisedInfo:
    for rel in ("tests/gold.patch", "patches/gold.patch"):
        text = _read(task_dir / rel)
        if not text:
            continue
        symbols, files, lines = _patch_minus_lines(text)
        if symbols:
            return ExcisedInfo(
                frozenset(symbols), frozenset(files), f"{rel} -lines", tuple(lines)
            )
    return ExcisedInfo(frozenset(), frozenset(), "")


def _from_excision_patch(task_dir: Path) -> ExcisedInfo:
    for rel in ("tests/excision.patch", "patches/excision.patch"):
        text = _read(task_dir / rel)
        if not text:
            continue
        symbols, files, lines = _patch_plus_stub_lines(text)
        # also collect the removed implementation's symbols? '-', lines are the
        # original code — its line numbers are the true excised lines.
        _ms, _mf, removed = _patch_minus_lines(text)
        if symbols or removed:
            return ExcisedInfo(
                frozenset(symbols), frozenset(files | _mf), f"{rel}", tuple(lines + removed)
            )
    return ExcisedInfo(frozenset(), frozenset(), "")


def _from_gold_funcs(task_dir: Path) -> ExcisedInfo:
    """No panic markers: gold.patch ``+`` lines re-add the removed functions."""
    for rel in ("tests/gold.patch", "patches/gold.patch"):
        text = _read(task_dir / rel)
        if not text:
            continue
        added = [ln[1:] for ln in text.splitlines() if ln.startswith("+") and not ln.startswith("+++")]
        symbols = set()
        for ln in added:
            symbols.update(FUNC_RE.findall(ln))
        _ms, files, lines = _patch_minus_lines(text)
        if symbols or files:
            return ExcisedInfo(
                frozenset(symbols), frozenset(files), f"{rel} +-lines funcs", tuple(lines)
            )
    return ExcisedInfo(frozenset(), frozenset(), "")


def _from_validation(task_dir: Path, rules_ctx: object) -> ExcisedInfo:
    data: dict = {}
    if rules_ctx is not None:
        data = dict(getattr(rules_ctx, "validation", {}) or {})
        extra = getattr(rules_ctx, "extra", {}) or {}
        for key in ("changed_symbols", "changed_files"):
            if not data.get(key) and extra.get(key):
                data[key] = extra[key]
    if not data:
        try:
            import json

            raw = json.loads(_read(task_dir / "validation.json") or "{}")
            if isinstance(raw, dict):
                data = raw
        except (ValueError, OSError):
            pass
    symbols = {str(s) for s in data.get("changed_symbols") or [] if s}
    files = {str(f) for f in data.get("changed_files") or [] if f}
    return ExcisedInfo(frozenset(symbols), frozenset(files), "validation.json changed_*")


def excised_info(task_dir: Path | str, *, rules_ctx: object = None) -> ExcisedInfo:
    td = Path(task_dir)
    for fn in (_from_tree, _from_gold, _from_excision_patch, _from_gold_funcs):
        info = fn(td)
        if info.symbols or info.files:
            return info
    return _from_validation(td, rules_ctx)
