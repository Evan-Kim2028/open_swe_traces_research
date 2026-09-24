"""Repair authored patches so the excised tree compiles (build-clean bare).

Two defect classes, both found by the preflight gate's ``bare=infra`` verdict:

1. Excision left imports unused (``imported and not used``). K's recipe:
   blank each unused import in the excision patch (``-`` original line,
   ``+`` ``\t_ "path"``), and give gold the mirror restore hunk
   (``-`` blanked, ``+`` original) so the gold tree un-blanks them. The restore
   hunk sorts first in the file's gold section, so the authored gold hunks see
   the original lines again. Cheat applies to the still-blanked excised tree —
   any old-side line naming an original import is rewritten to the blanked
   form, scoped to that file's section (a no-op when cheat never references
   the import block, as in practice).
2. Excision declared ``func _keepExcisedImports()`` in several files of one
   package (redeclaration). The 2nd+ declarations are renamed with the file
   stem (``_keepExcisedImportsWrapDoer``) and the rename is mirrored through
   gold (``-`` side) and cheat (context side).

Build errors are detected by ``go build`` of the excised tree inside the repo's
base image — the compiler's own diagnostic, not a guess. Every amended patch is
re-applied with patch(1) ``--fuzz=0`` against a scratch tree before it is
written back, then propagated to the unit's packaged level dirs (``tests/`` and
``patches/`` copies of gold/cheat, fresh ``environment/src``).

Usage: uv run python scripts/blank_unused_imports.py <task-dir> [<task-dir>...]
"""

from __future__ import annotations

import argparse
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

from openswe_traces.pipeline.config import load_config
from openswe_traces.pipeline.materialize import (
    base_tree_for,
    materialize_task,
    rebrand_text,
)

_UNUSED_RE = re.compile(
    r'^(\S+\.go):(\d+):\d+:\s+"([^"]+)" imported (?:as \w+ )?and not used',
    re.MULTILINE,
)
_DIAG_RE = re.compile(r"^\S+\.go:\d+:\d+:", re.MULTILINE)
_HUNK_HDR = re.compile(r"^@@ -(\d+)")
_BLANKED_RE = re.compile(r'^(\s*)_ "([^"]+)"(.*)$')
_IMPORT_RE_TMPL = r'^(\s*)(?:\w+\s+)?"{}"(.*)$'
_KEEP_FUNC_RE = re.compile(r"^([ +-])func _keepExcisedImports\(\)")

GO_BUILD = (
    "rm -rf /app && cp -a /src /app && cd /app && "
    "go build -gcflags=all=-e ./... 2>&1"
)


def _apply_fuzz0(tree: Path, patch_text: str) -> tuple[int, str]:
    with tempfile.NamedTemporaryFile("w", suffix=".patch", delete=False) as tmp:
        tmp.write(patch_text)
        tmpname = tmp.name
    try:
        proc = subprocess.run(
            ["patch", "-p1", "--forward", "--batch", "--fuzz=0", "-i", tmpname],
            cwd=tree,
            capture_output=True,
            text=True,
            check=False,
        )
        return proc.returncode, (proc.stderr or proc.stdout)[-600:]
    finally:
        Path(tmpname).unlink(missing_ok=True)


def build_errors(tree: Path, image: str) -> tuple[dict[str, list[str]], list[str]]:
    """(unused imports per file, other diagnostics) from go build in-image."""
    proc = subprocess.run(
        ["docker", "run", "--rm", "--network=none", "-e", "GOPROXY=off",
         "-v", f"{tree}:/src", image, "bash", "-c", GO_BUILD],
        capture_output=True,
        text=True,
        timeout=900,
        check=False,
    )
    blob = proc.stdout + "\n" + proc.stderr
    unused: dict[str, list[str]] = {}
    for m in _UNUSED_RE.finditer(blob):
        unused.setdefault(m.group(1), []).append(m.group(3))
    other = [
        ln.strip()[:200]
        for ln in blob.splitlines()
        if _DIAG_RE.match(ln) and "imported and not used" not in ln
    ]
    return unused, other


