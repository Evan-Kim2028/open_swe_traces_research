"""Adaptive ladder (amends C6).

Climb ``[L2, L5, L6]`` with one attempt per level. The first passing level is
the candidate flip. Then run two more attempts at the candidate and two more at
the level just below (L0 once when the candidate is L2). Confirmed iff the
candidate is >= 2/3 and the level below is < 2/3. Otherwise extend one level
in the direction the results point, again with two extra attempts. Levels that
failed on the climb get no extra attempts. L1/L3/L4 are generated only on
demand (returned as optional actions with reasons).
"""

from __future__ import annotations

import argparse
import json
import sys
from collections.abc import Iterable, Iterator, Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

# Harbor L{k} ↔ affordance.py A-level. L = A + 2. See synth/affordance.py.
LADDER_TO_AFFORDANCE: dict[int, int] = {0: -2, 1: -1, 2: 0, 3: 1, 4: 2, 5: 3, 6: 4}
AFFORDANCE_TO_LADDER: dict[int, int] = {v: k for k, v in LADDER_TO_AFFORDANCE.items()}

CLIMB_ORDER: tuple[int, ...] = (2, 5, 6)
OPTIONAL_LEVELS: tuple[int, ...] = (1, 3, 4)
ALL_LEVELS: tuple[int, ...] = (0, 1, 2, 3, 4, 5, 6)
PASS_NEED = 1  # one pass at a level is enough (2026-09-19 rule)
PASS_OF = 2  # two runs at the flip
EXTRA_ATTEMPTS = 1
L0_PROBE_ATTEMPTS = 1

OPTIONAL_REASONS: dict[int, str] = {
    1: "L2 passed and L0 failed; interpolate L0/L2 (generate L1 only on demand)",
    3: "L2 failed and a higher climb level passed; interpolate L2/L5 (generate L3 only on demand)",
    4: "L5 is in play; interpolate L2/L5 (generate L4 only on demand unless it is the confirm-below)",
}


@dataclass(frozen=True)
class LevelAttempt:
    """One Harbor (or author-session) attempt at a ladder level."""

    level: int
    passed: bool
    timeout: bool = False
    """Timeouts are class (d), not failures — they do not count toward 2/3."""


@dataclass(frozen=True)
class LadderAction:
    """Next work: run ``n_attempts`` at ``level``.

    Unpackable as ``(level, n_attempts)``. Optional actions are not required
    for the adaptive policy; the orchestrator packages them only on demand.
    """

    level: int
    n_attempts: int
    optional: bool = False
    reason: str = ""

    def as_tuple(self) -> tuple[int, int]:
        return (self.level, self.n_attempts)

    def __iter__(self) -> Iterator[int]:
        yield self.level
        yield self.n_attempts


@dataclass(frozen=True)
class FlipEvidence:
    candidate: int | None
    candidate_passes: int
    candidate_attempts: int
    below: int | None
    below_passes: int
    below_attempts: int
    climb_failed: tuple[int, ...]
    notes: str


@dataclass(frozen=True)
class FlipResult:
    level: int | None
    confirmed: bool
    evidence: FlipEvidence

    def as_tuple(self) -> tuple[int | None, bool, FlipEvidence]:
        return (self.level, self.confirmed, self.evidence)

    def __iter__(self) -> Iterator[Any]:
        yield self.level
        yield self.confirmed
        yield self.evidence


@dataclass(frozen=True)
class TaskState:
    """Per (repo, unit, solver) ladder state the orchestrator feeds us."""

    attempts: tuple[LevelAttempt, ...] = ()
    request_optional: frozenset[int] = field(default_factory=frozenset)
    """L1/L3/L4 (and extra L0) the caller explicitly asked to generate/run."""


def affordance_level(ladder: int) -> int:
    if ladder not in LADDER_TO_AFFORDANCE:
        raise ValueError(f"ladder level must be 0..6, got {ladder}")
    return LADDER_TO_AFFORDANCE[ladder]


def ladder_level(affordance: int) -> int:
    if affordance not in AFFORDANCE_TO_LADDER:
        raise ValueError(f"unknown affordance level {affordance}")
    return AFFORDANCE_TO_LADDER[affordance]


def level_below(level: int) -> int | None:
    """Numeric neighbour used for confirmation.

    Candidate L2 probes L0 (not L1). Nothing sits below L0.
    """
    if level == 2:
        return 0
    if level <= 0:
        return None
    return level - 1


