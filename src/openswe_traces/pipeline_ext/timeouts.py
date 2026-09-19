"""Backend-specific time budgets. Timeouts are class (d), not failures (C1)."""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from typing import Any

TIMEOUT_CLASS = "d"
DEFAULT_BACKEND = "composer"

# Harbor trial agent timeout (seconds).
AGENT_TIMEOUT_SEC: dict[str, int] = {
    "devin": 14400,
    "composer": 3600,
    "cursor": 3600,
    "grok": 3600,
}

# Author / verifier session timeout (seconds).
SESSION_TIMEOUT_SEC: dict[str, int] = {
    "devin": 90 * 60,
    "grok": 60 * 60,
    "composer": 60 * 60,
    "cursor": 60 * 60,
}

POLL_INTERVAL_SEC: dict[str, int] = {
    "devin": 120,
    "composer": 60,
    "cursor": 60,
    "grok": 60,
}

_ALIASES: dict[str, str] = {
    "devin": "devin",
    "swe-2-max": "devin",
    "swe-2": "devin",
    "composer": "composer",
    "composer-2.5": "composer",
    "cursor": "cursor",
    "cursor-cli": "cursor",
    "cursor/composer-2.5": "composer",
    "grok": "grok",
    "grok-4.6": "grok",
    "cursor-grok-4.6-high": "grok",
}


def normalize_backend(name: str) -> str:
    key = (name or "").strip().lower()
    if key in _ALIASES:
        return _ALIASES[key]
    if "devin" in key or "swe-2" in key:
        return "devin"
    if "grok" in key:
        return "grok"
    if "composer" in key:
        return "composer"
    if "cursor" in key:
        return "cursor"
    return key or "composer"


@dataclass(frozen=True)
class TimeoutBudget:
    backend: str
    agent_timeout_sec: int
    session_timeout_sec: int
    poll_interval_sec: int
    timeout_class: str = TIMEOUT_CLASS

    def as_dict(self) -> dict[str, Any]:
        return {
            "backend": self.backend,
            "agent_timeout_sec": self.agent_timeout_sec,
            "session_timeout_sec": self.session_timeout_sec,
            "poll_interval_sec": self.poll_interval_sec,
            "timeout_class": self.timeout_class,
        }


def agent_timeout_sec(backend: str) -> int:
    n = normalize_backend(backend)
    return AGENT_TIMEOUT_SEC.get(n, AGENT_TIMEOUT_SEC[DEFAULT_BACKEND])


def session_timeout_sec(backend: str) -> int:
    n = normalize_backend(backend)
    return SESSION_TIMEOUT_SEC.get(n, SESSION_TIMEOUT_SEC[DEFAULT_BACKEND])


def poll_interval_sec(backend: str) -> int:
    n = normalize_backend(backend)
    return POLL_INTERVAL_SEC.get(n, POLL_INTERVAL_SEC[DEFAULT_BACKEND])


def timeout_budget(backend: str) -> TimeoutBudget:
    n = normalize_backend(backend)
    if n not in AGENT_TIMEOUT_SEC:
        n = DEFAULT_BACKEND
    return TimeoutBudget(
        backend=n,
        agent_timeout_sec=AGENT_TIMEOUT_SEC[n],
        session_timeout_sec=SESSION_TIMEOUT_SEC[n],
        poll_interval_sec=POLL_INTERVAL_SEC[n],
        timeout_class=TIMEOUT_CLASS,
    )


def classify_attempt(*, timed_out: bool, passed: bool | None = None) -> str:
    """C1 class. Timeouts are ``d`` (infra) and must not count as solver fails."""
    if timed_out:
        return TIMEOUT_CLASS
    if passed is True:
        return "a"
    if passed is False:
        return "a"
    return TIMEOUT_CLASS


def counts_as_failure(*, timed_out: bool, passed: bool) -> bool:
    if timed_out:
        return False
    return not passed


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Print backend timeout budget as JSON")
    parser.add_argument("--backend", required=True)
    args = parser.parse_args(argv)
    json.dump(timeout_budget(args.backend).as_dict(), sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
