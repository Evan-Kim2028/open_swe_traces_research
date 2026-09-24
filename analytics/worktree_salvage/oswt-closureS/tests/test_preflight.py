"""Preflight gate: language-agnostic classification + launch refusal."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path

import pytest

from openswe_traces.pipeline.preflight import (
    FAIL,
    INFRA,
    PASS,
    PreflightError,
    classify_run,
    detect_language,
    ensure_preflight,
    infra_evidence,
    preflight_task,
)


def test_classify_pass_fail_infra() -> None:
    assert classify_run(0, "ok pkg\nREWARD=1\n", "")[0] == PASS
    assert classify_run(1, "--- FAIL: TestX\nREWARD=0\n", "")[0] == FAIL
    assert classify_run(1, "panic: TODO\nREWARD=0\n", "", language="go")[0] == FAIL
    # infra: the runner never reached assertions
    v, ev = classify_run(1, "[setup failed] cannot find module providing package x\nREWARD=0\n", "", language="go")
    assert v == INFRA and "setup failed" in ev
    assert classify_run(1, "pkg/a.go:12:3: undefined: F\n", "", language="go")[0] == INFRA
    assert classify_run(1, "ModuleNotFoundError: No module named 'x'\n", "", language="python")[0] == INFRA
    assert classify_run(1, "Cannot find module 'x'\n", "", language="node")[0] == INFRA
    assert classify_run(1, "error[E0432]: unresolved import\n", "", language="rust")[0] == INFRA
    assert classify_run(1, "cannot find symbol\nBUILD FAILURE\n", "", language="java")[0] == INFRA
    assert classify_run(124, "", "", timed_out=True)[0] == INFRA
    assert classify_run(125, "", "docker: error")[0] == INFRA
    assert classify_run(0, "all green\n", "")[0] == PASS
    # nonzero exit, no verdict line, no failure markers: infra, not a capability fail
    assert classify_run(2, "", "weird crash\n")[0] == INFRA


def test_classify_sibling_no_tests_does_not_mask_fail() -> None:
    # helm/storage: `go test ./pkg/storage/...` fails on pkg/storage (panic) while
    # sibling pkg/storage/driver reports "no tests to run" — that's a FAIL, not infra.
    out = (
        "FAIL\texample.internal/helm/pkg/storage\t0.013s\n"
        "ok  \texample.internal/helm/pkg/storage/driver\t0.014s [no tests to run]\n"
        "FAIL\n"
    )
    assert classify_run(1, out, "", language="go")[0] == FAIL
    # but a genuinely empty suite (no fail markers anywhere) is infra
    out2 = "ok  \texample.internal/helm/pkg/storage/driver\t0.014s [no tests to run]\n"
    assert classify_run(0, out2, "", language="go")[0] == INFRA
    # and a build failure still beats a bare FAIL line
    out3 = "FAIL\texample.internal/helm/pkg/x\t[build failed]\n"
    assert classify_run(1, out3, "", language="go")[0] == INFRA


def test_detect_language(tmp_path: Path) -> None:
    td = tmp_path / "t"
    (td / "environment" / "src").mkdir(parents=True)
    (td / "environment" / "src" / "go.mod").write_text("module example.internal/x\n")
    assert detect_language(td) == "go"
    td2 = tmp_path / "t2"
    (td2 / "tests" / "hidden").mkdir(parents=True)
    (td2 / "tests" / "hidden" / "x_test.py").write_text("def test_x(): pass\n")
    assert detect_language(td2) == "python"


def test_infra_evidence_union() -> None:
    assert infra_evidence("ImportError: no module", language=None)
    assert infra_evidence("go: downloading module x", language=None)
    assert infra_evidence("--- FAIL: TestX", language=None) is None


def _task_dir(tmp: Path) -> Path:
    td = tmp / "u1-L0"
    (td / "environment").mkdir(parents=True)
    (td / "environment" / "Dockerfile").write_text("FROM scratch\n")
    (td / "environment" / "src").mkdir()
    (td / "environment" / "src" / "go.mod").write_text("module example.internal/helm\n")
    (td / "tests").mkdir()
    (td / "tests" / "test.sh").write_text("#!/bin/bash\necho hi\n")
    (td / "tests" / "gold.patch").write_text("diff --git a/x b/x\n# GOLD\n")
    (td / "tests" / "cheat.patch").write_text("diff --git a/x b/x\n# CHEAT\n")
    (td / "task.toml").write_text('[task]\nname = "u1-L0"\n[verifier]\ntimeout_sec = 60\n')
    return td


def _fake_docker(run_log: list, *, bare_out: str, gold_out: str, cheat_out: str):
    def run(argv, *, input_text=None, timeout=None):
        run_log.append(list(argv))
        if argv[:3] == ["docker", "image", "inspect"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:img123\n", "")
        if argv[:2] == ["docker", "build"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:img123\n", "")
        assert argv[:2] == ["docker", "run"]
        if input_text and "GOLD" in input_text:
            return subprocess.CompletedProcess(argv, 0, gold_out, "")
        if input_text and "CHEAT" in input_text:
            return subprocess.CompletedProcess(argv, 1, cheat_out, "")
        return subprocess.CompletedProcess(argv, 1, bare_out, "")

    return run


def test_preflight_task_writes_verdicts(tmp_path: Path) -> None:
    td = _task_dir(tmp_path)
    run_log: list = []
    fake = _fake_docker(
        run_log,
        bare_out="--- FAIL: TestF\npanic: excised\nREWARD=0\n",
        gold_out="ok pkg\nREWARD=1\n",
        cheat_out="--- FAIL: TestF\nREWARD=0\n",
    )
    report = preflight_task(td, docker_run=fake, image="img:test")
    assert report.verdict == PASS
    data = json.loads((td / "validation.json").read_text())
    rows = {r["rule_id"]: r for r in data["rule_verdicts"]}
    assert rows["A8"]["passed"] is True and "panic" in rows["A8"]["evidence"]
    assert rows["A1"]["passed"] is True
    assert rows["A3"]["passed"] is True
    assert data["checks"]["buggy_fails"] is True
    assert data["preflight"]["verdict"] == "pass"


def test_preflight_refuses_infra_bare(tmp_path: Path) -> None:
    td = _task_dir(tmp_path)
    fake = _fake_docker(
        [],
        bare_out="[setup failed] cannot find module providing package example.internal/chartkit/v4/pkg/a\nREWARD=0\n",
        gold_out="ok\nREWARD=1\n",
        cheat_out="REWARD=0\n",
    )
    with pytest.raises(PreflightError, match="preflight not PASS"):
        ensure_preflight(td, docker_run=fake, image="img:test")
    data = json.loads((td / "validation.json").read_text())
    rows = {r["rule_id"]: r for r in data["rule_verdicts"]}
    assert rows["A8"]["passed"] is False
    assert "[setup failed]" in rows["A8"]["evidence"]


def test_preflight_cache_is_free(tmp_path: Path) -> None:
    td = _task_dir(tmp_path)
    run_log: list = []
    fake = _fake_docker(
        run_log,
        bare_out="panic: excised\nREWARD=0\n",
        gold_out="ok\nREWARD=1\n",
        cheat_out="REWARD=0\n",
    )
    ensure_preflight(td, docker_run=fake, image="img:test")
    n_runs = sum(1 for a in run_log if a[:2] == ["docker", "run"])
    # bare x2 + gold x2 (determinism) + cheat x1
    assert n_runs == 5
    run_log.clear()
    # unchanged image/tests/tree: verdict replays without a docker run
    report = ensure_preflight(td, docker_run=fake, image="img:test")
    assert report.cached is True and report.verdict == PASS
    assert not [a for a in run_log if a[:2] == ["docker", "run"]]
    # a changed tests/ dir re-proves
    (td / "tests" / "test.sh").write_text("#!/bin/bash\necho changed\n")
    ensure_preflight(td, docker_run=fake, image="img:test")
    assert [a for a in run_log if a[:2] == ["docker", "run"]]


def test_preflight_refuses_when_gold_fails(tmp_path: Path) -> None:
    td = _task_dir(tmp_path)
    fake = _fake_docker(
        [],
        bare_out="panic: excised\nREWARD=0\n",
        gold_out="--- FAIL: TestF\nREWARD=0\n",  # gold does not restore: packaging bug
        cheat_out="REWARD=0\n",
    )
    with pytest.raises(PreflightError):
        ensure_preflight(td, docker_run=fake, image="img:test")
    data = json.loads((td / "validation.json").read_text())
    rows = {r["rule_id"]: r for r in data["rule_verdicts"]}
    assert rows["A1"]["passed"] is False
