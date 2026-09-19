"""Minimal YAML subset loader (mappings, lists, scalars, comments, _-numbers)."""

from __future__ import annotations

import re
from typing import Any

_NUM_RE = re.compile(r"^-?\d+(?:\.\d+)?$")


def load_yaml(text: str) -> Any:
    lines = [_strip_comment(ln) for ln in text.splitlines()]
    cleaned = [(i, ln) for i, ln in enumerate(lines) if ln.strip()]
    if not cleaned:
        return {}
    value, _ = _parse_block(cleaned, 0, _indent(cleaned[0][1]))
    return value


def load_yaml_path(path: Any) -> Any:
    from pathlib import Path

    return load_yaml(Path(path).read_text(encoding="utf-8"))


def _strip_comment(line: str) -> str:
    out: list[str] = []
    quote = ""
    for ch in line:
        if quote:
            out.append(ch)
            if ch == quote:
                quote = ""
            continue
        if ch in {'"', "'"}:
            quote = ch
            out.append(ch)
            continue
        if ch == "#":
            break
        out.append(ch)
    return "".join(out).rstrip()


def _indent(line: str) -> int:
    return len(line) - len(line.lstrip(" "))


def _parse_block(lines: list[tuple[int, str]], idx: int, indent: int) -> tuple[Any, int]:
    if idx >= len(lines):
        return {}, idx
    _, first = lines[idx]
    if first.lstrip().startswith("- "):
        return _parse_list(lines, idx, indent)
    return _parse_map(lines, idx, indent)


def _parse_map(lines: list[tuple[int, str]], idx: int, indent: int) -> tuple[dict[str, Any], int]:
    out: dict[str, Any] = {}
    while idx < len(lines):
        _, raw = lines[idx]
        ind = _indent(raw)
        if ind < indent:
            break
        if ind > indent:
            break
        stripped = raw.strip()
        if stripped.startswith("- "):
            break
        if ":" not in stripped:
            raise ValueError(f"expected key: value, got {stripped!r}")
        key, rest = stripped.split(":", 1)
        key = key.strip()
        rest = rest.strip()
        idx += 1
        if rest:
            out[key] = _scalar(rest)
            continue
        if idx >= len(lines):
            out[key] = {}
            continue
        nxt_ind = _indent(lines[idx][1])
        if nxt_ind <= indent:
            out[key] = {}
            continue
        value, idx = _parse_block(lines, idx, nxt_ind)
        out[key] = value
    return out, idx


def _parse_list(lines: list[tuple[int, str]], idx: int, indent: int) -> tuple[list[Any], int]:
    out: list[Any] = []
    while idx < len(lines):
        _, raw = lines[idx]
        ind = _indent(raw)
        if ind < indent:
            break
        stripped = raw.strip()
        if not stripped.startswith("- "):
            break
        body = stripped[2:]
        idx += 1
        if body.startswith("- "):
            raise ValueError("nested dash lists are not supported")
        if ":" in body and not _looks_like_scalar_map_inline(body):
            key, rest = body.split(":", 1)
            item: dict[str, Any] = {key.strip(): _scalar(rest.strip()) if rest.strip() else {}}
            child_indent = ind + 2
            while idx < len(lines):
                _, nxt = lines[idx]
                nind = _indent(nxt)
                nstrip = nxt.strip()
                if nind < child_indent:
                    break
                if nstrip.startswith("- ") and nind == indent:
                    break
                if nind == child_indent and ":" in nstrip and not nstrip.startswith("- "):
                    k, r = nstrip.split(":", 1)
                    k, r = k.strip(), r.strip()
                    idx += 1
                    if r:
                        item[k] = _scalar(r)
                        continue
                    if idx < len(lines) and _indent(lines[idx][1]) > child_indent:
                        nested, idx = _parse_block(lines, idx, _indent(lines[idx][1]))
                        item[k] = nested
                    else:
                        item[k] = {}
                    continue
                if nind > child_indent:
                    nested, idx = _parse_block(lines, idx, nind)
                    last = next(reversed(item))
                    if item[last] in ({}, None):
                        item[last] = nested
                    continue
                break
            if item[next(iter(item))] == {} and len(item) == 1 and idx < len(lines):
                pass
            out.append(item)
            continue
        if body.endswith(":") and ":" in body:
            key = body[:-1].strip()
            if idx < len(lines) and _indent(lines[idx][1]) > ind:
                nested, idx = _parse_block(lines, idx, _indent(lines[idx][1]))
                out.append({key: nested})
            else:
                out.append({key: {}})
            continue
        out.append(_scalar(body) if body else {})
    return out, idx


def _looks_like_scalar_map_inline(body: str) -> bool:
    """`url: https://...` is a mapping entry, not a scalar. Always mapping if `:` present."""
    return False


def _scalar(text: str) -> Any:
    if text == "":
        return ""
    if text in {"true", "True", "yes", "YES"}:
        return True
    if text in {"false", "False", "no", "NO"}:
        return False
    if text in {"null", "None", "~"}:
        return None
    if (text.startswith('"') and text.endswith('"')) or (text.startswith("'") and text.endswith("'")):
        return text[1:-1]
    compact = text.replace("_", "")
    if _NUM_RE.fullmatch(compact):
        return float(compact) if "." in compact else int(compact)
    return text