def _find_import_line(file_lines: list[str], path: str) -> int | None:
    """Index of the file's import line for ``path`` (unique per file)."""
    rx = re.compile(_IMPORT_RE_TMPL.format(re.escape(path)))
    hits = [i for i, ln in enumerate(file_lines) if rx.match(ln)]
    return hits[0] if len(hits) == 1 else None


def _blanked_form(orig: str, path: str) -> str:
    """``\talias "path"`` -> ``\t_ "path"``, preserving indent and comments."""
    m = re.match(_IMPORT_RE_TMPL.format(re.escape(path)), orig)
    if not m:
        raise RuntimeError(f"not an import line for {path!r}: {orig!r}")
    return f'{m.group(1)}_ "{path}"{m.group(2)}'


def _change_hunk(file_lines: list[str], changes: dict[int, str], ctx: int = 3) -> str:
    """One hunk spanning all changed lines with ``ctx`` context lines around.

    ``changes`` maps 0-based file index to its new text; the old side is the
    file's own line. Ends never land on a changed line (GNU patch rejects).
    """
    idxs = sorted(changes)
    lo, hi = idxs[0], idxs[-1]
    c_lo = max(0, lo - ctx)
    c_hi = min(len(file_lines) - 1, hi + ctx)
    if c_lo in changes or c_hi in changes:
        raise RuntimeError("changed line at hunk boundary — widen the span")
    body: list[str] = []
    for i in range(c_lo, c_hi + 1):
        if i in changes:
            body.append(f"-{file_lines[i]}\n+{changes[i]}\n")
        else:
            body.append(f" {file_lines[i]}\n")
    n = c_hi - c_lo + 1
    return f"@@ -{c_lo + 1},{n} +{c_lo + 1},{n} @@\n" + "".join(body)


def _insert_sorted(patch_text: str, fname: str, hunk: str) -> str:
    """Insert ``hunk`` into ``fname``'s file section, hunks ascending by old-start.

    Appends a new ``--- a/F`` / ``+++ b/F`` section when the patch has none.
    """
    lines = patch_text.splitlines(keepends=True)
    plus = next(
        (i for i, ln in enumerate(lines)
         if ln.startswith("+++ ") and ln.split()[1].removeprefix("b/") == fname),
        None,
    )
    if plus is None:
        text = patch_text if patch_text.endswith("\n") else patch_text + "\n"
        return text + f"--- a/{fname}\n+++ b/{fname}\n" + hunk
    end = plus + 1
    while end < len(lines) and not lines[end].startswith("--- a/"):
        end += 1
    my_start = int(_HUNK_HDR.match(hunk).group(1))  # type: ignore[union-attr]
    pos = end
    for j in range(plus + 1, end):
        m = _HUNK_HDR.match(lines[j])
        if m and int(m.group(1)) > my_start:
            pos = j
            break
    return "".join(lines[:pos]) + hunk + "".join(lines[pos:])


def _file_sections(patch_text: str) -> list[tuple[str, int, int]]:
    """(file, section_start, section_end) line-index spans of each ---/+++ pair."""
    lines = patch_text.splitlines(keepends=True)
    heads = [
        i for i, ln in enumerate(lines)
        if ln.startswith(("--- a/", "--- /dev/null"))
    ]
    out = []
    for k, i in enumerate(heads):
        fname = lines[i].split()[1].removeprefix("a/")
        end = heads[k + 1] if k + 1 < len(heads) else len(lines)
        out.append((fname, i, end))
    return out


def _rewrite_old_side(patch_text: str, fname: str, orig: str, new: str) -> str:
    """In ``fname``'s section: context/removed lines equal to ``orig`` -> ``new``."""
    lines = patch_text.splitlines(keepends=True)
    for f, s, e in _file_sections(patch_text):
        if f != fname:
            continue
        for i in range(s, e):
            ln = lines[i]
            if ln[:1] in (" ", "-") and ln[1:].rstrip("\n") == orig:
                lines[i] = ln[0] + new + "\n"
    return "".join(lines)


