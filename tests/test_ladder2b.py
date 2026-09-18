from __future__ import annotations

from pathlib import Path

import pytest

from openswe_traces.synth.affordance import (
    B4PackagingError,
    HiddenTest,
    assert_b4_pass,
    build_affordance_levels,
    render_unsolv_task_toml,
    with_no_web,
)
from openswe_traces.synth.ladder2 import (
    ONEPC_INSTRUCTION,
    _patch_touches_tests,
    alt_onepc,
    cheat_onepc,
    stub_onepc,
    unified_patch,
)
from openswe_traces.synth.ladder2b import (
    CHECK_ASYNC,
    CHECK_ONEPC,
    ensure_check_onepc_probe,
    onepc_bb_spec,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE
from openswe_traces.synth.rules import check_b4, load_context

GOLD_ONEPC = """package transaction

func (c *twoPhaseCommitter) checkAsyncCommit() bool {
	if c.txn.GetScope() != oracle.GlobalTxnScope {
		return false
	}
	return c.txn.enableAsyncCommit
}

func (c *twoPhaseCommitter) checkOnePC() bool {
	if c.txn.GetScope() != oracle.GlobalTxnScope {
		return false
	}
	return c.txn.enable1PC
}
"""


def test_onepc_bb_spec_and_hidden_is_blackbox() -> None:
    unit = onepc_bb_spec()
    assert unit.family == "property-1pc"
    assert unit.kind == "property"
    assert unit.entry == "SetEnable1PC"
    assert "go test" in unit.instruction
    assert "-run" not in unit.instruction
    for name, _sentence in unit.coverage:
        assert name.startswith("Test")
        assert name not in unit.instruction
    for sym in unit.changed_symbols:
        assert sym not in unit.instruction
    for fp in unit.changed_files:
        assert Path(fp).name.lower() not in unit.instruction.lower()
    assert NO_WEB_CLAUSE in with_no_web(unit.instruction)
    hidden = (
        Path(__file__).resolve().parents[1]
        / "src/openswe_traces/synth/testdata/ladder2"
        / unit.testdata_name
    )
    text = hidden.read_text(encoding="utf-8")
    assert "checkOnePC(" not in text
    assert "checkAsyncCommit(" not in text
    assert "CheckOnePC(" in text
    assert "CheckAsyncCommit(" in text
    assert "SetEnable1PC(" in text
    assert "NewTiKVTxn(" in text
    assert "onepcBBCases = 10000" in text
    assert "onepcBBSeed = 20260918" in text


def test_onepc_patches_skip_tests() -> None:
    buggy = stub_onepc(GOLD_ONEPC)
    gold_patch = unified_patch("txnkv/transaction/2pc.go", buggy, GOLD_ONEPC)
    cheat_patch = unified_patch("txnkv/transaction/2pc.go", buggy, cheat_onepc(GOLD_ONEPC))
    alt_patch = unified_patch("txnkv/transaction/2pc.go", buggy, alt_onepc(GOLD_ONEPC))
    for p in (gold_patch, cheat_patch, alt_patch):
        assert p
        assert not _patch_touches_tests(p)
        assert "_test.go" not in p


def test_ensure_check_onepc_probe(tmp_path: Path) -> None:
    probe = tmp_path / "txnkv" / "transaction" / "test_probe.go"
    probe.parent.mkdir(parents=True)
    probe.write_text("package transaction\n\n" + CHECK_ASYNC, encoding="utf-8")
    ensure_check_onepc_probe(tmp_path)
    text = probe.read_text(encoding="utf-8")
    assert "func (c CommitterProbe) CheckOnePC()" in text
    ensure_check_onepc_probe(tmp_path)
    assert probe.read_text(encoding="utf-8").count("CheckOnePC()") == 1
    assert CHECK_ONEPC.split("CheckOnePC")[0] in text or "CheckOnePC" in text


def test_assert_b4_pass_and_packaging_refuse(tmp_path: Path) -> None:
    task = tmp_path / "whitebox-A0"
    src = task / "environment" / "src" / "txn"
    src.mkdir(parents=True)
    (src / "dec.go").write_text("package txn\n\nfunc checkOnePC() bool { return true }\n")
    (task / "instruction.md").write_text(ONEPC_INSTRUCTION)
    (task / "task.toml").write_text(render_unsolv_task_toml())
    (task / "environment" / "Dockerfile").write_text("FROM golang:1.23\n")
    hidden_go = """package txn

import "testing"

func TestDecision(t *testing.T) {
	c := &twoPhaseCommitter{}
	if !c.checkOnePC() {
		t.Fatal("no")
	}
	if !c.checkAsyncCommit() {
		t.Fatal("no")
	}
}
"""
    dest = task / "tests" / "hidden" / "txn" / "dec_test.go"
    dest.parent.mkdir(parents=True)
    dest.write_text(hidden_go)
    hidden = [
        HiddenTest(
            relpath="txn/dec_test.go",
            content=hidden_go,
            one_liner="calls unexported decision helpers",
            test_names=("TestDecision",),
        )
    ]
    with pytest.raises(B4PackagingError):
        build_affordance_levels(
            task,
            hidden,
            dest_root=tmp_path / "levels",
            family="whitebox-dec",
            instruction_a0=ONEPC_INSTRUCTION,
            packages=("txn",),
        )
    ctx = load_context(task)
    v = check_b4(ctx)
    assert v.passed is False
    with pytest.raises(B4PackagingError):
        assert_b4_pass(task)


def test_instruction_is_l2() -> None:
    assert ONEPC_INSTRUCTION.startswith("# Missing behavior")
    assert "expected refuse" in ONEPC_INSTRUCTION.lower() or "expected refuse" in ONEPC_INSTRUCTION
