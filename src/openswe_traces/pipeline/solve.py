"""Harbor solve with adaptive ladder (C6), B9 hack audit, and backend timeouts."""

from __future__ import annotations

import json
import os
import re
import subprocess
from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import load_env_file
from openswe_traces.pipeline.audit import TrialAudit, audit_class, audit_job
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.package import ensure_level
from openswe_traces.pipeline.prepare import image_tag
from openswe_traces.pipeline.resources import harbor_concurrency, run_cleanup, wait_for_load
from openswe_traces.pipeline.safety import apply_solver_network, assert_harbor_safe
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget
from openswe_traces.pipeline_ext.hack_audit import HackVerdict, audit_passing_attempt
from openswe_traces.pipeline_ext.ladder_policy import (
    FlipResult,
    LevelAttempt,
    TaskState,
    flip_point,
    next_actions,
)
from openswe_traces.pipeline_ext.timeouts import (
    TIMEOUT_CLASS,
    agent_timeout_sec,
    session_timeout_sec,
)


@dataclass
class HarborJobResult:
    job_dir: Path
    trials: list[TrialAudit] = field(default_factory=list)
    raw: dict[str, Any] = field(default_factory=dict)
    timed_out: bool = False


HackAuditFn = Callable[..., HackVerdict]


def solver_pair(cfg: PipelineConfig, name: str) -> tuple[str, str]:
    if name == "devin":
        return cfg.devin_harbor_agent, cfg.devin_harbor_model
    return cfg.cursor_harbor_agent, cfg.cursor_harbor_model


def solver_backends(cfg: PipelineConfig) -> tuple[str, ...]:
    return cfg.solver_order or cfg.solver_backends


def apply_agent_timeout(task_dir: Path, seconds: int) -> None:
    """Set ``[agent] timeout_sec`` in task.toml for this backend."""
    path = task_dir / "task.toml"
    if not path.is_file():
        return
    text = path.read_text(encoding="utf-8")
    if "[agent]" not in text:
        return
    head, rest = text.split("[agent]", 1)
    rest, n = re.subn(r"(timeout_sec\s*=\s*)\S+", rf"\g<1>{int(seconds)}", rest, count=1)
    if n == 0:
        # insert after [agent]
        rest = f"\ntimeout_sec = {int(seconds)}\n" + rest
    path.write_text(head + "[agent]" + rest, encoding="utf-8")


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
        "--timeout-multiplier",
        "1.0",
        "--jobs-dir",
        str(cfg.jobs_dir),
        "--job-name",
        job_name,
        "--yes",
    ]


DEVIN_CREDENTIALS = Path.home() / ".local" / "share" / "devin" / "credentials.toml"


def devin_api_key(path: Path | str = DEVIN_CREDENTIALS) -> str | None:
    """Read `windsurf_api_key` from the Devin CLI credentials file. Never log the value."""
    path = Path(path)
    if not path.is_file():
        return None
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw.strip()
        if line.startswith("windsurf_api_key") and "=" in line:
            return line.split("=", 1)[1].strip().strip('"').strip("'") or None
    return None


def solver_env(cfg: PipelineConfig, solver: str) -> dict[str, str]:
    """Environment for a Harbor solve: cursor env file for Composer, DEVIN_API_KEY for Devin."""
    env = os.environ.copy()
    if solver == "devin":
        if not env.get("DEVIN_API_KEY"):
            key = devin_api_key()
            if key:
                env["DEVIN_API_KEY"] = key
    else:
        load_env_file(cfg.cursor_env_file, environ=env)
    return env


def _env_test_signature(task: Path) -> tuple[tuple[str, str], ...]:
    """(relpath, sha256) of every *_test.go under environment/ — what the solver sees."""
    import hashlib

    env = task / "environment"
    if not env.is_dir():
        return ()
    out = []
    for f in sorted(env.rglob("*_test.go")):
        out.append((str(f.relative_to(env)), hashlib.sha256(f.read_bytes()).hexdigest()))
    return tuple(out)


def duplicate_of_lower_level(task: Path, lower: Path) -> bool:
    """True when L6 ships exactly the test files L5 already shipped (single-file suites)."""
    sig = _env_test_signature(task)
    return bool(sig) and sig == _env_test_signature(lower)


