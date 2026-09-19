from __future__ import annotations

from pathlib import Path

from openswe_traces.pipeline_ext.author_meta import (
    AuthorUnit,
    author_calibration,
    parse_author_dir,
    parse_difficulty_md,
    parse_predicted_flip,
    predicted_flip_line,
)
from openswe_traces.pipeline_ext.calibration import AttemptRecord
from openswe_traces.pipeline_ext.controls import (
    UnitMeta,
    control_status,
    is_control,
    pick_control,
)
from openswe_traces.pipeline_ext.timeouts import (
    TIMEOUT_CLASS,
    classify_attempt,
    counts_as_failure,
    timeout_budget,
)

FIXTURES = Path(__file__).parent / "fixtures" / "pipeline_ext"


def test_predicted_flip_format() -> None:
    assert predicted_flip_line(2) == "predicted_flip: L2"
    text = (FIXTURES / "difficulty.md").read_text()
    hit = parse_predicted_flip(text)
    assert hit is not None and hit.level == 2
    from_file = parse_difficulty_md(FIXTURES / "difficulty.md")
    assert from_file is not None and from_file.level == 2
    assert parse_author_dir(FIXTURES) is not None
    assert parse_predicted_flip("nope") is None


def test_author_calibration() -> None:
    units = [
        AuthorUnit("devin", "a", "client-go", predicted=2, measured=2, confirmed=True),
        AuthorUnit("devin", "b", "client-go", predicted=2, measured=5, confirmed=True),
        AuthorUnit("grok", "c", "client-go", predicted=0, measured=0, confirmed=False),
    ]
    rows = author_calibration(units)
    by = {r.backend: r for r in rows}
    assert by["devin"].n == 2
    assert by["devin"].exact == 1
    assert by["devin"].mean_abs_error == 1.5
    assert by["grok"].exact == 1


def test_control_is_easiest_predicted_l2() -> None:
    units = [
        UnitMeta("client-go", "hard", predicted_flip=5, n_lines=10),
        UnitMeta("client-go", "easy", predicted_flip=2, n_lines=80),
        UnitMeta("client-go", "easier", predicted_flip=2, n_lines=20),
        UnitMeta("other", "x", predicted_flip=2, n_lines=1),
    ]
    ctrl = pick_control(units, repo="client-go")
    assert ctrl is not None and ctrl.name == "easier"
    assert is_control("easier", units, repo="client-go")
    assert not is_control("easy", units, repo="client-go")
    attempts = [
        AttemptRecord(
            repo="client-go",
            unit="easier",
            solver="composer",
            level=2,
            attempt=i,
            passed=True,
        )
        for i in range(3)
    ]
    st = control_status("client-go", units, attempts)
    assert st.ok is True
    assert st.unit == "easier"


def test_timeout_budgets() -> None:
    d = timeout_budget("devin")
    assert d.agent_timeout_sec == 14400
    assert d.session_timeout_sec == 90 * 60
    assert d.poll_interval_sec == 120
    c = timeout_budget("cursor/composer-2.5")
    assert c.agent_timeout_sec == 3600
    assert c.poll_interval_sec == 60
    g = timeout_budget("cursor-grok-4.6-high")
    assert g.session_timeout_sec == 60 * 60
    assert timeout_budget("mystery").backend == "composer"
    assert classify_attempt(timed_out=True, passed=False) == TIMEOUT_CLASS
    assert counts_as_failure(timed_out=True, passed=False) is False
    assert counts_as_failure(timed_out=False, passed=False) is True
    assert classify_attempt(timed_out=False, passed=False) == "a"