def _unrebrand(text: str, rebrand: tuple[tuple[str, str], ...]) -> str:
    """Base-tree tokens -> authored tokens (reverse map, reversed order)."""
    return rebrand_text(text, tuple((r, lft) for lft, r in reversed(rebrand)))


def fix_unused_imports(
    cfg,
    base: Path,
    repo: str,
    unit: str,
    image: str,
    unused: dict[str, list[str]],
) -> list[str]:
    """Blank excision-orphaned imports; mirror the restore in gold and cheat."""
    author = cfg.authored_dir / repo / unit / "_author"
    exc_p = author / "excised" / "excision.patch"
    gold_p = author / "gold.patch"
    cheat_p = author / "cheat.patch"
    rebrand = cfg.repo(repo).rebrand
    rows: list[str] = []

    exc_text = exc_p.read_text(encoding="utf-8")

    work = Path(tempfile.mkdtemp(prefix=f"blank-{unit}-"))
    try:
        for f, paths in sorted(unused.items()):
            rows.append(f"{unit}: {f}: unused {', '.join(paths)}")

        # excision: one blank hunk per file, old side against the BASE file.
        exc_new = exc_text
        for f, paths in sorted(unused.items()):
            base_lines = (base / f).read_text(encoding="utf-8").splitlines()
            changes: dict[int, str] = {}
            for p in paths:
                i = _find_import_line(base_lines, p)
                if i is None:
                    raise RuntimeError(f"{unit}: {f}: no unique base import {p!r}")
                changes[i] = _blanked_form(base_lines[i], p)
            hunk = _change_hunk(base_lines, changes)
            exc_new = _insert_sorted(exc_new, f, _unrebrand(hunk, rebrand))
        exc_new_eff = rebrand_text(exc_new, rebrand)
        excised2 = work / "excised2"
        shutil.copytree(base, excised2, symlinks=True)
        rc, err = _apply_fuzz0(excised2, exc_new_eff)
        if rc != 0:
            raise RuntimeError(f"{unit}: blanked excision fails to apply: {err}")
        still, other2 = build_errors(excised2, image)
        if still or other2:
            raise RuntimeError(
                f"{unit}: excised still fails to build: {still} {other2[:3]}"
            )

        # gold: restore hunk per file, old side against the EXCISED file —
        # every import line the excision blanked (differing from base).
        exc_files = {f for f, _, _ in _file_sections(exc_new_eff)}
        gold_new = gold_p.read_text(encoding="utf-8") if gold_p.is_file() else ""
        for f in sorted(exc_files):
            exc_file = excised2 / f
            base_file = base / f
            if not exc_file.is_file() or not base_file.is_file():
                continue
            exc_lines = exc_file.read_text(encoding="utf-8").splitlines()
            base_lines = base_file.read_text(encoding="utf-8").splitlines()
            changes = {}
            for i, ln in enumerate(exc_lines):
                m = _BLANKED_RE.match(ln)
                if not m:
                    continue
                j = _find_import_line(base_lines, m.group(2))
                if j is None or base_lines[j] == ln:
                    continue
                changes[i] = base_lines[j]
            if not changes:
                continue
            hunk = _change_hunk(exc_lines, changes)
            gold_new = _insert_sorted(gold_new, f, _unrebrand(hunk, rebrand))
        if gold_new and gold_p.is_file():
            trial = work / "trial_gold"
            shutil.copytree(excised2, trial, symlinks=True)
            rc, err = _apply_fuzz0(trial, rebrand_text(gold_new, rebrand))
            if rc != 0:
                raise RuntimeError(f"{unit}: restored gold fails to apply: {err}")

        # cheat: old-side references to the original import lines -> blanked.
        cheat_new = ""
        if cheat_p.is_file():
            cheat_new = cheat_p.read_text(encoding="utf-8")
            for f, paths in sorted(unused.items()):
                base_lines = (base / f).read_text(encoding="utf-8").splitlines()
                for p in paths:
                    i = _find_import_line(base_lines, p)
                    if i is None:
                        continue
                    orig = _unrebrand(base_lines[i], rebrand)
                    blanked = _blanked_form(orig, _unrebrand(p, rebrand))
                    cheat_new = _rewrite_old_side(cheat_new, f, orig, blanked)
            trial = work / "trial_cheat"
            shutil.copytree(excised2, trial, symlinks=True)
            rc, err = _apply_fuzz0(trial, rebrand_text(cheat_new, rebrand))
            if rc != 0:
                raise RuntimeError(f"{unit}: cheat fails to apply: {err}")

        exc_p.write_text(exc_new, encoding="utf-8")
        if gold_new and gold_p.is_file():
            gold_p.write_text(gold_new, encoding="utf-8")
        if cheat_new and cheat_p.is_file():
            cheat_p.write_text(cheat_new, encoding="utf-8")
        rows.append(f"{unit}: excision+gold blank/restore written, cheat verified")
        rows.extend(_propagate(cfg, base, repo, unit))
        return rows
    finally:
        shutil.rmtree(work, ignore_errors=True)


