from __future__ import annotations

from pathlib import Path

import pytest

from openswe_traces.synth.adversarial_contract import (
    LIES,
    apply_lie,
    dir_digest,
    package_one,
)
from openswe_traces.synth.contract_gold_audit import parse_coverage_rows, parse_hidden_tests


def test_each_lie_matches_source_instruction_once() -> None:
    for lie in LIES:
        text = (lie.src_dir / "instruction.md").read_text(encoding="utf-8")
        lied = apply_lie(text, lie)
        assert lie.prose_old not in lied
        assert lie.row_old not in lied
        assert lie.prose_new in lied
        assert lie.row_new in lied
        # everything else is untouched
        assert lied.replace(lie.prose_new, lie.prose_old).replace(
            lie.row_new, lie.row_old
        ) == text


def test_apply_lie_rejects_missing_needle() -> None:
    lie = LIES[0]
    with pytest.raises(ValueError, match="prose match count 0"):
        apply_lie("# not the contract\n", lie)


def test_dir_digest_content_not_mtime(tmp_path: Path) -> None:
    a = tmp_path / "a"
    b = tmp_path / "b"
    (a / "x").mkdir(parents=True)
    (b / "x").mkdir(parents=True)
    (a / "x" / "f.txt").write_text("hello\n")
    (b / "x" / "f.txt").write_text("hello\n")
    assert dir_digest(a) == dir_digest(b)
    (b / "x" / "f.txt").write_text("hello\nworld\n")
    assert dir_digest(a) != dir_digest(b)


def test_package_one_only_changes_instruction(tmp_path: Path) -> None:
    lie = LIES[0]
    dest_root = tmp_path / "sweep_lie"
    report = package_one(lie, dest_root=dest_root)
    assert report.hidden_match and report.tree_match
    assert report.diffs_vs_l2 == ["instruction.md"]
    dest = Path(report.dest)
    assert (dest / "instruction.md").read_text(encoding="utf-8") != (
        lie.src_dir / "instruction.md"
    ).read_text(encoding="utf-8")
    assert (dest / "tests" / "hidden").is_dir()
    assert (dest / "environment" / "src").is_dir()


def test_parse_coverage_and_hidden() -> None:
    instruction = (
        "# Contract\n\n"
        "## Coverage of original in-tree tests\n\n"
        "| original test | contract sentence |\n"
        "|---|---|\n"
        "| `TestFoo` | empty input is not an error |\n"
        "\nReproduce with:\n"
    )
    rows = parse_coverage_rows(instruction)
    assert rows[0].original_test == "TestFoo"
    tests = parse_hidden_tests(
        'func TestFoo(t *testing.T) {\n\tt.Fatal("empty input")\n}\n'
    )
    assert tests[0].name == "TestFoo"
    assert "empty input" in tests[0].fatals[0]
