"""Harbor web-use audit (same signals as scripts/ops/harbor_web_audit.py)."""

from __future__ import annotations

import json
import re
from collections.abc import Mapping
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

WEB_FETCH_RE = re.compile(r"webFetchToolCall")
WEB_SEARCH_RE = re.compile(r"webSearchToolCall")
CHECKSUM_RE = re.compile(r"test file modified|sha256sum -c|checksum.?guard", re.IGNORECASE)


@dataclass(frozen=True)
class TrialAudit:
    trial_dir: Path
    task: str
    verdict: str
    reward: float | None
    wall_minutes: float | None
    web_fetch: int
    web_search: int
    checksum_guard: bool
    tokens_in: int
    tokens_out: int
    result: dict[str, Any]


def _read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def _load_json(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {}
    try:
        data = json.loads(_read(path) or "{}")
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def _wall_minutes(result: Mapping[str, Any]) -> float | None:
    try:
        start = datetime.fromisoformat(str(result["started_at"]))
        end = datetime.fromisoformat(str(result["finished_at"]))
        return round((end - start).total_seconds() / 60, 1)
    except (KeyError, TypeError, ValueError):
        return None


def _reward(result: Mapping[str, Any]) -> float | None:
    vr = result.get("verifier_result")
    if isinstance(vr, dict):
        rewards = vr.get("rewards")
        if isinstance(rewards, dict) and rewards.get("reward") is not None:
            try:
                return float(rewards["reward"])
            except (TypeError, ValueError):
                return None
    if result.get("reward") is not None:
        try:
            return float(result["reward"])
        except (TypeError, ValueError):
            return None
    return None


def _trajectory_tokens(trial_dir: Path) -> tuple[int, int]:
    path = trial_dir / "agent" / "trajectory.json"
    if not path.is_file():
        return 0, 0
    try:
        metrics = (
            json.loads(path.read_text(encoding="utf-8", errors="replace")).get("final_metrics")
            or {}
        )
    except (OSError, ValueError, AttributeError):
        return 0, 0
    return int(metrics.get("total_prompt_tokens") or 0), int(
        metrics.get("total_completion_tokens") or 0
    )


def audit_trial_dir(trial_dir: Path) -> TrialAudit:
    from openswe_traces.pipeline.tokens import parse_harbor_trial_tokens

    trial_dir = Path(trial_dir)
    agent_txt = ""
    agent = trial_dir / "agent"
    if agent.is_dir():
        for path in agent.iterdir():
            if path.is_file():
                agent_txt += _read(path)
    result = _load_json(trial_dir / "result.json")
    wf = len(WEB_FETCH_RE.findall(agent_txt))
    ws = len(WEB_SEARCH_RE.findall(agent_txt))
    blob = agent_txt + json.dumps(result)
    checksum = bool(CHECKSUM_RE.search(blob))
    reward = _reward(result)
    if wf or ws:
        verdict = "CONTAMINATED"
    elif checksum:
        verdict = "CHECKSUM"
    elif reward == 1.0:
        verdict = "PASS"
    elif reward == 0.0:
        verdict = "FAIL"
    else:
        verdict = "PENDING"
    tin, tout = parse_harbor_trial_tokens(result)
    if not tin and not tout:
        # Devin CLI trials report no tokens to Harbor: use the ATIF trajectory totals
        # (prompt tokens include cached context; the session db under-counts by ~10x).
        tin, tout = _trajectory_tokens(trial_dir)
        if not tin and not tout:
            from openswe_traces.results import _devin_tokens

            din, dout = _devin_tokens(trial_dir)
            tin, tout = int(din or 0), int(dout or 0)
    name = trial_dir.name.split("__")[0]
    return TrialAudit(
        trial_dir=trial_dir,
        task=name,
        verdict=verdict,
        reward=reward,
        wall_minutes=_wall_minutes(result),
        web_fetch=wf,
        web_search=ws,
        checksum_guard=checksum,
        tokens_in=tin,
        tokens_out=tout,
        result=result,
    )


def iter_trial_dirs(job_dir: Path | str) -> list[Path]:
    root = Path(job_dir)
    if not root.is_dir():
        return []
    out: list[Path] = []
    for child in sorted(root.iterdir()):
        if child.is_dir() and (child / "agent").is_dir():
            out.append(child)
    return out


def audit_job(job_dir: Path | str) -> list[TrialAudit]:
    return [audit_trial_dir(p) for p in iter_trial_dirs(job_dir)]


def audit_class(verdict: str) -> str:
    return {
        "CONTAMINATED": "contaminated",
        "CHECKSUM": "checksum",
        "PASS": "clean",
        "FAIL": "clean",
        "PENDING": "infra",
    }.get(verdict, "infra")
