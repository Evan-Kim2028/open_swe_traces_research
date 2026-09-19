"""Orchestrate prepare → author → verifier → package → solve → aggregate. Resumable."""

from __future__ import annotations

import logging
import traceback
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import AgentRunner
from openswe_traces.pipeline.aggregate import aggregate, format_table
from openswe_traces.pipeline.author import run_author
from openswe_traces.pipeline.config import PipelineConfig, RepoSpec, load_config
from openswe_traces.pipeline.package import package_unit as default_package_unit
from openswe_traces.pipeline.prepare import BaselineError, prepare_repo
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.solve import HarborJobResult, default_harbor, solve_unit
from openswe_traces.pipeline.state import DONE, FAILED, REJECTED, PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget
from openswe_traces.pipeline.verifier import VerifierReject, run_verifier

log = logging.getLogger("openswe.pipeline")


@dataclass
class Runtime:
    """Injectable side effects so tests never launch Harbor / Docker / agents."""

    runner: AgentRunner
    harbor: Callable[..., HarborJobResult]
    prepare: Callable[..., dict[str, Any]]
    prove: Callable[..., dict[str, Any]] | None = None
    package: Callable[..., dict[int, Any]] | None = None
    cleanup: Callable[[], str] | None = None
    wait_load: Callable[[], None] | None = None
    after_job: Callable[[PipelineStore, PipelineConfig], None] | None = None


def setup_logging(cfg: PipelineConfig) -> Path:
    cfg.logs_dir.mkdir(parents=True, exist_ok=True)
    dest = cfg.logs_dir / "pipeline.log"
    dest.parent.mkdir(parents=True, exist_ok=True)
    root = logging.getLogger("openswe.pipeline")
    root.setLevel(logging.INFO)
    if not any(isinstance(h, logging.FileHandler) and getattr(h, "baseFilename", "") == str(dest) for h in root.handlers):
        fmt = logging.Formatter("%(asctime)s %(levelname)s %(message)s")
        fh = logging.FileHandler(dest, encoding="utf-8")
        fh.setFormatter(fmt)
        sh = logging.StreamHandler()
        sh.setFormatter(fmt)
        root.addHandler(fh)
        root.addHandler(sh)
    return dest


def default_runtime(cfg: PipelineConfig, store: PipelineStore, sem: DevinSemaphore, budget: TokenBudget) -> Runtime:
    runner = AgentRunner(
        cursor_env_file=cfg.cursor_env_file,
        semaphore=sem,
        budget=budget,
    )
    return Runtime(
        runner=runner,
        harbor=lambda **kw: default_harbor(kw.pop("cfg", cfg), **kw),
        prepare=lambda spec, c: prepare_repo(spec, c),
    )


def _refresh_dashboard(store: PipelineStore, cfg: PipelineConfig) -> None:
    try:
        aggregate(store, cfg)
    except Exception as exc:  # noqa: BLE001 — dashboard must not kill the night
        log.warning("aggregate failed: %s", exc)


def run_repo(
    spec: RepoSpec,
    cfg: PipelineConfig,
    store: PipelineStore,
    rt: Runtime,
    *,
    budget: TokenBudget,
    semaphore: DevinSemaphore,
    max_units: int | None = None,
    host: str | None = None,
) -> None:
    store.upsert_repo(spec.name, "running")
    if store.should_run(spec.name, "prepare"):
        with store.running(spec.name, "prepare"):
            try:
                payload = rt.prepare(spec, cfg)
            except BaselineError as exc:
                store.mark_step(spec.name, "prepare", FAILED, error=str(exc))
                store.upsert_repo(spec.name, FAILED, error=str(exc))
                log.error("prepare failed %s: %s", spec.name, exc)
                return
            store.mark_step(spec.name, "prepare", DONE, payload=payload)
            log.info("prepare done %s image=%s", spec.name, payload.get("image"))
    else:
        log.info("prepare skip %s", spec.name)

    n = max_units if max_units is not None else cfg.units_per_author_batch
    if store.should_run(spec.name, "author"):
        with store.running(spec.name, "author"):
            units = run_author(spec.name, cfg, store, runner=rt.runner, n_units=n)
            store.mark_step(spec.name, "author", DONE, payload={"units": [u.get("name") for u in units]})
            log.info("author done %s n=%s", spec.name, len(units))
    else:
        log.info("author skip %s", spec.name)

    unit_rows = [u for u in store.list_units(spec.name)]
    if max_units is not None:
        unit_rows = unit_rows[:max_units]
    for urow in unit_rows:
        unit = urow["unit"]
        if urow["status"] == REJECTED:
            continue
        if store.should_run(spec.name, "verifier", unit):
            try:
                with store.running(spec.name, "verifier", unit):
                    run_verifier(
                        spec.name,
                        unit,
                        cfg,
                        store,
                        runner=rt.runner,
                        prove=rt.prove,
                    )
                    store.mark_step(spec.name, "verifier", DONE, unit=unit)
                    log.info("verifier pass %s/%s", spec.name, unit)
            except VerifierReject as exc:
                store.mark_step(spec.name, "verifier", REJECTED, unit=unit, error=str(exc))
                log.info("verifier reject %s/%s %s", spec.name, unit, exc.rule_id)
                _refresh_dashboard(store, cfg)
                continue
            except Exception as exc:
                store.mark_step(spec.name, "verifier", FAILED, unit=unit, error=str(exc)[:2000])
                log.exception("verifier error %s/%s", spec.name, unit)
                continue
        if store.should_run(spec.name, "package", unit):
            try:
                with store.running(spec.name, "package", unit):
                    pack = rt.package or default_package_unit
                    built = pack(spec.name, unit, cfg)
                    store.mark_step(
                        spec.name,
                        "package",
                        DONE,
                        unit=unit,
                        payload={"levels": sorted(built)},
                    )
                    store.upsert_unit(spec.name, unit, status="packaged")
                    log.info("package %s/%s levels=%s", spec.name, unit, sorted(built))
            except Exception as exc:
                store.mark_step(spec.name, "package", FAILED, unit=unit, error=str(exc)[:2000])
                log.exception("package error %s/%s", spec.name, unit)
                continue
        if store.should_run(spec.name, "solve", unit):
            try:
                with store.running(spec.name, "solve", unit):
                    solve_unit(
                        spec.name,
                        unit,
                        cfg,
                        store,
                        budget=budget,
                        semaphore=semaphore,
                        harbor=rt.harbor,
                        cleanup=rt.cleanup,
                        wait_load=rt.wait_load,
                        host=host,
                    )
                    store.upsert_unit(spec.name, unit, status="solved")
                    log.info("solve done %s/%s", spec.name, unit)
            except Exception as exc:
                store.mark_step(spec.name, "solve", FAILED, unit=unit, error=str(exc)[:2000])
                log.exception("solve error %s/%s", spec.name, unit)
        _refresh_dashboard(store, cfg)
        if rt.after_job:
            rt.after_job(store, cfg)

    store.upsert_repo(spec.name, DONE)
    _refresh_dashboard(store, cfg)


