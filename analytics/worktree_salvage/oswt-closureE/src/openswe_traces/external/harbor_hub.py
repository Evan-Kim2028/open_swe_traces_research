"""Harvest per-trial Terminal-Bench results from Harbor Hub public leaderboards.

Harbor Hub is a Supabase project. The `harbor` CLI gates its `hub job`/`hub trial`
commands behind `harbor auth login`, but the underlying RPCs and tables serve
public data to anonymous callers under the project's publishable key — the same
key embedded in the CLI (`harbor.auth.constants`). This module calls those
read-only endpoints directly over HTTP, so no login is required.

Endpoints used (all read-only):
  GET  /rest/v1/leaderboard                  board catalog
  POST /functions/v1/leaderboard-read        board definition + ranked rows
  GET  /rest/v1/leaderboard_row_trial        row -> trial associations
  GET  /rest/v1/trial?select=id,job_id       trial -> job map
  POST /rest/v1/rpc/get_job_overview         job header/stats
  POST /rest/v1/rpc/get_job_trials           per-trial rows (paginated)
  POST /rest/v1/rpc/get_job_tasks            per-task rows (paginated)

Resume-safe: every response is cached under <out_dir>/raw/ and existing files
are skipped, so re-running continues where it stopped. Assembly reads the raw
cache and writes trials/jobs/tasks/leaderboard_rows/row_trials parquet
atomically.
"""

from __future__ import annotations

import argparse
import json
import os
import random
import re
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from collections.abc import Callable
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import pandas as pd

ROOT = Path(__file__).resolve().parents[3]
DEFAULT_OUT_DIR = ROOT / "traces_external" / "harbor_hub"
LOG_PATH = ROOT / "outputs" / "closure_E.log"

SUPABASE_URL = os.environ.get(
    "HARBOR_SUPABASE_URL", "https://ofhuhcpkvzjlejydnvyd.supabase.co"
)
PUBLISHABLE_KEY = os.environ.get(
    "HARBOR_SUPABASE_PUBLISHABLE_KEY", "sb_publishable_Z-vuQbpvpG-PStjbh4yE0Q_e-d3MTIH"
)

REQUEST_TIMEOUT_S = 30.0
DEFAULT_SLEEP_S = 0.4
MAX_TRIES = 6
TRIAL_PAGE_SIZE = 500
ROW_TRIAL_PAGE_SIZE = 1000
JOB_ID_BATCH = 120

# (package slug, leaderboard name) -> benchmark version label.
BOARD_VERSIONS = {
    ("terminal-bench/terminal-bench-2", "2-0"): "2.0",
    ("terminal-bench/terminal-bench-2-1", "main"): "2.1",
    ("terminal-bench/terminal-bench", "3-0-0"): "3.0",
    ("terminal-bench/terminal-bench", "4-0-0"): "4.0",
}

JOB_URL_RE = re.compile(r"/jobs/([0-9a-fA-F-]{36})")


class HubError(RuntimeError):
    pass


def _headers() -> dict[str, str]:
    return {
        "apikey": PUBLISHABLE_KEY,
        "Authorization": f"Bearer {PUBLISHABLE_KEY}",
        "Content-Type": "application/json",
        "Accept": "application/json",
    }


def _http_json(
    method: str,
    url: str,
    payload: dict[str, Any] | None = None,
    *,
    sleep_s: float = DEFAULT_SLEEP_S,
) -> Any:
    """One HTTP call with retry on 429/5xx/network errors (backoff + jitter)."""
    body = json.dumps(payload).encode() if payload is not None else None
    delay = 1.0
    for attempt in range(1, MAX_TRIES + 1):
        req = urllib.request.Request(
            url, data=body, headers=_headers(), method=method
        )
        try:
            with urllib.request.urlopen(req, timeout=REQUEST_TIMEOUT_S) as resp:
                text = resp.read().decode()
                time.sleep(sleep_s)
                return json.loads(text) if text else None
        except urllib.error.HTTPError as exc:
            if exc.code in (400, 401, 403, 404):
                raise HubError(f"{method} {url} -> HTTP {exc.code}: {exc.read()[:300]}") from exc
            retry_after = exc.headers.get("Retry-After") if exc.headers else None
            wait = float(retry_after) if retry_after else delay + random.random()
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            if attempt == MAX_TRIES:
                raise HubError(f"{method} {url} failed after {MAX_TRIES} tries: {exc}") from exc
            wait = delay + random.random()
        if attempt == MAX_TRIES:
            raise HubError(f"{method} {url} failed after {MAX_TRIES} tries")
        time.sleep(wait)
        delay = min(delay * 2, 60.0)
    raise HubError(f"{method} {url} failed")


