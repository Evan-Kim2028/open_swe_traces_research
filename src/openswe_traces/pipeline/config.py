"""Pipeline knobs: experiments/pipeline/config.yaml + repos.yaml."""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.data import ROOT
from openswe_traces.pipeline.yamlutil import load_yaml_path

DEFAULT_DIR = ROOT / "experiments" / "pipeline"
DEFAULT_CONFIG = DEFAULT_DIR / "config.yaml"
DEFAULT_REPOS = DEFAULT_DIR / "repos.yaml"
CURSOR_ENV_FILE = Path("/home/evan/Documents/eval_tasks/.env")
DEVIN_SLOTS_PATH = Path("/tmp/devin.slots")


@dataclass(frozen=True)
class RepoSpec:
    name: str
    url: str
    commit: str
    language: str = "go"
    module: str = ""


@dataclass(frozen=True)
class HostSpec:
    name: str
    docker_concurrency: int = 4
    ssh: str = ""
    enabled: bool = True


@dataclass(frozen=True)
class PipelineConfig:
    units_per_author_batch: int = 10
    author_minutes: int = 30
    solver_order: tuple[str, ...] = ("cursor", "devin")
    solver_backends: tuple[str, ...] = ("cursor", "devin")
    climb_levels: tuple[int, ...] = (2, 5, 6)
    attempts: int = 3
    composer_token_cap: int = 1_000_000_000
    devin_slots: int = 6
    devin_slots_path: Path = DEVIN_SLOTS_PATH
    default_host: str = "laptop"
    hosts: dict[str, HostSpec] = field(default_factory=dict)
    cursor_model: str = "composer-2.5"
    cursor_harbor_agent: str = "cursor-cli"
    cursor_harbor_model: str = "cursor/composer-2.5"
    cursor_env_file: Path = CURSOR_ENV_FILE
    devin_model: str = "swe-2-max"
    devin_harbor_agent: str = "devin"
    devin_harbor_model: str = "swe-2-max"
    author_backend: str = "cursor"
    author_model: str = "composer-2.5"
    verifier_backend: str = "cursor"
    verifier_model: str = "composer-2.5"
    vps_enabled: bool = False
    repos: tuple[RepoSpec, ...] = ()
    root: Path = DEFAULT_DIR
    state_db: Path = DEFAULT_DIR / "state.db"
    logs_dir: Path = DEFAULT_DIR / "logs"
    work_dir: Path = DEFAULT_DIR / "work"
    authored_dir: Path = DEFAULT_DIR / "authored"
    tasks_dir: Path = DEFAULT_DIR / "tasks"
    jobs_dir: Path = DEFAULT_DIR / "jobs"
    results_parquet: Path = DEFAULT_DIR / "results.parquet"
    results_md: Path = DEFAULT_DIR / "results.md"
    cleanup_script: Path = ROOT / "scripts" / "ops" / "docker_cleanup.sh"
    load_mult: float = 2.0

    def host(self, name: str | None = None) -> HostSpec:
        key = name or self.default_host
        if key in self.hosts:
            return self.hosts[key]
        return HostSpec(name=key, docker_concurrency=4 if key == "laptop" else 2)

    def docker_concurrency(self, host: str | None = None) -> int:
        return self.host(host).docker_concurrency

    def repo(self, name: str) -> RepoSpec:
        for spec in self.repos:
            if spec.name == name:
                return spec
        raise KeyError(f"unknown repo {name!r}")


def _as_path(value: Any, default: Path) -> Path:
    if value in (None, ""):
        return default
    path = Path(str(value))
    return path if path.is_absolute() else ROOT / path


def _hosts(raw: Any) -> dict[str, HostSpec]:
    if not isinstance(raw, dict) or not raw:
        return {
            "laptop": HostSpec(name="laptop", docker_concurrency=4),
            "vps": HostSpec(name="vps", docker_concurrency=2, ssh="lake-vps", enabled=False),
        }
    out: dict[str, HostSpec] = {}
    for name, body in raw.items():
        body = body or {}
        out[str(name)] = HostSpec(
            name=str(name),
            docker_concurrency=int(body.get("docker_concurrency") or (4 if name == "laptop" else 2)),
            ssh=str(body.get("ssh") or ("lake-vps" if name == "vps" else "")),
            enabled=bool(body.get("enabled", name != "vps")),
        )
    return out


def _repos(raw: Any) -> tuple[RepoSpec, ...]:
    if not isinstance(raw, list):
        return ()
    out: list[RepoSpec] = []
    for row in raw:
        if not isinstance(row, dict):
            continue
        name = str(row.get("name") or "").strip()
        url = str(row.get("url") or row.get("git") or "").strip()
        commit = str(row.get("commit") or "").strip()
        if not (name and url and commit):
            continue
        out.append(
            RepoSpec(
                name=name,
                url=url,
                commit=commit,
                language=str(row.get("language") or "go"),
                module=str(row.get("module") or ""),
            )
        )
    return tuple(out)


