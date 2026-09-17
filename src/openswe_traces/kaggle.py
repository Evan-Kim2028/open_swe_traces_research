"""Kaggle kernel helpers: push, poll status, and download outputs.

Shells out to the kaggle CLI installed as a project dependency; the access token at
~/.kaggle/access_token is picked up automatically.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
import time
from pathlib import Path

TERMINAL_STATUS_RE = re.compile(r"complete|error|cancel", re.IGNORECASE)
DEFAULT_TIMEOUT = 43_200
DEFAULT_POLL_SECONDS = 20.0


def _kaggle(*args: str) -> str:
    result = subprocess.run(
        [sys.executable, "-m", "kaggle", *args],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip()
        raise RuntimeError(f"kaggle {' '.join(args)} failed ({result.returncode}): {detail}")
    return result.stdout.strip()


def read_kernel_id(kernel_dir: Path) -> str:
    metadata = json.loads((Path(kernel_dir) / "kernel-metadata.json").read_text())
    return metadata["id"]


def push(kernel_dir: Path, *, timeout: int = DEFAULT_TIMEOUT) -> str:
    """Push the kernel in kernel_dir; timeout caps the run and saves GPU quota."""
    return _kaggle("kernels", "push", "-p", str(kernel_dir), "-t", str(int(timeout)))


def status(kernel_id: str) -> str:
    """Current KernelWorkerStatus string for a pushed kernel."""
    return _kaggle("kernels", "status", kernel_id)


def wait(kernel_id: str, *, poll_seconds: float = DEFAULT_POLL_SECONDS) -> str:
    """Poll status until it reaches complete/error/cancel; return the final status."""
    while True:
        current = status(kernel_id)
        if TERMINAL_STATUS_RE.search(current):
            return current
        time.sleep(poll_seconds)


def output(kernel_id: str, out_dir: Path) -> Path:
    """Download the kernel's output files into out_dir."""
    _kaggle("kernels", "output", kernel_id, "-p", str(out_dir))
    return Path(out_dir)
