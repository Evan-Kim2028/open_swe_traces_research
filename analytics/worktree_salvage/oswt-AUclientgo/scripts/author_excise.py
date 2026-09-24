"""Stub Go function bodies with `panic("excised: <name>")`, preserving declarations.

Usage:
    uv run python scripts/author_excise.py FILE [FUNC ...]
    uv run python scripts/author_excise.py FILE --all
    uv run python scripts/author_excise.py FILE --all --except NAME,NAME2

Replaces each named (or all) top-level `func` body with a panic stub.
Doc comments and signatures are preserved. Multi-line signatures and
one-line bodies are handled. Interface methods / type / var / const
declarations are never touched (only lines starting with `func `).
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

FUNC_RE = re.compile(r"^func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(")


def stub_file(path: Path, only: set[str] | None, except_: set[str]) -> list[str]:
    lines = path.read_text().splitlines(keepends=True)
    out: list[str] = []
    stubbed: list[str] = []
    i = 0
    n = len(lines)
    while i < n:
        line = lines[i]
        m = FUNC_RE.match(line)
        if not m:
            out.append(line)
            i += 1
            continue
        name = m.group(1)
        # find end of signature: first line containing '{'
        j = i
        while j < n and "{" not in lines[j]:
            j += 1
        if j >= n:
            # no body (shouldn't happen for a top-level func)
            out.append(line)
            i += 1
            continue
        if only is not None and name not in only or name in except_:
            # keep func untouched: copy through its body
            # find matching closing brace at column 0 or same-line '}'
            k = j
            depth = lines[j].count("{") - lines[j].count("}")
            if depth <= 0:
                k = j
            else:
                k = j + 1
                while k < n and not lines[k].startswith("}"):
                    k += 1
            out.extend(lines[i : k + 1])
            i = k + 1
            continue
        # stub it
        brace_idx = lines[j].index("{")
        sig = "".join(lines[i:j]) + lines[j][: brace_idx + 1] + "\n"
        out.append(sig)
        # skip body: if depth <=0 body is same-line; else find ^}
        depth = lines[j][brace_idx:].count("{") - lines[j][brace_idx:].count("}")
        k = j
        if depth > 0:
            k = j + 1
            while k < n and not lines[k].startswith("}"):
                k += 1
        out.append(f'\tpanic("excised: {name}")\n')
        out.append("}\n")
        stubbed.append(name)
        i = k + 1
    path.write_text("".join(out))
    return stubbed


def main() -> None:
    args = sys.argv[1:]
    if not args:
        print(__doc__)
        sys.exit(1)
    path = Path(args[0])
    only: set[str] | None = None
    except_: set[str] = set()
    rest = args[1:]
    if "--all" in rest:
        only = None
        rest.remove("--all")
    for a in list(rest):
        if a.startswith("--except="):
            except_ = set(a.split("=", 1)[1].split(","))
            rest.remove(a)
    if only is None and not rest and "--all" not in args:
        pass
    if rest:
        only = set(rest)
    stubbed = stub_file(path, only, except_)
    print(f"{path}: stubbed {len(stubbed)}: {', '.join(stubbed)}")


if __name__ == "__main__":
    main()
