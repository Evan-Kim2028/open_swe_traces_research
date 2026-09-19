"""Fixture-job tests for Harbor results aggregation (C6 flips, audits, dashboard)."""

from __future__ import annotations

import json
from pathlib import Path

import pandas as pd
import pytest

from openswe_traces.results import (
    audit_trial,
    collect_jobs,
    flip_point_for,
    infer_repo,
    is_ssh_spec,
    parse_trial_name,
    read_token_cap,
    render_dashboard,
    results_dashboard,
    scan_job,
)


def _write_trial(
    job: Path,
    name: str,
    *,
    reward: float | None,
    agent_log: str = "",
    verifier_out: str = "",
    exception: dict | None = None,
    tokens_in: int | None = 100,
    tokens_out: int | None = 10,
    agent_name: str = "cursor-cli",
    model_name: str = "cursor/composer-2.5",
    started: str = "2026-09-18T10:00:00",
    finished: str = "2026-09-18T10:05:00",
    task_path: str = "experiments/harbor_nex/tasks/client-go-memget",
) -> Path:
    trial = job / name
    (trial / "agent").mkdir(parents=True)
    (trial / "verifier").mkdir(parents=True)
    (trial / "agent" / "cursor-cli.txt").write_text(agent_log)
    if verifier_out:
        (trial / "verifier" / "test-stdout.txt").write_text(verifier_out)
    if reward is not None:
        (trial / "verifier" / "reward.txt").write_text(str(int(reward)))
    result = {
        "task_name": name.split("__")[0],
        "task_id": {"path": task_path},
        "config": {"agent": {"name": agent_name, "model_name": model_name}},
        "agent_info": {"name": agent_name, "model_info": {"name": model_name}},
        "agent_result": {
            "n_input_tokens": tokens_in,
            "n_output_tokens": tokens_out,
        },
        "verifier_result": {"rewards": {"reward": reward}} if reward is not None else {},
        "exception_info": exception,
        "started_at": started,
        "finished_at": finished,
    }
    (trial / "result.json").write_text(json.dumps(result))
    (trial / "config.json").write_text(
        json.dumps({"agent": {"name": agent_name, "model_name": model_name}})
    )
    return trial


def _fixture_job(root: Path) -> Path:
    job = root / "jobs" / "fixture-composer"
    job.mkdir(parents=True)
    (job / "config.json").write_text(
        json.dumps(
            {
                "job_name": "fixture-composer",
                "agents": [{"name": "cursor-cli", "model_name": "cursor/composer-2.5"}],
            }
        )
    )
    (job / "result.json").write_text(json.dumps({"n_total_trials": 8}))
    # L1: 0/3 — not a flip. Older A-2 name maps to L0 and is unused here.
    for i, suffix in enumerate(("aaa", "bbb", "ccc")):
        _write_trial(
            job,
            f"client-go-memget-L1__{suffix}",
            reward=0.0,
            tokens_in=20 + i,
            tokens_out=2,
        )
    # L2: 2/3 — C6 flip.
    _write_trial(job, "client-go-memget-L2__p1", reward=1.0, tokens_in=50, tokens_out=5)
    _write_trial(job, "client-go-memget-L2__p2", reward=1.0, tokens_in=50, tokens_out=5)
    _write_trial(job, "client-go-memget-L2__f1", reward=0.0, tokens_in=50, tokens_out=5)
    # Older A0 name → L2 on a second unit, 3/3 pass so flip at L2.
    for suffix in ("x", "y", "z"):
        _write_trial(
            job,
            f"dynamic-pipeline-A0__{suffix}",
            reward=1.0,
            tokens_in=10,
            tokens_out=1,
            task_path="experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A0",
        )
    # Contamination examples (own units so they don't pollute memget rates).
    _write_trial(
        job,
        "client-go-web-L3__w1",
        reward=1.0,
        agent_log='{"webFetchToolCall": {"url": "https://example.com"}}',
        tokens_in=5,
        tokens_out=1,
    )
    _write_trial(
        job,
        "client-go-edit-L3__e1",
        reward=0.0,
        verifier_out="test file modified: hidden/foo_test.go\n",
        tokens_in=5,
        tokens_out=1,
    )
    _write_trial(
        job,
        "client-go-boom-L3__i1",
        reward=None,
        exception={"exception_type": "CancelledError", "exception_message": "killed"},
        tokens_in=None,
        tokens_out=None,
    )
    # Implicit L6 (no -L/-A suffix).
    _write_trial(job, "client-go-batchcmds__old", reward=1.0, tokens_in=8, tokens_out=1)
    return job


