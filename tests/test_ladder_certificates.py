"""Certificate and escalation rules, on a synthetic jobs tree.

A certificate is the lowest rung at which ONE solver flips from fail to pass, and
`rung_established` says whether the rung directly below it has a recorded failure.
"""
import json

import pytest

from openswe_traces.ladder import escalate, ledger

MODELS = {"devin": "swe-2-max", "composer": "composer-2.5"}


def write_trial(jobs, job, unit, n, solver, reward, errored=False):
    d = jobs / job / f"{unit}__t{n}"
    d.mkdir(parents=True)
    (d / "result.json").write_text(json.dumps({
        "verifier_result": None if errored else {"rewards": {"reward": reward}},
        "exception_info": {"type": "boom"} if errored else None,
        "agent_info": {"name": solver, "model_info": {"name": MODELS[solver]}},
        "agent_result": {"n_input_tokens": 10, "n_output_tokens": 5},
    }))


@pytest.fixture
def jobs(tmp_path):
    j = tmp_path / "jobs"
    n = iter(range(1000))
    # advrefs as it stood with L4 open: devin fails L0, L2, L3; passes L5 and L6.
    for rung, r in (("0", 0), ("2", 0), ("3", 0), ("5", 1), ("6", 1)):
        write_trial(j, "sweep_devin", f"advrefs-L{rung}", next(n), "devin", r)
    # composer flips advrefs at the contract.
    for rung, r in (("0", 0), ("2", 1)):
        write_trial(j, "sweep_cursor", f"advrefs-L{rung}", next(n), "composer", r)
    # A unit whose only L2 verdict is an error: errors are not failures.
    write_trial(j, "sweep_cursor", "flaky-L0", next(n), "composer", 0)
    write_trial(j, "sweep_cursor", "flaky-L2", next(n), "composer", 0, errored=True)
    return str(j)


def test_certificates_are_per_solver(jobs):
    by = ledger.certificates_by_solver(jobs)["advrefs"]
    assert by["composer"]["rung"] == 2 and by["composer"]["rung_established"]
    assert by["devin"]["rung"] == 5
    # L4 is unmeasured, so devin "flips by L5" but is not shown to NEED L5.
    assert by["devin"]["rung_established"] is False


def test_pooled_certificate_binds_at_the_lowest_solver(jobs):
    c = ledger.certificates(jobs)["advrefs"]
    assert c["rung"] == 2 and c["binds_for"] == ["composer"]
    assert c["solvers"] == ["composer", "devin"]


def test_l4_failure_establishes_l5(jobs, tmp_path):
    write_trial(tmp_path / "jobs", "sweep_devin_holes", "advrefs-L4", 999, "devin", 0)
    by = ledger.certificates_by_solver(jobs)["advrefs"]["devin"]
    assert by["rung"] == 5 and by["rung_established"] is True


def test_errored_trial_is_not_a_failure(jobs):
    assert "2" not in ledger.ledger(jobs)["flaky"]
    assert "flaky" not in ledger.certificates_by_solver(jobs)


@pytest.mark.parametrize("model,solver", [
    ("swe-2-max", "devin"), ("composer-2.5", "composer"), ("cursor-x", "composer"),
    ("grok-4.7", "grok"), (None, "other")])
def test_solver_of(model, solver):
    assert ledger.solver_of(model) == solver


@pytest.mark.parametrize("hist,rung", [
    ({0: True}, None),                          # passed L0: not hard
    ({0: False}, None),                         # no L2 failure yet
    ({0: False, 2: False}, 3),                  # climb one rung at a time
    ({0: False, 2: False, 3: False, 5: True}, 4),   # narrow the gap below a pass
    ({0: False, 2: False, 3: True}, None),      # settled: adjacent fail/pass
    ({0: False, 2: False, 6: False}, None),     # exhausted at the top rung
])
def test_next_rung(hist, rung):
    assert escalate.next_rung(hist)[0] == rung
