"""Harbor launch safety: agent allowlist, verifier no-network, checksums, no .git."""

from __future__ import annotations

import json
import os
import re
import tomllib
from pathlib import Path
from typing import Any

from openswe_traces.pipeline_ext.timeouts import normalize_backend

CURSOR_HOSTS: tuple[str, ...] = (
    "cursor.com",
    "*.cursor.com",
    "*.cursor.sh",
    "downloads.cursor.com",
)
DEVIN_HOSTS: tuple[str, ...] = (
    "devin.ai",
    "*.devin.ai",
    "cli.devin.ai",
    "cognition.ai",
    "*.cognition.ai",
    "api.devin.ai",
    "server.codeium.com",
)

_ALLOWED_HOSTS_RE = re.compile(r"allowed_hosts\s*=\s*\[[^\]]*\]")


class TaskSafetyError(RuntimeError):
    """Raised when a Harbor task.toml / tree is unsafe to launch."""


def solver_hosts(solver: str) -> tuple[str, ...]:
    backend = normalize_backend(solver)
    if backend == "devin":
        return DEVIN_HOSTS
    return CURSOR_HOSTS


def render_safe_task_toml(solver: str = "cursor", *, timeout_sec: int = 3600) -> str:
    hosts = ", ".join(f'"{h}"' for h in solver_hosts(solver))
    return (
        'schema_version = "1.3"\n\n'
        "[metadata]\n"
        'category = "software-engineering"\n'
        'tags = ["go", "bugfix"]\n\n'
        "[verifier]\n"
        'network_mode = "no-network"\n'
        "timeout_sec = 1800.0\n\n"
        "[agent]\n"
        'network_mode = "allowlist"\n'
        f"allowed_hosts = [{hosts}]\n"
        f"timeout_sec = {int(timeout_sec)}\n\n"
        "[environment]\n"
        "build_timeout_sec = 1800.0\n"
        'network_mode = "public"\n'
    )


def write_safe_task_skeleton(task_dir: Path, *, solver: str = "cursor") -> Path:
    """Minimal Harbor task tree that passes :func:`assert_harbor_safe`."""
    task_dir = Path(task_dir)
    task_dir.mkdir(parents=True, exist_ok=True)
    (task_dir / "task.toml").write_text(render_safe_task_toml(solver), encoding="utf-8")
    tests = task_dir / "tests"
    tests.mkdir(parents=True, exist_ok=True)
    (tests / "test.sh").write_text(
        "#!/bin/bash\n"
        "set -euo pipefail\n"
        "checksum_fail() { echo \"test file modified: $1\" >&2; exit 1; }\n"
        'echo "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  '
        '$HIDDEN/empty.go" | sha256sum -c --status || checksum_fail "hidden/empty.go"\n',
        encoding="utf-8",
    )
    src = task_dir / "environment" / "src"
    src.mkdir(parents=True, exist_ok=True)
    (task_dir / "environment" / "Dockerfile").write_text(
        "FROM golang:1.23\nWORKDIR /app\nCOPY src/ /app/\n"
        "RUN find /app -name .git -type d -prune -exec rm -rf {} + || true\n",
        encoding="utf-8",
    )
    return task_dir


def load_task_toml(task_dir: Path) -> dict[str, Any]:
    path = Path(task_dir) / "task.toml"
    if not path.is_file():
        raise TaskSafetyError(f"refuse: missing task.toml in {task_dir}")
    try:
        data = tomllib.loads(path.read_text(encoding="utf-8"))
    except tomllib.TOMLDecodeError as exc:
        raise TaskSafetyError(f"refuse: invalid task.toml in {task_dir}: {exc}") from exc
    if not isinstance(data, dict):
        raise TaskSafetyError(f"refuse: task.toml is not a table in {task_dir}")
    return data


def _section(data: dict[str, Any], name: str) -> dict[str, Any]:
    raw = data.get(name)
    return raw if isinstance(raw, dict) else {}


def checksum_guarded(task_dir: Path) -> bool:
    sh = Path(task_dir) / "tests" / "test.sh"
    if not sh.is_file():
        return False
    try:
        text = sh.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return False
    return "sha256sum -c" in text


