"""Aggregate Harbor job dirs into a task-level results table and markdown dashboard.

Called by the pipeline orchestrator and by ``scripts/results_dashboard.py``.
Flip points follow verifier_rules.md C6: lowest level with >= 2/3 passes over
>= 3 attempts. Older ``-A<k>`` trial names map to L via L = A + 2.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
from collections import Counter, defaultdict
from collections.abc import Iterable, Mapping, Sequence
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

import pandas as pd

from openswe_traces.data import ROOT

DEFAULT_OUT_DIR = ROOT / "experiments" / "pipeline"
DEFAULT_CONFIG_PATH = DEFAULT_OUT_DIR / "config.yaml"
DEFAULT_TOKEN_CAP = 1e9
WEB_TOOL_RE = re.compile(r"webFetchToolCall|webSearchToolCall")
CHECKSUM_RE = re.compile(
    r"test file modified|consumer test files were modified",
    re.IGNORECASE,
)
TRIAL_STEM_RE = re.compile(
    r"^(?P<unit>.+?)-(?P<kind>[LA])(?P<n>-?\d+)$",
)
SSH_SPEC_RE = re.compile(
    r"^(?:(?P<user>[^@/]+)@)?(?P<host>[^:/]+):(?P<path>.+)$",
)
REPO_PREFIXES: tuple[tuple[str, str], ...] = (
    ("dailycodingproblem-go-", "dailycodingproblem-go"),
    ("client-go-", "client-go"),
)
CLIENT_GO_UNITS = frozenset(
    {
        "dynamic-pipeline",
        "spec-reimpl",
        "spec-reimpl-bb",
        "property-backoff",
        "property-1pc",
        "property-policy",
        "dynamic-snapshot",
        "dynamic-latch",
        "spec-bb-chain",
        "spec-bb-bucket",
        "batch-delete",
        "delete-range",
        "decode",
        "next",
        "batchcmds-obf",
        "keyspacecodec-obf",
        "interceptor-obf",
        "memget-obf",
        "memsetvalue-obf",
        "onepc-scope-obf",
    }
)
AUDIT_RANK = {"test-edit": 0, "contaminated": 1, "infra": 2, "clean": 3}
RESULT_COLUMNS = (
    "repo",
    "unit",
    "level",
    "solver",
    "attempts",
    "passes",
    "pass_rate",
    "flip_point",
    "audit_class",
    "tokens_in",
    "tokens_out",
    "wall_minutes",
    "job_name",
)


@dataclass(frozen=True)
class ParsedTrialName:
    unit: str
    level: int
    raw_kind: str


@dataclass
class TrialRecord:
    job_name: str
    repo: str
    unit: str
    level: int
    solver: str
    passed: bool | None
    audit_class: str
    tokens_in: float | None
    tokens_out: float | None
    wall_minutes: float | None
    agent_name: str


def is_ssh_spec(spec: str) -> bool:
    """True for ``host:path`` / ``user@host:path`` that is not an existing local path."""
    if Path(spec).exists():
        return False
    match = SSH_SPEC_RE.match(spec)
    if not match:
        return False
    host = match.group("host") or ""
    return bool(host) and "/" not in host and spec.split(":", 1)[1] != ""


def parse_trial_name(dirname: str) -> ParsedTrialName:
    """Parse ``<unit>-L<k>__<id>`` or older ``<unit>-A<k>__<id>`` (L = A + 2).

    Names with no level suffix (original in-tree-test tasks) are L6.
    """
    stem = dirname.split("__", 1)[0]
    match = TRIAL_STEM_RE.match(stem)
    if not match:
        return ParsedTrialName(unit=stem, level=6, raw_kind="implicit-L6")
    kind = match.group("kind")
    n = int(match.group("n"))
    level = n + 2 if kind == "A" else n
    return ParsedTrialName(unit=match.group("unit"), level=level, raw_kind=kind)


def infer_repo(unit: str, task_path: str = "") -> str:
    blob = f"{unit} {task_path}".lower()
    for prefix, repo in REPO_PREFIXES:
        if unit.startswith(prefix) or prefix.rstrip("-") in Path(task_path).name:
            return repo
    if "revive" in blob or unit.startswith(("graph-", "nograph-")):
        return "revive"
    stripped = unit.removesuffix("-obf")
    if "client-go" in blob or unit in CLIENT_GO_UNITS or stripped in CLIENT_GO_UNITS:
        return "client-go"
    if "dailycodingproblem" in blob:
        return "dailycodingproblem-go"
    return "unknown"


def load_json(path: Path) -> dict[str, Any]:
    try:
        data = json.loads(path.read_text())
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def solver_from_config(config: Mapping[str, Any]) -> tuple[str, str]:
    """Return ``(solver_label, agent_name)`` from a job or trial config."""
    agents = config.get("agents")
    agent: dict[str, Any]
    if isinstance(agents, list) and agents and isinstance(agents[0], dict):
        agent = agents[0]
    else:
        nested = config.get("agent")
        agent = nested if isinstance(nested, dict) else {}
    name = str(agent.get("name") or "unknown")
    model = str(agent.get("model_name") or "")
    label = f"{name}/{model}" if model else name
    return label, name


def parse_dt(value: str | None) -> datetime | None:
    if not value:
        return None
    text = value.replace("Z", "+00:00")
    try:
        dt = datetime.fromisoformat(text)
    except ValueError:
        return None
    if dt.tzinfo is not None:
        dt = dt.replace(tzinfo=None)
    return dt


def wall_minutes(result: dict[str, Any]) -> float | None:
    start = parse_dt(result.get("started_at") if isinstance(result.get("started_at"), str) else None)
    finish = parse_dt(
        result.get("finished_at") if isinstance(result.get("finished_at"), str) else None
    )
    if start is None or finish is None:
        return None
    return round((finish - start).total_seconds() / 60.0, 2)


def _read_tree_text(directory: Path) -> str:
    if not directory.is_dir():
        return ""
    chunks: list[str] = []
    for path in directory.rglob("*"):
        if not path.is_file():
            continue
        try:
            chunks.append(path.read_text(errors="ignore"))
        except OSError:
            continue
    return "\n".join(chunks)


def trial_reward(trial_dir: Path, result: dict[str, Any]) -> float | None:
    verifier = result.get("verifier_result")
    if isinstance(verifier, dict):
        rewards = verifier.get("rewards")
        if isinstance(rewards, dict) and "reward" in rewards:
            try:
                return float(rewards["reward"])
            except (TypeError, ValueError):
                pass
    reward_path = trial_dir / "verifier" / "reward.txt"
    if reward_path.is_file():
        raw = reward_path.read_text(errors="ignore").strip().splitlines()
        if raw:
            try:
                return float(raw[0].strip())
            except ValueError:
                pass
    return None


def exception_present(result: dict[str, Any]) -> bool:
    info = result.get("exception_info")
    if not info:
        return False
    if isinstance(info, dict):
        return bool(info.get("exception_type") or info.get("exception_message") or info)
    return True


def audit_trial(trial_dir: Path, result: dict[str, Any] | None = None) -> str:
    """Classify one trial: test-edit / contaminated / infra / clean."""
    result = result if result is not None else load_json(trial_dir / "result.json")
    verifier_txt = _read_tree_text(trial_dir / "verifier")
    if CHECKSUM_RE.search(verifier_txt):
        return "test-edit"
    agent_txt = _read_tree_text(trial_dir / "agent")
    if WEB_TOOL_RE.search(agent_txt):
        return "contaminated"
    if exception_present(result):
        return "infra"
    return "clean"


def rollup_audit(classes: Iterable[str]) -> str:
    ranked = [c for c in classes if c in AUDIT_RANK]
    if not ranked:
        return "clean"
    return min(ranked, key=lambda c: AUDIT_RANK[c])


def flip_point_for(level_stats: dict[int, tuple[int, int]]) -> str:
    """C6: lowest level with attempts >= 3 and passes/attempts >= 2/3."""
    eligible: list[int] = []
    for level, (attempts, passes) in level_stats.items():
        if attempts >= 3 and passes * 3 >= attempts * 2:
            eligible.append(level)
    if not eligible:
        return ""
    return f"L{min(eligible)}"


def read_token_cap(config_path: Path | None = None) -> float:
    path = config_path if config_path is not None else DEFAULT_CONFIG_PATH
    if not path.is_file():
        return DEFAULT_TOKEN_CAP
    cap = DEFAULT_TOKEN_CAP
    for raw in path.read_text().splitlines():
        line = raw.split("#", 1)[0].strip()
        if not line or ":" not in line:
            continue
        key, value = line.split(":", 1)
        key = key.strip()
        if key in {"composer_token_cap", "token_cap", "composer_tokens_cap"}:
            try:
                cap = float(value.strip().strip("\"'"))
            except ValueError:
                continue
    return cap


def is_trial_dir(path: Path) -> bool:
    if not path.is_dir():
        return False
    return (path / "result.json").is_file() or (path / "agent").is_dir() or (path / "verifier").is_dir()


def is_job_dir(path: Path) -> bool:
    if not path.is_dir() or not (path / "config.json").is_file():
        return False
    if (path / "result.json").is_file():
        return True
    return any(is_trial_dir(child) for child in path.iterdir())


def iter_job_dirs(root: Path) -> list[Path]:
    if is_job_dir(root):
        return [root]
    jobs = [child for child in sorted(root.iterdir()) if child.is_dir() and is_job_dir(child)]
    return jobs


def materialize_source(spec: str | Path, cache_dir: Path) -> Path:
    text = str(spec)
    if not is_ssh_spec(text):
        path = Path(text)
        if not path.is_absolute():
            candidate = (ROOT / path).resolve()
            path = candidate if candidate.exists() else path.resolve()
        if not path.exists():
            raise FileNotFoundError(f"jobs path not found: {spec}")
        return path
    slug = re.sub(r"[^A-Za-z0-9._-]+", "_", text).strip("_")
    dest = cache_dir / slug
    dest.mkdir(parents=True, exist_ok=True)
    remote = text.rstrip("/") + "/"
    local = str(dest) + "/"
    rsync = _which("rsync")
    if rsync:
        cmd = [rsync, "-az", remote, local]
    else:
        scp = _which("scp")
        if not scp:
            raise RuntimeError("rsync or scp required to fetch ssh host:path jobs")
        cmd = [scp, "-r", text, str(dest)]
    subprocess.run(cmd, check=True)
    return dest


def _which(name: str) -> str | None:
    from shutil import which

    return which(name)


def collect_jobs(
    sources: Sequence[str | Path],
    *,
    cache_dir: Path | None = None,
) -> list[Path]:
    cache = cache_dir if cache_dir is not None else DEFAULT_OUT_DIR / ".ssh_cache"
    jobs: list[Path] = []
    seen: set[Path] = set()
    for spec in sources:
        root = materialize_source(spec, cache)
        for job in iter_job_dirs(root):
            resolved = job.resolve()
            if resolved not in seen:
                seen.add(resolved)
                jobs.append(resolved)
    return jobs


def _task_path(result: dict[str, Any], trial_config: dict[str, Any]) -> str:
    task_id = result.get("task_id")
    if isinstance(task_id, dict):
        path = task_id.get("path")
        if isinstance(path, str):
            return path
    task = trial_config.get("task")
    if isinstance(task, dict) and isinstance(task.get("path"), str):
        return str(task["path"])
    return ""


def _tokens(result: dict[str, Any]) -> tuple[float | None, float | None]:
    agent = result.get("agent_result")
    if not isinstance(agent, dict):
        stats = result.get("stats")
        agent = stats if isinstance(stats, dict) else {}
    tin = agent.get("n_input_tokens")
    tout = agent.get("n_output_tokens")
    try:
        tokens_in = float(tin) if tin is not None else None
    except (TypeError, ValueError):
        tokens_in = None
    try:
        tokens_out = float(tout) if tout is not None else None
    except (TypeError, ValueError):
        tokens_out = None
    return tokens_in, tokens_out


def scan_trial(
    trial_dir: Path,
    *,
    job_name: str,
    job_solver: str,
    job_agent: str,
) -> TrialRecord | None:
    result = load_json(trial_dir / "result.json")
    trial_config = load_json(trial_dir / "config.json")
    parsed = parse_trial_name(trial_dir.name)
    solver, agent_name = job_solver, job_agent
    if trial_config:
        s, a = solver_from_config(trial_config)
        if s != "unknown":
            solver, agent_name = s, a
    elif result:
        cfg = result.get("config")
        if isinstance(cfg, dict):
            s, a = solver_from_config(cfg)
            if s != "unknown":
                solver, agent_name = s, a
        info = result.get("agent_info")
        if isinstance(info, dict) and agent_name == "unknown":
            agent_name = str(info.get("name") or agent_name)
            model_info = info.get("model_info")
            model = ""
            if isinstance(model_info, dict):
                model = str(model_info.get("name") or "")
            if model:
                solver = f"{agent_name}/{model}"
    task_path = _task_path(result, trial_config)
    audit = audit_trial(trial_dir, result)
    reward = trial_reward(trial_dir, result)
    passed: bool | None
    if audit == "infra" and reward is None:
        passed = None
    elif reward is None and not result and not (trial_dir / "verifier" / "reward.txt").is_file():
        return None
    elif reward is None:
        passed = None
    else:
        passed = reward == 1.0
    tokens_in, tokens_out = _tokens(result)
    return TrialRecord(
        job_name=job_name,
        repo=infer_repo(parsed.unit, task_path),
        unit=parsed.unit,
        level=parsed.level,
        solver=solver,
        passed=passed,
        audit_class=audit,
        tokens_in=tokens_in,
        tokens_out=tokens_out,
        wall_minutes=wall_minutes(result),
        agent_name=agent_name,
    )


def scan_job(job_dir: Path) -> list[TrialRecord]:
    job_config = load_json(job_dir / "config.json")
    job_name = str(job_config.get("job_name") or job_dir.name)
    job_solver, job_agent = solver_from_config(job_config)
    records: list[TrialRecord] = []
    for child in sorted(job_dir.iterdir()):
        if not is_trial_dir(child):
            continue
        rec = scan_trial(
            child,
            job_name=job_name,
            job_solver=job_solver,
            job_agent=job_agent,
        )
        if rec is not None:
            records.append(rec)
    return records


def _sum_optional(values: Iterable[float | None]) -> float:
    total = 0.0
    seen = False
    for value in values:
        if value is None:
            continue
        total += value
        seen = True
    return total if seen else 0.0


def aggregate_records(records: Sequence[TrialRecord]) -> pd.DataFrame:
    """One row per (job, unit, level, solver); flip_point pooled per (unit, solver)."""
    pooled: dict[tuple[str, str], dict[int, list[bool]]] = defaultdict(lambda: defaultdict(list))
    groups: dict[tuple[str, str, int, str, str], list[TrialRecord]] = defaultdict(list)
    for rec in records:
        key = (rec.job_name, rec.unit, rec.level, rec.solver, rec.repo)
        groups[key].append(rec)
        if rec.passed is not None:
            pooled[(rec.unit, rec.solver)][rec.level].append(rec.passed)

    flips: dict[tuple[str, str], str] = {}
    for us_key, by_level in pooled.items():
        stats = {
            level: (len(flags), sum(1 for f in flags if f)) for level, flags in by_level.items()
        }
        flips[us_key] = flip_point_for(stats)

    rows: list[dict[str, Any]] = []
    for (job_name, unit, level, solver, repo), recs in sorted(groups.items()):
        scored = [r for r in recs if r.passed is not None]
        attempts = len(scored)
        passes = sum(1 for r in scored if r.passed)
        pass_rate = (passes / attempts) if attempts else float("nan")
        rows.append(
            {
                "repo": repo,
                "unit": unit,
                "level": f"L{level}",
                "level_n": level,
                "solver": solver,
                "attempts": attempts,
                "passes": passes,
                "pass_rate": pass_rate,
                "flip_point": flips.get((unit, solver), ""),
                "audit_class": rollup_audit(r.audit_class for r in recs),
                "tokens_in": _sum_optional(r.tokens_in for r in recs),
                "tokens_out": _sum_optional(r.tokens_out for r in recs),
                "wall_minutes": _sum_optional(r.wall_minutes for r in recs),
                "job_name": job_name,
            }
        )
    if not rows:
        df = pd.DataFrame(columns=list(RESULT_COLUMNS) + ["level_n"])
        return df
    df = pd.DataFrame(rows)
    return df.sort_values(["repo", "unit", "level_n", "solver", "job_name"]).reset_index(drop=True)


def _md_table(headers: Sequence[str], rows: Sequence[Sequence[Any]]) -> str:
    head = "| " + " | ".join(headers) + " |"
    sep = "| " + " | ".join("---" for _ in headers) + " |"
    body = ["| " + " | ".join(str(c) for c in row) + " |" for row in rows]
    return "\n".join([head, sep, *body])


def composer_tokens_used(df: pd.DataFrame) -> float:
    if df.empty:
        return 0.0
    mask = df["solver"].astype(str).str.startswith("cursor-cli")
    used = df.loc[mask, "tokens_in"].fillna(0).sum() + df.loc[mask, "tokens_out"].fillna(0).sum()
    return float(used)


def render_dashboard(df: pd.DataFrame, *, token_cap: float) -> str:
    n_task_levels = len(df)
    n_tasks = int(df.drop_duplicates(["unit", "solver"]).shape[0]) if not df.empty else 0
    n_levels = int(df["level"].nunique()) if not df.empty else 0
    n_trials = int(df["attempts"].sum()) if not df.empty else 0
    lines = [
        "# Harbor results",
        "",
        (
            "Task-level aggregation. Flip points follow C6 (lowest level with "
            ">= 2/3 passes over >= 3 attempts)."
        ),
        "",
        "## Totals",
        "",
        _md_table(
            ["metric", "value"],
            [
                ("tasks", n_tasks),
                ("levels_run", n_levels),
                ("task_levels", n_task_levels),
                ("trials", n_trials),
            ],
        ),
        "",
        "## Flip-point histogram per solver",
        "",
    ]
    if df.empty:
        lines.append("_No trials._")
    else:
        task_flips = (
            df.drop_duplicates(["unit", "solver"])[["solver", "unit", "flip_point"]]
            .assign(flip_point=lambda d: d["flip_point"].replace("", "none"))
        )
        solvers = sorted(task_flips["solver"].unique())
        for solver in solvers:
            sub = task_flips[task_flips["solver"] == solver]
            counts = sub["flip_point"].value_counts()
            labels = [f"L{i}" for i in range(7)] + ["none"]
            rows = [(lab, int(counts.get(lab, 0))) for lab in labels]
            lines.append(f"### `{solver}`")
            lines.append("")
            lines.append(_md_table(["flip_point", "tasks"], rows))
            lines.append("")
    audit_counts = (
        Counter(df["audit_class"].tolist()) if not df.empty else Counter()
    )
    lines.extend(
        [
            "## Contamination",
            "",
            _md_table(
                ["audit_class", "task_levels"],
                [
                    (cls, int(audit_counts.get(cls, 0)))
                    for cls in ("clean", "contaminated", "test-edit", "infra")
                ],
            ),
            "",
        ]
    )
    used = composer_tokens_used(df)
    remaining = token_cap - used
    pct = (used / token_cap * 100.0) if token_cap else float("nan")
    lines.extend(
        [
            "## Composer tokens (cursor-cli trials)",
            "",
            _md_table(
                ["metric", "value"],
                [
                    ("tokens_used", f"{used:.0f}"),
                    ("cap", f"{token_cap:.0f}"),
                    ("remaining", f"{remaining:.0f}"),
                    ("used_pct", f"{pct:.2f}%"),
                ],
            ),
            "",
            "## Per-repo",
            "",
        ]
    )
    if df.empty:
        lines.append("_No rows._")
    else:
        repo_rows: list[tuple[Any, ...]] = []
        for repo, sub in df.groupby("repo", sort=True):
            flips = sub.drop_duplicates(["unit", "solver"])["flip_point"]
            confirmed = [f for f in flips if f]
            repo_rows.append(
                (
                    repo,
                    int(sub.drop_duplicates(["unit"]).shape[0]),
                    len(sub),
                    int(sub["attempts"].sum()),
                    int((sub["audit_class"] == "clean").sum()),
                    int((sub["audit_class"] == "contaminated").sum()),
                    int((sub["audit_class"] == "test-edit").sum()),
                    int((sub["audit_class"] == "infra").sum()),
                    len(confirmed),
                )
            )
        lines.append(
            _md_table(
                [
                    "repo",
                    "units",
                    "task_levels",
                    "trials",
                    "clean",
                    "contaminated",
                    "test-edit",
                    "infra",
                    "confirmed_flips",
                ],
                repo_rows,
            )
        )
    lines.append("")
    return "\n".join(lines)


def write_outputs(
    df: pd.DataFrame,
    out_dir: Path,
    *,
    token_cap: float,
) -> tuple[Path, Path, str]:
    out_dir.mkdir(parents=True, exist_ok=True)
    parquet_path = out_dir / "results.parquet"
    md_path = out_dir / "results.md"
    export = df.drop(columns=["level_n"], errors="ignore")
    ordered = [c for c in RESULT_COLUMNS if c in export.columns]
    extra = [c for c in export.columns if c not in ordered]
    export = export[ordered + extra]
    export.to_parquet(parquet_path, index=False)
    markdown = render_dashboard(df, token_cap=token_cap)
    md_path.write_text(markdown)
    return parquet_path, md_path, markdown


def results_dashboard(
    sources: Sequence[str | Path],
    *,
    out_dir: Path | None = None,
    config_path: Path | None = None,
    cache_dir: Path | None = None,
) -> pd.DataFrame:
    """Scan Harbor jobs dirs (local and optional ``host:path``) and write outputs."""
    dest = Path(out_dir) if out_dir is not None else DEFAULT_OUT_DIR
    cap_path = config_path if config_path is not None else dest / "config.yaml"
    jobs = collect_jobs(sources, cache_dir=cache_dir if cache_dir is not None else dest / ".ssh_cache")
    records: list[TrialRecord] = []
    for job in jobs:
        records.extend(scan_job(job))
    df = aggregate_records(records)
    cap = read_token_cap(cap_path)
    write_outputs(df, dest, token_cap=cap)
    return df


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Aggregate Harbor jobs into experiments/pipeline/results.{parquet,md}",
    )
    parser.add_argument(
        "jobs",
        nargs="+",
        help="Harbor jobs dirs (local paths and optional ssh host:path)",
    )
    parser.add_argument(
        "--out",
        default=str(DEFAULT_OUT_DIR),
        help="Output directory (default: experiments/pipeline)",
    )
    parser.add_argument(
        "--config",
        default=None,
        help="YAML with composer_token_cap (default: <out>/config.yaml)",
    )
    args = parser.parse_args(list(argv) if argv is not None else None)
    out_dir = Path(args.out)
    config_path = Path(args.config) if args.config else out_dir / "config.yaml"
    df = results_dashboard(args.jobs, out_dir=out_dir, config_path=config_path)
    parquet_path = out_dir / "results.parquet"
    md_path = out_dir / "results.md"
    print(f"wrote {parquet_path} ({len(df)} rows) and {md_path}", flush=True)
    return 0
