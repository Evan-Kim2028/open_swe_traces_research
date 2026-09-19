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
from openswe_traces.pipeline.author import (
    discover_task_unit_names,
    existing_author_units,
    mark_control,
    record_units,
    run_author,
)
from openswe_traces.pipeline.config import PipelineConfig, RepoSpec, load_config
from openswe_traces.pipeline.package import package_unit as default_package_unit
from openswe_traces.pipeline.package import packaged_levels
from openswe_traces.pipeline.prepare import BaselineError, prepare_repo
from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.solve import HarborJobResult, default_harbor, solve_unit
from openswe_traces.pipeline.state import DONE, FAILED, PAUSED, REJECTED, SKIPPED, PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget
from openswe_traces.pipeline.verifier import VerifierReject, run_verifier
from openswe_traces.pipeline_ext.calibration import AttemptRecord
from openswe_traces.pipeline_ext.controls import UnitMeta, control_status, pick_control
from openswe_traces.pipeline_ext.timeouts import TIMEOUT_CLASS

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
    hack_audit: Callable[..., Any] | None = None
    skip_hack_docker: bool = True


def setup_logging(cfg: PipelineConfig) -> Path:
    cfg.logs_dir.mkdir(parents=True, exist_ok=True)
    dest = cfg.logs_dir / "pipeline.log"
    dest.parent.mkdir(parents=True, exist_ok=True)
    root = logging.getLogger("openswe.pipeline")
    root.setLevel(logging.INFO)
    if not any(
        isinstance(h, logging.FileHandler) and getattr(h, "baseFilename", "") == str(dest)
        for h in root.handlers
    ):
        fmt = logging.Formatter("%(asctime)s %(levelname)s %(message)s")
        fh = logging.FileHandler(dest, encoding="utf-8")
        fh.setFormatter(fmt)
        sh = logging.StreamHandler()
        sh.setFormatter(fmt)
        root.addHandler(fh)
        root.addHandler(sh)
    return dest


def default_runtime(
    cfg: PipelineConfig, store: PipelineStore, sem: DevinSemaphore, budget: TokenBudget
) -> Runtime:
    runner = AgentRunner(
        cursor_env_file=cfg.cursor_env_file,
        semaphore=sem,
        budget=budget,
    )
    return Runtime(
        runner=runner,
        harbor=lambda **kw: default_harbor(kw.pop("cfg", cfg), **kw),
        prepare=lambda spec, c: prepare_repo(spec, c),
        skip_hack_docker=False,
    )


def _refresh_dashboard(store: PipelineStore, cfg: PipelineConfig) -> None:
    try:
        aggregate(store, cfg)
    except Exception as exc:  # noqa: BLE001 — dashboard must not kill the night
        log.warning("aggregate failed: %s", exc)


def _ingest_task_units(spec_name: str, cfg: PipelineConfig, store: PipelineStore) -> None:
    """Register units that already exist as Harbor L2 dirs (external Devin verifier)."""
    for name in discover_task_unit_names(cfg, spec_name):
        store.upsert_unit(spec_name, name, status="authored")


def _row_val(row: Any, key: str, default: Any = None) -> Any:
    try:
        val = row[key]
    except (KeyError, IndexError):
        return default
    return default if val is None else val


def _unit_metas(store: PipelineStore, repo: str) -> list[UnitMeta]:
    metas: list[UnitMeta] = []
    for u in store.list_units(repo):
        metas.append(
            UnitMeta(
                repo=repo,
                name=u["unit"],
                predicted_flip=_row_val(u, "predicted_flip"),
                n_lines=u["n_lines"] or 0,
                is_control=bool(_row_val(u, "is_control", False)),
            )
        )
    return metas


def _sort_control_first(store: PipelineStore, repo: str, rows: list[Any]) -> list[Any]:
    ctrl = pick_control(_unit_metas(store, repo), repo=repo)
    ctrl_name = ctrl.name if ctrl else None
    if ctrl_name:
        current = next((r["status"] for r in store.list_units(repo) if r["unit"] == ctrl_name), "authored")
        store.upsert_unit(repo, ctrl_name, is_control=True, status=current)

    def key(row: Any) -> tuple[int, str]:
        return (0 if row["unit"] == ctrl_name else 1, row["unit"])

    return sorted(rows, key=key)


def _attempt_records_for(store: PipelineStore, repo: str) -> list[AttemptRecord]:
    units = {u["unit"]: u for u in store.list_units(repo)}
    out: list[AttemptRecord] = []
    for t in store.list_trials(repo=repo, include_excluded=True):
        u = units.get(t.unit)
        timeout = bool(t.timeout) or t.audit_class in {TIMEOUT_CLASS, "infra"}
        out.append(
            AttemptRecord(
                repo=t.repo,
                unit=t.unit,
                solver=t.solver,
                level=t.level,
                attempt=t.attempt,
                passed=t.reward == 1.0
                and not timeout
                and t.audit_class != "hacked"
                and not t.excluded,
                timeout=timeout,
                audit_class=t.audit_class or "clean",
                is_control=bool(_row_val(u, "is_control", False)) if u is not None else False,
                valid_unit=u is not None and u["status"] != "rejected",
            )
        )
    return out


def _control_unit_name(store: PipelineStore, repo: str) -> str | None:
    ctrl = pick_control(_unit_metas(store, repo), repo=repo)
    return None if ctrl is None else ctrl.name


