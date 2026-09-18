import numpy as np
import pandas as pd
import pytest
from scipy.special import expit
from scipy.stats import spearmanr

from openswe_traces.irt import (
    EASIEST_ORDER,
    UNKNOWN,
    add_irt_bucket,
    assign_quantile_buckets,
    bucket_cuts,
    build_irt_data,
    fit_irt,
    heldout_comparison,
    irt_bucket_frame,
)

SEED = 0
N_TASKS = 150
N_COMBOS = 7
PER_COMBO = 20


def synthetic_two_pl(
    seed: int = SEED,
    n_tasks: int = N_TASKS,
    n_combos: int = N_COMBOS,
    per_combo: int = PER_COMBO,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    """Rollouts from a known 2PL: P = sigmoid(a_task * (theta_combo - b_task))."""
    rng = np.random.default_rng(seed)
    theta = rng.normal(size=n_combos)
    b = rng.normal(size=n_tasks)
    a = np.exp(rng.normal(scale=0.5, size=n_tasks))
    task_idx = np.repeat(np.arange(n_tasks), n_combos * per_combo)
    combo_idx = np.tile(np.repeat(np.arange(n_combos), per_combo), n_tasks)
    p = expit(a[task_idx] * (theta[combo_idx] - b[task_idx]))
    y = (rng.random(len(p)) < p).astype(np.float64)
    return task_idx, combo_idx, y, b, theta, a


def aligned_mae(estimate: np.ndarray, truth: np.ndarray) -> float:
    """MAE after removing the joint (theta, b) location shift the likelihood cannot fix."""
    return float(np.mean(np.abs(estimate + (truth.mean() - estimate.mean()) - truth)))


@pytest.fixture(scope="module")
def synthetic() -> tuple[np.ndarray, ...]:
    return synthetic_two_pl()


@pytest.fixture(scope="module")
def fits(synthetic: tuple[np.ndarray, ...]) -> dict[str, object]:
    task_idx, combo_idx, y, _, _, _ = synthetic
    n_tasks = int(task_idx.max()) + 1
    n_combos = int(combo_idx.max()) + 1
    return {
        "2pl": fit_irt(task_idx, combo_idx, y, n_tasks, n_combos, two_pl=True),
        "1pl": fit_irt(task_idx, combo_idx, y, n_tasks, n_combos, two_pl=False),
    }


def test_synthetic_2pl_recovers_difficulty_and_ability(
    synthetic: tuple[np.ndarray, ...], fits: dict[str, object]
) -> None:
    _, _, _, b_true, theta_true, _ = synthetic
    fit = fits["2pl"]
    assert fit.two_pl
    assert np.all(fit.a > 0)
    rho = float(spearmanr(fit.b, b_true).statistic)
    assert rho > 0.9
    assert aligned_mae(fit.b, b_true) < 0.3
    assert float(spearmanr(fit.theta, theta_true).statistic) > 0.9
    # success probability is strictly increasing in combo ability
    order = np.argsort(fit.theta)
    p = fit.predict_proba(np.zeros(len(order), dtype=int), order)
    assert np.all(np.diff(p) > 0)


def test_1pl_recovers_difficulty_order(
    synthetic: tuple[np.ndarray, ...], fits: dict[str, object]
) -> None:
    _, _, _, b_true, _, _ = synthetic
    fit = fits["1pl"]
    assert not fit.two_pl
    assert np.allclose(fit.a, 1.0)
    assert float(spearmanr(fit.b, b_true).statistic) > 0.85
    assert aligned_mae(fit.b, b_true) < 0.45


def test_2pl_objective_beats_1pl(
    synthetic: tuple[np.ndarray, ...], fits: dict[str, object]
) -> None:
    assert fits["2pl"].objective <= fits["1pl"].objective + 1e-6 * len(synthetic[2])
    # the priors anchor the ability scale; a is shrunk toward 1, difficulty near 0 on average
    assert abs(float(fits["1pl"].b.mean())) < 1.0
    assert abs(float(fits["2pl"].theta.mean())) < 3.0


def test_build_irt_data_filters_and_orders() -> None:
    rows = [
        ("t-b", 1, 0),
        ("t-b", 0, 0),
        ("t-b", 1, 0),
        ("t-a", 1, 0),
        ("t-a", 0, 0),
        ("t-a", 1, 0),
        ("t-c", 1, 0),  # only 2 labeled rollouts -> dropped
        ("t-a", -1, 0),  # unlabeled -> dropped by the caller upstream
    ]
    frame = pd.DataFrame(rows, columns=["instance_id", "resolved", "unused"]).iloc[:, :2]
    frame["trajectory_id"] = [f"r{i}" for i in range(len(frame))]
    frame["combo"] = "h/t"
    frame = frame[frame["resolved"].isin([0, 1])]
    data = build_irt_data(frame)
    assert data.tasks == ["t-a", "t-b"]
    assert data.combos == ["h/t"]
    assert data.n_rollouts == 6

    shuffled = build_irt_data(frame.sample(frac=1.0, random_state=3).reset_index(drop=True))
    np.testing.assert_array_equal(data.task_idx, shuffled.task_idx)
    np.testing.assert_array_equal(data.y, shuffled.y)


def test_quantile_buckets_match_shares() -> None:
    rng = np.random.default_rng(1)
    b = rng.normal(size=5000)
    shares = {"all_pass": 0.3, "easy": 0.2, "mid": 0.1, "hard": 0.2, "all_fail": 0.2}
    cuts = bucket_cuts(b, shares)
    labels = assign_quantile_buckets(b, cuts)
    counts = pd.Series(labels).value_counts()
    for name, share in shares.items():
        assert abs(counts.get(name, 0) - share * len(b)) <= 2
    assert labels[np.argmin(b)] == EASIEST_ORDER[0]
    assert labels[np.argmax(b)] == EASIEST_ORDER[-1]
    assert np.all(np.diff(cuts) > 0)


def test_irt_bucket_frame_and_parquet_write(tmp_path) -> None:
    rng = np.random.default_rng(2)
    task_irt = pd.DataFrame(
        {
            "instance_id": [f"i{i}" for i in range(100)],
            "b_2pl": rng.normal(size=100),
        }
    )
    shares = {"all_pass": 0.2, "easy": 0.3, "mid": 0.2, "hard": 0.2, "all_fail": 0.1}
    buckets = irt_bucket_frame(task_irt, bucket_cuts(task_irt["b_2pl"].to_numpy(), shares))
    assert set(buckets["irt_bucket"]) <= set(EASIEST_ORDER)

    difficulty = pd.DataFrame(
        {
            "instance_id": ["i0", "i1", "other"],
            "solve_rate": [0.0, 1.0, 0.5],
            "difficulty_bucket": ["all_fail", "all_pass", "mid"],
        }
    )
    path = tmp_path / "task_difficulty.parquet"
    difficulty.to_parquet(path)
    out_path, n_unknown = add_irt_bucket(path, buckets)
    assert out_path == path
    assert n_unknown == 1
    back = pd.read_parquet(path)
    assert list(back.columns) == [*difficulty.columns, "irt_bucket"]
    assert back.loc[back["instance_id"] == "other", "irt_bucket"].iloc[0] == UNKNOWN
    assert (
        back.loc[back["instance_id"] == "i0", "irt_bucket"].iloc[0]
        == buckets.set_index("instance_id").loc["i0", "irt_bucket"]
    )


def test_heldout_comparison_runs() -> None:
    task_idx, combo_idx, y, b_true, _, _ = synthetic_two_pl(seed=5, n_tasks=60, per_combo=10)
    n_tasks = int(task_idx.max()) + 1
    n_combos = int(combo_idx.max()) + 1
    frame = pd.DataFrame(
        {
            "trajectory_id": [f"r{i}" for i in range(len(y))],
            "instance_id": [f"t{t}" for t in task_idx],
            "combo": [f"h/c{c}" for c in combo_idx],
            "resolved": y.astype(int),
        }
    )
    data = build_irt_data(frame)
    buckets = pd.Series(
        {name: ("all_pass" if i % 2 == 0 else "mid") for i, name in enumerate(data.tasks)}
    )
    held = heldout_comparison(data, buckets)
    assert 0 < held.n_test < data.n_rollouts
    assert held.n_degenerate + held.n_mixed == held.n_test
    for value in (
        held.mean_ll_1pl,
        held.mean_ll_2pl,
        held.mean_ll_rate,
        held.mean_ll_1pl_degenerate,
        held.mean_ll_2pl_mixed,
        held.mean_ll_rate_mixed,
    ):
        assert np.isfinite(value)
        assert -5.0 < value < 0.0
    assert b_true.size == n_tasks
    assert n_combos == 7
