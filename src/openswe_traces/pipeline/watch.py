"""Solve-as-verified: discover L2 units ready for Harbor and launch solve_unit."""

from __future__ import annotations

import json
import logging
import os
import re
import shlex
import subprocess
import time
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import load_env_file
from openswe_traces.pipeline.aggregate import aggregate
from openswe_traces.pipeline.audit import audit_job
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.package import packaged_levels, task_dir
from openswe_traces.pipeline.resources import docker_n_concurrent, run_cleanup, wait_for_load
from openswe_traces.pipeline.safety import TaskSafetyError, apply_solver_network, assert_harbor_safe
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.solve import HarborJobResult, default_harbor, harbor_argv, solve_unit
from openswe_traces.pipeline.state import DONE, REJECTED, RUNNING, SKIPPED, PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget

log = logging.getLogger("openswe.pipeline.watch")

L2_NAME_RE = re.compile(r"^(.+?)[-_]L2$")
SOLVE_BUSY = frozenset({DONE, SKIPPED, REJECTED, RUNNING})
UNIT_BUSY = frozenset({"solved", "solving", "running"})
PROOF_GOLD = ("gold_restore", "gold_pass", "gold_passes")
PROOF_BUGGY = ("buggy_fails",)
PROOF_CHEAT = ("cheat_rejected",)
VPS_TASKS_REL = "openswe/pipeline_tasks"
VPS_JOBS_REL = "openswe/pipeline_jobs"

RunFn = Callable[..., subprocess.CompletedProcess[str]]
HarborFn = Callable[..., HarborJobResult]


def setup_watch_logging(cfg: PipelineConfig) -> Path:
    cfg.logs_dir.mkdir(parents=True, exist_ok=True)
    dest = cfg.logs_dir / "solve_watch.log"
    root = logging.getLogger("openswe.pipeline.watch")
    root.setLevel(logging.INFO)
    if not any(
        isinstance(h, logging.FileHandler) and getattr(h, "baseFilename", "") == str(dest)
        for h in root.handlers
    ):
        fmt = logging.Formatter("%(asctime)s %(levelname)s %(message)s")
        fh = logging.FileHandler(dest, encoding="utf-8")
        fh.setFormatter(fmt)
        root.addHandler(fh)
        sh = logging.StreamHandler()
        sh.setFormatter(fmt)
        root.addHandler(sh)
    return dest


def _load_validation(path: Path) -> dict[str, Any]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def _verdict_map(data: dict[str, Any]) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    for row in data.get("rule_verdicts") or []:
        if not isinstance(row, dict):
            continue
        rid = str(row.get("rule_id") or "")
        if rid:
            out[rid] = row
    return out


def _row_passed(row: dict[str, Any] | None) -> bool:
    if not row:
        return False
    if row.get("skipped"):
        return False
    return bool(row.get("passed"))


def _flag(data: dict[str, Any], names: tuple[str, ...]) -> bool:
    checks = data.get("checks") if isinstance(data.get("checks"), dict) else {}
    for name in names:
        if name in checks:
            return bool(checks[name])
        if name in data:
            return bool(data[name])
    return False


def proof_flags_ok(data: dict[str, Any]) -> bool:
    """Packager image-proof flags that stand in for A1/A8/A3 verdicts."""
    return _flag(data, PROOF_GOLD) and _flag(data, PROOF_BUGGY) and _flag(data, PROOF_CHEAT)


def b4_passed(data: dict[str, Any]) -> bool:
    vm = _verdict_map(data)
    if "B4" in vm:
        return _row_passed(vm["B4"])
    return _flag(data, ("blackbox_hygiene",))


def unit_verified(data: dict[str, Any]) -> bool:
    """B4 pass and (A1/A8/A3 pass or packager proof flags)."""
    if not b4_passed(data):
        return False
    vm = _verdict_map(data)
    rules_ok = all(_row_passed(vm.get(rid)) for rid in ("A1", "A8", "A3"))
    return rules_ok or proof_flags_ok(data)