def find_git_entries(task_dir: Path) -> list[Path]:
    """Return paths named ``.git`` under the task tree (must be empty)."""
    hits: list[Path] = []
    root = Path(task_dir)
    if not root.is_dir():
        return hits
    for dirpath, dirnames, filenames in os.walk(root):
        if ".git" in dirnames:
            hits.append(Path(dirpath) / ".git")
        if ".git" in filenames:
            hits.append(Path(dirpath) / ".git")
        dirnames[:] = [d for d in dirnames if d != ".git"]
    return hits


def dockerfile_copies_git(task_dir: Path) -> bool:
    docker = Path(task_dir) / "environment" / "Dockerfile"
    if not docker.is_file():
        docker = Path(task_dir) / "Dockerfile"
    if not docker.is_file():
        return False
    text = docker.read_text(encoding="utf-8", errors="replace")
    if re.search(r"^\s*COPY\s+.*\.git\b", text, re.MULTILINE | re.IGNORECASE):
        return not ("rm -rf" in text and ".git" in text)
    return False


def apply_solver_network(task_dir: Path, solver: str) -> None:
    """Set ``[agent]`` allowlist to the solver's API hosts only."""
    path = Path(task_dir) / "task.toml"
    if not path.is_file():
        return
    hosts_lit = json.dumps(list(solver_hosts(solver)))
    text = path.read_text(encoding="utf-8")
    if "[agent]" not in text:
        text += (
            "\n[agent]\n"
            'network_mode = "allowlist"\n'
            f"allowed_hosts = {hosts_lit}\n"
        )
        path.write_text(text, encoding="utf-8")
        return
    head, rest = text.split("[agent]", 1)
    nxt = rest.find("\n[")
    body, tail = (rest, "") if nxt < 0 else (rest[:nxt], rest[nxt:])
    if _ALLOWED_HOSTS_RE.search(body):
        body = _ALLOWED_HOSTS_RE.sub(f"allowed_hosts = {hosts_lit}", body, count=1)
    else:
        body = body.rstrip() + f"\nallowed_hosts = {hosts_lit}\n"
    path.write_text(head + "[agent]" + body + tail, encoding="utf-8")


def assert_harbor_safe(task_dir: Path, *, solver: str = "cursor") -> None:
    """Read task.toml and refuse unless the launch is allowlisted + checksum-guarded."""
    task_dir = Path(task_dir)
    data = load_task_toml(task_dir)
    agent = _section(data, "agent")
    verifier = _section(data, "verifier")
    agent_mode = str(agent.get("network_mode") or "").strip().strip('"')
    if agent_mode != "allowlist":
        raise TaskSafetyError(
            f"refuse: [agent] network_mode must be allowlist, got {agent_mode!r} in {task_dir}"
        )
    allowed = agent.get("allowed_hosts") or []
    if isinstance(allowed, str):
        allowed = [allowed]
    if not isinstance(allowed, list) or not allowed:
        raise TaskSafetyError(f"refuse: [agent] allowed_hosts empty in {task_dir}")
    permitted = set(solver_hosts(solver))
    extra = [str(h) for h in allowed if str(h) not in permitted]
    if extra:
        raise TaskSafetyError(
            f"refuse: [agent] allowed_hosts has hosts outside {solver} allowlist: {extra}"
        )
    ver_mode = str(verifier.get("network_mode") or "").strip().strip('"')
    if ver_mode not in {"no-network", "none"}:
        raise TaskSafetyError(
            f"refuse: [verifier] network_mode must be no-network, got {ver_mode!r} in {task_dir}"
        )
    if not checksum_guarded(task_dir):
        raise TaskSafetyError(f"refuse: tests/test.sh is not checksum-guarded in {task_dir}")
    git_hits = find_git_entries(task_dir)
    if git_hits:
        raise TaskSafetyError(f"refuse: image/tree carries .git: {git_hits[0]}")
    if dockerfile_copies_git(task_dir):
        raise TaskSafetyError(f"refuse: Dockerfile copies .git in {task_dir}")
