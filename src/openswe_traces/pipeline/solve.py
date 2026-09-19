"""Harbor solve with a lazy ladder, web-use audit, and token-cap solver switch."""

from __future__ import annotations

import json
import subprocess
from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.audit import TrialAudit, audit_class, audit_job
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.ladder import attempts_for_level, next_solve_levels
from openswe_traces.pipeline.package import ensure_level
from openswe_traces.pipeline.resources import harbor_concurrency, run_cleanup, wait_for_load
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget


@dataclass
class HarborJobResult:
    job_dir: Path
    trials: list[TrialAudit] = field(default_factory=list)
    raw: dict[str, Any] = field(default_factory=dict)


def solver_pair(cfg: PipelineConfig, name: str) -> tuple[str, str]:
    if name == "devin":
        return cfg.devin_harbor_agent, cfg.devin_harbor_model
    return cfg.cursor_harbor_agent, cfg.cursor_harbor_model


def harbor_argv(
    cfg: PipelineConfig,
    *,
    path: Path,
    solver: str,
    n_attempts: int,
    n_concurrent: int,
    job_name: str,
) -> list[str]:
    agent, model = solver_pair(cfg, solver)
    return [
        "harbor",
        "run",
        "--path",
        str(path),
        "--agent",
        agent,
        "--model",
        model,
        "--n-attempts",
        str(n_attempts),
        "--n-concurrent",
        str(max(1, n_concurrent)),
        "--max-retries",
        "2",
        "--jobs-dir",
        str(cfg.jobs_dir),
        "--job-name",
        job_name,
        "--yes",
    ]


def run_harbor(
    argv: list[str],
    *,
    timeout: int | None = None,
    run: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run,
) -> subprocess.CompletedProcess[str]:
    return run(argv, capture_output=True, text=True, timeout=timeout, check=False)


def _record_audits(
    store: PipelineStore,
    budget: TokenBudget,
    *,
    repo: str,
    unit: str,
    level: int,
    solver: str,
    job_dir: Path,
    audits: list[TrialAudit],
    rerun: bool = False,
) -> list[TrialAudit]:
    for i, audit in enumerate(audits, start=1):
        klass = audit_class(audit.verdict)
        excluded = klass in {"contaminated", "checksum"} and rerun
        if klass == "checksum":
            store.add_event("rule", f"B1 checksum-guard {repo}/{unit}-L{level} {audit.task}")
        store.add_trial(
            repo=repo,
            unit=unit,
            level=level,
            solver=solver,
            attempt=i,
            reward=audit.reward,
            tokens_in=audit.tokens_in,
            tokens_out=audit.tokens_out,
            audit_class=klass,
            wall_minutes=audit.wall_minutes,
            job_dir=str(job_dir),
            excluded=excluded or klass == "contaminated" and rerun,
        )
        if solver != "devin":
            budget.add(audit.tokens_in, audit.tokens_out, "cursor-cli")
    return audits


def _contaminated(audits: list[TrialAudit]) -> bool:
    return any(a.verdict == "CONTAMINATED" for a in audits)