def below_target_attempts(below: int) -> int:
    return L0_PROBE_ATTEMPTS if below == 0 else EXTRA_ATTEMPTS


def valid_attempts(attempts: Sequence[LevelAttempt], level: int) -> tuple[LevelAttempt, ...]:
    return tuple(a for a in attempts if a.level == level and not a.timeout)


def n_valid(attempts: Sequence[LevelAttempt], level: int) -> int:
    return len(valid_attempts(attempts, level))


def n_pass(attempts: Sequence[LevelAttempt], level: int) -> int:
    return sum(1 for a in valid_attempts(attempts, level) if a.passed)


def pass_rate(passes: int, attempts: int) -> float:
    return passes / attempts if attempts else 0.0


def meets_two_thirds(passes: int, attempts: int) -> bool:
    return attempts >= PASS_OF and passes >= PASS_NEED


def majority_pass(passes: int, attempts: int) -> bool:
    """Enough signal that this level looks like a pass (incl. 2/2 extras)."""
    if attempts <= 0:
        return False
    if meets_two_thirds(passes, attempts):
        return True
    return attempts >= EXTRA_ATTEMPTS and pass_rate(passes, attempts) >= (PASS_NEED / PASS_OF)


def majority_fail(passes: int, attempts: int) -> bool:
    if attempts <= 0:
        return False
    return pass_rate(passes, attempts) < (PASS_NEED / PASS_OF)


def climb_failed_levels(attempts: Sequence[LevelAttempt]) -> tuple[int, ...]:
    failed: list[int] = []
    for level in CLIMB_ORDER:
        if n_valid(attempts, level) >= 1 and n_pass(attempts, level) == 0:
            failed.append(level)
    return tuple(failed)


def climb_candidate(attempts: Sequence[LevelAttempt]) -> int | None:
    """First climb level with a valid pass, or None if climb is unfinished/all-fail."""
    for level in CLIMB_ORDER:
        n = n_valid(attempts, level)
        if n == 0:
            return None
        if n_pass(attempts, level) > 0:
            return level
    return None


def _next_climb_level(attempts: Sequence[LevelAttempt]) -> int | None:
    for level in CLIMB_ORDER:
        if n_valid(attempts, level) == 0:
            return level
        if n_pass(attempts, level) > 0:
            return None
    return None


def _need(have: int, want: int) -> int:
    return max(0, want - have)


def _extend_level(
    candidate: int,
    below: int | None,
    direction: str,
    failed: frozenset[int],
    requested: frozenset[int],
) -> int | None:
    if direction == "up":
        nxt = candidate + 1
        while nxt <= 6:
            if nxt in failed:
                nxt += 1
                continue
            return nxt
        return None
    # down
    start = (below if below is not None else candidate) - 1
    nxt = start
    while nxt >= 0:
        if nxt in failed:
            nxt -= 1
            continue
        if nxt == 1 and nxt not in requested:
            nxt -= 1
            continue
        return nxt
    return None


def _optional_reason(level: int, state: TaskState, candidate: int | None) -> str | None:
    attempts = state.attempts
    if level == 1:
        if n_pass(attempts, 2) > 0 and n_valid(attempts, 0) >= 1 and n_pass(attempts, 0) == 0:
            return OPTIONAL_REASONS[1]
        return None
    if level == 3:
        if n_valid(attempts, 2) >= 1 and n_pass(attempts, 2) == 0 and (
            candidate in {5, 6} or n_pass(attempts, 5) > 0 or n_pass(attempts, 6) > 0
        ):
            return OPTIONAL_REASONS[3]
        return None
    if level == 4:
        if candidate in {5, 6} or n_pass(attempts, 5) > 0 or n_pass(attempts, 6) > 0:
            return OPTIONAL_REASONS[4]
        return None
    return None