def _rest(path: str, params: dict[str, str], *, sleep_s: float) -> Any:
    url = f"{SUPABASE_URL}/rest/v1/{path}?{urllib.parse.urlencode(params)}"
    return _http_json("GET", url, sleep_s=sleep_s)


def _rpc(name: str, payload: dict[str, Any], *, sleep_s: float) -> Any:
    return _http_json(
        "POST", f"{SUPABASE_URL}/rest/v1/rpc/{name}", payload, sleep_s=sleep_s
    )


def _fn(name: str, payload: dict[str, Any], *, sleep_s: float) -> Any:
    return _http_json(
        "POST", f"{SUPABASE_URL}/functions/v1/{name}", payload, sleep_s=sleep_s
    )


# ---------------------------------------------------------------------------
# API wrappers
# ---------------------------------------------------------------------------


def list_leaderboards(*, sleep_s: float = DEFAULT_SLEEP_S) -> list[dict[str, Any]]:
    """All public leaderboards with embedded package/org names."""
    return _rest(
        "leaderboard",
        {
            "select": "id,name,title,visibility,created_at,"
            "package:package_id!inner(name,organization:org_id!inner(name))",
            "order": "created_at.desc",
        },
        sleep_s=sleep_s,
    )


def terminal_bench_boards(
    *, sleep_s: float = DEFAULT_SLEEP_S
) -> dict[str, dict[str, Any]]:
    """Map bench_version -> leaderboard row for the public TB boards."""
    boards: dict[str, dict[str, Any]] = {}
    for lb in list_leaderboards(sleep_s=sleep_s):
        pkg = lb.get("package") or {}
        org = (pkg.get("organization") or {}).get("name", "")
        slug = f"{org}/{pkg.get('name', '')}"
        version = BOARD_VERSIONS.get((slug, lb.get("name", "")))
        if version:
            boards[version] = lb
    return boards


def leaderboard_read(
    leaderboard_id: str, *, sleep_s: float = DEFAULT_SLEEP_S
) -> dict[str, Any]:
    """Board definition + ranked rows via the leaderboard-read edge function."""
    return _fn(
        "leaderboard-read",
        {"leaderboard_id": leaderboard_id, "page": 1, "page_size": 1000},
        sleep_s=sleep_s,
    )


def list_row_trials(
    row_id: str, *, sleep_s: float = DEFAULT_SLEEP_S
) -> list[dict[str, Any]]:
    """All trial associations for one leaderboard row."""
    items: list[dict[str, Any]] = []
    offset = 0
    while True:
        page = _rest(
            "leaderboard_row_trial",
            {
                "select": "trial_id,created_at",
                "row_id": f"eq.{row_id}",
                "order": "created_at.asc,trial_id.asc",
                "offset": str(offset),
                "limit": str(ROW_TRIAL_PAGE_SIZE),
            },
            sleep_s=sleep_s,
        )
        items.extend(page)
        if len(page) < ROW_TRIAL_PAGE_SIZE:
            return items
        offset += ROW_TRIAL_PAGE_SIZE


