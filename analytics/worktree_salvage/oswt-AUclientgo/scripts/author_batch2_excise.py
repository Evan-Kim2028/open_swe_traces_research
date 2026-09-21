#!/usr/bin/env python3
"""Authoring driver for batch2 units: copy pristine tree, stub funcs, truncate
mapped test files, compile-check, emit excision.patch + gold.patch.

Usage: uv run python scripts/author_batch2_excise.py <spec.json>
spec.json:
  {
    "unit": "<name>",
    "files": {"<rel.go>": ["Func", "Recv.Method", ...]},
    "truncate_tests": ["<rel_test.go>", ...]
  }

Emits:
  experiments/pipeline/authored_batch2/client-go/<unit>/_author/excised/excision.patch
  experiments/pipeline/authored_batch2/client-go/<unit>/_author/gold.patch
and leaves the stubbed tree at outputs/excise_work/<unit>/src for cheat work.
"""
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PRISTINE = ROOT / "outputs/clientgo-src-pristine"
WORK = ROOT / "outputs/excise_work"
UNITS = ROOT / "experiments/pipeline/authored_batch2/client-go"


def find_func_spans(lines, wanted):
    spans = []
    n = len(lines)
    i = 0
    while i < n:
        m = re.match(r"^func\s+(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(", lines[i])
        if m:
            recv, name = m.group(1), m.group(2)
            qname = f"{recv.strip('() ').split()[-1].lstrip('*')}.{name}" if recv else name
            # The body brace is the first "{" after the parameter list closes.
            # Track () [] {} depth so type literals (interface{}, struct{},
            # map/slice types) inside the signature don't count.
            depth_paren, depth_brack, depth_brace = 1, 0, 0
            params_closed = False
            brace_idx = -1
            brace_col = -1
            j = i
            col = m.end()  # m ends just past the "(" of the param list
            while j < n:
                line = lines[j]
                while col < len(line):
                    ch = line[col]
                    if ch == "(":
                        depth_paren += 1
                    elif ch == ")":
                        depth_paren -= 1
                        if depth_paren == 0:
                            params_closed = True
                    elif ch == "[":
                        depth_brack += 1
                    elif ch == "]":
                        depth_brack -= 1
                    elif ch == "{":
                        if params_closed and depth_paren == 0 and depth_brack == 0 and depth_brace == 0:
                            prev = line[:col].rstrip()
                            if prev.endswith("interface") or prev.endswith("struct"):
                                depth_brace += 1
                            else:
                                brace_idx = j
                                brace_col = col
                                break
                        else:
                            depth_brace += 1
                    elif ch == "}":
                        if depth_brace > 0:
                            depth_brace -= 1
                    col += 1
                if brace_idx != -1:
                    break
                j += 1
                col = 0
            if brace_idx == -1:
                i += 1
                continue
            # Body ends where depth returns to 0, counting from the body brace.
            depth = 1
            k = brace_idx
            c = brace_col + 1
            end = -1
            while k < n:
                line = lines[k]
                while c < len(line):
                    ch = line[c]
                    if ch == "{":
                        depth += 1
                    elif ch == "}":
                        depth -= 1
                        if depth == 0:
                            end = k
                            break
                    c += 1
                if end != -1:
                    break
                k += 1
                c = 0
            if end == -1:
                i += 1
                continue
            if qname in wanted or name in wanted:
                spans.append((i, brace_idx, end, qname, brace_col))
            i = end + 1
        else:
            i += 1
    return spans


def stub_file(path, funcs):
    lines = Path(path).read_text().splitlines(keepends=True)
    spans = find_func_spans(lines, set(funcs))
    found = {s[3] for s in spans}
    found_short = {q.split(".")[-1] for q in found}
    missing = {f for f in funcs if f not in found and f.split(".")[-1] not in found_short}
    out = []
    prev = 0
    for start, bopen, end, qname, bcol in spans:
        out.extend(lines[prev:bopen])
        head = lines[bopen]
        out.append(head[:bcol].rstrip() + " {\n")
        out.append(f'\tpanic("excised: {qname}")\n')
        out.append("}\n")
        prev = end + 1
    out.extend(lines[prev:])
    Path(path).write_text("".join(out))
    return missing, sorted(found)


