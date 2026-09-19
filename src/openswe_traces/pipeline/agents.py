"""Model-agnostic `run_agent(role, brief, cwd, model)` with cursor and Devin backends."""

from __future__ import annotations

import os
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.semaphore import DevinSemaphore
from openswe_traces.pipeline.tokens import TokenBudget, parse_cursor_agent_output


@dataclass(frozen=True)
class AgentResult:
    text: str
    tokens_in: int
    tokens_out: int
    backend: str
    model: str
    cwd: Path
    returncode: int


def load_env_file(path: Path | str, *, environ: dict[str, str] | None = None) -> dict[str, str]:
    """Load KEY=VALUE lines. Never log values."""
    dest = environ if environ is not None else os.environ
    path = Path(path)
    if not path.is_file():
        return dest
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        if key:
            dest.setdefault(key, value)
    return dest


class AgentRunner:
    def __init__(
        self,
        *,
        cursor_env_file: Path | str | None = None,
        semaphore: DevinSemaphore | None = None,
        budget: TokenBudget | None = None,
        popen: Any = subprocess.run,
    ) -> None:
        self.cursor_env_file = Path(cursor_env_file) if cursor_env_file else None
        self.semaphore = semaphore
        self.budget = budget
        self._run = popen

    def run_agent(
        self,
        role: str,
        brief: str,
        cwd: Path | str,
        model: str,
        *,
        backend: str = "cursor",
        timeout_sec: int | None = None,
        output_format: str = "json",
    ) -> AgentResult:
        cwd = Path(cwd)
        cwd.mkdir(parents=True, exist_ok=True)
        backend = (backend or "cursor").lower()
        if backend == "cursor":
            return self._run_cursor(role, brief, cwd, model, timeout_sec, output_format)
        if backend == "devin":
            return self._run_devin(role, brief, cwd, model, timeout_sec)
        raise ValueError(f"unknown agent backend {backend!r}")

    def _run_cursor(
        self,
        role: str,
        brief: str,
        cwd: Path,
        model: str,
        timeout_sec: int | None,
        output_format: str,
    ) -> AgentResult:
        env = os.environ.copy()
        if self.cursor_env_file:
            load_env_file(self.cursor_env_file, environ=env)
        cmd = [
            "cursor-agent",
            "-p",
            brief,
            "-f",
            "--model",
            model,
            "--output-format",
            output_format,
        ]
        proc = self._run(
            cmd,
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=timeout_sec,
            check=False,
            env=env,
        )
        text = (proc.stdout or "") + (proc.stderr or "")
        tin, tout = parse_cursor_agent_output(proc.stdout or "")
        if self.budget is not None:
            self.budget.add(tin, tout, "cursor-agent")
        _ = role
        return AgentResult(
            text=text,
            tokens_in=tin,
            tokens_out=tout,
            backend="cursor",
            model=model,
            cwd=cwd,
            returncode=int(proc.returncode or 0),
        )

    def _run_devin(
        self,
        role: str,
        brief: str,
        cwd: Path,
        model: str,
        timeout_sec: int | None,
    ) -> AgentResult:
        cmd = ["devin", "-p", brief, "--model", model, "--permission-mode", "dangerous"]
        sem = self.semaphore
        if sem is None:
            proc = self._invoke(cmd, cwd, timeout_sec)
        else:
            with sem.hold(kind=f"agent:{role}", n=1, timeout=timeout_sec):
                proc = self._invoke(cmd, cwd, timeout_sec)
        text = (proc.stdout or "") + (proc.stderr or "")
        return AgentResult(
            text=text,
            tokens_in=0,
            tokens_out=0,
            backend="devin",
            model=model,
            cwd=cwd,
            returncode=int(proc.returncode or 0),
        )

    def _invoke(self, cmd: list[str], cwd: Path, timeout_sec: int | None) -> Any:
        return self._run(
            cmd,
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=timeout_sec,
            check=False,
        )


def run_agent(
    role: str,
    brief: str,
    cwd: Path | str,
    model: str,
    *,
    backend: str = "cursor",
    timeout_sec: int | None = None,
    runner: AgentRunner | None = None,
    **kwargs: Any,
) -> AgentResult:
    inst = runner or AgentRunner()
    return inst.run_agent(
        role, brief, cwd, model, backend=backend, timeout_sec=timeout_sec, **kwargs
    )
