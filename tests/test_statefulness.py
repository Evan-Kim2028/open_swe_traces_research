from __future__ import annotations

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.statefulness import (
    flip_to_number,
    score_test_body,
    score_test_file,
    score_unit,
    spearman,
)

FIXTURE_TEST = ROOT / "experiments/codegraph_bugs/fixture_host/mathx/mathx_test.go"
MATHX_API = ("Add", "SumClamped", "Clamp", "Next", "Put", "Get")


def test_mathx_fixture_components() -> None:
    text = FIXTURE_TEST.read_text(encoding="utf-8")
    scores = {s.name: s for s in score_test_file(text, MATHX_API)}
    assert scores["TestAdd"].api_calls_before_assert == 1
    assert scores["TestAdd"].sequence_dependent is False
    assert scores["TestSumClamped"].api_calls_before_assert == 1
    assert scores["TestClamp"].api_calls_before_assert == 1
    assert scores["TestNextOnce"].api_calls_before_assert == 1
    assert scores["TestNextOnce"].sequence_dependent is False
    seq = scores["TestNextSequence"]
    assert seq.api_calls_before_assert == 8
    assert seq.sequence_dependent is True
    assert seq.distinct_entry_points == 1
    bag = scores["TestBag"]
    assert bag.api_calls_before_assert == 2
    assert bag.sequence_dependent is True
    assert bag.distinct_entry_points == 2
    assert {s.name for s in scores.values()} == {
        "TestAdd",
        "TestSumClamped",
        "TestClamp",
        "TestNextOnce",
        "TestNextSequence",
        "TestBag",
    }


def test_dynamic_flag_on_goroutine_body() -> None:
    body = """
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = Add(1, 2)
	}()
	wg.Wait()
	if Add(2, 2) != 4 {
		t.Fatal("no")
	}
"""
    s = score_test_body("TestGo", body, ("Add",))
    assert s.uses_dynamic is True
    assert s.api_calls_before_assert >= 2
    assert s.sequence_dependent is True


def test_property_loop_counts_one_iteration() -> None:
    body = """
	for i := 0; i < 10000; i++ {
		got := expo(2, 8, 1)
		if got != 4 {
			t.Fatalf("case %d", i)
		}
	}
"""
    s = score_test_body("TestExpoProp", body, ("expo",))
    assert s.api_calls_before_assert == 1
    assert s.sequence_dependent is False


def test_score_unit_from_fixture_dir() -> None:
    unit = score_unit(
        "mathx",
        family="fixture",
        flip="A0",
        test_files=[FIXTURE_TEST],
        api=MATHX_API,
    )
    assert unit.n_tests == 6
    assert unit.mean_calls > 1.0
    assert 0.0 < unit.sequence_fraction < 1.0
    assert unit.any_dynamic is False


def test_spearman_known() -> None:
    # Perfect increasing ranks.
    assert spearman([1.0, 2.0, 3.0, 4.0], [10.0, 20.0, 30.0, 40.0]) == pytest.approx(1.0)
    assert spearman([4.0, 3.0, 2.0, 1.0], [10.0, 20.0, 30.0, 40.0]) == pytest.approx(-1.0)
    assert flip_to_number("A3") == 3.0
    assert flip_to_number("A-1") == -1.0
    assert flip_to_number("excluded") is None