def load_config(
    config_path: Path | str | None = None,
    repos_path: Path | str | None = None,
    *,
    overrides: dict[str, Any] | None = None,
) -> PipelineConfig:
    cfg_path = Path(config_path) if config_path else DEFAULT_CONFIG
    repo_path = Path(repos_path) if repos_path else DEFAULT_REPOS
    raw: dict[str, Any] = {}
    if cfg_path.is_file():
        loaded = load_yaml_path(cfg_path)
        if isinstance(loaded, dict):
            raw.update(loaded)
    repo_raw: Any = raw.get("repos")
    if repo_path.is_file():
        extra = load_yaml_path(repo_path)
        if isinstance(extra, dict) and extra.get("repos"):
            repo_raw = extra["repos"]
        elif isinstance(extra, list):
            repo_raw = extra
    if overrides:
        raw.update(overrides)
        if "repos" in overrides:
            repo_raw = overrides["repos"]

    paths = raw.get("paths") if isinstance(raw.get("paths"), dict) else {}
    cursor = raw.get("cursor") if isinstance(raw.get("cursor"), dict) else {}
    devin = raw.get("devin") if isinstance(raw.get("devin"), dict) else {}
    author = raw.get("author") if isinstance(raw.get("author"), dict) else {}
    verifier = raw.get("verifier") if isinstance(raw.get("verifier"), dict) else {}
    hosts = _hosts(raw.get("hosts"))
    backends = (
        raw.get("solver_order")
        or raw.get("solver_backends")
        or raw.get("solver backend order")
        or ("cursor", "devin")
    )
    if isinstance(backends, str):
        backends = [s.strip() for s in backends.split(",") if s.strip()]
    climb = raw.get("climb_levels") or (2, 5, 6)
    if isinstance(climb, str):
        climb = [int(s.strip()) for s in climb.split(",") if s.strip()]
    root = _as_path(paths.get("root"), DEFAULT_DIR)
    vps = hosts.get("vps")
    return PipelineConfig(
        units_per_author_batch=int(raw.get("units_per_author_batch") or 10),
        author_minutes=int(raw.get("author_minutes") or 30),
        solver_order=tuple(str(x) for x in backends),
        solver_backends=tuple(str(x) for x in backends),
        climb_levels=tuple(int(x) for x in climb),
        attempts=int(raw.get("attempts") or 3),
        composer_token_cap=int(raw.get("composer_token_cap") or 1_000_000_000),
        devin_slots=int(raw["devin_slots"]) if raw.get("devin_slots") is not None else 6,
        devin_slots_path=_as_path(raw.get("devin_slots_path"), DEVIN_SLOTS_PATH),
        default_host=str(raw.get("default_host") or "laptop"),
        hosts=hosts,
        cursor_model=str(cursor.get("model") or "composer-2.5"),
        cursor_harbor_agent=str(cursor.get("agent") or "cursor-cli"),
        cursor_harbor_model=str(cursor.get("harbor_model") or "cursor/composer-2.5"),
        cursor_env_file=_as_path(cursor.get("env_file"), CURSOR_ENV_FILE),
        devin_model=str(devin.get("model") or "swe-2-max"),
        devin_harbor_agent=str(devin.get("harbor_agent") or "devin"),
        devin_harbor_model=str(devin.get("harbor_model") or "swe-2-max"),
        author_backend=str(author.get("backend") or "cursor"),
        author_model=str(author.get("model") or cursor.get("model") or "composer-2.5"),
        verifier_backend=str(verifier.get("backend") or "cursor"),
        verifier_model=str(verifier.get("model") or cursor.get("model") or "composer-2.5"),
        vps_enabled=bool(raw.get("vps_enabled", vps.enabled if vps else False)),
        repos=_repos(repo_raw),
        root=root,
        state_db=_as_path(paths.get("state_db"), root / "state.db"),
        logs_dir=_as_path(paths.get("logs"), root / "logs"),
        work_dir=_as_path(paths.get("work"), root / "work"),
        authored_dir=_as_path(paths.get("authored"), root / "authored"),
        tasks_dir=_as_path(paths.get("tasks"), root / "tasks"),
        jobs_dir=_as_path(paths.get("jobs"), root / "jobs"),
        results_parquet=_as_path(paths.get("results_parquet"), root / "results.parquet"),
        results_md=_as_path(paths.get("results_md"), root / "results.md"),
        cleanup_script=_as_path(raw.get("cleanup_script"), ROOT / "scripts" / "ops" / "docker_cleanup.sh"),
        load_mult=float(raw.get("load_mult") or 2.0),
    )
