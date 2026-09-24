"""Composer (cursor-agent) client for the gate stack.

The gate's LLM calls go through one-shot ``cursor-agent -p`` invocations —
never OpenRouter, whose shared free-tier rate limits stalled four jobs for
two hours while the account spent $0.00. Composer is already paid for and
answers a ~6k-token prompt in ~40s.

Two traps already debugged elsewhere (``scripts/ops/ask_composer.py``):

- the credentials in ``/home/evan/Documents/eval_tasks/.env`` must WIN over
  any inherited ``CURSOR_API_KEY``; a stale shell key belongs to a free-plan
  account and the CLI then refuses named models
  (``ActionRequiredError: Named models unavailable``).
- calls run in an empty temp cwd, or the agent wanders into the repo and
  starts using tools.

``--output-format json`` returns real usage (``inputTokens`` /
``outputTokens`` / cache tokens), which is what the loop's per-unit cost
accounting records. Answers are cached under ``outputs/composer_cache/``
keyed by a caller-supplied ``cache_key`` (or the prompt), sharing the file
format of ``scripts/ops/ask_composer.py`` so caches interop.
"""
from __future__ import annotations

import hashlib
import json
import os
import subprocess
import tempfile
from pathlib import Path

CACHE = Path(os.environ.get("COMPOSER_CACHE", "outputs/composer_cache"))
MODEL = os.environ.get("COMPOSER_MODEL", "composer-2.5")
TIMEOUT = int(os.environ.get("COMPOSER_TIMEOUT", "300"))

_ENV_FILES = (
    Path("/home/evan/Documents/eval_tasks/.env"),
)


def _key(prompt: str, cache_key: str | None) -> str:
    return hashlib.sha256((cache_key or prompt).encode()).hexdigest()[:32]


def _env() -> dict[str, str]:
    env = os.environ.copy()
    for envfile in _ENV_FILES:
        if not envfile.is_file():
            continue
        for raw in envfile.read_text(errors="replace").splitlines():
            raw = raw.strip()
            if raw.startswith("#") or "=" not in raw:
                continue
            k, v = raw.split("=", 1)
            k, v = k.strip(), v.strip().strip('"').strip("'")
            # the env file is authoritative over inherited keys
            if v:
                env[k] = v
    return env


def _usage_tokens(usage: dict) -> int:
    return int(usage.get("inputTokens") or 0) + int(usage.get("outputTokens") or 0)


def ask(prompt: str, *, cache_key: str | None = None, model: str = MODEL,
        timeout: int = TIMEOUT) -> tuple[str, int]:
    """One prompt, one ``cursor-agent`` call. Returns (text, tokens).

    Raises RuntimeError on CLI failure or an error result; callers do their
    own retry/budget accounting.
    """
    CACHE.mkdir(parents=True, exist_ok=True)
    hit = CACHE / f"{_key(prompt, cache_key)}.json"
    if hit.is_file():
        try:
            d = json.loads(hit.read_text())
            return d["text"], int(d.get("tokens", 0))
        except (OSError, json.JSONDecodeError, KeyError, TypeError, ValueError):
            pass
    with tempfile.TemporaryDirectory() as td:
        proc = subprocess.run(
            ["cursor-agent", "-p", "--output-format", "json",
             "--model", model, "--trust", "--mode", "ask", prompt],
            cwd=td, capture_output=True, text=True, timeout=timeout,
            env=_env(), check=False,
        )
    raw = (proc.stdout or "").strip()
    text, tokens, is_error, err_msg = "", 0, False, ""
    for line in raw.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            data = json.loads(line)
        except json.JSONDecodeError:
            continue
        if data.get("type") == "result":
            text = str(data.get("result") or "").strip()
            usage = data.get("usage") or {}
            tokens = _usage_tokens(usage)
            is_error = bool(data.get("is_error"))
            if is_error:
                err_msg = text or "cursor-agent reported is_error"
            break
    if not text:
        tail = (proc.stderr or raw)[-300:]
        raise RuntimeError(
            f"composer failed rc={proc.returncode} is_error={is_error}: "
            f"{err_msg or tail}")
    hit.write_text(json.dumps(
        {"text": text, "model": model, "cache_key": cache_key,
         "tokens": tokens}))
    return text, tokens


def check_ready() -> None:
    """Fail fast if the CLI or its credentials are missing."""
    import shutil
    if not shutil.which("cursor-agent"):
        raise SystemExit("gate: cursor-agent not on PATH")
    if not any(p.is_file() for p in _ENV_FILES):
        raise SystemExit(
            "gate: no credential env file "
            "(expected /home/evan/Documents/eval_tasks/.env)")