def next_actions(task_state: TaskState) -> list[LadderAction]:
    """Required (and optional, with reasons) (level, n_attempts) pairs."""
    attempts = task_state.attempts
    requested = frozenset(task_state.request_optional)
    if n_pass(attempts, 2) > 0 and n_valid(attempts, 0) >= 1 and n_pass(attempts, 0) == 0:
        requested = requested | {1}  # L2 passed, L0 failed: probe L1 automatically
    failed = frozenset(climb_failed_levels(attempts))
    required: list[LadderAction] = []
    seen: set[int] = set()

    def add(level: int, n: int, *, optional: bool = False, reason: str = "") -> None:
        if n <= 0 or level in seen:
            return
        seen.add(level)
        required.append(LadderAction(level, n, optional=optional, reason=reason))

    nxt = _next_climb_level(attempts)
    candidate = climb_candidate(attempts)
    if nxt is not None and candidate is None:
        add(nxt, 1, reason=f"climb {list(CLIMB_ORDER)}: one attempt at L{nxt}")
        required.extend(_optional_actions(task_state, candidate, seen))
        return _promote_optional(required, requested, attempts)

    if candidate is None:
        # Every climb level failed. Held out at L6; nothing more unless asked.
        required.extend(_optional_actions(task_state, None, seen))
        return _promote_optional(required, requested, attempts)

    below = level_below(candidate)
    if below is not None and below not in failed:
        # probe below FIRST: a pass there moves the flip down before we spend confirmations
        add(
            below,
            _need(n_valid(attempts, below), below_target_attempts(below)),
            reason=(
                "probe L0 once after L2 pass"
                if below == 0
                else f"confirm-below L{below}: 1 extra (climb-failed levels skipped)"
            ),
        )
    add(
        candidate,
        _need(n_valid(attempts, candidate), PASS_OF),
        reason=f"confirm candidate L{candidate}: 1 extra (total 2)",
    )

    if not any(not a.optional for a in required):
        cand_p, cand_n = n_pass(attempts, candidate), n_valid(attempts, candidate)
        below_p = n_pass(attempts, below) if below is not None else 0
        below_n = n_valid(attempts, below) if below is not None else 0
        confirmed = _is_confirmed(candidate, cand_p, cand_n, below, below_p, below_n, failed)
        if not confirmed:
            direction = _extend_direction(cand_p, cand_n, below, below_p, below_n, failed)
            if direction is not None:
                ext = _extend_level(candidate, below, direction, failed, requested)
                if ext is not None and ext not in failed:
                    add(
                        ext,
                        _need(n_valid(attempts, ext), EXTRA_ATTEMPTS),
                        reason=f"extend {direction} to L{ext} (1 extra)",
                    )

    required.extend(_optional_actions(task_state, candidate, seen))
    return _promote_optional(required, requested, attempts)


def _is_confirmed(
    candidate: int,
    cand_p: int,
    cand_n: int,
    below: int | None,
    below_p: int,
    below_n: int,
    failed: frozenset[int],
) -> bool:
    if not meets_two_thirds(cand_p, cand_n):
        return False
    if below is None:
        return True
    if below in failed:
        return True
    if below == 0:
        # Single L0 probe: fail → L2 confirmed; pass → L0 is the flip, unconfirmed.
        return below_n >= 1 and below_p == 0
    if below_n == 0:
        return False
    return majority_fail(below_p, below_n)


def _extend_direction(
    cand_p: int,
    cand_n: int,
    below: int | None,
    below_p: int,
    below_n: int,
    failed: frozenset[int],
) -> str | None:
    if below is not None and below not in failed and majority_pass(below_p, below_n):
        return "down"
    if cand_n > 0 and not meets_two_thirds(cand_p, cand_n) and majority_fail(cand_p, cand_n):
        return "up"
    if below == 0 and below_n >= 1 and below_p >= 1:
        return "down"
    return None


def _optional_actions(
    state: TaskState,
    candidate: int | None,
    seen: set[int],
) -> list[LadderAction]:
    out: list[LadderAction] = []
    for level in OPTIONAL_LEVELS:
        if level in seen:
            continue
        reason = _optional_reason(level, state, candidate)
        if reason is None:
            continue
        want = PASS_OF
        n = _need(n_valid(state.attempts, level), want)
        if n <= 0:
            continue
        out.append(LadderAction(level, n, optional=True, reason=reason))
    return out


def _promote_optional(
    actions: list[LadderAction],
    requested: frozenset[int],
    attempts: Sequence[LevelAttempt],
) -> list[LadderAction]:
    out: list[LadderAction] = []
    for act in actions:
        if act.optional and act.level in requested:
            n = _need(n_valid(attempts, act.level), PASS_OF)
            if n <= 0:
                continue
            out.append(
                LadderAction(
                    act.level,
                    n,
                    optional=False,
                    reason=act.reason + " (requested)",
                )
            )
        else:
            out.append(act)
    return out