def dedupe_keep_funcs(cfg, base: Path, repo: str, unit: str, image: str) -> list[str]:
    """Rename duplicate ``_keepExcisedImports`` decls within one package."""
    author = cfg.authored_dir / repo / unit / "_author"
    exc_p = author / "excised" / "excision.patch"
    gold_p = author / "gold.patch"
    cheat_p = author / "cheat.patch"
    rebrand = cfg.repo(repo).rebrand
    rows: list[str] = []

    exc_text = exc_p.read_text(encoding="utf-8")
    lines = exc_text.splitlines(keepends=True)
    seen: dict[str, int] = {}
    cur = ""
    renames: dict[str, str] = {}  # file -> new func name
    for i, ln in enumerate(lines):
        if ln.startswith("+++ "):
            cur = ln.split()[1].removeprefix("b/")
        m = _KEEP_FUNC_RE.match(ln)
        if m and m.group(1) == "+" and cur:
            pkg = cur.rsplit("/", 1)[0]
            seen[pkg] = seen.get(pkg, 0) + 1
            if seen[pkg] > 1:
                stem = cur.rsplit("/", 1)[1].removesuffix(".go")
                camel = "".join(w.title() for w in re.split(r"[_\-]+", stem))
                new_name = f"_keepExcisedImports{camel}"
                lines[i] = ln.replace("_keepExcisedImports()", f"{new_name}()")
                renames[cur] = new_name
    if not renames:
        rows.append(f"{unit}: no keep-func collisions")
        return rows
    exc_new = "".join(lines)

    def mirror(patch_text: str) -> str:
        plines = patch_text.splitlines(keepends=True)
        for f, s, e in _file_sections(patch_text):
            new_name = renames.get(f)
            if new_name is None:
                continue
            for i in range(s, e):
                if _KEEP_FUNC_RE.match(plines[i]):
                    plines[i] = plines[i].replace(
                        "_keepExcisedImports()", f"{new_name}()"
                    )
        return "".join(plines)

    gold_new = mirror(gold_p.read_text(encoding="utf-8")) if gold_p.is_file() else ""
    cheat_new = mirror(cheat_p.read_text(encoding="utf-8")) if cheat_p.is_file() else ""

    work = Path(tempfile.mkdtemp(prefix=f"dedupe-{unit}-"))
    try:
        excised = work / "excised"
        shutil.copytree(base, excised, symlinks=True)
        rc, err = _apply_fuzz0(excised, rebrand_text(exc_new, rebrand))
        if rc != 0:
            raise RuntimeError(f"{unit}: renamed excision fails to apply: {err}")
        unused, other = build_errors(excised, image)
        if unused or other:
            raise RuntimeError(
                f"{unit}: excised still fails to build: {unused} {other[:3]}"
            )
        for name, text in (("gold", gold_new), ("cheat", cheat_new)):
            if not text:
                continue
            trial = work / f"trial_{name}"
            shutil.copytree(excised, trial, symlinks=True)
            rc, err = _apply_fuzz0(trial, rebrand_text(text, rebrand))
            if rc != 0:
                raise RuntimeError(f"{unit}: renamed {name} fails to apply: {err}")
        exc_p.write_text(exc_new, encoding="utf-8")
        if gold_new and gold_p.is_file():
            gold_p.write_text(gold_new, encoding="utf-8")
        if cheat_new and cheat_p.is_file():
            cheat_p.write_text(cheat_new, encoding="utf-8")
        rows.append(f"{unit}: renamed keep-funcs in {sorted(renames)}")
        rows.extend(_propagate(cfg, base, repo, unit))
        return rows
    finally:
        shutil.rmtree(work, ignore_errors=True)


