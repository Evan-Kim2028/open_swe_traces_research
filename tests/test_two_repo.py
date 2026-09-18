from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.two_repo import (
    parse_consumer_clientgo_calls,
    render_two_repo_dockerfile,
    render_two_repo_test_sh,
    two_repo_instruction,
)

CONSUMER_GO = """package app

import (
	"github.com/tikv/client-go/v2/tikv"
	txn "github.com/tikv/client-go/v2/txnkv/transaction"
)

func Open() {
	_, _ = tikv.NewTestTiKVStore(nil, nil, nil, nil, 0)
	_ = txn.NewTxn()
}
"""


def test_parse_consumer_clientgo_calls(tmp_path: Path) -> None:
    (tmp_path / "util_test.go").write_text(CONSUMER_GO)
    calls = parse_consumer_clientgo_calls(tmp_path)
    names = {(c.symbol, c.alias) for c in calls}
    assert ("NewTestTiKVStore", "tikv") in names
    assert ("NewTxn", "txn") in names


def test_two_repo_instruction_hides_fix_site() -> None:
    text = two_repo_instruction(
        ["TestOnePC/Test1PC"],
        "--- FAIL: TestOnePC/Test1PC (0.04s)\n"
        "        	Error:      	Should be true\n"
        "        	Error:      	Not equal:\n"
        "        	            	expected: 1\n"
        "        	            	actual  : 0\n",
        redact_terms=["checkOnePC", "2pc.go"],
    )
    assert "TestOnePC/Test1PC" in text
    assert "checkOnePC" not in text
    assert "2pc.go" not in text
    assert "diff --git" not in text
    assert "library" in text.lower()
    assert "checksum" in text.lower()
    check = instruction_self_check(
        text,
        test_sh=render_two_repo_test_sh(["TestOnePC/Test1PC"], consumer_digest="abc"),
        f2p_tests=["TestOnePC/Test1PC"],
        changed_symbols=["checkOnePC"],
        changed_files=["txnkv/transaction/2pc.go"],
        diff_hunk="-if c.txn.GetScope() != oracle.GlobalTxnScope {\n+if c.txn.GetScope() == oracle.GlobalTxnScope {",
    )
    assert check["ok"]
    assert check["leaked_symbols"] == []
    assert check["leaked_files"] == []


def test_render_two_repo_test_sh_checksum_and_ldflags() -> None:
    script = render_two_repo_test_sh(["TestOnePC/Test1PC", "TestOnePC/Test1PCIsolation"], consumer_digest="deadbeef")
    assert "expected='deadbeef'" in script
    assert "consumer test files were modified" in script
    assert "-ldflags=-checklinkname=0" in script
    assert "-run '^(TestOnePC)$'" in script
    assert "cd /app/integration_tests" in script


def test_render_two_repo_dockerfile_downloads_both_modules() -> None:
    df = render_two_repo_dockerfile()
    assert "FROM golang:1.23" in df
    assert "WORKDIR /app/integration_tests" in df
    assert df.count("go mod download") >= 2