_PKGNAME_CACHE = {}


def pkg_name_for(ipath):
    """Resolve the real package name for an import path.

    In-repo paths under the module may declare a package name different from
    the directory basename (e.g. wirerpc -> tikvrpc, error -> tikverr).
    """
    if ipath in _PKGNAME_CACHE:
        return _PKGNAME_CACHE[ipath]
    name = ipath.rsplit("/", 1)[-1].replace("-", "_")
    marker = "example.internal/kvstore/v2/"
    if ipath.startswith(marker):
        d = PRISTINE / ipath[len(marker):]
        if d.is_dir():
            for f in sorted(d.glob("*.go")):
                if f.name.endswith("_test.go"):
                    continue
                for line in f.read_text(errors="replace").splitlines():
                    pm = re.match(r"package\s+(\w+)", line)
                    if pm:
                        name = pm.group(1)
                        break
                if name != ipath.rsplit("/", 1)[-1].replace("-", "_"):
                    break
    _PKGNAME_CACHE[ipath] = name
    return name


def fix_unused_imports(path):
    txt = Path(path).read_text()
    m = re.search(r'import\s*\((.*?)\)', txt, re.S)
    single = None
    if not m:
        sm = re.search(r'^import\s+(?:(\w+|\.|_)\s+)?"([^"]+)"', txt, re.M)
        if not sm:
            return
        block = sm.group(0) + "\n"
        single = sm
    else:
        block = m.group(1)
    ispan = single or m
    # Strip // comments so "pkg." inside doc comments doesn't look like a use.
    body = re.sub(r"//[^\n]*", "", txt[: ispan.start()] + txt[ispan.end():])
    new_lines = []
    for iline in block.splitlines(keepends=True):
        indent = alias = ipath = None
        if single is not None:
            im = re.match(r'(import\s+)(?:(\w+|\.|_)\s+)?"([^"]+)"', iline)
            if im:
                indent, alias, ipath = "import ", im.group(2), im.group(3)
        else:
            im = re.match(r'(\s*)(?:(\w+|\.|_)\s+)?"([^"]+)"', iline)
            if im:
                indent, alias, ipath = im.groups()
        if ipath is None:
            new_lines.append(iline)
            continue
        name = alias or pkg_name_for(ipath)
        if alias in (".", "_") or re.search(rf'\b{re.escape(name)}\.', body):
            new_lines.append(iline)
            continue
        if single is not None:
            new_lines.append(f'import _ "{ipath}"\n')
        else:
            new_lines.append(f'{indent}_ "{ipath}"\n')
    if single is not None:
        Path(path).write_text(txt[: single.start()] + "".join(new_lines) + txt[single.end():])
    else:
        Path(path).write_text(txt[: m.start()] + "import (" + "".join(new_lines) + ")" + txt[m.end():])
    subprocess.run(["gofmt", "-w", str(path)], capture_output=True)


def truncate_test(path):
    pkg = None
    for line in Path(path).read_text().splitlines():
        if line.startswith("package "):
            pkg = line.strip()
            break
    if pkg is None:
        raise SystemExit(f"no package decl in {path}")
    Path(path).write_text(pkg + "\n")


def diff_file(old, new, rel):
    r = subprocess.run(
        ["diff", "-u", "--label", f"a/{rel}", "--label", f"b/{rel}", str(old), str(new)],
        capture_output=True, text=True,
    )
    return r.stdout if r.returncode == 1 else ""


def new_file_diff(new, rel):
    r = subprocess.run(
        ["diff", "-u", "--label", "/dev/null", "--label", f"b/{rel}", "/dev/null", str(new)],
        capture_output=True, text=True,
    )
    return r.stdout if r.returncode == 1 else ""


