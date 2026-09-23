#!/usr/bin/env python3
"""One-shot Composer calls: the replacement for OpenRouter free-tier in the gate stack.

Why: four gate jobs were all pointed at OpenRouter's `:free` model variants and spent an
hour queueing behind that provider's shared per-model rate limits, while the account had
spent $0.00. Composer is already paid for, is not rate-limited that way, and answers a
one-shot prompt in ~10s.

Not a chat client -- a cached, resumable, batch-friendly "prompt in, text out" helper:

  from openswe_traces.ops.ask_composer import ask, ask_many
  text = ask("summarise this contract", cache_key="unit/repindex/summary")
  out  = ask_many([(key, prompt), ...], workers=3)

Every answer is cached under outputs/composer_cache/<sha>.json, so a re-run after a crash
costs nothing for prompts already answered. Cache hits are free and instant.
"""
from __future__ import annotations
from openswe_traces.paths import REPO as _REPO

import hashlib
import json
import os
import subprocess
import sys
import tempfile
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

CACHE = Path(os.environ.get("COMPOSER_CACHE", "outputs/composer_cache"))
MODEL = os.environ.get("COMPOSER_MODEL", "composer-2.5")
TIMEOUT = int(os.environ.get("COMPOSER_TIMEOUT", "300"))


def _key(prompt: str, cache_key: str | None) -> str:
    return hashlib.sha256((cache_key or prompt).encode()).hexdigest()[:32]


def ask(prompt: str, *, cache_key: str | None = None, model: str = MODEL,
        timeout: int = TIMEOUT) -> str:
    """Send one prompt, return the text. Cached by prompt (or by an explicit cache_key)."""
    CACHE.mkdir(parents=True, exist_ok=True)
    hit = CACHE / f"{_key(prompt, cache_key)}.json"
    if hit.is_file():
        try:
            return json.loads(hit.read_text())["text"]
        except Exception:
            pass
    # A fresh empty cwd keeps the agent from wandering into the repo and using tools.
    # The cursor credentials live in an env file the sweeps source; without them the CLI
    # falls back to a free plan and refuses a named model ("Free plans can only use Auto").
    env = os.environ.copy()
    for envfile in (Path("/home/evan/Documents/eval_tasks/.env"),
                    _REPO / ".env"):
        if not envfile.is_file():
            continue
        for raw in envfile.read_text(errors="replace").splitlines():
            raw = raw.strip()
            if raw.startswith("#") or "=" not in raw:
                continue
            k, v = raw.split("=", 1)
            k, v = k.strip(), v.strip().strip('"').strip("'")
            # The env file is authoritative: an inherited CURSOR_API_KEY from an earlier
            # shell can belong to a different (free-plan) account, which the CLI rejects
            # with "Named models unavailable".
            if v:
                env[k] = v
    with tempfile.TemporaryDirectory() as td:
        proc = subprocess.run(
            ["cursor-agent", "-p", "--output-format", "text", "--model", model, "--trust", prompt],
            cwd=td, capture_output=True, text=True, timeout=timeout, env=env,
        )
    text = (proc.stdout or "").strip()
    if proc.returncode != 0 and not text:
        raise RuntimeError(f"composer failed rc={proc.returncode}: {(proc.stderr or '')[-300:]}")
    hit.write_text(json.dumps({"text": text, "model": model, "cache_key": cache_key}))
    return text


def ask_many(items: list[tuple[str, str]], *, workers: int = 3, model: str = MODEL) -> dict[str, str]:
    """[(cache_key, prompt)] -> {cache_key: text}. Keep workers low; each call is an agent."""
    out: dict[str, str] = {}

    def one(kp: tuple[str, str]) -> tuple[str, str]:
        k, p = kp
        try:
            return k, ask(p, cache_key=k, model=model)
        except Exception as exc:  # noqa: BLE001
            return k, f"__ERROR__ {exc}"

    with ThreadPoolExecutor(max_workers=workers) as ex:
        for k, v in ex.map(one, items):
            out[k] = v
    return out


def cli():
    p = sys.stdin.read() if len(sys.argv) < 2 else " ".join(sys.argv[1:])
    print(ask(p))


if __name__ == "__main__":
    cli()