def iter_l2_dirs(tasks_dir: Path) -> list[tuple[str, str, Path]]:
    """``(repo, unit, l2_dir)`` for ``<repo>/<unit>-L2`` and ``<repo>/<unit>_L2``."""
    root = Path(tasks_dir)
    if not root.is_dir():
        return []
    out: list[tuple[str, str, Path]] = []
    for repo_dir in sorted(root.iterdir()):
        if not repo_dir.is_dir():
            continue
        repo = repo_dir.name
        for child in sorted(repo_dir.iterdir()):
            if not child.is_dir():
                continue
            m = L2_NAME_RE.match(child.name)
            if not m:
                continue
            yield_path = child / "validation.json"
            if not yield_path.is_file():
                continue
            out.append((repo, m.group(1), child))
    return out


def discover_verified_units(cfg: PipelineConfig) -> list[tuple[str, str, Path]]:
    found: list[tuple[str, str, Path]] = []
    for repo, unit, l2 in iter_l2_dirs(cfg.tasks_dir):
        data = _load_validation(l2 / "validation.json")
        if unit_verified(data):
            found.append((repo, unit, l2))
    return found


def already_solving_or_solved(store: PipelineStore, repo: str, unit: str) -> bool:
    step = store.step(repo, "solve", unit)
    if step is not None and step.status in SOLVE_BUSY:
        return True
    for row in store.list_units(repo):
        if row["unit"] == unit and str(row["status"] or "") in UNIT_BUSY:
            return True
    return False


def cursor_api_key(env_file: Path | str | None) -> str:
    """Read CURSOR_API_KEY. Never log the value."""
    env: dict[str, str] = {}
    if env_file:
        load_env_file(env_file, environ=env)
    key = env.get("CURSOR_API_KEY") or os.environ.get("CURSOR_API_KEY") or ""
    return key


def _rsync(src: str, dest: str, *, run: RunFn, extra: list[str] | None = None) -> None:
    argv = ["rsync", "-az", "--exclude", ".git", *(extra or []), src, dest]
    proc = run(argv, capture_output=True, text=True, timeout=600, check=False)
    if proc.returncode != 0:
        raise RuntimeError(f"rsync failed: {(proc.stderr or proc.stdout or '')[-1500:]}")


def _ssh(
    host: str,
    remote: str,
    *,
    run: RunFn,
    env: dict[str, str] | None = None,
    timeout: int | None = None,
) -> subprocess.CompletedProcess[str]:
    argv = ["ssh", host, "--"]
    if env:
        argv.append("env")
        for key, value in env.items():
            argv.append(f"{key}={value}")
        argv.extend(["bash", "-lc", remote])
    else:
        argv.extend(["bash", "-lc", remote])
    return run(argv, capture_output=True, text=True, timeout=timeout, check=False)


def remeasure_timing_gate(
    ssh_host: str,
    remote_task: str,
    *,
    run: RunFn = subprocess.run,
) -> str:
    quoted = shlex.quote(remote_task)
    remote = (
        f"t={quoted}; "
        'if [ -f "$t/tests/measure_gold.sh" ]; then '
        'img=measure-$(basename "$t" | tr "A-Z" "a-z"); '
        'docker build -q -t "$img" "$t/environment" >/dev/null 2>&1 || exit 0; '
        'docker run --rm -v "$t/tests:/task/tests" "$img" '
        "bash -c 'cd /app && bash /task/tests/measure_gold.sh'; "
        "fi"
    )
    proc = _ssh(ssh_host, remote, run=run, timeout=1800)
    return ((proc.stdout or "") + (proc.stderr or "")).strip()


