#!/usr/bin/env python3
"""mkcheat.py — emit a git-format patch excised->(gold|cheat) for a list of files.

Usage: mkcheat.py <srcroot> <out.patch> <relpath>...
Reads the excised (pre-patch) copy from $EXCDIR/<relpath> and diffs it against
the current file at <srcroot>/<relpath>, rewriting headers to
a/<relpath> / b/<relpath>.
"""
import os
import subprocess
import sys

srcroot, out, files = sys.argv[1], sys.argv[2], sys.argv[3:]
excdir = os.environ["EXCDIR"]
chunks = []
for rel in files:
    old = os.path.join(excdir, rel)
    new = os.path.join(srcroot, rel)
    p = subprocess.run(
        ["git", "diff", "--no-index", "--", old, new],
        capture_output=True, text=True,
    )
    if p.returncode not in (0, 1):
        sys.exit(p.stderr)
    text = p.stdout
    if not text:
        continue
    lines = []
    for ln in text.splitlines():
        if ln.startswith("diff --git "):
            lines.append(f"diff --git a/{rel} b/{rel}")
        elif ln.startswith("--- "):
            lines.append(f"--- a/{rel}")
        elif ln.startswith("+++ "):
            lines.append(f"+++ b/{rel}")
        elif ln.startswith("index "):
            continue
        else:
            lines.append(ln)
    chunks.append("\n".join(lines) + "\n")
with open(out, "w") as f:
    f.write("".join(chunks))
print(f"wrote {out}: {len(chunks)} file(s)")
