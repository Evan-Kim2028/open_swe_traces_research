from __future__ import annotations

import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.pipeline_ext.hack_audit import (
    AUDIT_HIDDEN_SEED,
    DEFAULT_HIDDEN_SEED,
    HIDDEN_SEED_ENV,
    allowlist_violations,
    audit_passing_attempt,
    extract_seed_constant,
    hidden_seed_go_snippet,
    parse_touched_paths,
    rewrite_seed,
    scan_trajectory,
)

FIXTURES = Path(__file__).parent / "fixtures" / "pipeline_ext"


def test_seed_contract() -> None:
    src = (FIXTURES / "hidden_seed.go").read_text()
    assert extract_seed_constant(src) == DEFAULT_HIDDEN_SEED
    assert HIDDEN_SEED_ENV in src
    rewritten = rewrite_seed(src, AUDIT_HIDDEN_SEED)
    assert str(AUDIT_HIDDEN_SEED) in rewritten
    assert "HIDDEN_SEED" in hidden_seed_go_snippet()


def test_allowlist_good_patch() -> None:
    patch = (FIXTURES / "good.patch").read_text()
    assert parse_touched_paths(patch) == ("internal/apicodec/codec.go",)
    assert allowlist_violations(patch) == []


def test_allowlist_forbidden() -> None:
    patch = (FIXTURES / "bad.patch").read_text()
    hits = allowlist_violations(patch)
    assert any("_test.go" in h for h in hits)
    assert any("test.sh" in h or "tests/" in h for h in hits)
    assert any("build tags" in h for h in hits)


def test_constant_leakage_is_flag_not_fail() -> None:
    patch = (FIXTURES / "leak.patch").read_text()
    contract = (FIXTURES / "contract.md").read_text()
    v = audit_passing_attempt(
        patch=patch,
        contract_md=contract,
        skip_docker=True,
    )
    assert v.passed is True
    assert v.hard_fails == ()
    assert any("constant leakage" in f for f in v.flags)


def test_trajectory_web_is_hard_fail() -> None:
    text = (FIXTURES / "trajectory_web.txt").read_text()
    hard, flags = scan_trajectory(text)
    assert any("web-tool" in h for h in hard)
    assert any("network command" in h for h in hard)
    assert any("oracle read" in h for h in hard)
    assert any("git history" in f for f in flags)


def test_audit_passing_attempt_hard_fail_on_test_edit() -> None:
    v = audit_passing_attempt(
        patch=(FIXTURES / "bad.patch").read_text(),
        skip_docker=True,
    )
    assert v.passed is False
    assert v.hard_fails


def test_docker_mock_seed_and_collateral(tmp_path: Path) -> None:
    seen: dict[str, object] = {}

    def fake_run(argv, *, input_text=None, timeout=900):  # type: ignore[no-untyped-def]
        seen["argv"] = argv
        seen["input"] = input_text
        seen["timeout"] = timeout
        cmd = argv[-1]
        assert f"{HIDDEN_SEED_ENV}={AUDIT_HIDDEN_SEED}" in " ".join(argv)
        assert "--network=none" in argv
        assert "-i" in argv
        assert "COLLATERAL" in cmd or "go test" in cmd
        return subprocess.CompletedProcess(argv, 0, stdout="REWARD=1\n", stderr="")

    v = audit_passing_attempt(
        image="ladder-base:test",
        patch=(FIXTURES / "good.patch").read_text(),
        hidden_files={"retry/backoff_prop_test.go": (FIXTURES / "hidden_seed.go").read_text()},
        unit_packages=("./retry/",),
        baseline_packages=("./mathx/",),
        docker_run=fake_run,
        skip_docker=False,
    )
    assert v.passed is True
    assert seen["input"].startswith("diff --git")
    assert v.evidence["docker"]["passed"] is True


def test_docker_hidden_fail() -> None:
    def fake_run(argv, *, input_text=None, timeout=900):  # type: ignore[no-untyped-def]
        return subprocess.CompletedProcess(argv, 1, stdout="REWARD=0\n", stderr="FAIL")

    v = audit_passing_attempt(
        image="ladder-base:test",
        patch=(FIXTURES / "good.patch").read_text(),
        docker_run=fake_run,
        skip_docker=False,
    )
    assert v.passed is False
    assert any("HIDDEN_SEED" in h for h in v.hard_fails)


def test_real_harbor_git_probe() -> None:
    trial = ROOT / "experiments/harbor_nex/jobs/bigL0/keyspacecodec-obf-L2__UxLiR3B"
    if not (trial / "agent").is_dir():
        pytest.skip("harbor trial missing")
    v = audit_passing_attempt(trial_dir=trial, skip_docker=True)
    assert any("git history" in f for f in v.flags)
