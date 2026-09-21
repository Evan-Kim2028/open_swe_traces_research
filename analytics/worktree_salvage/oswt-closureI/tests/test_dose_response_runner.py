"""Dose-response runner: dry-run planning, resume, backoff. No Harbor, no docker."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path

import pytest

from openswe_traces.synth import dose_response as dr


def _task(tmp_path: Path, name: str) -> Path:
    d = tmp_path / name
    (d / "environment" / "src").mkdir(parents=True)
    (d / "tests").mkdir()
    (d / "task.toml").write_text('[agent]\nnetwork_mode = "allowlist"\n', encoding="utf-8")
    return d


_TRIAL_SEQ = iter(range(1000))


def _trial_dir(job_dir: Path, name: str, reward: float) -> Path:
    t = job_dir / f"{name}__t{next(_TRIAL_SEQ)}"
    (t / "agent").mkdir(parents=True)
    (t / "agent" / "stdout.txt").write_text("done", encoding="utf-8")
    (t / "result.json").write_text(
        json.dumps(
            {
                "verifier_result": {"rewards": {"reward": reward}},
                "started_at": "2026-09-19T00:00:00",
                "finished_at": "2026-09-19T00:10:00",
            }
        ),
        encoding="utf-8",
    )
    return t


def test_task_unit_level(tmp_path: Path) -> None:
    task = _task(tmp_path, "store-s11-r020-L2")
    assert dr.task_unit_level(task) == ("store-s11-r020", 2)
    with pytest.raises(ValueError):
        dr.task_unit_level(_task(tmp_path, "noversion"))


def test_plan_jobs_resume_and_limit(tmp_path: Path) -> None:
    dose = tmp_path / "dose"
    t1 = _task(dose / "tasks", "store-s11-r020-L0")
    t2 = _task(dose / "tasks", "store-s11-r020-L2")
    done = {("store-s11-r020", 0, "openrouter/m"): 5}  # L0 already complete
    jobs = dr.plan_jobs([t1, t2], ["openrouter/m"], 5, done, dose_root=dose)
    assert [j.n_missing for j in jobs] == [5]
    assert jobs[0].level == 2 and jobs[0].arm == "1"

    jobs = dr.plan_jobs([t1, t2], ["openrouter/m"], 5, {}, dose_root=dose, limit=1)
    assert len(jobs) == 1 and jobs[0].n_missing == 1

    jobs = dr.plan_jobs([t1, t2], ["m1", "m2"], 5, {}, dose_root=dose, limit=7)
    assert [j.n_missing for j in jobs] == [5, 2]


def test_plan_jobs_arm2(tmp_path: Path) -> None:
    task = _task(tmp_path / "elsewhere", "coalesce-L0")
    jobs = dr.plan_jobs([task], ["m"], 3, {}, dose_root=tmp_path / "dose")
    assert jobs[0].arm == "2"


def test_solver_for() -> None:
    assert dr.solver_for("mini-swe-agent", "openrouter/deepseek-v4-flash:free") == "openrouter"
    assert dr.solver_for("cursor-cli", "cursor/composer-2.5") == "cursor"
    assert dr.solver_for("devin", "swe-2-max") == "devin"


def test_classify_job(tmp_path: Path) -> None:
    ok = subprocess.CompletedProcess([], 0, stdout="done", stderr="")
    rl = subprocess.CompletedProcess([], 1, stdout="", stderr="HTTP 429 rate limit exceeded")
    err = subprocess.CompletedProcess([], 1, stdout="", stderr="boom")
    assert dr.classify_job(ok, 2) == "ok"
    assert dr.classify_job(ok, 0) == "no_trials"
    assert dr.classify_job(rl, 0) == "rate_limit"
    assert dr.classify_job(err, 0) == "error"


def test_fab_l0_instruction() -> None:
    bug = (
        "# Bug report\n\nIt panics.\n\nExpected: works.\nGot: panic.\n\n"
        "Reproduce with:\n\n```\n./repro.sh\n```\n\n"
        "Run it from the unit root directory (the parent of `_author/`). blah\n"
    )
    out = dr.fab_l0_instruction(bug)
    assert "go test -count=1 ./..." in out
    assert "./repro.sh" not in out
    assert "_author" not in out
    assert "It panics." in out


def test_record_trials_and_resume(tmp_path: Path) -> None:
    jobs_dir = tmp_path / "jobs"
    job_dir = jobs_dir / "u-L2-m-seq1"
    _trial_dir(job_dir, "store-s11-r020-L2", 1.0)
    _trial_dir(job_dir, "store-s11-r020-L2", 0.0)
    t = _trial_dir(job_dir, "store-s11-r020-L2", 1.0)
    job = dr.PlannedJob(tmp_path / "tasks" / "store-s11-r020-L2", "store-s11-r020", 2, "1", "m", 3)
    trials_path = tmp_path / "trials.parquet"
    done: dict[tuple[str, int, str], int] = {}
    n = dr.record_trials(job, dr.RunResult(job_dir=job_dir, status="ok", n_trials=3), trials_path, done=done)
    assert n == 3
    assert done[("store-s11-r020", 2, "m")] == 3
    import pandas as pd

    df = pd.read_parquet(trials_path)
    assert list(df.columns) == list(dr.TRIAL_FIELDS)
    assert df["reward"].tolist() == [1.0, 0.0, 1.0]
    assert df["attempt"].tolist() == [1, 2, 3]
    assert str(t) in df["job_dir"].tolist()

    # resume: (unit, level, model) now complete -> not re-planned
    done2 = dr.load_done_counts(trials_path)
    jobs = dr.plan_jobs([job.task_dir], ["m"], 3, done2, dose_root=tmp_path)
    assert jobs == []


def test_run_job_retries_on_rate_limit(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(dr, "assert_harbor_safe", lambda *a, **k: None)  # type: ignore[attr-defined]
    monkeypatch.setattr(dr, "apply_solver_network", lambda *a, **k: None)  # type: ignore[attr-defined]
    jobs_dir = tmp_path / "jobs"
    calls = []

    def fake_run(argv, **kw):
        calls.append(argv)
        job_dir = jobs_dir / argv[argv.index("--job-name") + 1]
        if len(calls) == 1:
            return subprocess.CompletedProcess(argv, 1, stdout="", stderr="429 too many requests")
        _trial_dir(job_dir, "x-L2", 1.0)
        return subprocess.CompletedProcess(argv, 0, stdout="ok", stderr="")

    job = dr.PlannedJob(tmp_path / "tasks" / "x-L2", "x", 2, "1", "openrouter/m", 1)
    res = dr.run_job(
        job,
        agent="mini-swe-agent",
        jobs_dir=jobs_dir,
        n_concurrent=1,
        env={},
        seq=1,
        run=fake_run,
        sleep=lambda s: None,
    )
    assert res.status == "ok"
    assert len(calls) == 2
    argv = calls[0]
    assert argv[:2] == ["harbor", "run"]
    assert "openrouter/m" in argv and "mini-swe-agent" in argv
