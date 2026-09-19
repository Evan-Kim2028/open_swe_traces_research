"""Host load, Docker concurrency, Harbor-aware cleanup."""

from __future__ import annotations

import os
import subprocess
import time
from collections.abc import Callable
from pathlib import Path

from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.semaphore import DevinSemaphore


def n_cores() -> int:
    return os.cpu_count() or 1


def load_avg() -> float:
    try:
        return float(os.getloadavg()[0])
    except (OSError, AttributeError):
        return 0.0


def load_too_high(*, cores: int | None = None, load: float | None = None, mult: float = 2.0) -> bool:
    c = cores if cores is not None else n_cores()
    l = load if load is not None else load_avg()
    return l > mult * c


def wait_for_load(
    *,
    mult: float = 2.0,
    sleep_s: float = 30.0,
    max_wait_s: float = 3600.0,
    load_fn: Callable[[], float] = load_avg,
    cores_fn: Callable[[], int] = n_cores,
    sleep_fn: Callable[[float], None] = time.sleep,
) -> None:
    deadline = time.time() + max_wait_s
    while load_too_high(cores=cores_fn(), load=load_fn(), mult=mult):
        if time.time() >= deadline:
            raise RuntimeError(f"load still > {mult}× cores after {max_wait_s:.0f}s; skip launch")
        sleep_fn(sleep_s)


def docker_n_concurrent(cfg: PipelineConfig, host: str | None = None) -> int:
    return max(1, cfg.docker_concurrency(host))


def harbor_concurrency(
    cfg: PipelineConfig,
    solver: str,
    sem: DevinSemaphore,
    *,
    host: str | None = None,
) -> int:
    n = docker_n_concurrent(cfg, host)
    if solver == "devin":
        n = min(n, max(0, sem.available()))
    return n


def run_cleanup(script: Path | str, *, timeout: int = 180) -> str:
    script = Path(script)
    if not script.is_file():
        return f"cleanup skipped: {script} missing"
    proc = subprocess.run(
        ["bash", str(script)],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )
    return ((proc.stdout or "") + (proc.stderr or "")).strip()


def maybe_ssh(host: str, argv: list[str], *, enabled: bool) -> list[str]:
    """Prefix argv with `ssh lake-vps` when the VPS host is selected."""
    if enabled and host == "vps":
        return ["ssh", "lake-vps", "--", *argv]
    return argv