def test_parse_trial_name_l_and_a_mapping() -> None:
    l2 = parse_trial_name("spec-reimpl-bb-L2__abc")
    assert (l2.unit, l2.level) == ("spec-reimpl-bb", 2)
    a0 = parse_trial_name("dynamic-pipeline-A0__x")
    assert (a0.unit, a0.level) == ("dynamic-pipeline", 2)
    a_neg = parse_trial_name("dynamic-latch-A-2__y")
    assert (a_neg.unit, a_neg.level) == ("dynamic-latch", 0)
    implicit = parse_trial_name("client-go-batchcmds__NULhKkJ")
    assert (implicit.unit, implicit.level) == ("client-go-batchcmds", 6)


def test_infer_repo_and_ssh_spec() -> None:
    assert infer_repo("client-go-memget") == "client-go"
    assert infer_repo("dailycodingproblem-go-match") == "dailycodingproblem-go"
    assert infer_repo("dynamic-pipeline") == "client-go"
    assert infer_repo("nograph-file-filter") == "revive"
    assert is_ssh_spec("lake-vps:experiments/harbor_nex/jobs")
    assert is_ssh_spec("evan@lake-vps:/tmp/jobs")
    assert not is_ssh_spec("experiments/harbor_nex/jobs")
    assert not is_ssh_spec("/abs/path")


def test_c6_flip_requires_three_attempts() -> None:
    assert flip_point_for({2: (3, 2), 1: (3, 0)}) == "L2"
    assert flip_point_for({2: (2, 2)}) == ""
    assert flip_point_for({2: (3, 1)}) == ""
    assert flip_point_for({4: (3, 0), 5: (3, 2)}) == "L5"


def test_audit_classes_on_fixture(tmp_path: Path) -> None:
    job = _fixture_job(tmp_path)
    assert audit_trial(job / "client-go-web-L3__w1") == "contaminated"
    assert audit_trial(job / "client-go-edit-L3__e1") == "test-edit"
    assert audit_trial(job / "client-go-boom-L3__i1") == "infra"
    assert audit_trial(job / "client-go-memget-L2__p1") == "clean"


def test_aggregate_fixture_job(tmp_path: Path) -> None:
    job = _fixture_job(tmp_path)
    (tmp_path / "config.yaml").write_text("composer_token_cap: 10000\n")
    out = tmp_path / "out"
    df = results_dashboard([job.parent], out_dir=out, config_path=tmp_path / "config.yaml")
    assert (out / "results.parquet").is_file()
    assert (out / "results.md").is_file()
    loaded = pd.read_parquet(out / "results.parquet")
    assert set(loaded.columns) >= {
        "repo",
        "unit",
        "level",
        "solver",
        "attempts",
        "passes",
        "pass_rate",
        "flip_point",
        "audit_class",
        "tokens_in",
        "tokens_out",
        "wall_minutes",
        "job_name",
    }
    memget_l2 = df[(df["unit"] == "client-go-memget") & (df["level"] == "L2")].iloc[0]
    assert memget_l2["attempts"] == 3
    assert memget_l2["passes"] == 2
    assert memget_l2["pass_rate"] == pytest.approx(2 / 3)
    assert memget_l2["flip_point"] == "L2"
    memget_l1 = df[(df["unit"] == "client-go-memget") & (df["level"] == "L1")].iloc[0]
    assert memget_l1["passes"] == 0
    assert memget_l1["flip_point"] == "L2"
    pipe = df[df["unit"] == "dynamic-pipeline"].iloc[0]
    assert pipe["level"] == "L2"
    assert pipe["attempts"] == 3
    assert pipe["flip_point"] == "L2"
    web = df[df["unit"] == "client-go-web"].iloc[0]
    assert web["audit_class"] == "contaminated"
    edit = df[df["unit"] == "client-go-edit"].iloc[0]
    assert edit["audit_class"] == "test-edit"
    boom = df[df["unit"] == "client-go-boom"].iloc[0]
    assert boom["audit_class"] == "infra"
    assert boom["attempts"] == 0
    batch = df[df["unit"] == "client-go-batchcmds"].iloc[0]
    assert batch["level"] == "L6"
    md = (out / "results.md").read_text()
    assert "Flip-point histogram" in md
    assert "Contamination" in md
    assert "Composer tokens" in md
    assert "10000" in md
    assert "Per-repo" in md
    assert "client-go" in md


def test_collect_jobs_nested_and_token_cap(tmp_path: Path) -> None:
    job = _fixture_job(tmp_path)
    jobs = collect_jobs([job.parent])
    assert jobs == [job.resolve()]
    records = scan_job(job)
    assert records
    assert read_token_cap(tmp_path / "missing.yaml") == 1e9
    (tmp_path / "c.yaml").write_text("token_cap: 42\n")
    assert read_token_cap(tmp_path / "c.yaml") == 42.0
    empty = render_dashboard(pd.DataFrame(), token_cap=1e9)
    assert "tasks" in empty