def cheat_patch(spec):
    """Emit cheat.patch = diff(stubbed state -> current workdir state) for spec files.

    Reconstructs the stub state by re-stubbing a pristine copy in a temp dir.
    """
    import tempfile
    unit = spec["unit"]
    files = spec["files"]
    dst = WORK / unit / "src"
    outdir = UNITS / unit / "_author"
    tmp = Path(tempfile.mkdtemp())
    patch_text = ""
    for rel, funcs in files.items():
        tdir = tmp / Path(rel).parent
        tdir.mkdir(parents=True, exist_ok=True)
        tfile = tmp / rel
        shutil.copy(PRISTINE / rel, tfile)
        stub_file(tfile, funcs)
        fix_unused_imports(tfile)
        patch_text += diff_file(tfile, dst / rel, rel)
    (outdir / "cheat.patch").write_text(patch_text)
    print(f"wrote {outdir}/cheat.patch ({len(patch_text)} bytes)")
    r = subprocess.run(["go", "build", "./..."], cwd=dst, capture_output=True, text=True)
    print("cheat compiles:", r.returncode == 0)
    if r.returncode != 0:
        print(r.stderr[:3000])


def main():
    if sys.argv[1] == "--cheat":
        cheat_patch(json.loads(Path(sys.argv[2]).read_text()))
        return
    spec = json.loads(Path(sys.argv[1]).read_text())
    unit = spec["unit"]
    files = spec["files"]
    trunc = spec.get("truncate_tests", [])

    dst = WORK / unit / "src"
    if dst.exists():
        shutil.rmtree(dst)
    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(PRISTINE, dst, symlinks=True)

    all_missing = {}
    stubbed = []
    for rel, funcs in files.items():
        p = dst / rel
        missing, found = stub_file(p, funcs)
        fix_unused_imports(p)
        if missing:
            all_missing[rel] = sorted(missing)
        stubbed.append(rel)
        print(f"stubbed {rel}: {len(found)} funcs")

    for rel in trunc:
        truncate_test(dst / rel)
        print(f"truncated {rel}")

    appends = spec.get("append_files", {})
    creates = spec.get("create_files", {})
    for rel, text in appends.items():
        p = dst / rel
        p.write_text(p.read_text() + text)
        subprocess.run(["gofmt", "-w", str(p)], capture_output=True)
        print(f"appended {rel}")
    for rel, text in creates.items():
        p = dst / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(text)
        subprocess.run(["gofmt", "-w", str(p)], capture_output=True)
        print(f"created {rel}")

    # gofmt sanity
    bad = subprocess.run(["gofmt", "-l"] + [str(dst / r) for r in stubbed],
                         capture_output=True, text=True).stdout.strip()
    if bad:
        print(f"GOFMT-BAD: {bad}")

    # compile check (main module only; integration_tests is a separate module)
    r = subprocess.run(["go", "build", "./..."], cwd=dst, capture_output=True, text=True)
    compiles = r.returncode == 0
    if not compiles:
        print("BUILD-FAIL:")
        print(r.stderr[:4000])

    # test-file compile check for integration_tests module if touched
    integ = [t for t in trunc if t.startswith("integration_tests/")]
    if integ:
        r2 = subprocess.run(["go", "vet", "./..."], cwd=dst / "integration_tests",
                            capture_output=True, text=True)
        if r2.returncode != 0:
            print("INTEG-VET-FAIL (first 30 lines):")
            print("\n".join(r2.stderr.splitlines()[:30]))
        else:
            print("integration_tests vet: ok")

    outdir = UNITS / unit / "_author"
    (outdir / "excised").mkdir(parents=True, exist_ok=True)

    # excision.patch: prod hunks first, then truncated-test hunks
    exc = ""
    for rel in stubbed:
        exc += diff_file(PRISTINE / rel, dst / rel, rel)
    for rel in trunc:
        exc += diff_file(PRISTINE / rel, dst / rel, rel)
    for rel in appends:
        exc += diff_file(PRISTINE / rel, dst / rel, rel)
    for rel in creates:
        exc += new_file_diff(dst / rel, rel)
    (outdir / "excised/excision.patch").write_text(exc)

    # gold.patch: stubbed -> pristine, prod files only
    gold = ""
    for rel in stubbed:
        gold += diff_file(dst / rel, PRISTINE / rel, rel)
    (outdir / "gold.patch").write_text(gold)

    if all_missing:
        print(f"MISSING: {all_missing}")
    print(f"compiles: {compiles}")
    print(f"wrote {outdir}")


if __name__ == "__main__":
    main()
