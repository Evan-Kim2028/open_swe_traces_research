from __future__ import annotations

import json
from pathlib import Path

from openswe_traces.pipeline.agents import AgentResult, AgentRunner
from openswe_traces.pipeline.aggregate import aggregate, render_dashboard, task_rows
from openswe_traces.pipeline.audit import audit_class, audit_trial_dir
from openswe_traces.pipeline.author import discover_units
from openswe_traces.pipeline.config import PipelineConfig, RepoSpec, load_config
from openswe_traces.pipeline.ladder import (
    affordance_level,
    attempts_for_level,
    flip_point,
    next_solve_levels,
)
from openswe_traces.pipeline.prepare import BaselineError, parse_baseline_output
from openswe_traces.pipeline.run import Runtime, dry_run, run_pipeline, status_text
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.solve import HarborJobResult, solve_unit
from openswe_traces.pipeline.state import DONE, PipelineStore
from openswe_traces.pipeline.tokens import (
    TokenBudget,
    parse_cursor_agent_output,
    parse_harbor_trial_tokens,
)
from openswe_traces.pipeline.yamlutil import load_yaml


def _cfg(tmp: Path, **kw) -> PipelineConfig:
    base = PipelineConfig(
        units_per_author_batch=5,
        author_minutes=1,
        composer_token_cap=kw.pop("composer_token_cap", 1_000_000),
        devin_slots=4,
        devin_slots_path=tmp / "devin.slots",
        state_db=tmp / "state.db",
        logs_dir=tmp / "logs",
        work_dir=tmp / "work",
        authored_dir=tmp / "authored",
        tasks_dir=tmp / "tasks",
        jobs_dir=tmp / "jobs",
        results_parquet=tmp / "results.parquet",
        results_md=tmp / "results.md",
        repos=(
            RepoSpec(
                name="mathx",
                url="https://example.invalid/mathx.git",
                commit="deadbeef",
                language="go",
            ),
        ),
        **kw,
    )
    return base


def _write_unit(root: Path, name: str, *, family: str = "state-machine", lines: int = 40) -> Path:
    author = root / "units" / name / "_author"
    author.mkdir(parents=True, exist_ok=True)
    (author / "api.md").write_text("# API\n\nfunc Add(a, b int) int\n")
    (author / "contract.md").write_text(
        "# Contract\n\n| test | sentence |\n|---|---|\n| TestAdd | adding two integers returns their sum |\n"
    )
    (author / "bugreport.md").write_text(
        "# Missing behavior\n\nAdding two small integers yields a value below the sum: expected 4, got 0.\n\n"
        "Reproduce with:\n\n```\ntests/test.sh\n```\n"
    )
    (author / "gold.patch").write_text("diff --git a/add.go b/add.go\n")
    (author / "cheat.patch").write_text("diff --git a/add.go b/add.go\n")
    (author / "closure.json").write_text(
        json.dumps({"functions": ["Add"], "files": ["add.go"], "lines": lines, "family": family})
    )
    tree = author / "tree" / "mathx"
    tree.mkdir(parents=True)
    (tree / "add.go").write_text("package mathx\n\nfunc Add(a, b int) int { return a - b }\n")
    return author


HIDDEN_GO = """package mathx

import (
	"math/rand"
	"testing"
)

func TestAddProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(20260919))
	for i := 0; i < 100; i++ {
		a := rng.Intn(50)
		if Add(a, 0) != a {
			t.Fatalf("expected %d actual %d", a, Add(a, 0))
		}
	}
}
"""