def _propagate(cfg, base: Path, repo: str, unit: str) -> list[str]:
    """Fresh environment/src + authored gold/cheat copies in every level dir.

    The authored copies land first; ``materialize_task`` then rebrands packaged
    artifacts (tests/, patches/) and rebuilds environment/src in one pass.
    """
    author = cfg.authored_dir / repo / unit / "_author"
    rows: list[str] = []
    for root in (cfg.tasks_dir, cfg.tasks_dir.parent / "tasks_composerver"):
        unit_root = root / repo
        if not unit_root.is_dir():
            continue
        for td in sorted(unit_root.iterdir()):
            if not td.is_dir() or not re.fullmatch(rf"{re.escape(unit)}-L\d+", td.name):
                continue
            for name in ("gold.patch", "cheat.patch"):
                src = author / name
                if not src.is_file():
                    continue
                for sub in ("tests", "patches"):
                    dest = td / sub / name
                    if dest.exists():
                        dest.write_bytes(src.read_bytes())
            materialize_task(cfg, base, repo, unit, td)
            rows.append(f"{unit}: rebuilt {td.name}")
    return rows


def fix_unit(cfg, repo: str, unit: str, image: str) -> list[str]:
    """Route to the repair for this unit's excised-tree build failure."""
    base = base_tree_for(cfg.repo(repo), cfg)
    author = cfg.authored_dir / repo / unit / "_author"
    exc_p = author / "excised" / "excision.patch"
    rebrand = cfg.repo(repo).rebrand
    work = Path(tempfile.mkdtemp(prefix=f"probe-{unit}-"))
    try:
        excised = work / "excised"
        shutil.copytree(base, excised, symlinks=True)
        rc, err = _apply_fuzz0(excised, rebrand_text(exc_p.read_text(encoding="utf-8"), rebrand))
        if rc != 0:
            return [f"{unit}: excision does not apply: {err[:160]}"]
        unused, other = build_errors(excised, image)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    if unused:
        return fix_unused_imports(cfg, base, repo, unit, image, unused)
    if any("redeclared" in o or "other declaration" in o for o in other):
        return dedupe_keep_funcs(cfg, base, repo, unit, image)
    if other:
        return [f"{unit}: unhandled build errors: {other[:4]}"]
    return [f"{unit}: excised tree builds clean"]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("task_dirs", nargs="+", type=Path)
    ap.add_argument("--config", default="experiments/pipeline/config.yaml")
    ap.add_argument("--repos", default="experiments/pipeline/repos.yaml")
    ap.add_argument(
        "--image",
        default=None,
        help="base image for build diagnostics (default: the task Dockerfile's FROM)",
    )
    args = ap.parse_args()
    cfg = load_config(args.config, args.repos)
    all_rows: list[str] = []
    for td in args.task_dirs:
        m = re.match(r"^(.+?)-L\d+$", td.name)
        if not m:
            print(f"skip {td}: not a level dir")
            continue
        unit = m.group(1)
        repo = td.parent.name
        image = args.image
        if image is None:
            dfile = td / "environment" / "Dockerfile"
            fm = re.search(r"^FROM\s+(\S+)", dfile.read_text(), re.MULTILINE) if dfile.is_file() else None
            image = fm.group(1) if fm else f"ladder-base:{repo}"
        print(f"== {repo}/{unit} ({td.name}) [{image}]", flush=True)
        all_rows.extend(fix_unit(cfg, repo, unit, image))
    print("\n".join(all_rows))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