def solve_unit(
    repo: str,
    unit: str,
    cfg: PipelineConfig,
    store: PipelineStore,
    *,
    budget: TokenBudget,
    semaphore: DevinSemaphore,
    harbor: Callable[..., HarborJobResult],
    cleanup: Callable[[], str] | None = None,
    wait_load: Callable[..., None] | None = None,
    host: str | None = None,
) -> dict[str, Any]:
    """Lazy ladder from L2. Token cap may flip the solver to Devin mid-unit."""
    wait = wait_load or (
        lambda: wait_for_load(mult=cfg.load_mult)
    )
    clean = cleanup or (lambda: run_cleanup(cfg.cleanup_script))
    solver = budget.choose_solver(cfg.solver_backends)
    if solver != cfg.solver_backends[0]:
        store.add_event("token_cap", f"composer cap reached ({budget.used}/{budget.cap}); solver={solver}")
        store.set_meta("solver_backend", solver)

    results: dict[int, tuple[int, int]] = {}
    # seed from already-scored trials (resume)
    for level in range(7):
        p, n = store.level_counts(repo, unit, level, solver)
        if n:
            results[level] = (p, n)

    pending = next_solve_levels(results, start=2, attempts=cfg.attempts)
    while pending:
        level = pending[0]
        if budget.exhausted() and solver != "devin":
            nxt = budget.choose_solver(cfg.solver_backends)
            if nxt != solver:
                store.add_event("token_cap", f"switch solver {solver} -> {nxt}")
                store.set_meta("solver_backend", nxt)
                solver = nxt
                results = {}
                for lv in range(7):
                    p, n = store.level_counts(repo, unit, lv, solver)
                    if n:
                        results[lv] = (p, n)
                pending = next_solve_levels(results, start=2, attempts=cfg.attempts)
                continue
        task = ensure_level(repo, unit, cfg, level)
        n_att = attempts_for_level(level, default=cfg.attempts)
        wait()
        n_conc = harbor_concurrency(cfg, solver, semaphore, host=host)
        if solver == "devin":
            if n_conc < 1:
                with semaphore.hold(kind="harbor", n=1, timeout=cfg.author_minutes * 60):
                    n_conc = 1
                    job = _launch(
                        cfg,
                        store,
                        budget,
                        harbor,
                        repo=repo,
                        unit=unit,
                        level=level,
                        solver=solver,
                        task=task,
                        n_att=n_att,
                        n_conc=n_conc,
                    )
            else:
                with semaphore.hold(kind="harbor", n=n_conc, timeout=cfg.author_minutes * 60):
                    job = _launch(
                        cfg,
                        store,
                        budget,
                        harbor,
                        repo=repo,
                        unit=unit,
                        level=level,
                        solver=solver,
                        task=task,
                        n_att=n_att,
                        n_conc=n_conc,
                    )
        else:
            job = _launch(
                cfg,
                store,
                budget,
                harbor,
                repo=repo,
                unit=unit,
                level=level,
                solver=solver,
                task=task,
                n_att=n_att,
                n_conc=max(1, n_conc),
            )
        clean()
        p, n = store.level_counts(repo, unit, level, solver)
        results[level] = (p, n)
        _ = job
        pending = next_solve_levels(results, start=2, attempts=cfg.attempts)

    flip = None
    for level in range(7):
        p, n = results.get(level, (0, 0))
        if n >= (1 if level == 0 else cfg.attempts) and p >= (1 if level == 0 else 2):
            flip = level
            break
    # L0 distinction: a single pass counts as flip=L0; a single fail keeps flip at L2 if L2 passed.
    if 2 in results:
        p2, n2 = results[2]
        if n2 >= cfg.attempts and p2 >= 2:
            if 0 in results and results[0][0] >= 1:
                flip = 0
            else:
                flip = 2 if flip is None or flip > 2 else flip
    store.mark_step(repo, "solve", "done", unit=unit, payload={"flip": flip, "solver": solver, "results": results})
    return {"flip": flip, "solver": solver, "results": {str(k): list(v) for k, v in results.items()}}


def _launch(
    cfg: PipelineConfig,
    store: PipelineStore,
    budget: TokenBudget,
    harbor: Callable[..., HarborJobResult],
    *,
    repo: str,
    unit: str,
    level: int,
    solver: str,
    task: Path,
    n_att: int,
    n_conc: int,
) -> HarborJobResult:
    job_name = f"{repo}-{unit}-L{level}-{solver}"
    job = harbor(
        cfg=cfg,
        path=task,
        solver=solver,
        n_attempts=n_att,
        n_concurrent=n_conc,
        job_name=job_name,
    )
    audits = list(job.trials or audit_job(job.job_dir))
    if not audits:
        for i in range(n_att):
            store.add_trial(
                repo=repo,
                unit=unit,
                level=level,
                solver=solver,
                attempt=i + 1,
                reward=None,
                audit_class="infra",
                job_dir=str(job.job_dir),
            )
        return job
    _record_audits(
        store, budget, repo=repo, unit=unit, level=level, solver=solver, job_dir=job.job_dir, audits=audits
    )
    if _contaminated(audits):
        store.add_event("audit", f"contaminated {job_name}; rerun once")
        job2 = harbor(
            cfg=cfg,
            path=task,
            solver=solver,
            n_attempts=n_att,
            n_concurrent=n_conc,
            job_name=job_name + "-rerun",
        )
        audits2 = job2.trials or audit_job(job2.job_dir)
        _record_audits(
            store,
            budget,
            repo=repo,
            unit=unit,
            level=level,
            solver=solver,
            job_dir=job2.job_dir,
            audits=audits2,
            rerun=True,
        )
        if _contaminated(audits2):
            store.add_event("audit", f"contaminated after rerun; excluded {job_name}")
        return job2
    return job


def default_harbor(cfg: PipelineConfig | None = None, **kw: Any) -> HarborJobResult:
    if cfg is None:
        cfg = kw.pop("cfg")
    argv = harbor_argv(
        cfg,
        path=kw["path"],
        solver=kw["solver"],
        n_attempts=kw["n_attempts"],
        n_concurrent=kw["n_concurrent"],
        job_name=kw["job_name"],
    )
    run_harbor(argv)
    job_dir = cfg.jobs_dir / kw["job_name"]
    raw: dict[str, Any] = {}
    result = job_dir / "result.json"
    if result.is_file():
        try:
            loaded = json.loads(result.read_text(encoding="utf-8"))
            if isinstance(loaded, dict):
                raw = loaded
        except json.JSONDecodeError:
            raw = {}
    return HarborJobResult(job_dir=job_dir, trials=audit_job(job_dir), raw=raw)