def _pause_repo(store: PipelineStore, repo: str, remaining: list[str], detail: str) -> None:
    store.upsert_repo(repo, PAUSED, error=detail)
    store.add_event("control", detail)
    for unit in remaining:
        store.mark_step(repo, "solve", SKIPPED, unit=unit, error=detail)
        store.mark_step(repo, "package", SKIPPED, unit=unit, error=detail)
        store.mark_step(repo, "verifier", SKIPPED, unit=unit, error=detail)
    log.warning("paused %s: %s", repo, detail)


def _run_verifier_unit(
    spec_name: str,
    unit: str,
    cfg: PipelineConfig,
    store: PipelineStore,
    rt: Runtime,
) -> None:
    with store.running(spec_name, "verifier", unit):
        run_verifier(
            spec_name,
            unit,
            cfg,
            store,
            runner=rt.runner,
            prove=rt.prove,
        )
        store.mark_step(spec_name, "verifier", DONE, unit=unit)


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
    _ingest_task_units(spec.name, cfg, store)
    if store.should_run(spec.name, "author"):
        with store.running(spec.name, "author"):
            units = run_author(spec.name, cfg, store, runner=rt.runner, n_units=n)
            store.mark_step(
                spec.name, "author", DONE, payload={"units": [u.get("name") for u in units]}
            )
            log.info("author done %s n=%s", spec.name, len(units))
    else:
        log.info("author skip %s", spec.name)
        leftover = existing_author_units(cfg, spec.name)
        if leftover:
            record_units(spec.name, store, leftover, author_backend=cfg.author_backend)
            mark_control(store, spec.name, leftover, author_backend=cfg.author_backend)

    unit_rows = list(store.list_units(spec.name))
    unit_rows = _sort_control_first(store, spec.name, unit_rows)
    if max_units is not None:
        unit_rows = unit_rows[:max_units]
    ctrl_name = _control_unit_name(store, spec.name)

    for idx, urow in enumerate(unit_rows):
        unit = urow["unit"]
        if urow["status"] == REJECTED or (urow["rejected_rule"] if "rejected_rule" in urow.keys() else None):  # noqa: SIM118
            continue
        if store.should_run(spec.name, "verifier", unit):
            try:
                _run_verifier_unit(spec.name, unit, cfg, store, rt)
                log.info("verifier pass %s/%s", spec.name, unit)
            except VerifierReject as exc:
                store.mark_step(spec.name, "verifier", REJECTED, unit=unit, error=str(exc))
                log.info("verifier reject %s/%s %s", spec.name, unit, exc.rule_id)
                _refresh_dashboard(store, cfg)
                if unit == ctrl_name:
                    rest = [r["unit"] for r in unit_rows[idx + 1 :]]
                    _pause_repo(store, spec.name, rest, f"control {unit} rejected at verifier")
                    return
                continue
            except Exception as exc:
                store.mark_step(spec.name, "verifier", FAILED, unit=unit, error=str(exc)[:2000])
                log.exception("verifier error %s/%s", spec.name, unit)
                continue
        if store.should_run(spec.name, "package", unit):
            have = packaged_levels(cfg, spec.name, unit)
            if have:
                store.mark_step(
                    spec.name,
                    "package",
                    DONE,
                    unit=unit,
                    payload={"levels": sorted(have), "skipped": True},
                )
                store.upsert_unit(spec.name, unit, status="packaged")
                log.info("package skip %s/%s existing=%s", spec.name, unit, sorted(have))
            else:
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
                        hack_audit=rt.hack_audit,
                        skip_hack_docker=rt.skip_hack_docker,
                    )
                    store.upsert_unit(spec.name, unit, status="solved")
                    log.info("solve done %s/%s", spec.name, unit)
            except Exception as exc:
                store.mark_step(spec.name, "solve", FAILED, unit=unit, error=str(exc)[:2000])
                log.exception("solve error %s/%s", spec.name, unit)
        if unit == ctrl_name:
            st = control_status(
                spec.name, _unit_metas(store, spec.name), _attempt_records_for(store, spec.name)
            )
            if not st.ok:
                rest = [r["unit"] for r in unit_rows[idx + 1 :]]
                _pause_repo(store, spec.name, rest, st.detail)
                _refresh_dashboard(store, cfg)
                return
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
    cfg.authored_dir.mkdir(parents=True, exist_ok=True)
    cfg.tasks_dir.mkdir(parents=True, exist_ok=True)
    cfg.jobs_dir.mkdir(parents=True, exist_ok=True)
    own_store = store is None
    store = store or PipelineStore(cfg.state_db)
    sem = DevinSemaphore(cfg.devin_slots_path, cfg.devin_slots)
    budget = TokenBudget(store, cfg.composer_token_cap)
    rt = runtime or default_runtime(cfg, store, sem, budget)
    selected = list(repos) if repos is not None else list(cfg.repos)
    log.info(
        "pipeline start repos=%s cap=%s slots=%s",
        [r.name for r in selected],
        cfg.composer_token_cap,
        cfg.devin_slots,
    )
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
            f"devin slots: {cfg.devin_slots} ({cfg.devin_slots_path})",
            f"solver order: {list(cfg.solver_order)}",
            f"climb levels: {list(cfg.climb_levels)}",
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
