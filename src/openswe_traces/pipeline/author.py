"""AUTHOR stage: ingest existing batches or run one session. Never writes tests."""

from __future__ import annotations

import json
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import AgentResult, AgentRunner
from openswe_traces.pipeline.briefs import author_brief
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline_ext.author_meta import parse_control_flag, parse_predicted_flip
from openswe_traces.pipeline_ext.controls import UnitMeta, pick_control
from openswe_traces.pipeline_ext.timeouts import session_timeout_sec

REQUIRED = ("api.md", "contract.md", "bugreport.md", "gold.patch", "cheat.patch")


def author_root(cfg: PipelineConfig, repo: str) -> Path:
    return cfg.work_dir / repo / "author_batch"


def authored_repo_dir(cfg: PipelineConfig, repo: str) -> Path:
    return cfg.authored_dir / repo


def unit_complete(author_dir: Path) -> bool:
    return all((author_dir / name).is_file() for name in REQUIRED)


def resolve_author_dir(cfg: PipelineConfig, repo: str, unit: str) -> Path:
    """Prefer external Devin artifacts, then the in-pipeline author batch."""
    candidates = (
        authored_repo_dir(cfg, repo) / unit / "_author",
        author_root(cfg, repo) / "units" / unit / "_author",
        cfg.work_dir / repo / "units" / unit / "_author",
    )
    for path in candidates:
        if unit_complete(path):
            return path
    return candidates[0]


def _read_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None


def _closure_from_author(author: Path) -> dict[str, Any]:
    cpath = author / "closure.json"
    raw = _read_json(cpath) if cpath.is_file() else None
    if isinstance(raw, dict):
        return raw
    tree = author / "tree"
    files = [p.relative_to(tree).as_posix() for p in tree.rglob("*") if p.is_file()] if tree.is_dir() else []
    gold = author / "gold.patch"
    n_lines = 0
    if gold.is_file():
        n_lines = sum(
            1
            for ln in gold.read_text(encoding="utf-8", errors="replace").splitlines()
            if ln.startswith("+") and not ln.startswith("+++")
        )
    return {"files": files, "n_files": len(files), "n_lines": n_lines, "lines": n_lines}


def _difficulty_meta(author: Path) -> tuple[int | None, bool]:
    for name in ("difficulty.md", "DIFFICULTY.md"):
        path = author / name
        if not path.is_file():
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except OSError:
            continue
        pred = parse_predicted_flip(text, path=str(path))
        return (None if pred is None else pred.level, parse_control_flag(text))
    return None, False


def _row_from_author(name: str, author: Path, batch: Path | None = None) -> dict[str, Any]:
    closure = _closure_from_author(author)
    predicted, is_ctrl = _difficulty_meta(author)
    rel = ""
    if batch is not None:
        try:
            rel = str(author.relative_to(batch))
        except ValueError:
            rel = f"units/{name}/_author"
    return {
        "name": name,
        "dir": rel or f"{name}/_author",
        "author_dir": str(author),
        "family": closure.get("family") or "",
        "n_files": closure.get("n_files") or len(closure.get("files") or []),
        "n_lines": closure.get("n_lines") or closure.get("lines") or 0,
        "closure": closure,
        "predicted_flip": predicted,
        "is_control": is_ctrl,
    }


def discover_units(batch: Path) -> list[dict[str, Any]]:
    manifest = batch / "units.json"
    rows: list[dict[str, Any]] = []
    raw = _read_json(manifest) if manifest.is_file() else None
    if isinstance(raw, list):
        rows = [r for r in raw if isinstance(r, dict) and r.get("name")]
        out: list[dict[str, Any]] = []
        for row in rows:
            name = str(row["name"])
            author = batch / str(row.get("dir") or f"units/{name}/_author")
            if not unit_complete(author):
                continue
            merged = _row_from_author(name, author, batch)
            merged.update({k: v for k, v in row.items() if k not in merged or merged[k] in ("", 0, None)})
            merged["name"] = name
            merged["author_dir"] = str(author)
            pred, is_ctrl = _difficulty_meta(author)
            if pred is not None:
                merged["predicted_flip"] = pred
            if is_ctrl:
                merged["is_control"] = True
            out.append(merged)
        if out:
            return out
    units_dir = batch / "units"
    if not units_dir.is_dir():
        return []
    found: list[dict[str, Any]] = []
    for child in sorted(units_dir.iterdir()):
        author = child / "_author"
        if author.is_dir() and unit_complete(author):
            found.append(_row_from_author(child.name, author, batch))
    return found


