"""Lazy L0–L6 selection (C2/C6). Package L0+L2 first; climb only on failure."""

from __future__ import annotations

from collections.abc import Mapping

# Harbor dir L{k} ↔ affordance.py level (A = L - 2).
LADDER_TO_AFFORDANCE: dict[int, int] = {0: -2, 1: -1, 2: 0, 3: 1, 4: 2, 5: 3, 6: 4}
AFFORDANCE_TO_LADDER: dict[int, int] = {v: k for k, v in LADDER_TO_AFFORDANCE.items()}
INITIAL_PACKAGE_LEVELS: tuple[int, ...] = (0, 2)
CLIMB_LEVELS: tuple[int, ...] = (3, 4, 5, 6)
PASS_RATE_NUM = 2
PASS_RATE_DEN = 3


def affordance_level(ladder: int) -> int:
    if ladder not in LADDER_TO_AFFORDANCE:
        raise ValueError(f"ladder level must be 0..6, got {ladder}")
    return LADDER_TO_AFFORDANCE[ladder]


def ladder_level(affordance: int) -> int:
    if affordance not in AFFORDANCE_TO_LADDER:
        raise ValueError(f"unknown affordance level {affordance}")
    return AFFORDANCE_TO_LADDER[affordance]


def level_passed(passes: int, attempts: int, *, need: int = PASS_RATE_NUM, of: int = PASS_RATE_DEN) -> bool:
    return attempts >= of and passes >= need


def flip_point(results: Mapping[int, tuple[int, int]]) -> int | None:
    """Lowest level with >= 2/3 passes. None if no level has a stable pass rate."""
    for level in range(7):
        if level not in results:
            continue
        p, n = results[level]
        if level_passed(p, n):
            return level
    return None


def next_solve_levels(
    results: Mapping[int, tuple[int, int]],
    *,
    start: int = 2,
    attempts: int = PASS_RATE_DEN,
) -> list[int]:
    """Levels to run next. Empty means the unit is done for this solver.

    L2 first. If L2 >= 2/3: record flip<=L2 and run L0 once (1 attempt) for the
    L0/L2 distinction. If L2 fails: climb L3..L6 until >= 2/3 pass.
    """
    if start not in results or results[start][1] < attempts:
        return [start]
    p0, n0 = results[start]
    if level_passed(p0, n0, of=attempts):
        if 0 not in results or results[0][1] < 1:
            return [0]
        return []
    for level in CLIMB_LEVELS:
        if level not in results or results[level][1] < attempts:
            return [level]
        p, n = results[level]
        if level_passed(p, n, of=attempts):
            return []
    return []


def attempts_for_level(level: int, *, default: int = PASS_RATE_DEN) -> int:
    """L0 run after an L2 pass is a single distinction attempt."""
    return 1 if level == 0 else default


def levels_to_package(needed: int, already: set[int]) -> list[int]:
    """Generate L1 / L3..L6 only when a lower level failed and the solver asks."""
    if needed in already:
        return []
    if needed in INITIAL_PACKAGE_LEVELS:
        return [needed] if needed not in already else []
    return [needed]
