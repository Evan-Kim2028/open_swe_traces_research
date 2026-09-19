#!/usr/bin/env python3
"""Excise Go function bodies -> panic("excised: <name>") stubs.

Usage: excise_funcs.py <repo_root> <spec.json>
spec.json: {"unit": {"relpath.go": ["Func", "Recv.Method", ...]}}
Keeps signatures/doc comments. Blank-imports imports left unused.
Repo must be a clean git repo; caller runs git diff per unit.
"""
import json
import re
import subprocess
import sys
from pathlib import Path


def find_func_spans(lines, wanted):
    spans = []
    n = len(lines)
    i = 0
    while i < n:
        m = re.match(r"^func\s+(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(", lines[i])
        if m:
            recv, name = m.group(1), m.group(2)
            qname = f"{recv.strip('() ').split()[-1].lstrip('*')}.{name}" if recv else name
            j = i
            brace_idx = -1
            while j < n:
                if "{" in lines[j]:
                    brace_idx = j
                    break
                j += 1
            if brace_idx == -1:
                i += 1
                continue
            depth = 0
            k = brace_idx
            end = -1
            started = False
            while k < n:
                for ch in lines[k]:
                    if ch == "{":
                        depth += 1
                        started = True
                    elif ch == "}":
                        depth -= 1
                        if started and depth == 0:
                            end = k
                            break
                if end != -1:
                    break
                k += 1
            if end == -1:
                i += 1
                continue
            if qname in wanted or name in wanted:
                spans.append((i, brace_idx, end, qname))
            i = end + 1
        else:
            i += 1
    return spans


def stub_file(path, funcs):
    lines = Path(path).read_text().splitlines(keepends=True)
    spans = find_func_spans(lines, set(funcs))
    found = {q for _, _, _, q in spans}
    missing = {f for f in funcs if f not in found and f.split(".")[-1] not in {q.split(".")[-1] for q in found}}
    out = []
    prev = 0
    for start, bopen, end, qname in spans:
        out.extend(lines[prev:bopen])
        head = lines[bopen]
        brace_pos = head.index("{")
        out.append(head[:brace_pos].rstrip() + " {\n")
        out.append(f'\tpanic("excised: {qname}")\n')
        out.append("}\n")
        prev = end + 1
    out.extend(lines[prev:])
    Path(path).write_text("".join(out))
    return missing


def fix_unused_imports(path):
    txt = Path(path).read_text()
    m = re.search(r'import\s*\((.*?)\)', txt, re.S)
    if not m:
        return
    block = m.group(1)
    body = txt[: m.start()] + txt[m.end():]
    new_lines = []
    changed = False
    for iline in block.splitlines(keepends=True):
        im = re.match(r'(\s*)(?:(\w+|\.|_)\s+)?"([^"]+)"', iline)
        if not im:
            new_lines.append(iline)
            continue
        indent, alias, ipath = im.groups()
        name = alias or ipath.rsplit("/", 1)[-1].replace("-", "_")
        if alias in (".", "_") or re.search(rf'\b{re.escape(name)}\.', body):
            new_lines.append(iline)
            continue
        new_lines.append(f'{indent}_ "{ipath}"\n')
        changed = True
    if changed:
        Path(path).write_text(txt[: m.start()] + "import (" + "".join(new_lines) + ")" + txt[m.end():])


def main():
    root = Path(sys.argv[1])
    spec = json.loads(Path(sys.argv[2]).read_text())
    for unit, files in spec.items():
        for rel, funcs in files.items():
            p = root / rel
            missing = stub_file(p, funcs)
            if missing:
                print(f"WARN {unit} {rel}: not found: {sorted(missing)}")
            fix_unused_imports(p)
            r = subprocess.run(["gofmt", "-l", str(p)], capture_output=True, text=True)
            if r.stdout.strip():
                print(f"GOFMT-BAD {unit} {rel}")
    print("done")


if __name__ == "__main__":
    main()
