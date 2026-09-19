"""AUTHOR stage: one batched session, no tests."""

from __future__ import annotations

import json
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import AgentResult, AgentRunner
from openswe_traces.pipeline.briefs import author_brief
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.state import PipelineStore

REQUIRED = ("api.md", "contract.md", "bugreport.md", "gold.patch", "cheat.patch")


def author_root(cfg: PipelineConfig, repo: str) -> Path:
    return cfg.work_dir / repo / "author_batch"


def discover_units(batch: Path) -> list[dict[str, Any]]:
    manifest = batch / "units.json"
    rows: list[dict[str, Any]] = []
    if manifest.is_file():
        try:
            data = json.loads(manifest.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            data = []
        if isinstance(data, list):
            rows = [r for r in data if isinstance(r, dict) and r.get("name")]
    if rows:
        return rows
    units_dir = batch / "units"
    if not units_dir.is_dir():
        return []
    for child in sorted(units_dir.iterdir()):
        author = child / "_author"
        if author.is_dir() and all((author / name).is_file() for name in REQUIRED):
            closure = {}
            cpath = author / "closure.json"
            if cpath.is_file():
                try:
                    closure = json.loads(cpath.read_text(encoding="utf-8"))
                except json.JSONDecodeError:
                    closure = {}
            rows.append(
                {
                    "name": child.name,
                    "dir": str(author.relative_to(batch)),
                    "family": closure.get("family") or "",
                    "n_files": closure.get("n_files") or len(closure.get("files") or []),
                    "n_lines": closure.get("n_lines") or closure.get("lines") or 0,
                    "closure": closure,
                }
            )
    return rows


def unit_complete(author_dir: Path) -> bool:
    return all((author_dir / name).is_file() for name in REQUIRED)


def run_author(
    repo: str,
    cfg: PipelineConfig,
    store: PipelineStore,
    *,
    runner: AgentRunner,
    n_units: int | None = None,
    run: Callable[..., AgentResult] | None = None,
) -> list[dict[str, Any]]:
    n = n_units if n_units is not None else cfg.units_per_author_batch
    batch = author_root(cfg, repo)
    batch.mkdir(parents=True, exist_ok=True)
    existing = discover_units(batch)
    if existing:
        _record_units(repo, store, batch, existing)
        return existing
    tree = cfg.work_dir / repo / "tree"
    brief = author_brief(
        n=n,
        minutes=cfg.author_minutes,
        repo=repo,
        tree=str(tree),
        out=str(batch),
    )
    invoke = run or runner.run_agent
    invoke(
        "author",
        brief,
        batch,
        cfg.author_model,
        backend=cfg.author_backend,
        timeout_sec=cfg.author_minutes * 60,
    )
    found = discover_units(batch)
    if not found:
        raise RuntimeError(f"author session produced no units under {batch}")
    _record_units(repo, store, batch, found)
    return found[:n]


def _record_units(repo: str, store: PipelineStore, batch: Path, rows: list[dict[str, Any]]) -> None:
    for row in rows:
        name = str(row["name"])
        author = batch / str(row.get("dir") or f"units/{name}/_author")
        closure = row.get("closure") if isinstance(row.get("closure"), dict) else {}
        cpath = author / "closure.json"
        if cpath.is_file() and not closure:
            try:
                closure = json.loads(cpath.read_text(encoding="utf-8"))
            except json.JSONDecodeError:
                closure = {}
        files = closure.get("files") or []
        store.upsert_unit(
            repo,
            name,
            status="authored",
            family=str(row.get("family") or closure.get("family") or ""),
            closure=closure,
            n_files=int(row.get("n_files") or len(files) or 0),
            n_lines=int(row.get("n_lines") or closure.get("lines") or closure.get("n_lines") or 0),
        )
