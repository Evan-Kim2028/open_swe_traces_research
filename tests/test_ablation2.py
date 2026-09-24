from __future__ import annotations

from pathlib import Path

import pytest

from openswe_traces.synth.ablation2 import (
    FILE_FILTER_INSTRUCTION,
    REVIVELIB_INSTRUCTION,
    ValidUnit,
    extract_go_func,
    hidden_tests_for,
    valid_units,
)
from openswe_traces.synth.affordance import with_no_web
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE


def test_valid_units_match_judge() -> None:
    units = valid_units()
    assert [(u.cond, u.name) for u in units] == [
        ("graph", "file-exclude-filter"),
        ("nograph", "revivelib-runner"),
        ("nograph", "file-filter"),
    ]
    assert all(u.family == f"{u.cond}-{u.name}" for u in units)
    # experiments/ablation_graph/repos/ is gitignored, so a fresh clone has no excision
    # patches to check; only assert they are all present when the round-1 tree is local.
    present = [u.excision.is_file() for u in units]
    if not any(present):
        pytest.skip("ablation_graph/repos not materialised on this host")
    assert all(present)


def test_l2_instructions_have_symptom_no_leaks() -> None:
    for unit in valid_units():
        text = with_no_web(unit.instruction)
        assert NO_WEB_CLAUSE in text
        assert "-run" not in unit.instruction
        chk = instruction_self_check(
            text,
            test_sh="go test -count=1 -timeout 15m ./x/",
            f2p_tests=unit.listed_tests,
            changed_symbols=unit.changed_symbols,
            changed_files=unit.changed_files,
            locality=2,
            packages=unit.packages,
        )
        assert chk["ok"], chk
        for name in unit.listed_tests:
            assert name not in unit.instruction


def test_extract_go_func() -> None:
    src = (
        "package p\n\nfunc Keep() {}\n\n"
        "func TestGetConfig(t *testing.T) {\n\tif true {\n\t\treturn\n\t}\n}\n\n"
        "func Other() {}\n"
    )
    got = extract_go_func(src, "func TestGetConfig(")
    assert got.startswith("func TestGetConfig(")
    assert "func Other" not in got
    assert "func Keep" not in got


def test_hidden_extracts_mixed_file(tmp_path: Path) -> None:
    src = tmp_path / "src"
    cfg = src / "config"
    cfg.mkdir(parents=True)
    (cfg / "config_test.go").write_text(
        "package config_test\n\n"
        'import (\n\t"testing"\n)\n\n'
        "func TestGetConfig(t *testing.T) {\n\tt.Log(\"exclude\")\n}\n\n"
        "func TestDefault(t *testing.T) {}\n",
        encoding="utf-8",
    )
    (src / "lint").mkdir()
    (src / "lint" / "filefilter_test.go").write_text(
        "package lint_test\n\nimport \"testing\"\n\nfunc TestFileFilter(t *testing.T) {}\n",
        encoding="utf-8",
    )
    (src / "test").mkdir()
    (src / "test" / "file_filter_test.go").write_text(
        "package test_test\n\nimport \"testing\"\n\n"
        "func TestFileExcludeFilterAtRuleLevel(t *testing.T) {}\n",
        encoding="utf-8",
    )
    unit = ValidUnit(
        cond="graph",
        name="file-exclude-filter",
        units_root=tmp_path,
        listed_tests=("TestFileFilter", "TestFileExcludeFilterAtRuleLevel", "TestGetConfig"),
        test_files=(
            "lint/filefilter_test.go",
            "test/file_filter_test.go",
            "config/config_test.go",
        ),
        packages=("lint", "test", "config"),
        changed_symbols=("ParseFileFilter",),
        changed_files=("filefilter.go",),
        instruction=FILE_FILTER_INSTRUCTION,
        one_liners={"config/getconfig_bb_test.go": "exclude compile"},
    )
    hidden = hidden_tests_for(unit, src)
    rels = {h.relpath: h for h in hidden}
    assert "lint/filefilter_test.go" in rels
    assert "test/file_filter_test.go" in rels
    assert "config/getconfig_bb_test.go" in rels
    assert "func TestGetConfig" in rels["config/getconfig_bb_test.go"].content
    leftover = (cfg / "config_test.go").read_text(encoding="utf-8")
    assert "func TestGetConfig" not in leftover
    assert "func TestDefault" in leftover
    assert '"strings"' not in leftover


def test_revivelib_instruction_constant_ok() -> None:
    assert "expected five findings" in REVIVELIB_INSTRUCTION
    assert "TestReviveLint" not in REVIVELIB_INSTRUCTION
    assert "ParseFileFilter" not in FILE_FILTER_INSTRUCTION
