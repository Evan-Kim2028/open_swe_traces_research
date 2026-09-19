"""Global Devin session cap (author + verifier + Harbor --agent devin)."""

from __future__ import annotations

import fcntl
import json
import os
import time
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass
from pathlib import Path
from typing import Any

DEFAULT_PATH = Path("/tmp/devin.slots")
DEFAULT_SLOTS = 4


def pid_alive(pid: int) -> bool:
    if pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


@dataclass(frozen=True)
class Slot:
    id: str
    kind: str
    pid: int


class DevinSemaphore:
    """File-lock semaphore shared across processes. Default 4 slots at /tmp/devin.slots."""

    def __init__(self, path: Path | str = DEFAULT_PATH, slots: int = DEFAULT_SLOTS) -> None:
        self.path = Path(path)
        self.slots = int(slots)
        self.path.mkdir(parents=True, exist_ok=True)
        self._lock_path = self.path / "lock"
        self._state_path = self.path / "holders.json"
        self._mine: list[str] = []

    def _lock_fh(self):
        self._lock_path.touch(exist_ok=True)
        return self._lock_path.open("a+")

    def _load_unlocked(self) -> list[dict[str, Any]]:
        if not self._state_path.is_file():
            return []
        try:
            data = json.loads(self._state_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return []
        if isinstance(data, list):
            return [h for h in data if isinstance(h, dict)]
        if isinstance(data, dict) and isinstance(data.get("holders"), list):
            return [h for h in data["holders"] if isinstance(h, dict)]
        return []

    def _save_unlocked(self, holders: list[dict[str, Any]]) -> None:
        self._state_path.write_text(json.dumps(holders, indent=2) + "\n", encoding="utf-8")

    def _reap(self, holders: list[dict[str, Any]]) -> list[dict[str, Any]]:
        live: list[dict[str, Any]] = []
        for h in holders:
            pid = int(h.get("pid") or 0)
            if pid_alive(pid):
                live.append(h)
        return live

    def _with_lock(self, fn):
        with self._lock_fh() as fh:
            fcntl.flock(fh, fcntl.LOCK_EX)
            try:
                holders = self._reap(self._load_unlocked())
                result = fn(holders)
                self._save_unlocked(holders)
                return result
            finally:
                fcntl.flock(fh, fcntl.LOCK_UN)

    def available(self) -> int:
        def _fn(holders: list[dict[str, Any]]) -> int:
            return max(0, self.slots - len(holders))

        return int(self._with_lock(_fn))

    def held(self) -> int:
        return self.slots - self.available()

    def harbor_n_concurrent(self, requested: int) -> int:
        """Cap Harbor --n-concurrent so Devin sessions stay <= slots."""
        return max(0, min(int(requested), self.available()))

    def acquire(self, *, kind: str = "session", n: int = 1, timeout: float | None = None) -> list[Slot]:
        if n <= 0:
            return []
        deadline = None if timeout is None else time.time() + timeout
        while True:

            def _fn(holders: list[dict[str, Any]]) -> list[Slot] | None:
                free = self.slots - len(holders)
                if free < n:
                    return None
                got: list[Slot] = []
                for i in range(n):
                    sid = f"{os.getpid()}-{time.time_ns()}-{i}"
                    holders.append({"id": sid, "kind": kind, "pid": os.getpid()})
                    self._mine.append(sid)
                    got.append(Slot(id=sid, kind=kind, pid=os.getpid()))
                return got

            got = self._with_lock(_fn)
            if got is not None:
                return got
            if deadline is not None and time.time() >= deadline:
                raise TimeoutError(f"devin semaphore: {n} slots not free (cap={self.slots})")
            time.sleep(0.05)

    def release(self, ids: list[str] | None = None) -> None:
        drop = set(ids if ids is not None else list(self._mine))

        def _fn(holders: list[dict[str, Any]]) -> None:
            keep = [h for h in holders if str(h.get("id")) not in drop]
            holders[:] = keep

        self._with_lock(_fn)
        self._mine = [s for s in self._mine if s not in drop]

    @contextmanager
    def hold(self, *, kind: str = "session", n: int = 1, timeout: float | None = None) -> Iterator[list[Slot]]:
        slots = self.acquire(kind=kind, n=n, timeout=timeout)
        try:
            yield slots
        finally:
            self.release([s.id for s in slots])