def map_trials_to_jobs(
    trial_ids: list[str], *, sleep_s: float = DEFAULT_SLEEP_S
) -> dict[str, str]:
    """trial_id -> job_id via the public trial table, batched."""
    mapping: dict[str, str] = {}
    ids = list(dict.fromkeys(trial_ids))
    for i in range(0, len(ids), JOB_ID_BATCH):
        batch = ids[i : i + JOB_ID_BATCH]
        rows = _rest(
            "trial",
            {"select": "id,job_id", "id": f"in.({','.join(batch)})"},
            sleep_s=sleep_s,
        )
        for row in rows:
            mapping[str(row["id"])] = str(row["job_id"])
    return mapping


def get_job_overview(job_id: str, *, sleep_s: float = DEFAULT_SLEEP_S) -> dict[str, Any]:
    return _rpc(
        "get_job_overview",
        {"p_job_ids": [job_id], "p_force_combined": False},
        sleep_s=sleep_s,
    )


def get_job_trials(
    job_id: str, *, sleep_s: float = DEFAULT_SLEEP_S
) -> list[dict[str, Any]]:
    """All trial executions of a job (every attempt, not just latest)."""
    items: list[dict[str, Any]] = []
    page = 1
    while True:
        payload = _rpc(
            "get_job_trials",
            {
                "p_job_ids": [job_id],
                "p_page": page,
                "p_page_size": TRIAL_PAGE_SIZE,
                "p_attempts": "all",
            },
            sleep_s=sleep_s,
        )
        items.extend(payload.get("items") or [])
        total_pages = payload.get("total_pages") or 1
        if page >= total_pages:
            return items
        page += 1


def get_job_tasks(job_id: str, *, sleep_s: float = DEFAULT_SLEEP_S) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    page = 1
    while True:
        payload = _rpc(
            "get_job_tasks",
            {"p_job_id": job_id, "p_page": page, "p_page_size": TRIAL_PAGE_SIZE},
            sleep_s=sleep_s,
        )
        items.extend(payload.get("items") or [])
        total_pages = payload.get("total_pages") or 1
        if page >= total_pages:
            return items
        page += 1


# ---------------------------------------------------------------------------
# Parsing (pure functions — exercised by tests against recorded fixtures)
# ---------------------------------------------------------------------------


def _link_label(value: Any) -> str | None:
    """Leaderboard metadata cells are {label, url} objects or plain strings."""
    if isinstance(value, dict):
        return value.get("label")
    if isinstance(value, str):
        return value
    return None


def parse_row_meta(metadata: dict[str, Any]) -> dict[str, Any]:
    """Normalize both leaderboard metadata schemas (2.0 vs 2.1+)."""
    model_names = metadata.get("model_names")
    model = _link_label(metadata.get("model_display"))
    if model is None and isinstance(model_names, list):
        model = ", ".join(str(m) for m in model_names)
    return {
        "agent": _link_label(metadata.get("agent_display")) or metadata.get("agent_name"),
        "model": model,
        "agent_org": _link_label(metadata.get("agent_org")),
        "model_org": _link_label(metadata.get("model_org")),
        "reasoning_effort": metadata.get("reasoning_effort"),
        "date": metadata.get("date") or metadata.get("release_date"),
        "job_url": (metadata.get("pr_url") or {}).get("url")
        if isinstance(metadata.get("pr_url"), dict)
        else None,
        "source_url": metadata.get("source_url"),
    }


def parse_leaderboard_row(
    row: dict[str, Any], *, bench_version: str, leaderboard_id: str
) -> dict[str, Any]:
    meta = parse_row_meta(row.get("metadata") or {})
    metrics = row.get("metrics") or {}
    return {
        "bench_version": bench_version,
        "leaderboard_id": leaderboard_id,
        "row_id": row.get("id"),
        "rank": row.get("rank"),
        "status": row.get("status"),
        **meta,
        "accuracy": metrics.get("accuracy"),
        "accuracy_ci95_half_width": metrics.get("accuracy_ci95_half_width"),
        "n_trials": (
            row.get("n_trials") if row.get("n_trials") is not None
            else metrics.get("n_trials")
        ),
        "total_tokens": metrics.get("total_tokens"),
        "total_cost_usd": metrics.get("total_cost_usd"),
        "created_at": row.get("created_at"),
        "updated_at": row.get("updated_at"),
    }


