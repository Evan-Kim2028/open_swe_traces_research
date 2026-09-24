"""Gate tests: tier precedence, merge semantics, and the new rules on fixtures."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path

import pytest

from openswe_traces.gate import (
    DOCUMENTARY,
    EXECUTED,
    STATIC,
    GateContext,
    GateError,
    Verdict,
    ensure_gate,
    evaluate_gate,
    gate_blockers,
    gate_task,
    record_verdicts,
)
from openswe_traces.gate.alt_fix import compare_patches, record_trial
from openswe_traces.gate.core import load_validation, merge_verdict_rows
from openswe_traces.gate.excision import excised_info
from openswe_traces.gate.static_rules import check_coverage, check_nesting
from openswe_traces.pipeline.preflight import PreflightError, ensure_preflight
from openswe_traces.pipeline.safety import TaskSafetyError
from openswe_traces.synth.rules import write_task_validation as doc_write


def _v(rid: str, tier: str, passed: bool, *, at: str = "2026-09-19T00:00:00+00:00", skipped=False):
    return Verdict(rid, passed, skipped, tier, f"{tier} evidence", at, "test")


def _rows(data: dict) -> dict[str, dict]:
    return {r["rule_id"]: r for r in data.get("rule_verdicts") or []}


# --- tier precedence ------------------------------------------------------------


def test_executed_beats_documentary(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    record_verdicts(td, [_v("A1", EXECUTED, True)])
    record_verdicts(td, [_v("A1", DOCUMENTARY, False, at="2026-09-19T01:00:00+00:00")])
    data = load_validation(td)
    prim = _rows(data)["A1"]
    assert prim["tier"] == EXECUTED and prim["passed"] is True
    secs = [r for r in data["rule_verdicts_secondary"] if r["rule_id"] == "A1"]
    assert len(secs) == 1 and secs[0]["tier"] == DOCUMENTARY


def test_static_beats_documentary_and_loses_to_executed(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    record_verdicts(td, [_v("B7", DOCUMENTARY, True)])
    record_verdicts(td, [_v("B7", STATIC, False, at="2026-09-19T01:00:00+00:00")])
    record_verdicts(td, [_v("B7", EXECUTED, True, at="2026-09-19T02:00:00+00:00")])
    data = load_validation(td)
    assert _rows(data)["B7"]["tier"] == EXECUTED
    tiers = {r["tier"] for r in data["rule_verdicts_secondary"] if r["rule_id"] == "B7"}
    assert tiers == {DOCUMENTARY, STATIC}


def test_legacy_rows_are_documentary(tmp_path: Path) -> None:
    """Rows written before tiers exist merge as documentary, never as primary
    over a fresh executed verdict."""
    td = tmp_path / "t-L0"
    td.mkdir()
    (td / "validation.json").write_text(
        json.dumps({"rule_verdicts": [{"rule_id": "A1", "passed": True, "evidence": "old"}]})
    )
    record_verdicts(td, [_v("A1", EXECUTED, False)])
    prim = _rows(load_validation(td))["A1"]
    assert prim["tier"] == EXECUTED and prim["passed"] is False


def test_same_tier_newest_wins(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    record_verdicts(td, [_v("A1", EXECUTED, False, at="2026-09-19T00:00:00+00:00")])
    record_verdicts(td, [_v("A1", EXECUTED, True, at="2026-09-19T03:00:00+00:00")])
    assert _rows(load_validation(td))["A1"]["passed"] is True


def test_documentary_write_cannot_overwrite_executed(tmp_path: Path) -> None:
    """The reported defect: rules.write_task_validation re-deriving A1 from
    notes must not replace an executed in-image verdict."""
    td = tmp_path / "t-L0"
    td.mkdir()
    (td / "instruction.md").write_text("instr\n")
    (td / "task.toml").write_text("[verifier]\nnetwork_mode = \"no-network\"\n")
    record_verdicts(td, [_v("A1", EXECUTED, True)])
    doc_write(td, {"checks": {}})
    prim = _rows(load_validation(td))["A1"]
    assert prim["tier"] == EXECUTED and prim["passed"] is True
    secs = [
        r
        for r in load_validation(td).get("rule_verdicts_secondary", [])
        if r["rule_id"] == "A1"
    ]
    assert any(r["tier"] == DOCUMENTARY for r in secs)


# --- launch decision ------------------------------------------------------------


def test_gate_blockers_missing_executed(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    # no verdicts at all: every executed-impl rule lacks an executed verdict
    blockers = gate_blockers(td)
    assert any("A1" in b for b in blockers)
    assert any("A8" in b for b in blockers)


def test_gate_blockers_failed_verdict(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    record_verdicts(td, [_v("B1", STATIC, False)])
    assert any("B1 failed" in b for b in gate_blockers(td))


def test_ensure_gate_raises_task_safety(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    with pytest.raises((GateError, TaskSafetyError)):
        ensure_gate(td)


def test_skipped_does_not_fail(tmp_path: Path) -> None:
    td = tmp_path / "t-L0"
    td.mkdir()
    record_verdicts(td, [_v("A2", EXECUTED, False, skipped=True)])
    assert not any("A2" in b for b in gate_blockers(td))


# --- fixtures for rule impls ------------------------------------------------------


def _task(root: Path, name: str = "u1-L0", *, excised: str = "SecretFunc") -> Path:
    td = root / name
    (td / "environment" / "src" / "pkg").mkdir(parents=True)
    (td / "environment" / "src" / "go.mod").write_text("module example.internal/x\n")
    (td / "environment" / "src" / "pkg" / "a.go").write_text(
        f'package pkg\n\nfunc Broken() {{\n\tpanic("excised: {excised}")\n}}\n'
    )
    (td / "environment" / "Dockerfile").write_text("FROM scratch\n")
    (td / "tests" / "hidden").mkdir(parents=True)
    (td / "tests" / "hidden" / "a_test.go").write_text(
        "package pkg\n\n// property check: seeded\nfunc TestBroken_Works(t *testing.T) {}\n"
    )
    (td / "tests" / "test.sh").write_text(
        "#!/bin/bash\necho 'x  /app/pkg/a.go' | sha256sum -c --status || exit 1\n"
        "go test -run '^(TestBroken_Works)$' ./pkg/...\n"
    )
    (td / "tests" / "gold.patch").write_text(
        "# GOLD\n"
        "diff --git a/pkg/a.go b/pkg/a.go\n"
        "--- a/pkg/a.go\n+++ b/pkg/a.go\n@@ -2,5 +2,5 @@\n"
        f' func Broken() {{\n-\tpanic("excised: {excised}")\n+\t// fixed\n }}\n'
    )
    (td / "task.toml").write_text(
        '[verifier]\nnetwork_mode = "no-network"\n'
        '[agent]\nnetwork_mode = "allowlist"\nallowed_hosts = ["x.com"]\n'
    )
    return td


def test_excision_info_from_tree(tmp_path: Path) -> None:
    td = _task(tmp_path)
    info = excised_info(td)
    assert "SecretFunc" in info.symbols
    assert "pkg/a.go" in info.files
    assert ("a.go", 4) in info.removed_lines


def test_b7_leaks_symbol_file_line(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "instruction.md").write_text(
        "The bug is in pkg/a.go:4 — fix SecretFunc.\nDo not use web search.\n"
    )
    v = evaluate_gate(td)
    b7 = next(x for x in v if x.rule_id == "B7")
    assert b7.passed is False
    assert "symbol:SecretFunc" in b7.evidence
    assert "file:a.go" in b7.evidence or "file:pkg/a.go" in b7.evidence
    assert "line:pkg/a.go:4" in b7.evidence


def test_b7_clean_instruction(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "instruction.md").write_text(
        "The endpoint returns the wrong count.\nDo not use web search.\n"
    )
    v = evaluate_gate(td)
    b7 = next(x for x in v if x.rule_id == "B7")
    assert b7.passed is True, b7.evidence


def test_coverage_wiring(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "instruction.md").write_text("x\n")
    ctx = GateContext(task_dir=td)
    v = check_coverage(ctx)
    assert v.passed is True, v.evidence
    # remove the -run wiring: hidden test no longer wired
    (td / "tests" / "test.sh").write_text("#!/bin/bash\ngo test -run '^(TestOther)$' ./pkg/...\n")
    ctx2 = GateContext(task_dir=td)
    v2 = check_coverage(ctx2)
    assert v2.passed is False
    assert "TestBroken_Works" in v2.evidence


def test_coverage_table_gap(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "instruction.md").write_text(
        "| property | test |\n|---|---|\n| x | TestSomethingElse |\n"
    )
    ctx = GateContext(task_dir=td)
    v = check_coverage(ctx)
    assert v.passed is False
    assert "TestBroken_Works" in v.evidence


def test_nesting_monotone(tmp_path: Path) -> None:
    td = _task(tmp_path / "repo", "u1-L0")
    td2 = _task(tmp_path / "repo", "u1-L2")
    (td / "instruction.md").write_text("see `pkg` and `Broken`\n")
    (td2 / "instruction.md").write_text("see `pkg` and `Broken` and `helper.go`\n")
    v = check_nesting(GateContext(task_dir=td))
    assert v.passed is True, v.evidence
    # L2 drops a token L0 revealed
    (td2 / "instruction.md").write_text("see `pkg`\n")
    v2 = check_nesting(GateContext(task_dir=td))
    assert v2.passed is False
    assert "drops information" in v2.evidence
    # different hidden tests
    (td2 / "instruction.md").write_text("see `pkg` and `Broken`\n")
    (td2 / "tests" / "hidden" / "a_test.go").write_text("changed\n")
    v3 = check_nesting(GateContext(task_dir=td))
    assert v3.passed is False
    assert "differ across levels" in v3.evidence


# --- executed rules via a fake docker ------------------------------------------


def _fake_docker(run_log: list, *, bare_out: str, gold_out: str, cheat_out: str = ""):
    def run(argv, *, input_text=None, timeout=None):
        run_log.append(list(argv))
        if argv[:3] == ["docker", "image", "inspect"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:img1\n", "")
        if argv[:2] == ["docker", "build"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:img1\n", "")
        assert argv[:2] == ["docker", "run"]
        if input_text and "GOLD" in input_text:
            return subprocess.CompletedProcess(argv, 0, gold_out, "")
        if input_text and "CHEAT" in input_text:
            return subprocess.CompletedProcess(argv, 1, cheat_out or "REWARD=0\n", "")
        return subprocess.CompletedProcess(argv, 1, bare_out, "")

    return run


def test_executed_rules_full_pass(tmp_path: Path) -> None:
    td = _task(tmp_path)
    fake = _fake_docker(
        [], bare_out="panic: excised\nREWARD=0\n", gold_out="ok\nREWARD=1\n"
    )
    rep = gate_task(td, docker_run=fake)
    by = {v.rule_id: v for v in rep.primaries}
    assert by["A1"].tier == EXECUTED and by["A1"].passed
    assert by["A8"].tier == EXECUTED and by["A8"].passed
    assert by["A5"].tier == EXECUTED and by["A5"].passed
    assert by["A10"].tier == EXECUTED and by["A10"].passed
    assert by["A3"].skipped  # no cheat.patch in fixture
    assert rep.ok, rep.blockers


def test_a5_flake_fails(tmp_path: Path) -> None:
    td = _task(tmp_path)
    calls = {"n": 0}

    def fake(argv, *, input_text=None, timeout=None):
        if argv[:3] == ["docker", "image", "inspect"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:i\n", "")
        if argv[:2] == ["docker", "build"]:
            return subprocess.CompletedProcess(argv, 0, "sha256:i\n", "")
        if input_text and "GOLD" in input_text:
            calls["n"] += 1
            out = "ok\nREWARD=1\n" if calls["n"] == 1 else "panic\nREWARD=0\n"
            return subprocess.CompletedProcess(argv, 0 if calls["n"] == 1 else 1, out, "")
        return subprocess.CompletedProcess(argv, 1, "panic\nREWARD=0\n", "")

    rep = gate_task(td, docker_run=fake)
    by = {v.rule_id: v for v in rep.primaries}
    assert by["A5"].passed is False
    assert "nondeterministic" in by["A5"].evidence
    assert any("A5" in b for b in rep.blockers)


def test_seed_env_passed_second_run(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "tests" / "hidden" / "a_test.go").write_text(
        'package pkg\nimport "os"\nvar _ = os.Getenv("HIDDEN_SEED")\n'
        "func TestBroken_Works(t *testing.T) {}\n"
    )
    envs: list[list[str]] = []
    fake = _fake_docker(
        [], bare_out="panic\nREWARD=0\n", gold_out="ok\nREWARD=1\n"
    )

    def spy(argv, *, input_text=None, timeout=None):
        if argv[:2] == ["docker", "run"]:
            envs.append([a for a in argv if a.startswith("HIDDEN_SEED")])
        return fake(argv, input_text=input_text, timeout=timeout)

    gate_task(td, docker_run=spy)
    seeded = [e for e in envs if e]
    assert seeded and all("HIDDEN_SEED=" in e[0] for e in seeded)


def test_ensure_preflight_adapter(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "tests" / "cheat.patch").write_text("diff --git a/x b/x\n# CHEAT\n")
    fake = _fake_docker(
        [], bare_out="panic\nREWARD=0\n", gold_out="ok\nREWARD=1\n", cheat_out="REWARD=0\n"
    )
    rep = ensure_preflight(td, docker_run=fake, image="img:t")
    assert rep.verdict == "pass"
    # replay: no docker runs needed
    rep2 = ensure_preflight(td, docker_run=fake, image="img:t")
    assert rep2.cached and rep2.verdict == "pass"


def test_gate_refuses_bare_pass(tmp_path: Path) -> None:
    td = _task(tmp_path)
    fake = _fake_docker(
        [], bare_out="ok\nREWARD=1\n", gold_out="ok\nREWARD=1\n"
    )
    with pytest.raises(PreflightError):
        ensure_preflight(td, docker_run=fake, image="img:t")
    with pytest.raises(GateError):
        ensure_gate(td, docker_run=fake)


# --- A2 alt-fix ----------------------------------------------------------------


GOLD_PATCH = """diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -2,5 +2,7 @@
 func Broken() {
-\tpanic("excised: SecretFunc")
+\ttotal := 0
+\tfor _, x := range items {
+\t\ttotal += x
+\t}
+\treturn total
 }