def flip_point(task_state: TaskState) -> FlipResult:
    """``(level|None, confirmed, evidence)`` for the current task state."""
    attempts = task_state.attempts
    failed = climb_failed_levels(attempts)
    candidate = climb_candidate(attempts)
    notes: list[str] = []

    if candidate is None:
        if _next_climb_level(attempts) is not None:
            notes.append("climb unfinished")
        else:
            notes.append("held out: L2, L5, and L6 all failed on the climb")
        evidence = FlipEvidence(
            candidate=None,
            candidate_passes=0,
            candidate_attempts=0,
            below=None,
            below_passes=0,
            below_attempts=0,
            climb_failed=failed,
            notes="; ".join(notes),
        )
        return FlipResult(None, False, evidence)

    below = level_below(candidate)
    cand_p, cand_n = n_pass(attempts, candidate), n_valid(attempts, candidate)
    below_p = n_pass(attempts, below) if below is not None else 0
    below_n = n_valid(attempts, below) if below is not None else 0
    confirmed = _is_confirmed(candidate, cand_p, cand_n, below, below_p, below_n, frozenset(failed))

    level: int | None = candidate
    if below == 0 and below_n >= 1 and below_p >= 1:
        level = 0
        confirmed = False
        notes.append("L0 probe passed; flip is L0 (provisional, 1 attempt)")
    elif confirmed:
        notes.append(f"confirmed L{candidate}: {cand_p}/{cand_n} and below < 2/3")
    else:
        notes.append(f"unconfirmed candidate L{candidate}: {cand_p}/{cand_n}")

    evidence = FlipEvidence(
        candidate=candidate,
        candidate_passes=cand_p,
        candidate_attempts=cand_n,
        below=below,
        below_passes=below_p,
        below_attempts=below_n,
        climb_failed=failed,
        notes="; ".join(notes) if notes else "",
    )
    return FlipResult(level, confirmed, evidence)


def required_actions(task_state: TaskState) -> list[tuple[int, int]]:
    return [a.as_tuple() for a in next_actions(task_state) if not a.optional]


def attempts_from_pairs(
    rows: Iterable[tuple[int, bool] | tuple[int, bool, bool]],
) -> tuple[LevelAttempt, ...]:
    out: list[LevelAttempt] = []
    for row in rows:
        if len(row) == 3:
            level, passed, timeout = row  # type: ignore[misc]
            out.append(LevelAttempt(int(level), bool(passed), bool(timeout)))
        else:
            level, passed = row  # type: ignore[misc]
            out.append(LevelAttempt(int(level), bool(passed), False))
    return tuple(out)


def task_state_from_dict(data: dict[str, Any]) -> TaskState:
    raw = data.get("attempts") or []
    attempts: list[LevelAttempt] = []
    for row in raw:
        if isinstance(row, dict):
            attempts.append(
                LevelAttempt(
                    level=int(row["level"]),
                    passed=bool(row.get("passed")),
                    timeout=bool(row.get("timeout", False)),
                )
            )
        elif isinstance(row, (list, tuple)) and len(row) >= 2:
            timeout = bool(row[2]) if len(row) > 2 else False
            attempts.append(LevelAttempt(int(row[0]), bool(row[1]), timeout))
    requested = frozenset(int(x) for x in (data.get("request_optional") or ()))
    return TaskState(attempts=tuple(attempts), request_optional=requested)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Adaptive ladder next_actions / flip_point")
    parser.add_argument("--state", type=Path, required=True, help="TaskState JSON")
    args = parser.parse_args(argv)
    state = task_state_from_dict(json.loads(args.state.read_text(encoding="utf-8")))
    actions = next_actions(state)
    level, confirmed, evidence = flip_point(state)
    json.dump(
        {
            "next_actions": [
                {
                    "level": a.level,
                    "n_attempts": a.n_attempts,
                    "optional": a.optional,
                    "reason": a.reason,
                }
                for a in actions
            ],
            "flip_point": {
                "level": level,
                "confirmed": confirmed,
                "evidence": {
                    "candidate": evidence.candidate,
                    "candidate_passes": evidence.candidate_passes,
                    "candidate_attempts": evidence.candidate_attempts,
                    "below": evidence.below,
                    "below_passes": evidence.below_passes,
                    "below_attempts": evidence.below_attempts,
                    "climb_failed": list(evidence.climb_failed),
                    "notes": evidence.notes,
                },
            },
        },
        sys.stdout,
        indent=2,
    )
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
