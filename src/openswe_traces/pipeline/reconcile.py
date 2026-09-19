"""Record finished Harbor jobs that no trial row references (orphans from a restarted watcher)."""

from __future__ import annotations

import logging
import re
from pathlib import Path

from openswe_traces.pipeline.audit import audit_job
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.solve import _record_audits
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline.tokens import TokenBudget

log = logging.getLogger("openswe.pipeline.reconcile")

JOB_RE = re.compile(
    r"^(?P<unit>.+?)-L(?P<level>\d)-(?P<solver>devin|cursor)-n(?P<n>\d+)-k(?P<k>\d+)(?P<rerun>-rerun)?$"
)


def parse_job_name(name: str, repos: set[str]) -> dict[str, str] | None:
    """Split ``<repo>-<unit>-L<level>-<solver>-n<k>-k<idx>`` using the known repo names as prefixes."""
    for repo in sorted(repos, key=len, reverse=True):
        if not name.startswith(repo + "-"):
            continue
        m = JOB_RE.match(name[len(repo) + 1 :])
        if m:
            d = m.groupdict()
            d["repo"] = repo
            return d
    return None


def reconcile_jobs(
    cfg: PipelineConfig,
    store: PipelineStore,
    *,
    budget: TokenBudget | None = None,
    skip_hack_docker: bool = False,
) -> int:
    """Add trial rows for completed jobs with a trial result but no row. Returns rows added."""
    known = {t.job_dir for t in store.list_trials(include_excluded=True)}
    repos = {r.name for r in cfg.repos}
    budget = budget or TokenBudget(store, cfg.composer_token_cap)
    added = 0
    for job_dir in sorted(p for p in Path(cfg.jobs_dir).iterdir() if p.is_dir()):
        if str(job_dir) in known:
            continue
        meta = parse_job_name(job_dir.name, repos)
        if meta is None:
            continue
        audits = [a for a in audit_job(job_dir) if a.result]
        if not audits or any(a.reward is None and not a.result.get("finished_at") for a in audits):
            continue  # still running
        repo, unit, level, solver = meta["repo"], meta["unit"], int(meta["level"]), meta["solver"]
        task_dir = Path(cfg.tasks_dir) / repo / f"{unit}-L{level}"
        _record_audits(
            store,
            budget,
            repo=repo,
            unit=unit,
            level=level,
            solver=solver,
            job_dir=job_dir,
            audits=audits,
            rerun=bool(meta.get("rerun")),
            skip_hack_docker=skip_hack_docker,
            task_dir=task_dir if task_dir.is_dir() else None,
        )
        store.add_event("reconcile", f"recorded orphaned job {job_dir.name} ({len(audits)} trial)")
        # the parent solve-unit is gone; free the unit so the watcher relaunches and resumes the ladder
        store.upsert_unit(repo, unit, status="resume")
        log.info("reconciled orphaned job %s", job_dir.name)
        added += len(audits)
    return added