def vps_harbor(
    cfg: PipelineConfig,
    **kw: Any,
) -> HarborJobResult:
    """Rsync the unit's task dirs to lake-vps, run Harbor there, rsync the job back."""
    path: Path = Path(kw["path"])
    solver: str = str(kw["solver"])
    n_attempts: int = int(kw["n_attempts"])
    n_concurrent: int = int(kw["n_concurrent"])
    job_name: str = str(kw["job_name"])
    timeout_sec = kw.get("timeout_sec")
    run: RunFn = kw.get("run") or subprocess.run
    apply_solver_network(path, solver)
    assert_harbor_safe(path, solver=solver)
    ssh_host = cfg.host("vps").ssh or "lake-vps"
    repo = path.parent.name
    remote_task = f"{VPS_TASKS_REL}/{repo}/{path.name}"
    local_parent = str(path.parent) + "/"
    remote_parent = f"{ssh_host}:{VPS_TASKS_REL}/{repo}/"
    _rsync(local_parent, remote_parent, run=run)
    measured = remeasure_timing_gate(ssh_host, f"~/{remote_task}", run=run)
    if measured:
        log.info("vps measure_gold %s: %s", path.name, measured[-400:])
    remote_argv = harbor_argv(
        cfg,
        path=path,
        solver=solver,
        n_attempts=n_attempts,
        n_concurrent=n_concurrent,
        job_name=job_name,
    )
    try:
        jidx = remote_argv.index("--jobs-dir")
        remote_argv[jidx + 1] = f"~/{VPS_JOBS_REL}"
    except ValueError:
        remote_argv.extend(["--jobs-dir", f"~/{VPS_JOBS_REL}"])
    try:
        pidx = remote_argv.index("--path")
        remote_argv[pidx + 1] = f"~/{remote_task}"
    except ValueError:
        pass
    quoted: list[str] = []
    for arg in remote_argv:
        if arg.startswith("~/"):
            quoted.append(arg)
        else:
            quoted.append(shlex.quote(arg))
    remote_cmd = f"mkdir -p ~/{VPS_JOBS_REL} && " + " ".join(quoted)
    env: dict[str, str] = {}
    if solver != "devin":
        key = cursor_api_key(cfg.cursor_env_file)
        if not key:
            raise RuntimeError("CURSOR_API_KEY missing; export from eval_tasks/.env in the ssh command")
        env["CURSOR_API_KEY"] = key
    timed_out = False
    try:
        _ssh(ssh_host, remote_cmd, run=run, env=env or None, timeout=timeout_sec)
    except subprocess.TimeoutExpired:
        timed_out = True
    local_job = cfg.jobs_dir / "vps" / job_name
    local_job.mkdir(parents=True, exist_ok=True)
    _rsync(
        f"{ssh_host}:{VPS_JOBS_REL}/{job_name}/",
        str(local_job) + "/",
        run=run,
    )
    raw: dict[str, Any] = {}
    result = local_job / "result.json"
    if result.is_file():
        try:
            loaded = json.loads(result.read_text(encoding="utf-8"))
            if isinstance(loaded, dict):
                raw = loaded
        except json.JSONDecodeError:
            raw = {}
    return HarborJobResult(
        job_dir=local_job, trials=audit_job(local_job), raw=raw, timed_out=timed_out
    )


def make_harbor(cfg: PipelineConfig, host: str | None) -> HarborFn:
    chosen = host or cfg.default_host

    def _local(**kw: Any) -> HarborJobResult:
        return default_harbor(kw.pop("cfg", cfg), **kw)

    def _vps(**kw: Any) -> HarborJobResult:
        kw.pop("cfg", None)
        return vps_harbor(cfg, **kw)

    return _vps if chosen == "vps" else _local


