"""Mechanical half of L0 unit authoring: excise, trim, patch, verify.

One spec per unit (JSON)::

    {
      "unit": "randgen",
      "excise": {"expr/random.go": ["NewFakerRandomizer", "FakerRandomizer.Int", ...]},
      "test_delete": ["expr/example_test.go"],
      "test_trim": {"expr/x_test.go": ["TestFoo", "TestBar"]},
      "test_add": {"expr/helpers_test.go": "<full file content>"},
      "verify_pkgs": ["./expr/..."]
    }

Layout produced under <out>/<unit>/_author/:
    excised/excision.patch   orig -> stubbed + test edits (all changed files)
    excised/<relpath>        every changed file, post-excision content
    gold.patch               stubbed -> orig, non-test files only
    cheat.patch              stubbed -> cheat impl, non-test files only (written by `cheat`)

Verification: `go build ./...` and `go test -count=1 -run '^$' <verify_pkgs>` in
the excised tree must both pass before patches are emitted.
"""

from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

FUNC_RE = re.compile(r"^func\s+(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(")
STR_LIT_RE = re.compile(r'"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'')


def _strip_brace_noise(line: str) -> str:
    """Blank signature self-balancing braces and string/char literals with
    same-length spaces so brace counting only sees structural braces while
    offsets stay aligned with the original text."""
    line = STR_LIT_RE.sub(lambda m: " " * len(m.group(0)), line)
    for tok in ("struct{}", "struct {}", "interface{}", "interface {}"):
        line = line.replace(tok, " " * len(tok))
    return line


def find_func_spans(lines: list[str], wanted: set[str]) -> list[tuple[int, int, int, str]]:
    """(decl_line, brace_line, end_line, qualified_name) for each wanted func."""
    spans = []
    n = len(lines)
    i = 0
    while i < n:
        m = FUNC_RE.match(lines[i])
        if not m:
            i += 1
            continue
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
        end = -1
        for k in range(brace_idx, n):
            # struct{}/interface{} braces inside signatures self-balance;
            # strip them so only the function body braces count.
            countable = _strip_brace_noise(lines[k])
            for ch in countable:
                if ch == "{":
                    depth += 1
                elif ch == "}":
                    depth -= 1
                    if depth == 0:
                        end = k
                        break
            if end != -1:
                break
        if end == -1:
            i += 1
            continue
        if qname in wanted or name in wanted:
            spans.append((i, brace_idx, end, qname))
        i = end + 1
    return spans


def stub_file(path: Path, funcs: list[str]) -> list[str]:
    lines = path.read_text().splitlines(keepends=True)
    spans = find_func_spans(lines, set(funcs))
    found = {q for _, _, _, q in spans}
    missing = {f for f in funcs if f not in found and f.split(".")[-1] not in {q.split(".")[-1] for q in found}}
    out: list[str] = []
    prev = 0
    for start, bopen, end, qname in spans:
        out.extend(lines[prev:start])
        # The body brace is the LAST `{` seen at depth 0 before the func's
        # closing `}` — signature braces (struct{}, interface{}) self-balance.
        sig = "".join(lines[start : end + 1])
        countable = _strip_brace_noise(sig)
        depth = 0
        body_open = -1
        for idx, ch in enumerate(countable):
            if ch == "{":
                if depth == 0:
                    body_open = idx
                depth += 1
            elif ch == "}":
                depth -= 1
        out.append(sig[:body_open].rstrip() + " {\n")
        out.append(f'\tpanic("excised: {qname}")\n')
        out.append("}\n")
        prev = end + 1
    out.extend(lines[prev:])
    path.write_text("".join(out))
    return sorted(missing)


def fix_unused_imports(path: Path) -> None:
    txt = path.read_text()
    m = re.search(r'import\s*\((.*?)\)', txt, re.S)
    if not m:
        return
    block = m.group(1)
    body = txt[: m.start()] + txt[m.end():]
    new_lines = []
    for iline in block.splitlines(keepends=True):
        im = re.match(r'(\s*)(?:(\w+|\.|_)\s+)?"([^"]+)"', iline)
        if not im:
            new_lines.append(iline)
            continue
        indent, alias, ipath = im.groups()
        name = alias or ipath.rsplit("/", 1)[-1].replace("-", "_")
        if alias in (".", "_") or re.search(rf'\b{re.escape(name)}\.[A-Za-z_]', body):
            new_lines.append(iline)
            continue
        new_lines.append(f'{indent}_ "{ipath}"\n')
    new_txt = txt[: m.start()] + "import (" + "".join(new_lines) + ")" + txt[m.end():]
    if new_txt != txt:
        path.write_text(new_txt)


def drop_unused_imports(path: Path) -> None:
    """Delete (not blank) unused import specs — for trimmed test files where a
    blank import would be visible noise in the task tree."""
    txt = path.read_text()
    m = re.search(r'import\s*\((.*?)\)', txt, re.S)
    if not m:
        return
    block = m.group(1)
    body = txt[: m.start()] + txt[m.end():]
    new_lines = []
    for iline in block.splitlines(keepends=True):
        im = re.match(r'(\s*)(?:(\w+|\.|_)\s+)?"([^"]+)"', iline)
        if not im:
            new_lines.append(iline)
            continue
        indent, alias, ipath = im.groups()
        name = alias or ipath.rsplit("/", 1)[-1].replace("-", "_")
        if alias in (".", "_") or re.search(rf'\b{re.escape(name)}\.[A-Za-z_]', body):
            new_lines.append(iline)
    new_txt = txt[: m.start()] + "import (" + "".join(new_lines) + ")" + txt[m.end():]
    if new_txt != txt:
        path.write_text(new_txt)


def trim_test_funcs(path: Path, funcs: list[str]) -> list[str]:
    """Remove whole test functions (doc comment through closing brace)."""
    lines = path.read_text().splitlines(keepends=True)
    spans = find_func_spans(lines, set(funcs))
    found = {q for _, _, _, q in spans}
    missing = sorted(set(funcs) - found - {q.split(".")[-1] for q in found})
    out: list[str] = []
    prev = 0
    for start, _bopen, end, _q in spans:
        # include the doc comment block immediately above the func decl
        c = start
        while c > 0 and lines[c - 1].lstrip().startswith("//"):
            c -= 1
        out.extend(lines[prev:c])
        prev = end + 1
    out.extend(lines[prev:])
    path.write_text("".join(out))
    return missing


def _rel_files(root: Path) -> dict[str, Path]:
    return {p.relative_to(root).as_posix(): p for p in root.rglob("*") if p.is_file() and ".git" not in p.parts}


def _diff_file(a: Path | None, b: Path | None, rel: str) -> str:
    """`diff -u` between optional paths, labelled a/<rel> b/<rel> (dev/null for absent)."""
    alabel = f"a/{rel}" if a is not None else "/dev/null"
    blabel = f"b/{rel}" if b is not None else "/dev/null"
    atext = a.read_text(errors="replace") if a is not None else ""
    btext = b.read_text(errors="replace") if b is not None else ""
    import tempfile
    with tempfile.NamedTemporaryFile("w", suffix=".go", delete=False) as fa, \
         tempfile.NamedTemporaryFile("w", suffix=".go", delete=False) as fb:
        fa.write(atext)
        fb.write(btext)
        fa.flush()
        fb.flush()
        proc = subprocess.run(
            ["diff", "-u", "--label", alabel, "--label", blabel, fa.name, fb.name],
            capture_output=True, text=True,
        )
    return proc.stdout


def build_unit(spec_path: Path, src: Path, out_author: Path, work: Path, *, verify: bool = True) -> int:
    spec = json.loads(spec_path.read_text())
    unit = spec["unit"]
    tree = work / unit / "src"
    if tree.exists():
        shutil.rmtree(tree)
    shutil.copytree(src, tree, symlinks=True)
    author = out_author / unit / "_author"
    excised_dir = author / "excised"
    excised_dir.mkdir(parents=True, exist_ok=True)

    # 1. stub
    for rel, funcs in spec.get("excise", {}).items():
        missing = stub_file(tree / rel, funcs)
        if missing:
            print(f"WARN {unit} {rel}: not found: {missing}")
        fix_unused_imports(tree / rel)

    # 2. test edits
    for rel in spec.get("test_delete", []):
        (tree / rel).unlink()
    for rel, funcs in spec.get("test_trim", {}).items():
        missing = trim_test_funcs(tree / rel, funcs)
        if missing:
            print(f"WARN {unit} trim {rel}: not found: {missing}")
        drop_unused_imports(tree / rel)
    for rel, content in spec.get("test_add", {}).items():
        (tree / rel).parent.mkdir(parents=True, exist_ok=True)
        (tree / rel).write_text(content)

    # 3. gofmt sanity + verify
    bad = subprocess.run(["gofmt", "-l", "./..."], cwd=tree, capture_output=True, text=True).stdout.strip()
    if bad:
        print(f"GOFMT-BAD {unit}: {bad}")
    if verify:
        b = subprocess.run(["go", "build", "./..."], cwd=tree, capture_output=True, text=True)
        if b.returncode != 0:
            print(f"BUILD-FAIL {unit}:\n{b.stdout[-2000:]}{b.stderr[-3000:]}")
            return 1
        for pkg in spec.get("verify_pkgs", []):
            t = subprocess.run(["go", "test", "-count=1", "-run", "^$", pkg], cwd=tree,
                               capture_output=True, text=True)
            if t.returncode != 0:
                print(f"TESTCOMPILE-FAIL {unit} {pkg}:\n{t.stdout[-2000:]}{t.stderr[-3000:]}")
                return 1

    # 4. patches
    orig = _rel_files(src)
    new = _rel_files(tree)
    changed = [r for r in sorted(set(orig) | set(new))
               if (r not in orig) or (r not in new) or orig[r].read_bytes() != new[r].read_bytes()]
    excision, gold = [], []
    for rel in changed:
        a = orig.get(rel)
        b = new.get(rel)
        excision.append(_diff_file(a, b, rel))
        if not rel.endswith("_test.go"):
            gold.append(_diff_file(b, a, rel))
    (excised_dir / "excision.patch").write_text("".join(excision))
    (author / "gold.patch").write_text("".join(gold))
    for rel in changed:
        if rel not in new:
            continue  # deleted file: recorded in excision.patch, nothing to copy
        dest = excised_dir / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(new[rel], dest)
    print(f"OK {unit}: {len(changed)} changed files, {len(gold)} gold hunks")
    return 0


def build_cheat(spec_path: Path, src: Path, out_author: Path, work: Path) -> int:
    """Diff the excised tree against <work>/<unit>/cheat for non-test files -> cheat.patch.

    The cheat tree must already exist at <work>/<unit>/cheat (copy of the excised
    tree with cheat implementations edited in).
    """
    spec = json.loads(spec_path.read_text())
    unit = spec["unit"]
    exc_tree = work / unit / "src"
    cheat_tree = work / unit / "cheat"
    if not cheat_tree.is_dir():
        print(f"CHEAT-MISSING {unit}: {cheat_tree}")
        return 1
    b = subprocess.run(["go", "build", "./..."], cwd=cheat_tree, capture_output=True, text=True)
    if b.returncode != 0:
        print(f"CHEAT-BUILD-FAIL {unit}:\n{b.stdout[-1500:]}{b.stderr[-2500:]}")
        return 1
    exc = _rel_files(exc_tree)
    cht = _rel_files(cheat_tree)
    changed = [r for r in sorted(set(exc) | set(cht))
               if not r.endswith("_test.go")
               and ((r not in exc) or (r not in cht) or exc[r].read_bytes() != cht[r].read_bytes())]
    cheat = "".join(_diff_file(exc.get(r), cht.get(r), r) for r in changed)
    author = out_author / unit / "_author"
    (author / "cheat.patch").write_text(cheat)
    gold_adds = len(re.findall(r"^\+(?!\+\+)", (author / "gold.patch").read_text(errors="replace"), re.M))
    cheat_adds = len(re.findall(r"^\+(?!\+\+)", cheat, re.M))
    ratio = cheat_adds / gold_adds if gold_adds else 0.0
    print(f"CHEAT {unit}: {cheat_adds} adds vs gold {gold_adds} -> ratio {ratio:.2f} "
          f"({'ok' if ratio < 0.6 else 'TOO BIG'})")
    return 0 if ratio < 0.6 else 2


def main(argv: list[str]) -> int:
    mode, spec_path, src, out_author, work = argv[1], Path(argv[2]), Path(argv[3]), Path(argv[4]), Path(argv[5])
    work.mkdir(parents=True, exist_ok=True)
    if mode == "excise":
        return build_unit(spec_path, src, out_author, work)
    if mode == "cheat":
        return build_cheat(spec_path, src, out_author, work)
    print(f"unknown mode {mode}", file=sys.stderr)
    return 2
