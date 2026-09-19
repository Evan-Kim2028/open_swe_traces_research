"""Rebuild a unit's ``excision.patch`` from committed author artifacts.

``_author/excised/`` is gitignored, so a fresh clone has ``gold.patch`` and
``contract.md`` but not the excision that produced the task tree. Both halves of
the excision are recoverable:

* the stubbed source is ``gold.patch`` applied in reverse to the pinned upstream
  tree (gold is by construction the diff from excised to restored);
* the in-tree tests dropped under rule B3 are exactly the ones named in the
  contract's "Coverage of original in-tree tests" table, which is committed.

The rebuilt patch is a candidate, not an authority: the batch builder still runs
the image proof (A8) and re-evaluates the rule verdicts over whatever tree this
produces.
"""

from __future__ import annotations

import re
import shutil
import subprocess
import tempfile
from pathlib import Path

COVERAGE_ROW_RE = re.compile(r"^\|\s*`(?P<test>Test\w+)`\s*\|")
FUNC_RE_TMPL = r"^func {name}\(.*?\n\}}\n"


class ExcisionRecoveryError(RuntimeError):
    """Raised when the excision cannot be rebuilt from committed artifacts."""


def covered_test_names(contract_md: str) -> tuple[str, ...]:
    """Test names from the contract's coverage table, in order, deduplicated."""
    names: list[str] = []
    for line in contract_md.splitlines():
        m = COVERAGE_ROW_RE.match(line.strip())
        if m:
            name = m.group("test")
            if name not in names:
                names.append(name)
    return tuple(names)


def strip_test_func(source: str, name: str) -> tuple[str, bool]:
    """Drop a top-level ``func <name>(...)`` block from gofmt'd Go source."""
    pattern = re.compile(FUNC_RE_TMPL.format(name=re.escape(name)), re.DOTALL | re.MULTILINE)
    stripped, n = pattern.subn("", source, count=1)
    if not n:
        return source, False
    return stripped.replace("\n\n\n", "\n\n"), True


def strip_covered_tests(tree: Path, names: tuple[str, ...]) -> dict[str, str]:
    """Remove each named test function from the tree. Returns name -> relpath."""
    removed: dict[str, str] = {}
    for path in sorted(tree.rglob("*_test.go")):
        text = path.read_text(encoding="utf-8", errors="replace")
        changed = False
        for name in names:
            if name in removed:
                continue
            text, hit = strip_test_func(text, name)
            if hit:
                removed[name] = str(path.relative_to(tree))
                changed = True
        if changed:
            path.write_text(text, encoding="utf-8")
    return removed


def _run(args: list[str], cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True, check=False)


def reconstruct_excision(
    author_dir: Path,
    upstream: Path,
    *,
    write: bool = True,
) -> dict[str, object]:
    """Rebuild ``<author_dir>/excised/excision.patch`` against ``upstream``."""
    gold = author_dir / "gold.patch"
    contract = author_dir / "contract.md"
    for required in (gold, contract):
        if not required.is_file():
            raise ExcisionRecoveryError(f"missing {required}")
    if not (upstream / "go.mod").is_file():
        raise ExcisionRecoveryError(f"upstream tree has no go.mod: {upstream}")

    names = covered_test_names(contract.read_text(encoding="utf-8", errors="replace"))

    with tempfile.TemporaryDirectory(prefix="excision-") as tmp:
        work = Path(tmp) / "src"
        shutil.copytree(upstream, work, symlinks=True)
        rev = _run(
            ["patch", "-p1", "-R", "--forward", "--batch", "-i", str(gold.resolve())],
            cwd=work,
        )
        if rev.returncode != 0:
            raise ExcisionRecoveryError(
                f"reverse gold.patch failed for {author_dir.parent.name}: "
                f"{(rev.stderr or rev.stdout)[-2000:]}"
            )
        removed = strip_covered_tests(work, names)
        missing = [n for n in names if n not in removed]

        diff = _run(["diff", "-ruN", str(upstream), str(work)])
        if diff.returncode not in (0, 1):
            raise ExcisionRecoveryError(f"diff failed: {diff.stderr[-2000:]}")
        patch_text = diff.stdout.replace(f"{upstream}/", "a/").replace(f"{work}/", "b/")

    if not patch_text.strip():
        raise ExcisionRecoveryError(f"empty excision for {author_dir.parent.name}")

    out = author_dir / "excised" / "excision.patch"
    if write:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(patch_text, encoding="utf-8")
    return {
        "patch": str(out),
        "covered_tests": list(names),
        "removed_tests": removed,
        "missing_tests": missing,
        "bytes": len(patch_text),
    }