def discover_authored_units(cfg: PipelineConfig, repo: str) -> list[dict[str, Any]]:
    root = authored_repo_dir(cfg, repo)
    if not root.is_dir():
        return []
    found: list[dict[str, Any]] = []
    for child in sorted(root.iterdir()):
        if not child.is_dir() or child.name.startswith(("_", ".")):
            continue
        author = child / "_author"
        if unit_complete(author):
            found.append(_row_from_author(child.name, author))
    return found


def discover_task_unit_names(cfg: PipelineConfig, repo: str) -> list[str]:
    root = cfg.tasks_dir / repo
    if not root.is_dir():
        return []
    names: list[str] = []
    for child in sorted(root.iterdir()):
        if child.is_dir() and child.name.endswith("-L2"):
            names.append(child.name[: -len("-L2")])
    return names


def existing_author_units(cfg: PipelineConfig, repo: str) -> list[dict[str, Any]]:
    """Units that already have _author artifacts (external Devin or prior run)."""
    by_name: dict[str, dict[str, Any]] = {}
    for row in discover_authored_units(cfg, repo):
        by_name[str(row["name"])] = row
    for row in discover_units(author_root(cfg, repo)):
        by_name.setdefault(str(row["name"]), row)
    return [by_name[k] for k in sorted(by_name)]


def unit_metas_from_rows(repo: str, rows: list[dict[str, Any]], *, author_backend: str = "") -> list[UnitMeta]:
    out: list[UnitMeta] = []
    for row in rows:
        pred = row.get("predicted_flip")
        out.append(
            UnitMeta(
                repo=repo,
                name=str(row["name"]),
                predicted_flip=None if pred is None else int(pred),
                n_lines=int(row.get("n_lines") or 0),
                n_files=int(row.get("n_files") or 0),
                family=str(row.get("family") or ""),
                author_backend=author_backend,
                is_control=bool(row.get("is_control")),
            )
        )
    return out


def mark_control(store: PipelineStore, repo: str, rows: list[dict[str, Any]], *, author_backend: str) -> str | None:
    metas = unit_metas_from_rows(repo, rows, author_backend=author_backend)
    ctrl = pick_control(metas, repo=repo)
    if ctrl is None:
        return None
    for row in rows:
        name = str(row["name"])
        store.upsert_unit(
            repo,
            name,
            is_control=(name == ctrl.name),
            status="authored",
        )
        row["is_control"] = name == ctrl.name
    return ctrl.name


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
    existing = existing_author_units(cfg, repo)
    if existing:
        record_units(repo, store, existing, author_backend=cfg.author_backend)
        mark_control(store, repo, existing, author_backend=cfg.author_backend)
        store.add_event("author", f"ingest {repo}: {len(existing)} existing _author units; skip re-author")
        return existing[:n] if n else existing
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
        timeout_sec=session_timeout_sec(cfg.author_backend),
    )
    found = discover_units(batch)
    if not found:
        raise RuntimeError(f"author session produced no units under {batch}")
    record_units(repo, store, found, author_backend=cfg.author_backend)
    mark_control(store, repo, found, author_backend=cfg.author_backend)
    return found[:n]


def record_units(
    repo: str,
    store: PipelineStore,
    rows: list[dict[str, Any]],
    *,
    author_backend: str = "",
) -> None:
    for row in rows:
        name = str(row["name"])
        author = Path(str(row.get("author_dir") or ""))
        if not str(row.get("author_dir") or "") or not author.is_dir():
            continue
        closure = row.get("closure") if isinstance(row.get("closure"), dict) else _closure_from_author(author)
        files = closure.get("files") or []
        predicted = row.get("predicted_flip")
        store.upsert_unit(
            repo,
            name,
            status="authored",
            family=str(row.get("family") or closure.get("family") or ""),
            closure=closure,
            n_files=int(row.get("n_files") or len(files) or 0),
            n_lines=int(row.get("n_lines") or closure.get("lines") or closure.get("n_lines") or 0),
            predicted_flip=None if predicted is None else int(predicted),
            is_control=bool(row.get("is_control")),
            author_backend=author_backend,
        )