class FakeRunner(AgentRunner):
    def __init__(self, work: Path, n: int = 2) -> None:
        super().__init__()
        self.work = work
        self.n = n
        self.calls: list[str] = []

    def run_agent(self, role, brief, cwd, model, **kwargs):  # type: ignore[no-untyped-def]
        self.calls.append(role)
        cwd = Path(cwd)
        if role == "author":
            batch = self.work / "mathx" / "author_batch"
            batch.mkdir(parents=True, exist_ok=True)
            rows = []
            for i in range(self.n):
                name = f"unit{i}"
                _write_unit(batch, name)
                rows.append(
                    {
                        "name": name,
                        "dir": f"units/{name}/_author",
                        "family": "state-machine",
                        "n_files": 1,
                        "n_lines": 40,
                    }
                )
            (batch / "units.json").write_text(json.dumps(rows), encoding="utf-8")
        if role == "verifier":
            hidden = cwd / "tests" / "hidden" / "mathx"
            hidden.mkdir(parents=True, exist_ok=True)
            (hidden / "add_prop_test.go").write_text(HIDDEN_GO)
            # also land under the verifier dir the pipeline copies from sandbox
        return AgentResult("ok", 10, 4, kwargs.get("backend") or "cursor", model, cwd, 0)


def _ok_proof() -> dict:
    return {
        "ok": True,
        "buggy": {"reward": "0"},
        "gold": {"reward": "1"},
        "cheat": {"reward": "0"},
        "image": "ladder-base:mathx",
    }


def test_yaml_config_and_repos_load() -> None:
    from openswe_traces.data import ROOT

    cfg = load_config(
        ROOT / "experiments/pipeline/config.yaml", ROOT / "experiments/pipeline/repos.yaml"
    )
    assert cfg.units_per_author_batch == 10
    assert cfg.author_minutes == 30
    assert cfg.solver_backends == cfg.solver_order
    assert set(cfg.solver_order) == {"cursor", "devin"}
    assert cfg.climb_levels == (2, 5, 6)
    assert cfg.attempts == 3
    assert cfg.composer_token_cap == 1_000_000_000
    assert cfg.devin_slots == 2
    assert cfg.hosts["laptop"].docker_concurrency == 4
    assert cfg.hosts["vps"].docker_concurrency == 2
    assert cfg.hosts["vps"].enabled is False
    assert all(r.language == "go" for r in cfg.repos)
    assert "tikv/client-go" not in {r.url for r in cfg.repos}


def test_yaml_underscored_int() -> None:
    data = load_yaml("composer_token_cap: 1_000_000_000\n")
    assert data["composer_token_cap"] == 1_000_000_000


def test_store_resumability(tmp_path: Path) -> None:
    store = PipelineStore(tmp_path / "s.db")
    store.mark_step("mathx", "prepare", DONE, payload={"image": "ladder-base:mathx"})
    store.mark_step("mathx", "author", "running")
    assert store.step_done("mathx", "prepare") is True
    assert store.should_run("mathx", "prepare") is False
    assert store.should_run("mathx", "author") is True  # crashed mid-run → retry
    store.mark_step("mathx", "author", DONE, payload={"units": ["unit0"]})
    assert store.should_run("mathx", "author") is False
    store.close()
    again = PipelineStore(tmp_path / "s.db")
    assert again.step_done("mathx", "prepare") is True
    assert again.step("mathx", "author").payload["units"] == ["unit0"]
    again.close()


