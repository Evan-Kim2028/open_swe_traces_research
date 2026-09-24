"""Excision reconstruction from committed author artifacts."""

from __future__ import annotations

import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.synth.excision_recover import (
    ExcisionRecoveryError,
    covered_test_names,
    reconstruct_excision,
    strip_covered_tests,
    strip_test_func,
)

CONTRACT = """# Contract (L2) — widget

Prose.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestWidgetDefault` | the dispatch table |
| `TestWidgetValidate` | the validate hook |
| `TestWidgetDefault` | duplicate row |
"""

RESTORED = """package widget

func Default(x int) int {
\treturn x + 1
}
"""

EXCISED = """package widget

func Default(x int) int {
\tpanic("excised: Default")
}
"""

TESTS = """package widget

import "testing"

func TestWidgetDefault(t *testing.T) {
\tif Default(1) != 2 {
\t\tt.Fatal("bad")
\t}
}

func TestOther(t *testing.T) {
\t_ = Default(0)
}

func TestWidgetValidate(t *testing.T) {
\tt.Log("hi")
}
"""


def _git(args: list[str], cwd: Path) -> None:
    subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True, text=True)


def test_covered_test_names_parses_and_dedupes() -> None:
    assert covered_test_names(CONTRACT) == ("TestWidgetDefault", "TestWidgetValidate")


def test_covered_test_names_without_table() -> None:
    assert covered_test_names("# Contract\n\nno table here\n") == ()


def test_strip_test_func_removes_one_block() -> None:
    out, hit = strip_test_func(TESTS, "TestWidgetDefault")
    assert hit
    assert "func TestWidgetDefault" not in out
    assert "func TestOther" in out
    assert "func TestWidgetValidate" in out


def test_strip_test_func_misses_absent_name() -> None:
    out, hit = strip_test_func(TESTS, "TestNope")
    assert not hit
    assert out == TESTS


def test_strip_covered_tests_reports_relpaths(tmp_path: Path) -> None:
    (tmp_path / "pkg").mkdir()
    (tmp_path / "pkg" / "w_test.go").write_text(TESTS, encoding="utf-8")
    removed = strip_covered_tests(tmp_path, ("TestWidgetDefault", "TestMissing"))
    assert removed == {"TestWidgetDefault": "pkg/w_test.go"}


@pytest.fixture
def unit(tmp_path: Path) -> tuple[Path, Path]:
    """An upstream tree plus an author dir holding gold.patch and contract.md."""
    upstream = tmp_path / "upstream"
    (upstream / "widget").mkdir(parents=True)
    (upstream / "go.mod").write_text("module example.internal/widget\n\ngo 1.23\n", encoding="utf-8")
    (upstream / "widget" / "widget.go").write_text(RESTORED, encoding="utf-8")
    (upstream / "widget" / "widget_test.go").write_text(TESTS, encoding="utf-8")

    # gold.patch = excised -> restored, produced the same way the author did.
    staging = tmp_path / "staging"
    shutil.copytree(upstream, staging)
    _git(["init", "-q"], staging)
    _git(["add", "-A"], staging)
    _git(["-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "base"], staging)
    (staging / "widget" / "widget.go").write_text(EXCISED, encoding="utf-8")
    diff = subprocess.run(
        ["git", "diff", "-R"], cwd=staging, capture_output=True, text=True, check=True
    )

    author = tmp_path / "authored" / "widget" / "_author"
    author.mkdir(parents=True)
    (author / "gold.patch").write_text(diff.stdout, encoding="utf-8")
    (author / "contract.md").write_text(CONTRACT, encoding="utf-8")
    return author, upstream


def test_reconstruct_excision_stubs_source_and_drops_covered_tests(
    unit: tuple[Path, Path], tmp_path: Path
) -> None:
    author, upstream = unit
    info = reconstruct_excision(author, upstream)

    patch = Path(str(info["patch"]))
    assert patch.is_file()
    assert info["removed_tests"] == {
        "TestWidgetDefault": "widget/widget_test.go",
        "TestWidgetValidate": "widget/widget_test.go",
    }
    assert info["missing_tests"] == []

    # Applying it to a fresh upstream copy yields the excised tree.
    applied = tmp_path / "applied"
    shutil.copytree(upstream, applied)
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch)],
        cwd=applied,
        capture_output=True,
        text=True,
        check=False,
    )
    assert proc.returncode in (0, 1), proc.stderr
    src = (applied / "widget" / "widget.go").read_text(encoding="utf-8")
    assert 'panic("excised: Default")' in src
    tests = (applied / "widget" / "widget_test.go").read_text(encoding="utf-8")
    assert "func TestWidgetDefault" not in tests
    assert "func TestWidgetValidate" not in tests
    assert "func TestOther" in tests


def test_reconstruct_excision_requires_artifacts(tmp_path: Path) -> None:
    author = tmp_path / "_author"
    author.mkdir()
    with pytest.raises(ExcisionRecoveryError, match="missing"):
        reconstruct_excision(author, tmp_path)


def test_reconstruct_excision_rejects_non_go_upstream(unit: tuple[Path, Path], tmp_path: Path) -> None:
    author, _ = unit
    with pytest.raises(ExcisionRecoveryError, match="no go.mod"):
        reconstruct_excision(author, tmp_path / "empty")