def run_pipeline(
    cfg: PipelineConfig,
    *,
    store: PipelineStore | None = None,
    runtime: Runtime | None = None,
    repos: Sequence[RepoSpec] | None = None,
    max_units: int | None = None,
    host: str | None = None,
) -> Path:
    setup_logging(cfg)
    cfg.logs_dir.mkdir(parents=True, exist_ok=True)
    cfg.work_dir.mkdir(parents=True, exist_ok=True)
    cfg.tasks_dir.mkdir(parents=True, exist_ok=True)
    cfg.jobs_dir.mkdir(parents=True, exist_ok=True)
    own_store = store is None
    store = store or PipelineStore(cfg.state_db)
    sem = DevinSemaphore(cfg.devin_slots_path, cfg.devin_slots)
    budget = TokenBudget(store, cfg.composer_token_cap)
    rt = runtime or default_runtime(cfg, store, sem, budget)
    selected = list(repos) if repos is not None else list(cfg.repos)
    log.info("pipeline start repos=%s cap=%s", [r.name for r in selected], cfg.composer_token_cap)
    try:
        for spec in selected:
            try:
                run_repo(
                    spec,
                    cfg,
                    store,
                    rt,
                    budget=budget,
                    semaphore=sem,
                    max_units=max_units,
                    host=host or cfg.default_host,
                )
            except Exception:  # noqa: BLE001 — one repo must not abort the night
                log.error("repo %s crashed:\n%s", spec.name, traceback.format_exc())
                store.upsert_repo(spec.name, FAILED, error="uncaught")
        aggregate(store, cfg)
    finally:
        if own_store:
            store.close()
    return cfg.results_md


def dry_run(
    repo: str,
    units: int,
    *,
    cfg: PipelineConfig | None = None,
    store: PipelineStore | None = None,
    runtime: Runtime | None = None,
) -> str:
    cfg = cfg or load_config()
    spec = cfg.repo(repo)
    run_pipeline(cfg, store=store, runtime=runtime, repos=[spec], max_units=units)
    if store is None:
        store = PipelineStore(cfg.state_db)
        try:
            import pandas as pd

            from openswe_traces.pipeline.aggregate import task_rows

            df = pd.DataFrame(task_rows(store))
            return format_table(df)
        finally:
            store.close()
    import pandas as pd

    from openswe_traces.pipeline.aggregate import task_rows

    return format_table(pd.DataFrame(task_rows(store)))


def status_text(cfg: PipelineConfig | None = None, store: PipelineStore | None = None) -> str:
    cfg = cfg or load_config()
    own = store is None
    store = store or PipelineStore(cfg.state_db)
    try:
        lines = [
            f"state: {cfg.state_db}",
            f"composer tokens: {store.composer_tokens()} / {cfg.composer_token_cap}",
            f"solver override: {store.get_meta('solver_backend') or '(default order)'}",
            "",
        ]
        rows = store.status_rows()
        if not rows:
            lines.append("no repos recorded (pipeline has not started)")
            return "\n".join(lines) + "\n"
        for row in rows:
            lines.append(f"## {row['repo']}  [{row['status']}]")
            if row["error"]:
                lines.append(f"  error: {row['error']}")
            for step in row["steps"]:
                unit = step["unit"] or "-"
                err = f"  {step['error']}" if step["error"] else ""
                lines.append(f"  {step['stage']:10} {unit:20} {step['status']}{err}")
            for unit in row["units"]:
                extra = f" rule={unit['rejected_rule']}" if unit["rejected_rule"] else ""
                lines.append(f"  unit {unit['unit']}: {unit['status']}{extra}")
            lines.append("")
        return "\n".join(lines)
    finally:
        if own:
            store.close()