def test_pipeline_skips_finished_prepare(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_repo("mathx")
    store.mark_step("mathx", "prepare", DONE, payload={"image": "x"})
    prepares: list[str] = []

    def prepare(spec, _cfg):
        prepares.append(spec.name)
        return {"image": "x"}

    runner = FakeRunner(cfg.work_dir, n=2)
    # pre-write units so author is a no-op even if called
    batch = cfg.work_dir / "mathx" / "author_batch"
    _write_unit(batch, "unit0")
    (batch / "units.json").write_text(json.dumps([{"name": "unit0", "dir": "units/unit0/_author"}]))

    launches: list[str] = []

    def harbor(**kw):
        launches.append(kw["job_name"])
        job = tmp_path / "jobs" / kw["job_name"]
        trial = job / "u__1"
        (trial / "agent").mkdir(parents=True, exist_ok=True)
        (trial / "result.json").write_text(
            json.dumps(
                {
                    "started_at": "2026-09-18T00:00:00+00:00",
                    "finished_at": "2026-09-18T00:03:00+00:00",
                    "verifier_result": {"rewards": {"reward": 1.0}},
                    "usage": {"input_tokens": 5, "output_tokens": 2},
                }
            )
        )
        return HarborJobResult(job_dir=job, trials=[audit_trial_dir(trial)])

    rt = Runtime(
        runner=runner,
        harbor=harbor,
        prepare=prepare,
        prove=_ok_proof,
        package=lambda *a, **k: {0: tmp_path / "L0", 2: tmp_path / "L2"},
        cleanup=lambda: "ok",
        wait_load=lambda: None,
    )
    _touch_levels(cfg, "mathx", "unit0")
    run_pipeline(cfg, store=store, runtime=rt, repos=cfg.repos, max_units=2)
    assert prepares == []  # prepare was already done
    store.close()


def test_devin_semaphore(tmp_path: Path) -> None:
    sem = DevinSemaphore(tmp_path / "slots", slots=4)
    got = sem.acquire(kind="author", n=4, timeout=0.2)
    assert len(got) == 4
    assert sem.available() == 0
    assert sem.harbor_n_concurrent(3) == 0
    try:
        sem.acquire(kind="extra", n=1, timeout=0.15)
        raise AssertionError("expected TimeoutError")
    except TimeoutError:
        pass
    sem.release([s.id for s in got[:2]])
    assert sem.available() == 2
    assert sem.harbor_n_concurrent(3) == 2
    more = sem.acquire(kind="harbor", n=2, timeout=0.2)
    assert len(more) == 2
    sem.release()


def test_token_cap_switches_solver(tmp_path: Path) -> None:
    store = PipelineStore(tmp_path / "s.db")
    budget = TokenBudget(store, cap=100)
    assert budget.choose_solver(("cursor", "devin")) == "cursor"
    budget.add(80, 30, "cursor-cli")
    assert budget.exhausted()
    assert budget.choose_solver(("cursor", "devin")) == "devin"
    store.close()


def test_token_cap_switches_mid_solve(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path, composer_token_cap=20)
    store = PipelineStore(cfg.state_db)
    store.upsert_unit("mathx", "unit0", status="packaged")
    budget = TokenBudget(store, cfg.composer_token_cap)
    sem = DevinSemaphore(tmp_path / "slots", slots=4)
    solvers: list[str] = []

    def harbor(**kw):
        solvers.append(kw["solver"])
        job = tmp_path / "jobs" / kw["job_name"]
        trials = []
        n = kw["n_attempts"]
        for i in range(n):
            trial = job / f"u__{i}"
            (trial / "agent").mkdir(parents=True, exist_ok=True)
            reward = 0.0 if kw["solver"] == "cursor" else 1.0
            (trial / "result.json").write_text(
                json.dumps(
                    {
                        "started_at": "2026-09-18T00:00:00+00:00",
                        "finished_at": "2026-09-18T00:01:00+00:00",
                        "verifier_result": {"rewards": {"reward": reward}},
                        "usage": {"input_tokens": 12, "output_tokens": 12},
                    }
                )
            )
            trials.append(audit_trial_dir(trial))
        return HarborJobResult(job_dir=job, trials=trials)

    _touch_levels(cfg, "mathx", "unit0")
    # L2 cursor fails (and blows the cap); climb should switch to devin
    solve_unit(
        "mathx",
        "unit0",
        cfg,
        store,
        budget=budget,
        semaphore=sem,
        harbor=harbor,
        cleanup=lambda: "ok",
        wait_load=lambda: None,
    )
    assert "cursor" in solvers
    assert "devin" in solvers
    events = store.events(kind="token_cap")
    assert events, "expected a token-cap switch event"
    store.close()


def test_lazy_ladder_selection() -> None:
    assert affordance_level(0) == -2
    assert affordance_level(2) == 0
    assert affordance_level(6) == 4
    assert next_solve_levels({}) == [2]
    assert next_solve_levels({2: (2, 3)}) == [0]
    assert next_solve_levels({2: (2, 3), 0: (1, 1)}) == []
    assert next_solve_levels({2: (1, 3)}) == [3]
    assert next_solve_levels({2: (1, 3), 3: (2, 3)}) == []
    assert next_solve_levels({2: (0, 3), 3: (0, 3), 4: (0, 3)}) == [5]
    assert attempts_for_level(0) == 1
    assert attempts_for_level(2) == 3
    assert flip_point({2: (2, 3), 5: (3, 3)}) == 2


def test_parse_tokens() -> None:
    tin, tout = parse_cursor_agent_output(
        '{"type":"result","usage":{"input_tokens":11,"output_tokens":7}}\n'
    )
    assert (tin, tout) == (11, 7)
    tin, tout = parse_harbor_trial_tokens({"usage": {"prompt_tokens": 3, "completion_tokens": 9}})
    assert (tin, tout) == (3, 9)


def test_baseline_zero_passing_fails() -> None:
    parsed = parse_baseline_output("FAIL\texample.com/mod/pkg\t0.1s\n")
    assert parsed["n_passing"] == 0
    try:
        if parsed["n_passing"] <= 0:
            raise BaselineError("baseline has 0 passing packages")
        raise AssertionError("expected BaselineError")
    except BaselineError:
        pass
    parsed = parse_baseline_output("ok  \texample.com/mod/a\t0.1s\nok  \texample.com/mod/b\t0.2s\n")
    assert parsed["n_passing"] == 2


def test_aggregator(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_unit(
        "mathx", "codec", status="solved", family="cross-file", n_files=5, n_lines=900
    )
    store.upsert_unit("mathx", "bad", status="rejected", rejected_rule="B4", family="other")
    for i in range(3):
        store.add_trial(
            repo="mathx",
            unit="codec",
            level=2,
            solver="cursor",
            attempt=i + 1,
            reward=1.0 if i < 2 else 0.0,
            tokens_in=100,
            tokens_out=20,
            audit_class="clean",
            wall_minutes=4.0,
        )
    store.add_trial(
        repo="mathx",
        unit="codec",
        level=0,
        solver="cursor",
        attempt=1,
        reward=0.0,
        tokens_in=10,
        tokens_out=2,
        audit_class="clean",
        wall_minutes=3.0,
    )
    store.add_trial(
        repo="mathx",
        unit="codec",
        level=2,
        solver="cursor",
        attempt=9,
        reward=1.0,
        tokens_in=1,
        tokens_out=1,
        audit_class="contaminated",
        excluded=True,
    )
    store.add_tokens("cursor-cli", 1000, 50)
    df = aggregate(store, cfg)
    assert cfg.results_parquet.is_file()
    assert cfg.results_md.is_file()
    md = cfg.results_md.read_text(encoding="utf-8")
    assert "Composer tokens used" in md
    assert "1000" in md or "1050" in md
    assert "B4" in md
    rows = task_rows(store)
    codec = next(r for r in rows if r["unit"] == "codec")
    assert codec["flip"] == 2
    assert codec["L2_pass"] == 2
    assert codec["L2_n"] == 3
    assert int(df.loc[df["unit"] == "codec", "closure_size"].iloc[0]) == 5
    dash = render_dashboard(store, rows)
    assert "rejected by rule" in dash.lower()
    store.close()


def test_audit_contaminated(tmp_path: Path) -> None:
    trial = tmp_path / "t"
    (trial / "agent").mkdir(parents=True)
    (trial / "agent" / "log").write_text("webFetchToolCall {url: x}\n")
    (trial / "result.json").write_text(
        json.dumps({"verifier_result": {"rewards": {"reward": 1.0}}})
    )
    audit = audit_trial_dir(trial)
    assert audit.verdict == "CONTAMINATED"
    assert audit_class(audit.verdict) == "contaminated"


def test_discover_units(tmp_path: Path) -> None:
    _write_unit(tmp_path, "alpha")
    rows = discover_units(tmp_path)
    assert [r["name"] for r in rows] == ["alpha"]


def _touch_levels(cfg: PipelineConfig, repo: str, unit: str) -> None:
    from openswe_traces.pipeline.safety import write_safe_task_skeleton

    for lv in range(7):
        write_safe_task_skeleton(cfg.tasks_dir / repo / f"{unit}-L{lv}")


def test_status_and_dry_run_table(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_repo("mathx", "done")
    store.upsert_unit(
        "mathx", "unit0", status="solved", family="state-machine", n_files=2, n_lines=40
    )
    store.add_trial(
        repo="mathx",
        unit="unit0",
        level=2,
        solver="cursor",
        attempt=1,
        reward=1.0,
        tokens_in=1,
        tokens_out=1,
        audit_class="clean",
        wall_minutes=1.0,
    )
    store.add_trial(
        repo="mathx",
        unit="unit0",
        level=2,
        solver="cursor",
        attempt=2,
        reward=1.0,
        tokens_in=1,
        tokens_out=1,
        audit_class="clean",
        wall_minutes=1.0,
    )
    store.add_trial(
        repo="mathx",
        unit="unit0",
        level=2,
        solver="cursor",
        attempt=3,
        reward=1.0,
        tokens_in=1,
        tokens_out=1,
        audit_class="clean",
        wall_minutes=1.0,
    )
    text = status_text(cfg, store)
    assert "mathx" in text
    assert "solved" in text

    runner = FakeRunner(cfg.work_dir, n=1)
    rt = Runtime(
        runner=runner,
        harbor=lambda **kw: HarborJobResult(job_dir=tmp_path / "empty"),
        prepare=lambda spec, c: {"image": "ladder-base:mathx", "baseline": {"n_passing": 3}},
        prove=_ok_proof,
        package=lambda *a, **k: {0: tmp_path / "L0", 2: tmp_path / "L2"},
        cleanup=lambda: "ok",
        wait_load=lambda: None,
    )
    _touch_levels(cfg, "mathx", "unit0")
    table = dry_run("mathx", 1, cfg=cfg, store=store, runtime=rt)
    assert "mathx" in table
    store.close()


def test_author_ingests_existing_without_agent(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    authored_root = cfg.authored_dir / "mathx"
    _write_unit(authored_root, "alpha")
    import shutil

    src = authored_root / "units" / "alpha" / "_author"
    dest = authored_root / "alpha" / "_author"
    dest.parent.mkdir(parents=True, exist_ok=True)
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest)
    shutil.rmtree(authored_root / "units")
    (dest / "difficulty.md").write_text("predicted_flip: L2\ncontrol: true\n")
    runner = FakeRunner(cfg.work_dir, n=9)
    from openswe_traces.pipeline.author import run_author

    rows = run_author("mathx", cfg, store, runner=runner, n_units=10)
    assert [r["name"] for r in rows] == ["alpha"]
    assert runner.calls == []
    assert rows[0]["predicted_flip"] == 2
    assert rows[0]["is_control"] is True
    store.close()


def test_verifier_skips_existing_l2_b4_pass(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    batch = cfg.work_dir / "mathx" / "author_batch"
    _write_unit(batch, "unit0")
    l2 = cfg.tasks_dir / "mathx" / "unit0-L2"
    hidden = l2 / "tests" / "hidden" / "pkg"
    hidden.mkdir(parents=True)
    (hidden / "p_test.go").write_text("package pkg\n")
    (l2 / "validation.json").write_text(
        json.dumps(
            {
                "rule_verdicts": [
                    {"rule_id": "B4", "passed": True, "skipped": False, "evidence": "ok"}
                ]
            }
        )
    )
    runner = FakeRunner(cfg.work_dir, n=1)
    from openswe_traces.pipeline.verifier import run_verifier

    out = run_verifier("mathx", "unit0", cfg, store, runner=runner, prove=_ok_proof)
    assert runner.calls == []
    assert out["proof"]["skipped"] is True
    assert store.step("mathx", "verifier", "unit0").status == "done"
    store.close()


def test_solve_adaptive_ladder_starts_one_l2_attempt(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_unit("mathx", "unit0", status="packaged")
    budget = TokenBudget(store, cfg.composer_token_cap)
    sem = DevinSemaphore(tmp_path / "slots", slots=6)
    seen: list[tuple[int, str]] = []

    def harbor(**kw):
        seen.append((kw["n_attempts"], kw["job_name"]))
        job = tmp_path / "jobs" / kw["job_name"]
        trials = []
        for i in range(kw["n_attempts"]):
            trial = job / f"u__{i}"
            (trial / "agent").mkdir(parents=True, exist_ok=True)
            (trial / "result.json").write_text(
                json.dumps(
                    {
                        "started_at": "2026-09-18T00:00:00+00:00",
                        "finished_at": "2026-09-18T00:01:00+00:00",
                        "verifier_result": {
                            "rewards": {"reward": 0.0 if "-L0-" in kw["job_name"] else 1.0}
                        },
                        "usage": {"input_tokens": 1, "output_tokens": 1},
                    }
                )
            )
            from openswe_traces.pipeline.audit import audit_trial_dir

            trials.append(audit_trial_dir(trial))
        return HarborJobResult(job_dir=job, trials=trials)

    _touch_levels(cfg, "mathx", "unit0")
    out = solve_unit(
        "mathx",
        "unit0",
        cfg,
        store,
        budget=budget,
        semaphore=sem,
        harbor=harbor,
        cleanup=lambda: "ok",
        wait_load=lambda: None,
        skip_hack_docker=True,
    )
    assert seen[0][0] == 1
    assert "-L2-" in seen[0][1]
    assert out["flip"] == 2
    assert out["confirmed"] is True
    store.close()


def test_hack_audit_hard_fail_excluded_from_flip(tmp_path: Path) -> None:
    from openswe_traces.pipeline_ext.hack_audit import HackVerdict

    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_unit("mathx", "unit0", status="packaged")
    budget = TokenBudget(store, cfg.composer_token_cap)
    sem = DevinSemaphore(tmp_path / "slots", slots=6)

    def harbor(**kw):
        job = tmp_path / "jobs" / kw["job_name"]
        trials = []
        for i in range(kw["n_attempts"]):
            trial = job / f"u__{i}"
            (trial / "agent").mkdir(parents=True, exist_ok=True)
            (trial / "result.json").write_text(
                json.dumps({"verifier_result": {"rewards": {"reward": 1.0}}})
            )
            from openswe_traces.pipeline.audit import audit_trial_dir

            trials.append(audit_trial_dir(trial))
        return HarborJobResult(job_dir=job, trials=trials)

    _touch_levels(cfg, "mathx", "unit0")
    solve_unit(
        "mathx",
        "unit0",
        cfg,
        store,
        budget=budget,
        semaphore=sem,
        harbor=harbor,
        cleanup=lambda: "ok",
        wait_load=lambda: None,
        hack_audit=lambda **k: HackVerdict(False, ("touched tests/",), (), {}),
        skip_hack_docker=True,
    )
    trials = store.list_trials(repo="mathx", unit="unit0", include_excluded=True)
    assert any(t.audit_class == "hacked" and t.excluded for t in trials)
    # hacked passes do not confirm L2
    from openswe_traces.pipeline.solve import flip_from_store

    flip = flip_from_store(store, "mathx", "unit0", "cursor")
    assert flip.level is None or flip.confirmed is False
    store.close()


def test_control_runs_first_and_pause_on_fail(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    batch = cfg.work_dir / "mathx" / "author_batch"
    _write_unit(batch, "ctrl")
    _write_unit(batch, "other")
    ctrl_author = batch / "units" / "ctrl" / "_author"
    (ctrl_author / "difficulty.md").write_text("predicted_flip: L2\ncontrol: true\n")
    (batch / "units" / "other" / "_author" / "difficulty.md").write_text("predicted_flip: L5\n")
    (batch / "units.json").write_text(
        json.dumps(
            [
                {"name": "ctrl", "dir": "units/ctrl/_author"},
                {"name": "other", "dir": "units/other/_author"},
            ]
        )
    )
    launches: list[str] = []

    def harbor(**kw):
        launches.append(kw["job_name"])
        job = tmp_path / "jobs" / kw["job_name"]
        trials = []
        for i in range(kw["n_attempts"]):
            trial = job / f"u__{i}"
            (trial / "agent").mkdir(parents=True, exist_ok=True)
            (trial / "result.json").write_text(
                json.dumps(
                    {
                        "started_at": "2026-09-18T00:00:00+00:00",
                        "finished_at": "2026-09-18T00:01:00+00:00",
                        "verifier_result": {"rewards": {"reward": 0.0}},
                    }
                )
            )
            from openswe_traces.pipeline.audit import audit_trial_dir

            trials.append(audit_trial_dir(trial))
        return HarborJobResult(job_dir=job, trials=trials)

    runner = FakeRunner(cfg.work_dir, n=2)
    rt = Runtime(
        runner=runner,
        harbor=harbor,
        prepare=lambda spec, c: {"image": "x"},
        prove=_ok_proof,
        package=lambda *a, **k: {2: tmp_path / "L2"},
        cleanup=lambda: "ok",
        wait_load=lambda: None,
    )
    _touch_levels(cfg, "mathx", "ctrl")
    _touch_levels(cfg, "mathx", "other")
    run_pipeline(cfg, store=store, runtime=rt, repos=cfg.repos, max_units=2)
    assert any("ctrl" in n and "-L2-" in n for n in launches)
    assert not any("other" in n for n in launches)
    assert store.repo_status("mathx") == "paused"
    store.close()


def test_aggregate_emits_calibration_and_flags(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.upsert_unit(
        "mathx",
        "codec",
        status="solved",
        family="cross-file",
        n_files=5,
        n_lines=900,
        predicted_flip=2,
        is_control=True,
        author_backend="cursor",
    )
    for i in range(3):
        store.add_trial(
            repo="mathx",
            unit="codec",
            level=2,
            solver="cursor",
            attempt=i + 1,
            reward=1.0 if i < 2 else 0.0,
            tokens_in=100,
            tokens_out=20,
            audit_class="clean",
            wall_minutes=4.0,
        )
    store.add_trial(
        repo="mathx",
        unit="codec",
        level=0,
        solver="cursor",
        attempt=1,
        reward=0.0,
        tokens_in=10,
        tokens_out=2,
        audit_class="clean",
        wall_minutes=3.0,
    )
    store.add_trial(
        repo="mathx",
        unit="codec",
        level=2,
        solver="cursor",
        attempt=9,
        reward=1.0,
        tokens_in=1,
        tokens_out=1,
        audit_class="hacked",
        excluded=True,
    )
    store.add_tokens("cursor-cli", 1000, 50)
    df = aggregate(store, cfg)
    md = cfg.results_md.read_text(encoding="utf-8")
    assert "Calibration curve" in md
    assert "Author calibration" in md
    assert "hacked" in md.lower()
    assert "Early-warning flags" in md
    assert "Action:" in md or "_none fired_" in md
    assert "inter-attempt" in md
    codec = next(r for r in task_rows(store) if r["unit"] == "codec")
    assert codec["flip"] == 2
    assert codec["hacked_n"] == 1
    assert int(df.loc[df["unit"] == "codec", "closure_size"].iloc[0]) == 5
    store.close()


def test_timeout_does_not_count_as_fail(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.add_trial(
        repo="mathx",
        unit="unit0",
        level=2,
        solver="cursor",
        attempt=1,
        reward=0.0,
        audit_class="d",
        timeout=True,
    )
    from openswe_traces.pipeline.solve import task_state_from_store
    from openswe_traces.pipeline_ext.ladder_policy import next_actions

    state = task_state_from_store(store, "mathx", "unit0", "cursor")
    req = [a for a in next_actions(state) if not a.optional]
    assert req[0].level == 2
    assert req[0].n_attempts == 1
    store.close()


def test_parse_job_name_with_hyphenated_repo() -> None:
    from openswe_traces.pipeline.reconcile import parse_job_name

    d = parse_job_name("client-go-connarray-cv-L2-devin-n1-k3", {"client-go", "helm"})
    assert d is not None
    assert (d["repo"], d["unit"], d["level"], d["solver"], d["rerun"]) == (
        "client-go",
        "connarray-cv",
        "2",
        "devin",
        None,
    )
    assert (
        parse_job_name("nats-server-x-L0-cursor-n3-k0-rerun", {"nats-server"})["rerun"] == "-rerun"
    )
    assert parse_job_name("unknown-x-L0-devin-n1-k0", {"helm"}) is None