def task_id_from_name(task_name: str | None) -> str | None:
    """'terminal-bench/photonic-waveguide-routing' -> 'photonic-waveguide-routing'."""
    if not task_name:
        return None
    return task_name.rsplit("/", 1)[-1]


def _reward_of(trial: dict[str, Any]) -> float | None:
    reward = trial.get("reward")
    if reward is None:
        metrics = ((trial.get("evals") or {}).get("reward") or {}).get("metrics") or []
        if metrics:
            reward = metrics[0].get("reward")
    try:
        return float(reward) if reward is not None else None
    except (TypeError, ValueError):
        return None


def _iso_to_epoch(value: str | None) -> float | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value).timestamp()
    except ValueError:
        return None


def parse_trial_row(
    trial: dict[str, Any],
    *,
    bench_version: str,
    leaderboard_id: str,
    leaderboard_row_id: str | None,
    harvested_at: str,
) -> dict[str, Any]:
    """One get_job_trials item -> one trials.parquet record."""
    reward = _reward_of(trial)
    started = _iso_to_epoch(trial.get("started_at"))
    finished = _iso_to_epoch(trial.get("finished_at"))
    error_class = (
        trial.get("error_type") or trial.get("hosted_error_code") or trial.get("hosted_error")
    )
    config_values = trial.get("config_values") or {}
    effort_raw = config_values.get("agent.kwargs.reasoning_effort")
    try:
        reasoning_effort = json.loads(effort_raw) if effort_raw else None
    except (TypeError, json.JSONDecodeError):
        reasoning_effort = effort_raw
    return {
        "bench_version": bench_version,
        "leaderboard_id": leaderboard_id,
        "leaderboard_row_id": leaderboard_row_id,
        "job_id": trial.get("job_id"),
        "job_title": trial.get("job_name"),
        "agent": trial.get("agent_name"),
        "agent_version": trial.get("agent_version"),
        "reasoning_effort": reasoning_effort,
        "config_json": json.dumps(config_values, sort_keys=True)
        if config_values
        else None,
        "model": trial.get("model_name"),
        "model_provider": trial.get("model_provider"),
        "task_id": task_id_from_name(trial.get("task_name")),
        "task_name": trial.get("task_name"),
        "trial_id": trial.get("id"),
        "trial_name": trial.get("name"),
        "attempt_index": trial.get("attempt"),
        "n_attempts": trial.get("n_attempts"),
        "reward": reward,
        "passed": (reward >= 1.0) if reward is not None else None,
        "status": trial.get("status"),
        "is_scored": trial.get("is_scored"),
        "error_class": error_class,
        "started_at": trial.get("started_at"),
        "finished_at": trial.get("finished_at"),
        "wall_seconds": (finished - started)
        if started is not None and finished is not None
        else None,
        "tokens_in": trial.get("input_tokens"),
        "tokens_out": trial.get("output_tokens"),
        "cache_tokens": trial.get("cache_tokens"),
        "cost_usd": trial.get("cost_usd"),
        "trajectory_available": trial.get("archive_path") is not None,
        "harvested_at": harvested_at,
    }


# ---------------------------------------------------------------------------
# Harvest orchestration
# ---------------------------------------------------------------------------


def _load_json(path: Path) -> Any | None:
    if path.exists():
        return json.loads(path.read_text())
    return None


