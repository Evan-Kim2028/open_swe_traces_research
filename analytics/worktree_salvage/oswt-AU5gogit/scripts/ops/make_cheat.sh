#!/bin/bash
# make_cheat.sh <unit_dir> — produce _author/cheat.patch by diffing a full
# excised tree against the same tree with _author/cheat/<relpath> files
# overlaid. Expects _author/cheat/ to already contain the cheat versions of
# the touched files (full contents).
set -uo pipefail
UNIT="$1"
AU="$UNIT/_author"
GOLD_SRC=${GOLD_SRC:-/home/evan/Documents/oswt-AU5gogit/outputs/scratch/gogit_gold}
EXC="$AU/excised"

python3 - "$GOLD_SRC" "$EXC/excision.patch" "$AU/cheat" "$AU/cheat.patch" <<'PY'
import subprocess, sys, tempfile
from pathlib import Path

gold_src, excision, cheat_overlay, out = sys.argv[1:5]
with tempfile.TemporaryDirectory() as td:
    a_dir, b_dir = Path(td) / "a", Path(td) / "b"
    subprocess.run(["cp", "-a", f"{gold_src}/.", str(a_dir)], check=True)
    subprocess.run(["patch", "-p1", "-s", "-d", str(a_dir)],
                   stdin=open(excision, "rb"), check=True)
    subprocess.run(["cp", "-al", str(a_dir) + "/.", str(b_dir)], check=True)
    # overlay cheat files onto b
    for f in Path(cheat_overlay).rglob("*"):
        if f.is_file():
            rel = f.relative_to(cheat_overlay)
            (b_dir / rel).parent.mkdir(parents=True, exist_ok=True)
            (b_dir / rel).unlink(missing_ok=True)
            subprocess.run(["cp", str(f), str(b_dir / rel)], check=True)
    p = subprocess.run(["git", "diff", "--no-index", "a", "b"],
                       cwd=td, capture_output=True, text=True)

    def norm(line):
        parts = line.split(" ")
        for i, tok in enumerate(parts):
            t = tok.rstrip("\n")
            for pre in ("a/", "b/"):
                if t.startswith(pre):
                    rest = t[len(pre):]
                    if rest.startswith(("a/", "b/")):
                        parts[i] = pre + rest[2:] + ("\n" if tok.endswith("\n") else "")
                    break
        return " ".join(parts)

    outl = []
    for line in p.stdout.splitlines(keepends=True):
        if line.startswith(("diff --git ", "rename from ", "rename to ",
                            "copy from ", "copy to ", "--- ", "+++ ")):
            line = norm(line)
        outl.append(line)
    Path(out).write_text("".join(outl))
    print("wrote", out, f"({len(outl)} lines)")
PY
