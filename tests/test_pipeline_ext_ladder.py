from __future__ import annotations

from pathlib import Path

from openswe_traces.pipeline_ext.ladder_policy import (
    CLIMB_ORDER,
    OPTIONAL_LEVELS,
    TaskState,
    affordance_level,
    attempts_from_pairs,
    flip_point,
    ladder_level,
    next_actions,
    required_actions,
    task_state_from_dict,
)


def _state(*rows: tuple[int, bool] | tuple[int, bool, bool], optional: frozenset[int] = frozenset()) -> TaskState:
    return TaskState(attempts=attempts_from_pairs(rows), request_optional=optional)


def _req(state: TaskState) -> list[tuple[int, int]]:
    return [(a.level, a.n_attempts) for a in next_actions(state) if not a.optional]


def test_affordance_mapping() -> None:
    assert CLIMB_ORDER == (2, 5, 6)
    assert OPTIONAL_LEVELS == (1, 3, 4)
    assert affordance_level(0) == -2
    assert affordance_level(2) == 0
    assert affordance_level(6) == 4
    assert ladder_level(0) == 2


def test_climb_starts_at_l2() -> None:
    acts = next_actions(_state())
    assert required_actions(_state()) == [(2, 1)]
    assert acts[0].reason.startswith("climb")


def test_l2_fail_climbs_l5() -> None:
    assert _req(_state((2, False))) == [(5, 1)]


def test_l2_l5_fail_climbs_l6() -> None:
    assert _req(_state((2, False), (5, False))) == [(6, 1)]


def test_all_climb_fail_held_out() -> None:
    state = _state((2, False), (5, False), (6, False))
    assert _req(state) == []
    level, confirmed, ev = flip_point(state)
    assert level is None
    assert confirmed is False
    assert ev.climb_failed == (2, 5, 6)
    assert "held out" in ev.notes


def test_l2_pass_then_two_extra_and_l0_probe() -> None:
    state = _state((2, True))
    req = {a.level: a.n_attempts for a in next_actions(state) if not a.optional}
    assert req[2] == 2
    assert req[0] == 1
    assert 1 not in req


def test_timeouts_do_not_count() -> None:
    state = _state((2, False, True))
    assert _req(state) == [(2, 1)]


def test_l2_confirmed_when_l0_fails() -> None:
    state = _state((2, True), (2, True), (2, True), (0, False))
    assert _req(state) == []
    level, confirmed, ev = flip_point(state)
    assert level == 2
    assert confirmed is True
    assert ev.below == 0
    assert ev.below_passes == 0


def test_l0_probe_pass_is_provisional_flip() -> None:
    state = _state((2, True), (2, True), (2, True), (0, True))
    level, confirmed, ev = flip_point(state)
    assert level == 0
    assert confirmed is False
    assert "L0" in ev.notes
    # nothing below L0; L1 only on demand after L0 *fail*
    optional = [a for a in next_actions(state) if a.optional]
    assert all(a.level != 1 for a in optional)


def test_l1_optional_only_if_l0_fails() -> None:
    state = _state((2, True), (2, True), (2, True), (0, False))
    opt = [a for a in next_actions(state) if a.optional]
    assert any(a.level == 1 for a in opt)
    promoted = _state((2, True), (2, True), (2, True), (0, False), optional=frozenset({1}))
    req = _req(promoted)
    assert req == [(1, 3)]


def test_climb_failed_levels_get_no_extra() -> None:
    # L2 failed climb, L5 pass → extras at L5 and L4, not more L2
    state = _state((2, False), (5, True))
    req = {a.level: a.n_attempts for a in next_actions(state) if not a.optional}
    assert req[5] == 2
    assert req[4] == 2
    assert 2 not in req


def test_l5_confirmed() -> None:
    state = _state(
        (2, False),
        (5, True),
        (5, True),
        (5, True),
        (4, False),
        (4, False),
    )
    level, confirmed, ev = flip_point(state)
    assert level == 5
    assert confirmed is True
    assert ev.below == 4
    assert _req(state) == []
    opt = [a for a in next_actions(state) if a.optional]
    assert any(a.level == 3 for a in opt)


def test_extend_down_when_below_passes() -> None:
    state = _state(
        (2, False),
        (5, True),
        (5, True),
        (5, True),
        (4, True),
        (4, True),
    )
    req = _req(state)
    assert req == [(3, 2)]
    level, confirmed, _ = flip_point(state)
    assert level == 5
    assert confirmed is False


def test_extend_up_when_candidate_fails_extras() -> None:
    state = _state((2, True), (2, False), (2, False), (0, False))
    req = dict(_req(state))
    assert req[3] == 2
    _, confirmed, _ = flip_point(state)
    assert confirmed is False


def test_l6_candidate_skips_failed_l5() -> None:
    state = _state((2, False), (5, False), (6, True))
    req = {a.level: a.n_attempts for a in next_actions(state) if not a.optional}
    assert req[6] == 2
    assert 5 not in req


def test_l6_confirmed_when_below_failed_climb() -> None:
    state = _state((2, False), (5, False), (6, True), (6, True), (6, True))
    level, confirmed, ev = flip_point(state)
    assert level == 6
    assert confirmed is True
    assert 5 in ev.climb_failed
    assert _req(state) == []


def test_unpack_actions_and_flip() -> None:
    state = _state()
    level, n = next_actions(state)[0]
    assert (level, n) == (2, 1)
    flip_level, confirmed, evidence = flip_point(state)
    assert flip_level is None
    assert confirmed is False
    assert evidence.notes


def test_task_state_from_dict() -> None:
    state = task_state_from_dict(
        {"attempts": [{"level": 2, "passed": False}, [5, True]], "request_optional": [3]}
    )
    assert len(state.attempts) == 2
    assert 3 in state.request_optional
    req = dict(_req(state))
    assert req[5] == 2
    assert req[4] == 2
    assert req[3] == 3


def test_cli_roundtrip(tmp_path: Path) -> None:
    from openswe_traces.pipeline_ext.ladder_policy import main

    p = tmp_path / "state.json"
    p.write_text('{"attempts": [[2, true]]}\n')
    assert main(["--state", str(p)]) == 0