def _save_json(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(json.dumps(payload))
    tmp.rename(path)


def _write_parquet(df: pd.DataFrame, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        dir=path.parent, suffix=".parquet.tmp", delete=False
    ) as tmp:
        tmp_path = Path(tmp.name)
    df.to_parquet(tmp_path, index=False)
    tmp_path.rename(path)


def _row_job_hint(row: dict[str, Any]) -> str | None:
    """Job UUID from a row's pr_url/job link metadata, if present."""
    meta = row.get("metadata") or {}
    pr_url = meta.get("pr_url")
    url = pr_url.get("url") if isinstance(pr_url, dict) else None
    if url:
        match = JOB_URL_RE.search(url)
        if match:
            return match.group(1)
    return None


def harvest(
    out_dir: Path = DEFAULT_OUT_DIR,
    *,
    versions: list[str] | None = None,
    sleep_s: float = DEFAULT_SLEEP_S,
    log: Callable[[str], None] = print,
) -> dict[str, int]:
    """Pull every job on the public TB boards into the raw cache, then parquet."""
    raw = out_dir / "raw"
    harvested_at = datetime.now(UTC).isoformat()

    boards_path = raw / "boards.json"
    boards = _load_json(boards_path)
    if boards is None:
        boards = terminal_bench_boards(sleep_s=sleep_s)
        _save_json(boards_path, boards)
        log(f"discovered boards: {sorted(boards)}")
    else:
        log(f"boards (cached): {sorted(boards)}")

    wanted = versions or sorted(BOARD_VERSIONS.values())
    stats: dict[str, int] = {"rows": 0, "jobs": 0, "trials": 0}

    for version in wanted:
        board = boards.get(version)
        if board is None:
            log(f"[{version}] no leaderboard found, skipping")
            continue
        lb_id = board["id"]
        board_path = raw / f"board_{version}.json"
        payload = _load_json(board_path)
        if payload is None:
            payload = leaderboard_read(lb_id, sleep_s=sleep_s)
            _save_json(board_path, payload)
        rows = payload.get("rows") or []
        log(f"[{version}] {board['title']}: {len(rows)} rows")

        # row -> trial ids (cached per row)
        row_trials: dict[str, list[str]] = {}
        job_hints: dict[str, str] = {}  # row_id -> job_id from pr_url
        for row in rows:
            row_id = row["id"]
            cached = _load_json(raw / "row_trials" / f"{row_id}.json")
            if cached is None:
                cached = list_row_trials(row_id, sleep_s=sleep_s)
                _save_json(raw / "row_trials" / f"{row_id}.json", cached)
            row_trials[row_id] = [t["trial_id"] for t in cached]
            hint = _row_job_hint(row)
            if hint:
                job_hints[row_id] = hint

        # trial -> job map (cached per board)
        map_path = raw / f"board_{version}.trial_jobs.json"
        trial_job = _load_json(map_path)
        all_trial_ids = [t for ids in row_trials.values() for t in ids]
        if trial_job is None:
            trial_job = map_trials_to_jobs(all_trial_ids, sleep_s=sleep_s)
            _save_json(map_path, trial_job)
        missing = [t for t in all_trial_ids if t not in trial_job]
        if missing:
            trial_job.update(map_trials_to_jobs(missing, sleep_s=sleep_s))
            _save_json(map_path, trial_job)

        # row_id -> job_ids (association map, plus pr_url fallback)
        row_jobs: dict[str, list[str]] = {}
        for row in rows:
            row_id = row["id"]
            job_ids = list(
                dict.fromkeys(
                    trial_job[t] for t in row_trials[row_id] if t in trial_job
                )
            )
            if not job_ids and row_id in job_hints:
                job_ids = [job_hints[row_id]]
            row_jobs[row_id] = job_ids

        job_ids_all = list(dict.fromkeys(j for ids in row_jobs.values() for j in ids))
        log(f"[{version}] {len(job_ids_all)} jobs, {len(all_trial_ids)} row trials")

        jobs_dir = raw / "jobs"
        for i, job_id in enumerate(job_ids_all, 1):
            for kind, fetcher in (
                ("overview", get_job_overview),
                ("trials", get_job_trials),
                ("tasks", get_job_tasks),
            ):
                path = jobs_dir / f"{job_id}.{kind}.json"
                if path.exists():
                    continue
                _save_json(path, fetcher(job_id, sleep_s=sleep_s))
            if i % 10 == 0 or i == len(job_ids_all):
                log(f"[{version}] jobs {i}/{len(job_ids_all)}")

        stats["rows"] += len(rows)
        stats["jobs"] += len(job_ids_all)
        stats["trials"] += len(all_trial_ids)

    assemble(out_dir, boards=boards, harvested_at=harvested_at, log=log)
    return stats


def assemble(
    out_dir: Path = DEFAULT_OUT_DIR,
    *,
    boards: dict[str, dict[str, Any]] | None = None,
    harvested_at: str | None = None,
    log: Callable[[str], None] = print,
) -> dict[str, Path]:
    """Read the raw cache and write trials/jobs/tasks/leaderboard_rows parquet."""
    raw = out_dir / "raw"
    if boards is None:
        boards = _load_json(raw / "boards.json") or {}
    harvested_at = harvested_at or datetime.now(UTC).isoformat()

    rows_rec: list[dict[str, Any]] = []
    row_trial_rec: list[dict[str, Any]] = []
    trial_rec: list[dict[str, Any]] = []
    job_rec: list[dict[str, Any]] = []

    for board_path in sorted(raw.glob("board_*.json")):
        version = board_path.stem.removeprefix("board_")
        if version.endswith(".trial_jobs"):
            continue
        payload = _load_json(board_path) or {}
        lb_id = (payload.get("leaderboard") or {}).get("id") or (
            boards.get(version) or {}
        ).get("id")
        rows = payload.get("rows") or []
        trial_job = _load_json(raw / f"board_{version}.trial_jobs.json") or {}

        for row in rows:
            rows_rec.append(
                parse_leaderboard_row(row, bench_version=version, leaderboard_id=lb_id)
            )

        for row in rows:
            row_id = row["id"]
            cached = _load_json(raw / "row_trials" / f"{row_id}.json") or []
            for assoc in cached:
                row_trial_rec.append(
                    {
                        "bench_version": version,
                        "leaderboard_id": lb_id,
                        "row_id": row_id,
                        "trial_id": assoc["trial_id"],
                    }
                )
            job_ids = list(
                dict.fromkeys(
                    trial_job[t["trial_id"]] for t in cached if t["trial_id"] in trial_job
                )
            )
            hint = _row_job_hint(row)
            if not job_ids and hint:
                job_ids = [hint]
            for job_id in job_ids:
                overview = _load_json(raw / "jobs" / f"{job_id}.overview.json") or {}
                trials = _load_json(raw / "jobs" / f"{job_id}.trials.json") or []
                for trial in trials:
                    trial_rec.append(
                        parse_trial_row(
                            trial,
                            bench_version=version,
                            leaderboard_id=lb_id,
                            leaderboard_row_id=row_id,
                            harvested_at=harvested_at,
                        )
                    )
                job_rec.append(
                    _job_record(
                        job_id,
                        overview,
                        trials,
                        row=row,
                        bench_version=version,
                        leaderboard_id=lb_id,
                    )
                )

    trials_df = pd.DataFrame(trial_rec)
    if not trials_df.empty:
        dupes = trials_df.duplicated(subset=["trial_id", "attempt_index"]).sum()
        if dupes:
            log(f"warning: dropping {dupes} duplicate (trial_id, attempt_index) rows")
            trials_df = trials_df.drop_duplicates(subset=["trial_id", "attempt_index"])

    jobs_df = pd.DataFrame(job_rec)
    if not jobs_df.empty:
        jobs_df = jobs_df.drop_duplicates(subset=["bench_version", "job_id"])

    if trials_df.empty:
        tasks_df = pd.DataFrame(
            columns=["bench_version", "task_id", "n_trials", "n_jobs", "mean_reward", "n_passed"]
        )
    else:
        tasks_df = (
            trials_df.groupby(["bench_version", "task_id"], dropna=False)
            .agg(
                n_trials=("trial_id", "count"),
                n_jobs=("job_id", "nunique"),
                mean_reward=("reward", "mean"),
                n_passed=("passed", "sum"),
            )
            .reset_index()
            .sort_values(["bench_version", "mean_reward"])
        )

    rows_df = pd.DataFrame(rows_rec)
    row_trials_df = pd.DataFrame(row_trial_rec)

    out_paths = {
        "trials": out_dir / "trials.parquet",
        "jobs": out_dir / "jobs.parquet",
        "tasks": out_dir / "tasks.parquet",
        "leaderboard_rows": out_dir / "leaderboard_rows.parquet",
        "row_trials": out_dir / "row_trials.parquet",
    }
    for name, df in (
        ("trials", trials_df),
        ("jobs", jobs_df),
        ("tasks", tasks_df),
        ("leaderboard_rows", rows_df),
        ("row_trials", row_trials_df),
    ):
        _write_parquet(df, out_paths[name])
        log(f"wrote {out_paths[name]} ({len(df)} rows)")
    return out_paths


def _job_record(
    job_id: str,
    overview: dict[str, Any],
    trials: list[dict[str, Any]],
    *,
    row: dict[str, Any],
    bench_version: str,
    leaderboard_id: str,
) -> dict[str, Any]:
    metrics = row.get("metrics") or {}
    meta = parse_row_meta(row.get("metadata") or {})
    t0 = trials[0] if trials else {}
    evals_rows = (overview.get("evals") or {}).get("rows") or []
    eval_key = evals_rows[0].get("key") if evals_rows else {}
    rewards = [_reward_of(t) for t in trials]
    scored = [r for r in rewards if r is not None]
    return {
        "bench_version": bench_version,
        "leaderboard_id": leaderboard_id,
        "leaderboard_row_id": row.get("id"),
        "job_id": job_id,
        "title": overview.get("name") or t0.get("job_name"),
        "agent": (eval_key or {}).get("agent") or t0.get("agent_name") or meta["agent"],
        "agent_version": (eval_key or {}).get("agent_version") or t0.get("agent_version"),
        "model": (eval_key or {}).get("model") or t0.get("model_name") or meta["model"],
        "model_provider": (eval_key or {}).get("provider") or t0.get("model_provider"),
        "n_tasks": len({task_id_from_name(t.get("task_name")) for t in trials}),
        "n_trials": overview.get("n_total_trials") or len(trials),
        "mean_reward": sum(scored) / len(scored) if scored else None,
        "rank": row.get("rank"),
        "accuracy": metrics.get("accuracy"),
        "cost_usd": overview.get("cost_usd"),
        "tokens_in": overview.get("input_tokens"),
        "tokens_out": overview.get("output_tokens"),
        "n_errors": overview.get("n_errors"),
        "visibility": overview.get("visibility"),
        "owner_org": (overview.get("owner_org") or {}).get("name"),
        "submitted_at": overview.get("started_at") or row.get("created_at"),
        "finished_at": overview.get("finished_at"),
    }


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Harvest per-trial Terminal-Bench results from Harbor Hub."
    )
    parser.add_argument("--out-dir", type=Path, default=DEFAULT_OUT_DIR)
    parser.add_argument(
        "--versions",
        nargs="*",
        default=None,
        help="Subset of bench versions (e.g. 4.0 3.0). Default: all discovered.",
    )
    parser.add_argument("--sleep", type=float, default=DEFAULT_SLEEP_S)
    parser.add_argument(
        "--assemble-only",
        action="store_true",
        help="Skip network; rebuild parquet from the raw cache.",
    )
    args = parser.parse_args()

    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    log_file = LOG_PATH.open("a")

    def log(msg: str) -> None:
        line = f"{datetime.now(UTC).isoformat()} {msg}"
        print(line)
        log_file.write(line + "\n")
        log_file.flush()

    if args.assemble_only:
        assemble(args.out_dir, log=log)
        return
    stats = harvest(args.out_dir, versions=args.versions, sleep_s=args.sleep, log=log)
    log(f"done: {stats}")
    log_file.close()


if __name__ == "__main__":
    main()