def run_solve_unit(
    repo: str,
    unit: str,
    cfg: PipelineConfig,
    *,
    store: PipelineStore | None = None,
    harbor: HarborFn | None = None,
    host: str | None = None,
    semaphore: DevinSemaphore | None = None,
    budget: TokenBudget | None = None,
    cleanup: Callable[[], str] | None = None,
    wait_load: Callable[[], None] | None = None,
    skip_hack_docker: bool = False,
) -> dict[str, Any]:
    """Run adaptive solve_unit for one packaged unit; aggregate results.md after."""
    own = store is None
    store = store or PipelineStore(cfg.state_db)
    try:
        have = packaged_levels(cfg, repo, unit)
        if 2 not in have and not have:
            raise FileNotFoundError(f"not packaged: {repo}/{unit}")
        l2 = task_dir(cfg, repo, unit, 2)
        if not l2.is_dir():
            # external verifier may have written unit_L2
            alt = cfg.tasks_dir / repo / f"{unit}_L2"
            if alt.is_dir():
                l2 = alt
        solver = (cfg.solver_order or ("cursor",))[0]
        if l2.is_dir() and (l2 / "task.toml").is_file():
            apply_solver_network(l2, solver)
            assert_harbor_safe(l2, solver=solver)
        sem = semaphore or DevinSemaphore(cfg.devin_slots_path, cfg.devin_slots)
        tok = budget or TokenBudget(store, cfg.composer_token_cap)
        harbor_fn = harbor or make_harbor(cfg, host)
        store.upsert_unit(repo, unit, status="solving")
        store.upsert_repo(repo, "running")
        with store.running(repo, "solve", unit):
            payload = solve_unit(
                repo,
                unit,
                cfg,
                store,
                budget=tok,
                semaphore=sem,
                harbor=harbor_fn,
                cleanup=cleanup or (lambda: run_cleanup(cfg.cleanup_script)),
                wait_load=wait_load or (lambda: wait_for_load(mult=cfg.load_mult)),
                host=host or cfg.default_host,
                skip_hack_docker=skip_hack_docker,
            )
        store.upsert_unit(repo, unit, status="solved")
        aggregate(store, cfg)
        return payload
    finally:
        if own:
            store.close()


def solve_watch(
    cfg: PipelineConfig,
    *,
    interval: int = 300,
    host: str | None = None,
    store: PipelineStore | None = None,
    harbor: HarborFn | None = None,
    sleep_fn: Callable[[float], None] = time.sleep,
    max_cycles: int | None = None,
    skip_hack_docker: bool = False,
    wait_load: Callable[[], None] | None = None,
    cleanup: Callable[[], str] | None = None,
) -> int:
    """Poll tasks/*/ *L2/validation.json and launch solve-unit for newly verified units."""
    setup_watch_logging(cfg)
    chosen = host or cfg.default_host
    slots = docker_n_concurrent(cfg, chosen)
    own = store is None
    store = store or PipelineStore(cfg.state_db)
    sem = DevinSemaphore(cfg.devin_slots_path, cfg.devin_slots)
    budget = TokenBudget(store, cfg.composer_token_cap)
    harbor_fn = harbor or make_harbor(cfg, chosen)
    launched = 0
    cycle = 0
    log.info("solve-watch start interval=%s host=%s docker_slots=%s", interval, chosen, slots)
    try:
        while True:
            cycle += 1
            ready = discover_verified_units(cfg)
            log.info("scan cycle=%s verified=%s docker_slots=%s", cycle, [(r, u) for r, u, _ in ready], slots)
            for repo, unit, l2 in ready:
                if already_solving_or_solved(store, repo, unit):
                    continue
                log.info("launch solve-unit %s/%s from %s", repo, unit, l2)
                try:
                    run_solve_unit(
                        repo,
                        unit,
                        cfg,
                        store=store,
                        harbor=harbor_fn,
                        host=chosen,
                        semaphore=sem,
                        budget=budget,
                        cleanup=cleanup,
                        wait_load=wait_load,
                        skip_hack_docker=skip_hack_docker,
                    )
                    launched += 1
                    log.info("solve-unit done %s/%s", repo, unit)
                except TaskSafetyError as exc:
                    log.error("refuse %s/%s: %s", repo, unit, exc)
                    store.mark_step(repo, "solve", "failed", unit=unit, error=str(exc)[:2000])
                    store.add_event("safety", f"{repo}/{unit}: {exc}")
                except Exception as exc:
                    log.exception("solve-unit error %s/%s", repo, unit)
                    store.mark_step(repo, "solve", "failed", unit=unit, error=str(exc)[:2000])
                try:
                    aggregate(store, cfg)
                except Exception as exc:  # noqa: BLE001
                    log.warning("aggregate failed: %s", exc)
            if max_cycles is not None and cycle >= max_cycles:
                break
            sleep_fn(float(interval))
    finally:
        if own:
            store.close()
    return launched