"""

ALT_PATCH = """diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -2,5 +2,5 @@
 func Broken() {
-\tpanic("excised: SecretFunc")
+\tsum := foldl(add, 0, items)
+\treturn sum
 }
"""

SAME_PATCH = """diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -2,5 +2,7 @@
 func Broken() {
-\tpanic("excised: SecretFunc")
+\ttotal := 0
+\tfor _, x := range items {
+\t\ttotal += x
+\t}
+\treturn total
 }
"""


def test_compare_patches_alternative() -> None:
    cmp = compare_patches(ALT_PATCH, GOLD_PATCH)
    assert cmp["alternative"] is True
    assert cmp["jaccard"] < 1.0


def test_compare_patches_identical() -> None:
    assert compare_patches(SAME_PATCH, GOLD_PATCH)["alternative"] is False


def test_compare_patches_whitespace_only() -> None:
    ws = GOLD_PATCH.replace("\ttotal := 0", "\ttotal:=0\n")
    assert compare_patches(ws, GOLD_PATCH)["alternative"] is False


def test_record_trial_writes_a2(tmp_path: Path) -> None:
    td = _task(tmp_path)
    (td / "tests" / "gold.patch").write_text(GOLD_PATCH)
    trial = tmp_path / "trial1"
    trial.mkdir()
    (trial / "agent.patch").write_text(ALT_PATCH)
    (trial / "result.json").write_text(json.dumps({"reward": 1.0}))
    v = record_trial(td, trial, trial_id="t1")
    assert v is not None and v.passed and v.tier == EXECUTED
    assert "alternative correct fix accepted" in v.evidence
    rows = _rows(load_validation(td))
    assert rows["A2"]["tier"] == EXECUTED and rows["A2"]["passed"] is True


def test_record_trial_failed_trial_no_evidence(tmp_path: Path) -> None:
    td = _task(tmp_path)
    trial = tmp_path / "trial1"
    trial.mkdir()
    (trial / "agent.patch").write_text(ALT_PATCH)
    (trial / "result.json").write_text(json.dumps({"reward": 0.0}))
    assert record_trial(td, trial, reward=0.0) is None


def test_merge_rows_keeps_secondary_for_report() -> None:
    stored = [_v("A1", EXECUTED, True).to_dict()]
    new = [_v("A1", DOCUMENTARY, False, at="2026-09-19T05:00:00+00:00")]
    prim, sec = merge_verdict_rows(stored, new)
    assert prim[0]["tier"] == EXECUTED and prim[0]["passed"] is True
    assert sec[0]["tier"] == DOCUMENTARY
