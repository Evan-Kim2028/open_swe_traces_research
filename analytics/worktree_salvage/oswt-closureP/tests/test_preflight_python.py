"""Python preflight classification: pytest collection errors are infra,
assertion failures are fail, and the two are never confused."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path

import pytest

from openswe_traces.pipeline.preflight import (
    FAIL,
    INFRA,
    PASS,
    classify_run,
    detect_language,
    ensure_preflight,
    preflight_task,
)


def test_pytest_collection_errors_are_infra() -> None:
    # ModuleNotFoundError at import time: the runner never reached assertions.
    out = (
        "============================= test session starts ==============================\n"
        "ERROR collecting tests/test_itsdangerous/test_signer_bb.py\n"
        "ImportError while importing test module 'test_signer_bb'.\n"
        "ModuleNotFoundError: No module named 'itsdangerous'\n"
    )
    v, ev = classify_run(1, out, "", language="python")
    assert v == INFRA
    assert "ModuleNotFoundError" in ev  # first matching infra pattern wins

    # Old-style bare import failure, no pytest wrapper.
    assert classify_run(1, "ModuleNotFoundError: No module named 'x'\n", "", language="python")[0] == INFRA
    # The ERRORS section header (fixture/setup errors) is infra too.
    assert (
        classify_run(1, "________________________ ERROR at setup ________________________\n", "", language="python")[0]
        == INFRA
    )
    # A pytest internal crash is never a test verdict.
    assert classify_run(1, "INTERNALERROR> ValueError: unknown log mark\n", "", language="python")[0] == INFRA
    assert classify_run(2, "Fatal Python error: Segmentation fault\n", "", language="python")[0] == INFRA


def test_pytest_assertion_failure_is_fail() -> None:
    # Realistic pytest -rf output for a failing hidden test.
    out = (
        "tests/test_itsdangerous/test_signer_bb.py FFFF.FF\n"
        "______________________ TestSignerProperties.test_roundtrip ______________________\n"
        "    def test_roundtrip(self):\n"
        "        signed = signer.sign(value)\n"
        ">       assert signer.unsign(signed) == value\n"
        "E       AssertionError: assert b'my string.' == b'my string.wh6tMHxLgJqB6oY1uT73iMlyrOA'\n"
        "FAILED tests/test_itsdangerous/test_signer_bb.py::TestSignerProperties::test_roundtrip - AssertionError\n"
        "========================= 1 failed, 1 passed in 0.12s =========================\n"
    )
    v, ev = classify_run(1, out, "", language="python")
    assert v == FAIL
    assert "AssertionError" in ev

    # Assertion failure with REWARD=0 (what test.sh writes) is still a fail.
    out2 = out + "REWARD=0\n"
    assert classify_run(1, out2, "", language="python")[0] == FAIL


def test_pytest_empty_suite_is_infra() -> None:
    # The python analogue of "no tests to run": pytest collected nothing.
    out = "============================= no tests ran in 0.01s =============================\n"
    assert classify_run(0, out, "", language="python")[0] == INFRA
    assert classify_run(5, "collected 0 items\n", "", language="python")[0] == INFRA


def test_sibling_no_tests_does_not_mask_python_fail() -> None:
    # A sibling module reporting "no tests ran" must not mask a real
    # assertion failure in the module under test.
    out = (
        "tests/test_itsdangerous/test_signer_bb.py FFFF\n"
        "FAILED tests/test_itsdangerous/test_signer_bb.py::test_x - AssertionError\n"
        "other_pkg/test_unrelated.py no tests ran\n"
    )
    assert classify_run(1, out, "", language="python")[0] == FAIL
    # But a genuinely empty run (no fail markers anywhere) is infra.
    assert classify_run(0, "no tests ran\n", "", language="python")[0] == INFRA


def test_detect_language_python_markers(tmp_path: Path) -> None:
    cases = ("pyproject.toml", "setup.py", "conftest.py", "pytest.ini")
    for marker in cases:
        td = tmp_path / marker.replace(".", "_")
        (td / "environment" / "src").mkdir(parents=True)
        (td / "environment" / "src" / marker).write_text("x\n")
        assert detect_language(td) == "python", marker
    td2 = tmp_path / "hidden_suffix"
    (td2 / "tests" / "hidden").mkdir(parents=True)
    (td2 / "tests" / "hidden" / "test_signer_bb.py").write_text("def test_x(): pass\n")
    assert detect_language(td2) == "python"


def _python_task_dir(tmp: Path) -> Path:
    td = tmp / "signer-L0"
    (td / "environment").mkdir(parents=True)
    (td / "environment" / "Dockerfile").write_text("FROM python:3.12-slim\n")
    (td / "environment" / "src").mkdir()
    (td / "environment" / "src" / "pyproject.toml").write_text("[project]\nname = 'x'\n")
    (td / "tests").mkdir()
    (td / "tests" / "test.sh").write_text("#!/bin/bash\necho hi\n")
    (td / "tests" / "gold.patch").write_text("diff --git a/x b/x\n# GOLD\n")
    (td / "tests" / "cheat.patch").write_text("diff --git a/x b/x\n# CHEAT\n")
    (td / "task.toml").write_text('[task]\nname = "signer-L0"\n[verifier]\ntimeout_sec = 60\n')
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


def test_preflight_python_bare_fail_gold_pass_cheat_fail(tmp_path: Path) -> None:
    td = _python_task_dir(tmp_path)
    run_log: list = []
    bare = (
        "tests/test_signer_bb.py FFF\n"
        "FAILED tests/test_signer_bb.py::test_roundtrip - AssertionError\n"
        "REWARD=0\n"
    )
    gold = "18 passed in 0.1s\nREWARD=1\n"
    cheat = (
        "FAILED tests/test_signer_bb.py::test_property - AssertionError\n"
        "REWARD=0\n"
    )
    report = preflight_task(td, docker_run=_fake_docker(run_log, bare_out=bare, gold_out=gold, cheat_out=cheat), image="img:test")
    assert report.verdict == PASS
    data = json.loads((td / "validation.json").read_text())
    rows = {r["rule_id"]: r for r in data["rule_verdicts"]}
    assert rows["A8"]["passed"] is True and "AssertionError" in rows["A8"]["evidence"]
    assert rows["A1"]["passed"] is True
    assert rows["A3"]["passed"] is True
    assert data["preflight"]["checks"] == {"bare": "fail", "gold": "pass", "cheat": "fail"}


def test_preflight_python_collection_error_refuses_launch(tmp_path: Path) -> None:
    td = _python_task_dir(tmp_path)
    fake = _fake_docker(
        [],
        bare_out="ERROR collecting tests/test_signer_bb.py\nModuleNotFoundError: No module named 'x'\nREWARD=0\n",
        gold_out="18 passed\nREWARD=1\n",
        cheat_out="REWARD=0\n",
    )
    with pytest.raises(Exception, match="preflight not PASS"):
        ensure_preflight(td, docker_run=fake, image="img:test")
    data = json.loads((td / "validation.json").read_text())
    rows = {r["rule_id"]: r for r in data["rule_verdicts"]}
    assert rows["A8"]["passed"] is False
    assert "ModuleNotFoundError" in rows["A8"]["evidence"]
