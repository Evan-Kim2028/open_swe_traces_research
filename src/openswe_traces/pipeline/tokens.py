"""Composer token budget. Exceeding the cap switches the solver to Devin."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from openswe_traces.pipeline.state import PipelineStore

COMPOSER_SOURCES = ("cursor-cli", "cursor-agent", "cursor")


def parse_usage_blob(obj: Mapping[str, Any] | None) -> tuple[int, int]:
    if not obj:
        return 0, 0
    usage = obj.get("usage")
    if not isinstance(usage, dict):
        nested = obj.get("result")
        if isinstance(nested, dict) and isinstance(nested.get("usage"), dict):
            usage = nested["usage"]
        elif isinstance(obj.get("metadata"), dict) and isinstance(obj["metadata"].get("usage"), dict):
            usage = obj["metadata"]["usage"]
        else:
            usage = obj if any(k in obj for k in ("input_tokens", "prompt_tokens", "tokens_in")) else {}
    if not isinstance(usage, dict):
        return 0, 0
    tin = usage.get("input_tokens") or usage.get("prompt_tokens") or usage.get("tokens_in") or 0
    tout = usage.get("output_tokens") or usage.get("completion_tokens") or usage.get("tokens_out") or 0
    try:
        return int(tin), int(tout)
    except (TypeError, ValueError):
        return 0, 0


def parse_cursor_agent_output(text: str) -> tuple[int, int]:
    """Read usage from cursor-agent --output-format json (last JSON object wins)."""
    import json

    tin = tout = 0
    for line in reversed((text or "").splitlines()):
        line = line.strip()
        if not (line.startswith("{") and line.endswith("}")):
            continue
        try:
            obj = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(obj, dict):
            continue
        a, b = parse_usage_blob(obj)
        if a or b:
            return a, b
    blob = (text or "").strip()
    if blob.startswith("{") and blob.endswith("}"):
        try:
            obj = json.loads(blob)
        except json.JSONDecodeError:
            return tin, tout
        if isinstance(obj, dict):
            return parse_usage_blob(obj)
    return tin, tout


def parse_harbor_trial_tokens(result: Mapping[str, Any] | None) -> tuple[int, int]:
    """Walk common Harbor result.json shapes for input/output tokens."""
    if not result:
        return 0, 0
    candidates: list[Any] = [
        result,
        result.get("agent_result"),
        result.get("agent"),
        (result.get("agent_result") or {}).get("metadata")
        if isinstance(result.get("agent_result"), dict)
        else None,
        result.get("usage"),
        result.get("stats"),
    ]
    for cand in candidates:
        if isinstance(cand, dict):
            a, b = parse_usage_blob(cand)
            if a or b:
                return a, b
    return 0, 0


class TokenBudget:
    def __init__(self, store: PipelineStore, cap: int) -> None:
        self.store = store
        self.cap = int(cap)

    @property
    def used(self) -> int:
        return self.store.composer_tokens()

    @property
    def remaining(self) -> int:
        return max(0, self.cap - self.used)

    def exhausted(self) -> bool:
        return self.used >= self.cap

    def add(self, tokens_in: int, tokens_out: int, source: str) -> None:
        if not tokens_in and not tokens_out:
            return
        self.store.add_tokens(source, tokens_in, tokens_out)

    def choose_solver(self, backends: tuple[str, ...] | list[str]) -> str:
        order = [str(b) for b in backends if b]
        if not order:
            return "devin"
        if self.exhausted():
            if "devin" in order:
                return "devin"
            return order[-1]
        return order[0]
