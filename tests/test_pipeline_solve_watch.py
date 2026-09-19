from __future__ import annotations

import json
from pathlib import Path

from openswe_traces.pipeline.audit import audit_trial_dir
from openswe_traces.pipeline.config import PipelineConfig, RepoSpec
from openswe_traces.pipeline.safety import (
    TaskSafetyError,
    assert_harbor_safe,
    write_safe_task_skeleton,
)
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.solve import HarborJobResult
from openswe_traces.pipeline.state import DONE, RUNNING, PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget
from openswe_traces.pipeline.watch import (
    already_solving_or_solved,
    discover_verified_units,
    iter_l2_dirs,
    run_solve_unit,
    solve_watch,
    unit_verified,
    vps_harbor,
)


def _cfg(tmp: Path, **kw: object) -> PipelineConfig:
    return PipelineConfig(
        units_per_author_batch=5,
        composer_token_cap=1_000_000,
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
        **kw,  # type: ignore[arg-type]
    )


def _verdicts(*pairs: tuple[str, bool, bool]) -> list[dict[str, object]]:
    return [
        {"rule_id": rid, "passed": passed, "skipped": skipped, "evidence": rid}
        for rid, passed, skipped in pairs
    ]


def _write_l2(
    cfg: PipelineConfig,
    repo: str,
    unit: str,
    *,
    verdicts: list[dict[str, object]] | None = None,
    checks: dict[str, object] | None = None,
    underscore: bool = False,
    extra: dict[str, object] | None = None,
) -> Path:
    name = f"{unit}_L2" if underscore else f"{unit}-L2"
    dest = cfg.tasks_dir / repo / name
    write_safe_task_skeleton(dest)
    payload: dict[str, object] = {}
    if verdicts is not None:
        payload["rule_verdicts"] = verdicts
    if checks is not None:
        payload["checks"] = checks
    if extra:
        payload.update(extra)
    (dest / "validation.json").write_text(json.dumps(payload), encoding="utf-8")
    return dest


PASS_RULES = _verdicts(("B4", True, False), ("A1", True, False), ("A8", True, False), ("A3", True, False))


def _fake_harbor(tmp: Path, *, reward: float = 1.0):
    def harbor(**kw):
        job = tmp / "jobs" / kw["job_name"]
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
                            "rewards": {
                                "reward": 0.0 if "-L0-" in kw["job_name"] else reward
                            }
                        },
                        "usage": {"input_tokens": 1, "output_tokens": 1},
                    }
                )
            )
            trials.append(audit_trial_dir(trial))
        return HarborJobResult(job_dir=job, trials=trials)

    return harbor


def test_unit_verified_rules_and_proof_flags() -> None:
    rules = {"rule_verdicts": PASS_RULES}
    assert unit_verified(rules) is True
    missing_a1 = {
        "rule_verdicts": _verdicts(
            ("B4", True, False), ("A1", False, False), ("A8", True, False), ("A3", True, False)
        )
    }
    assert unit_verified(missing_a1) is False
    proof = {
        "rule_verdicts": _verdicts(
            ("B4", True, False), ("A1", False, True), ("A8", False, True), ("A3", False, True)
        ),
        "checks": {"gold_restore": True, "buggy_fails": True, "cheat_rejected": True},
    }
    assert unit_verified(proof) is True
    no_b4 = {
        "rule_verdicts": _verdicts(
            ("B4", False, False), ("A1", True, False), ("A8", True, False), ("A3", True, False)
        )
    }
    assert unit_verified(no_b4) is False


