from __future__ import annotations

from openswe_traces.pipeline_ext.calibration import (
    FLAG_ACTIONS,
    AttemptRecord,
    calibrate,
    early_warning_flags,
    inter_attempt_agreement,
    nearest_50,
    pass_rate_curve,
)


def _rec(**kw: object) -> AttemptRecord:
    base: dict[str, object] = {
        "repo": "client-go",
        "unit": "codec",
        "solver": "composer",
        "level": 2,
        "attempt": 0,
        "passed": True,
    }
    base.update(kw)
    return AttemptRecord(**base)  # type: ignore[arg-type]


def test_pass_rate_curve_and_nearest_50() -> None:
    rows = [
        _rec(level=2, attempt=i, passed=True)
        for i in range(3)
    ] + [
        _rec(level=5, attempt=i, passed=(i == 0))
        for i in range(3)
    ]
    curve = pass_rate_curve(rows)
    by_level = {r.level: r for r in curve}
    assert by_level[2].rate == 1.0
    assert abs(by_level[5].rate - 1 / 3) < 1e-9
    assert nearest_50(curve, repo="client-go", solver="composer") == 5
    report = calibrate(rows)
    assert report.nearest_50[("client-go", "composer")] == 5


def test_timeouts_dropped_from_curve() -> None:
    rows = [_rec(passed=False, timeout=True), _rec(passed=True, attempt=1)]
    curve = pass_rate_curve(rows)
    assert curve[0].attempts == 1
    assert curve[0].passes == 1


def test_inter_attempt_agreement() -> None:
    split = [_rec(attempt=i, passed=(i < 2)) for i in range(3)]
    unanimous = [_rec(unit="easy", attempt=i, passed=True) for i in range(3)]
    frac = inter_attempt_agreement(split + unanimous)
    assert abs(frac - 0.5) < 1e-9


def test_gate_rejection_flag() -> None:
    rows = [
        _rec(unit=f"u{i}", rejected_rule="A4" if i < 5 else None)
        for i in range(10)
    ]
    flags = {f.flag: f for f in early_warning_flags(rows) if f.flag == "gate_rejection"}
    # 5/10 = 50% > 40%
    fired = [f for f in flags.values() if f.fired]
    assert fired
    assert "40%" in fired[0].action or "Pause" in fired[0].action


def test_control_miss_needs_three_attempts() -> None:
    pending = [_rec(is_control=True, attempt=i, passed=True) for i in range(2)]
    assert all(not f.fired for f in early_warning_flags(pending) if f.flag == "control_miss")
    miss = [_rec(is_control=True, attempt=i, passed=(i == 0)) for i in range(3)]
    fired = [f for f in early_warning_flags(miss) if f.flag == "control_miss"]
    assert fired and fired[0].fired
    assert fired[0].action == FLAG_ACTIONS["control_miss"]


def test_too_easy_first_15() -> None:
    rows = [
        _rec(unit=f"u{u}", attempt=a, passed=True)
        for u in range(15)
        for a in range(3)
    ]
    fired = [f for f in early_warning_flags(rows) if f.flag == "too_easy"]
    assert fired and fired[0].fired


def test_l6_fail_and_splits_and_contamination() -> None:
    rows = [
        _rec(level=6, passed=False, unit="hard"),
        *[_rec(unit="n", attempt=i, passed=(i < 2)) for i in range(3)],
        _rec(audit_class="test-edit", host="vps", unit="x0"),
        _rec(audit_class="contaminated", host="vps", unit="x1"),
        _rec(audit_class="checksum", host="vps", unit="x2"),
        _rec(audit_class="test-edit", host="vps", unit="x3"),
    ]
    flags = early_warning_flags(rows)
    by = {f.flag: f for f in flags if f.fired}
    assert "l6_fail" in by
    assert "noisy_splits" in by
    assert "contamination" in by


def test_author_yield_and_attempt_time() -> None:
    rows = [
        _rec(author_backend="devin", author_session="s1", unit="only", valid_unit=True),
        _rec(attempt=0, wall_seconds=10.0, unit="slow"),
        _rec(attempt=1, wall_seconds=25.0, unit="slow"),
    ]
    flags = early_warning_flags(rows)
    yield_f = [f for f in flags if f.flag == "author_yield"]
    time_f = [f for f in flags if f.flag == "attempt_time"]
    assert yield_f and yield_f[0].fired
    assert time_f and time_f[0].fired
    assert "class d" in time_f[0].action