def copy_level_trials(
    store: PipelineStore, *, repo: str, unit: str, solver: str, src_level: int, dst_level: int
) -> int:
    """Mirror src_level trials as dst_level rows (audit_class 'dup_l5') so the policy sees L6 = L5."""
    n = 0
    for t in store.list_trials(repo=repo, unit=unit, include_excluded=True):
        if t.solver != solver or t.level != src_level:
            continue
        store.add_trial(
            repo=repo,
            unit=unit,
            level=dst_level,
            solver=solver,
            attempt=t.attempt,
            reward=t.reward,
            tokens_in=0,
            tokens_out=0,
            audit_class="dup_l5",
            wall_minutes=0.0,
            job_dir=t.job_dir,
            excluded=bool(t.excluded),
            timeout=bool(getattr(t, "timeout", False)),
        )
        n += 1
    return n


def run_harbor(
    argv: list[str],
    *,
    timeout: int | None = None,
    env: dict[str, str] | None = None,
    run: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run,
) -> subprocess.CompletedProcess[str]:
    return run(argv, capture_output=True, text=True, timeout=timeout, check=False, env=env)


def trial_timed_out(audit: TrialAudit, *, job_timed_out: bool = False) -> bool:
    if job_timed_out:
        return True
    result = audit.result or {}
    if result.get("timed_out") or result.get("timeout"):
        return True
    blob = " ".join(
        str(result.get(k) or "")
        for k in ("exception", "error", "status", "failure_reason", "agent_result")
    ).lower()
    return "timeout" in blob or "timed out" in blob