def test_discover_hyphen_and_underscore_l2(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    _write_l2(cfg, "mathx", "alpha", verdicts=PASS_RULES)
    _write_l2(cfg, "mathx", "beta", verdicts=PASS_RULES, underscore=True)
    _write_l2(cfg, "mathx", "gamma", verdicts=_verdicts(("B4", False, False)))
    found = {(r, u) for r, u, _ in discover_verified_units(cfg)}
    assert ("mathx", "alpha") in found
    assert ("mathx", "beta") in found
    assert ("mathx", "gamma") not in found
    names = {p.name for _, _, p in iter_l2_dirs(cfg.tasks_dir)}
    assert "alpha-L2" in names
    assert "beta_L2" in names


def test_state_skips_solved_and_in_progress(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    store.mark_step("mathx", "solve", DONE, unit="doneu")
    store.mark_step("mathx", "solve", RUNNING, unit="live")
    store.upsert_unit("mathx", "solvedu", status="solved")
    assert already_solving_or_solved(store, "mathx", "doneu") is True
    assert already_solving_or_solved(store, "mathx", "live") is True
    assert already_solving_or_solved(store, "mathx", "solvedu") is True
    assert already_solving_or_solved(store, "mathx", "fresh") is False
    store.close()


def test_solve_watch_launches_then_skips(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    _write_l2(cfg, "mathx", "unit0", verdicts=PASS_RULES)
    for lv in (0, 5, 6):
        write_safe_task_skeleton(cfg.tasks_dir / "mathx" / f"unit0-L{lv}")
    store = PipelineStore(cfg.state_db)
    n = solve_watch(
        cfg,
        interval=0,
        host="laptop",
        store=store,
        harbor=_fake_harbor(tmp_path),
        max_cycles=1,
        skip_hack_docker=True,
        wait_load=lambda: None,
        cleanup=lambda: "ok",
        sleep_fn=lambda _s: None,
    )
    assert n == 1
    assert store.step("mathx", "solve", "unit0").status == DONE
    assert cfg.results_md.is_file()
    n2 = solve_watch(
        cfg,
        interval=0,
        host="laptop",
        store=store,
        harbor=_fake_harbor(tmp_path),
        max_cycles=1,
        skip_hack_docker=True,
        wait_load=lambda: None,
        cleanup=lambda: "ok",
        sleep_fn=lambda _s: None,
    )
    assert n2 == 0
    store.close()


def test_refuse_bad_task_toml(tmp_path: Path) -> None:
    dest = tmp_path / "bad"
    write_safe_task_skeleton(dest)
    toml = dest / "task.toml"
    toml.write_text(
        toml.read_text(encoding="utf-8").replace(
            'network_mode = "allowlist"', 'network_mode = "public"', 1
        ),
        encoding="utf-8",
    )
    try:
        assert_harbor_safe(dest, solver="cursor")
        raise AssertionError("expected TaskSafetyError")
    except TaskSafetyError as exc:
        assert "allowlist" in str(exc)

    dest2 = tmp_path / "extra-host"
    write_safe_task_skeleton(dest2)
    text = (dest2 / "task.toml").read_text(encoding="utf-8")
    (dest2 / "task.toml").write_text(
        text.replace("downloads.cursor.com", 'downloads.cursor.com", "evil.example'),
        encoding="utf-8",
    )
    try:
        assert_harbor_safe(dest2, solver="cursor")
        raise AssertionError("expected TaskSafetyError")
    except TaskSafetyError as exc:
        assert "evil.example" in str(exc)

    dest3 = tmp_path / "gitty"
    write_safe_task_skeleton(dest3)
    (dest3 / "environment" / "src" / ".git").mkdir()
    try:
        assert_harbor_safe(dest3, solver="cursor")
        raise AssertionError("expected TaskSafetyError")
    except TaskSafetyError as exc:
        assert ".git" in str(exc)

    dest4 = tmp_path / "nocheck"
    write_safe_task_skeleton(dest4)
    (dest4 / "tests" / "test.sh").write_text("#!/bin/bash\ngo test ./...\n", encoding="utf-8")
    try:
        assert_harbor_safe(dest4, solver="cursor")
        raise AssertionError("expected TaskSafetyError")
    except TaskSafetyError as exc:
        assert "checksum" in str(exc)


def test_solve_unit_refuses_public_agent(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    _write_l2(cfg, "mathx", "unit0", verdicts=PASS_RULES)
    l2 = cfg.tasks_dir / "mathx" / "unit0-L2"
    text = (l2 / "task.toml").read_text(encoding="utf-8")
    (l2 / "task.toml").write_text(text.replace('"allowlist"', '"bridge"', 1), encoding="utf-8")
    try:
        run_solve_unit(
            "mathx",
            "unit0",
            cfg,
            store=store,
            harbor=_fake_harbor(tmp_path),
            wait_load=lambda: None,
            cleanup=lambda: "ok",
            skip_hack_docker=True,
        )
        raise AssertionError("expected TaskSafetyError")
    except TaskSafetyError:
        pass
    step = store.step("mathx", "solve", "unit0")
    assert step is None or step.status != DONE
    store.close()


def test_run_solve_unit_records_and_aggregates(tmp_path: Path) -> None:
    cfg = _cfg(tmp_path)
    store = PipelineStore(cfg.state_db)
    _write_l2(cfg, "mathx", "unit0", verdicts=PASS_RULES)
    for lv in (0, 5, 6):
        write_safe_task_skeleton(cfg.tasks_dir / "mathx" / f"unit0-L{lv}")
    out = run_solve_unit(
        "mathx",
        "unit0",
        cfg,
        store=store,
        harbor=_fake_harbor(tmp_path),
        wait_load=lambda: None,
        cleanup=lambda: "ok",
        skip_hack_docker=True,
        semaphore=DevinSemaphore(tmp_path / "slots", slots=4),
        budget=TokenBudget(store, cfg.composer_token_cap),
    )
    assert out["flip"] == 2
    assert out["confirmed"] is True
    assert store.step("mathx", "solve", "unit0").status == DONE
    unit = next(u for u in store.list_units("mathx") if u["unit"] == "unit0")
    assert unit["status"] == "solved"
    assert cfg.results_md.is_file()
    store.close()


def test_cli_parses_solve_commands() -> None:
    from openswe_traces.pipeline.cli import build_parser

    parser = build_parser()
    a = parser.parse_args(["solve-unit", "--repo", "mathx", "--unit", "codec"])
    assert a.cmd == "solve-unit" and a.repo == "mathx" and a.unit == "codec"
    b = parser.parse_args(["solve-watch", "--interval", "12", "--host", "vps"])
    assert b.cmd == "solve-watch" and b.interval == 12 and b.host == "vps"


def test_vps_harbor_rsync_and_key_in_ssh(tmp_path: Path) -> None:
    env_file = tmp_path / "cursor.env"
    env_file.write_text("CURSOR_API_KEY=secret-test-key\n", encoding="utf-8")
    cfg = _cfg(tmp_path, cursor_env_file=env_file)
    dest = write_safe_task_skeleton(cfg.tasks_dir / "mathx" / "unit0-L2")
    calls: list[list[str]] = []

    def run(argv, **kw):  # type: ignore[no-untyped-def]
        calls.append(list(argv))
        from subprocess import CompletedProcess

        return CompletedProcess(argv, 0, stdout="", stderr="")

    job = vps_harbor(
        cfg,
        path=dest,
        solver="cursor",
        n_attempts=1,
        n_concurrent=2,
        job_name="mathx-unit0-L2-cursor-n1",
        timeout_sec=30,
        run=run,
    )
    assert job.job_dir == cfg.jobs_dir / "vps" / "mathx-unit0-L2-cursor-n1"
    joined = [" ".join(c) for c in calls]
    assert any(c[0] == "rsync" for c in calls)
    ssh_cmds = [c for c in calls if c and c[0] == "ssh"]
    assert ssh_cmds, joined
    assert any("CURSOR_API_KEY=secret-test-key" in " ".join(c) for c in ssh_cmds)
    assert any("harbor" in " ".join(c) for c in ssh_cmds)
    leaked = list(dest.rglob("*"))
    for path in leaked:
        if path.is_file() and path.suffix in {".env", ".txt", ".toml", ".sh"}:
            blob = path.read_text(encoding="utf-8", errors="replace")
            assert "secret-test-key" not in blob