def task_state_from_store(store: PipelineStore, repo: str, unit: str, solver: str) -> TaskState:
    attempts: list[LevelAttempt] = []
    for t in store.list_trials(repo=repo, unit=unit, include_excluded=True):
        if t.solver != solver:
            continue
        if t.excluded or t.audit_class in {"hacked", "contaminated", "checksum"}:
            continue
        timeout = bool(t.timeout) or t.audit_class == TIMEOUT_CLASS
        passed = t.reward == 1.0 and not timeout
        attempts.append(LevelAttempt(level=t.level, passed=passed, timeout=timeout))
    return TaskState(attempts=tuple(attempts))


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
    job_timed_out: bool = False,
    hack_audit: HackAuditFn | None = None,
    skip_hack_docker: bool = True,
    task_dir: Path | None = None,
) -> list[TrialAudit]:
    audit_fn = hack_audit or audit_passing_attempt
    for i, audit in enumerate(audits, start=1):
        timed_out = trial_timed_out(audit, job_timed_out=job_timed_out)
        klass = TIMEOUT_CLASS if timed_out else audit_class(audit.verdict)
        excluded = (klass in {"contaminated", "checksum"} and rerun) or klass == "hacked"
        reward = audit.reward
        if timed_out:
            excluded = False
            klass = TIMEOUT_CLASS
        elif klass == "clean" and reward == 1.0:
            verdict = audit_fn(
                image=image_tag(repo),
                trial_dir=audit.trial_dir,
                task_dir=task_dir,
                skip_docker=skip_hack_docker,
            )
            if not verdict.passed:
                klass = "hacked"
                excluded = True
                store.add_event(
                    "hack_audit",
                    f"B9 hard fail {repo}/{unit}-L{level} {audit.task}: {list(verdict.hard_fails)[:4]}",
                )
            elif verdict.flags:
                store.add_event(
                    "hack_audit",
                    f"B9 flags {repo}/{unit}-L{level}: {list(verdict.flags)[:4]}",
                )
        if klass == "checksum":
            store.add_event("rule", f"B1 checksum-guard {repo}/{unit}-L{level} {audit.task}")
        store.add_trial(
            repo=repo,
            unit=unit,
            level=level,
            solver=solver,
            attempt=i,
            reward=reward,
            tokens_in=audit.tokens_in,
            tokens_out=audit.tokens_out,
            audit_class=klass,
            wall_minutes=audit.wall_minutes,
            job_dir=str(job_dir),
            excluded=excluded or (klass == "contaminated" and rerun),
            timeout=timed_out,
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
    hack_audit: HackAuditFn | None = None,
    skip_hack_docker: bool = True,
) -> dict[str, Any]:
    """Adaptive climb L2→L5→L6; confirm at the flip; B9 every pass."""
    wait = wait_load or (lambda: wait_for_load(mult=cfg.load_mult))
    clean = cleanup or (lambda: run_cleanup(cfg.cleanup_script))
    order = solver_backends(cfg)
    solver = budget.choose_solver(order)
    if solver != order[0]:
        store.add_event(
            "token_cap", f"composer cap reached ({budget.used}/{budget.cap}); solver={solver}"
        )
        store.set_meta("solver_backend", solver)

    launches = 0
    while launches < 24:
        if budget.exhausted() and solver != "devin":
            nxt = budget.choose_solver(order)
            if nxt != solver:
                store.add_event("token_cap", f"switch solver {solver} -> {nxt}")
                store.set_meta("solver_backend", nxt)
                solver = nxt
                continue
        state = task_state_from_store(store, repo, unit, solver)
        required = [a for a in next_actions(state) if not a.optional]
        if not required:
            break
        action = required[0]
        launches += 1
        level, n_att = action.level, action.n_attempts
        task = ensure_level(repo, unit, cfg, level)
        if level == 6 and duplicate_of_lower_level(task, ensure_level(repo, unit, cfg, 5)):
            n = copy_level_trials(
                store, repo=repo, unit=unit, solver=solver, src_level=5, dst_level=6
            )
            store.add_event(
                "ladder",
                f"{repo}/{unit}: L6 test files identical to L5; mirrored {n} L5 trials as L6",
            )
            continue
        apply_agent_timeout(task, agent_timeout_sec(solver))
        wait()
        n_conc = harbor_concurrency(cfg, solver, semaphore, host=host)
        hold_timeout = session_timeout_sec(solver)
        if solver == "devin":
            n_hold = max(1, n_conc)
            with semaphore.hold(kind="harbor", n=n_hold, timeout=hold_timeout):
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
                    hack_audit=hack_audit,
                    skip_hack_docker=skip_hack_docker,
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
                hack_audit=hack_audit,
                skip_hack_docker=skip_hack_docker,
            )
        clean()
        _ = job

    result = flip_point(task_state_from_store(store, repo, unit, solver))
    payload = {
        "flip": result.level,
        "confirmed": result.confirmed,
        "solver": solver,
        "notes": result.evidence.notes,
        "candidate": result.evidence.candidate,
    }
    store.mark_step(repo, "solve", "done", unit=unit, payload=payload)
    return payload


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
    hack_audit: HackAuditFn | None = None,
    skip_hack_docker: bool = True,
) -> HarborJobResult:
    job_name = f"{repo}-{unit}-L{level}-{solver}-n{n_att}-k{len(list(store.list_trials(repo=repo, unit=unit)))}"
    apply_solver_network(task, solver)
    assert_harbor_safe(task, solver=solver)
    job = harbor(
        cfg=cfg,
        path=task,
        solver=solver,
        n_attempts=n_att,
        n_concurrent=n_conc,
        job_name=job_name,
        timeout_sec=n_att * agent_timeout_sec(solver) + 600,
    )
    job_timed_out = bool(getattr(job, "timed_out", False))
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
                audit_class=TIMEOUT_CLASS if job_timed_out else "infra",
                job_dir=str(job.job_dir),
                timeout=job_timed_out,
            )
        return job
    _record_audits(
        store,
        budget,
        repo=repo,
        unit=unit,
        level=level,
        solver=solver,
        job_dir=job.job_dir,
        audits=audits,
        job_timed_out=job_timed_out,
        hack_audit=hack_audit,
        skip_hack_docker=skip_hack_docker,
        task_dir=task,
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
            timeout_sec=n_att * agent_timeout_sec(solver) + 600,
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
            job_timed_out=bool(getattr(job2, "timed_out", False)),
            hack_audit=hack_audit,
            skip_hack_docker=skip_hack_docker,
            task_dir=task,
        )
        if _contaminated(audits2):
            store.add_event("audit", f"contaminated after rerun; excluded {job_name}")
        return job2
    return job


def default_harbor(cfg: PipelineConfig | None = None, **kw: Any) -> HarborJobResult:
    if cfg is None:
        cfg = kw.pop("cfg")
    timeout_sec = kw.pop("timeout_sec", None)
    path = Path(kw["path"])
    solver = str(kw["solver"])
    apply_solver_network(path, solver)
    assert_harbor_safe(path, solver=solver)
    argv = harbor_argv(
        cfg,
        path=path,
        solver=solver,
        n_attempts=kw["n_attempts"],
        n_concurrent=kw["n_concurrent"],
        job_name=kw["job_name"],
    )
    timed_out = False
    try:
        run_harbor(argv, timeout=timeout_sec, env=solver_env(cfg, solver))
    except subprocess.TimeoutExpired:
        timed_out = True
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
    return HarborJobResult(job_dir=job_dir, trials=audit_job(job_dir), raw=raw, timed_out=timed_out)


def flip_from_store(store: PipelineStore, repo: str, unit: str, solver: str) -> FlipResult:
    return flip_point(task_state_from_store(store, repo, unit, solver))
