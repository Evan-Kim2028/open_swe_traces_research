"""Synthetic-table tests for the framework tests (T1 interaction, T2 discrimination, T3 mechanism)."""

import numpy as np
import pandas as pd
import pytest

from openswe_traces.framework_tests import (
    _build_t1_base,
    _build_t1_design,
    _zscore,
    fit_task_irt,
    load_analysis_frame,
    t1_interaction,
    t1_table,
    t2_discrimination,
    t3_mechanism,
    top_combos,
)

N_INST = 240
N_REPOS = 24


def synthetic_frames(seed: int = 0) -> dict[str, pd.DataFrame]:
    """Trajectory frame + rung frame + closure frame with a planted T1/T2/T3 signal."""
    rng = np.random.default_rng(seed)
    inst_ids = [f"inst_{i}" for i in range(N_INST)]
    repos = [f"repo_{i % N_REPOS}" for i in range(N_INST)]
    languages = ["python" if i % 3 else "go" for i in range(N_INST)]

    # ratio: closure tree term; added_lines: size (independent of ratio)
    ratio = rng.lognormal(0.0, 0.8, N_INST)
    added_lines = rng.lognormal(2.8, 0.5, N_INST)
    n_files = rng.poisson(3, N_INST).astype(float)
    new_frac = rng.random(N_INST)
    rung = rng.integers(0, 5, N_INST)
    rung_binary = (rung >= 2).astype(int)

    # T2: a_2pl correlates with ratio, not with added_lines
    a_2pl = 1.0 + 0.6 * _zscore(pd.Series(ratio)) + 0.05 * _zscore(pd.Series(added_lines))
    a_2pl = np.clip(a_2pl, 0.3, 3.0)
    b_2pl = rng.normal(0.0, 0.8, N_INST)

    # per-instance solve prob: the rung benefit is LARGER on high-ratio AND on large
    # (tree-heavy) tasks -> positive interactions on both proxies
    ratio_s = _zscore(pd.Series(ratio))
    added_s = _zscore(pd.Series(added_lines))
    p = 1 / (1 + np.exp(-(0.3 * rung_binary + 0.6 * ratio_s + 0.8 * rung_binary * ratio_s
                          + 0.3 * added_s + 0.5 * rung_binary * added_s)))
    p = np.clip(p, 0.05, 0.95)

    combos = ["sweagent/qwen38_27b", "sweagent/qwen36_27b", "sweagent/qwen35_122b", "openhands/minimax_m25"]
    per_inst = 12
    rows = []
    for i in range(N_INST):
        for c in combos:
            for k in range(per_inst):
                if rng.random() < 0.12:  # unknown label (resolved == -1), as in the real corpus
                    resolved = -1
                    jaccard = 0.3
                else:
                    resolved = int(rng.random() < p[i])
                    jaccard = 0.5 + 0.4 * ratio_s[i] if resolved == 0 else 0.4
                rows.append(
                    {
                        "trajectory_id": f"t_{i}_{c}_{k}",
                        "instance_id": inst_ids[i],
                        "repo": repos[i],
                        "language": languages[i],
                        "harness": c.split("/")[0],
                        "teacher": c.split("/")[1],
                        "source": "synth",
                        "resolved": resolved,
                        "patch_file_jaccard": float(np.clip(jaccard, 0, 1)),
                        "combo": c,
                    }
                )
    traj = pd.DataFrame(rows)
    rung_df = pd.DataFrame({"instance_id": inst_ids, "rung": rung})
    solve_rate = rng.uniform(0.05, 0.95, N_INST)
    closure = pd.DataFrame(
        {
            "instance_id": inst_ids,
            "ratio": ratio,
            "new_frac": new_frac,
            "added_lines": added_lines,
            "n_files": n_files,
            "n_labeled": [48] * N_INST,
            "n_resolved": np.round(solve_rate * 48).astype(int),
        }
    )
    return {"traj": traj, "rung": rung_df, "closure": closure, "a_2pl": a_2pl, "b_2pl": b_2pl}


def test_load_analysis_frame_builds_rung_binary(tmp_path) -> None:
    frames = synthetic_frames()
    rung_path = tmp_path / "rung.parquet"
    closure_path = tmp_path / "closure.parquet"
    frames["rung"].to_parquet(rung_path, index=False)
    frames["closure"].to_parquet(closure_path, index=False)
    merged = load_analysis_frame(frames["traj"], rung_path, closure_path)
    assert {"rung_binary", "ratio", "added_lines", "n_files", "log1p_added_lines"} <= set(merged.columns)
    assert merged["rung_binary"].isin([0, 1]).all()
    # rung_binary is per-instance: all trajectories of one instance agree
    grouped = merged.groupby("instance_id")["rung_binary"].nunique()
    assert (grouped == 1).all()
    assert merged["rung"].isna().sum() == 0


def test_load_analysis_frame_drops_unlabeled_instances(tmp_path) -> None:
    """A missing rung label must drop the instance, not read it as rung_binary = 0 (L < 2)."""
    frames = synthetic_frames()
    rung_path = tmp_path / "rung_subset.parquet"
    closure_path = tmp_path / "closure.parquet"
    frames["rung"].head(200).to_parquet(rung_path, index=False)  # 40 of 240 instances unlabeled
    frames["closure"].to_parquet(closure_path, index=False)
    merged = load_analysis_frame(frames["traj"], rung_path, closure_path)
    assert merged["instance_id"].nunique() == 200
    assert merged["rung_binary"].notna().all()
    assert merged["rung_binary"].isin([0, 1]).all()


def _merge_with_frames(frames: dict[str, pd.DataFrame]) -> pd.DataFrame:
    traj = frames["traj"].copy()
    traj = traj.merge(frames["rung"], on="instance_id", how="left")
    traj = traj.merge(frames["closure"], on="instance_id", how="left")
    traj["rung_binary"] = (traj["rung"] >= 2).astype(float)
    return traj.reset_index(drop=True)


@pytest.mark.parametrize("variant", ["ratio", "log1p_added_lines"])
def test_t1_interaction_recovers_positive_interaction(variant: str) -> None:
    frames = synthetic_frames()
    merged = _merge_with_frames(frames)
    result = t1_interaction(
        merged, label="synth", variant=variant, n_bootstrap=25, use_repo_fe=True, n_jobs=1
    )
    assert result.n_instances == N_INST
    assert result.n_repos == N_REPOS
    assert 0 < result.auc_test <= 1.0
    assert result.interaction_coef > 0.05  # planted positive interaction
    assert result.ci_lo < result.ci_hi


def test_t1_table_solve_rate_rises_with_proxy_for_rung1() -> None:
    frames = synthetic_frames()
    merged = _merge_with_frames(frames)
    labeled = merged[merged["resolved"].isin([0, 1])]
    for column in ("added_lines", "ratio"):
        table = t1_table(labeled, column)
        assert table["rung_binary"].nunique() == 2
        assert set(table["tercile"]) == {"low", "mid", "high"}
        assert (table["n"] > 0).all()
        # solve rates are over labeled rollouts only: never negative, never above 1
        assert table["mean_solve_rate"].between(0.0, 1.0).all()
        rb1 = table[table["rung_binary"] == 1].set_index("tercile")["mean_solve_rate"]
        assert rb1["high"] > rb1["low"]  # rung>=2 tasks get easier as the proxy rises


def test_t1_prebuilt_base_matches_builtin() -> None:
    """A prebuilt lang+repo base (built on the labeled subset) must give identical results."""
    frames = synthetic_frames()
    merged = _merge_with_frames(frames)
    labeled = merged[merged["resolved"].isin([0, 1])].reset_index(drop=True)
    base = _build_t1_base(labeled, use_repo_fe=True)
    with_base = t1_interaction(
        labeled, label="synth", variant="ratio", n_bootstrap=25, use_repo_fe=True, n_jobs=1, base=base
    )
    without = t1_interaction(
        labeled, label="synth", variant="ratio", n_bootstrap=25, use_repo_fe=True, n_jobs=1
    )
    assert with_base.interaction_coef == pytest.approx(without.interaction_coef, abs=1e-9)
    assert with_base.ci_lo == pytest.approx(without.ci_lo, abs=1e-9)


def test_terciles_tie_safe_on_integer_proxy() -> None:
    """A mass point at the quantile must not collapse the tercile bins (qcut regression)."""
    frames = synthetic_frames()
    merged = _merge_with_frames(frames)
    inst_ids = merged["instance_id"].unique()
    spike = {iid: 3 for iid in inst_ids[:100]}  # 100 of 240 instances share added_lines == 3
    merged["added_lines"] = merged["instance_id"].map(spike).fillna(merged["added_lines"])
    table = t1_table(merged[merged["resolved"].isin([0, 1])], "added_lines")
    assert set(table["tercile"]) == {"low", "mid", "high"}
    assert (table["n"] > 0).all()
    t3 = t3_mechanism(merged, label="synth")  # same qcut path in _tercile_means
    assert set(t3.by_added["tercile"]) == {"low", "mid", "high"}
    assert (t3.by_added["n"] > 0).all()


def test_t2_discrimination_ratio_beats_size_on_a() -> None:
    frames = synthetic_frames()
    task_irt = pd.DataFrame(
        {
            "instance_id": frames["traj"]["instance_id"].unique(),
            "a_2pl": frames["a_2pl"],
            "b_2pl": frames["b_2pl"],
            "n_labeled": [48] * N_INST,
            "solve_rate": rng_solve_rates(frames),
        }
    )
    result = t2_discrimination(task_irt, frames["closure"], label="synth")
    assert result.n_mixed == N_INST
    a_rows = result.table[result.table["target"] == "a_2pl"].set_index("feature")
    assert a_rows.loc["ratio", "spearman"] > 0.5
    assert abs(a_rows.loc["log1p_added_lines", "spearman"]) < 0.2
    assert a_rows.loc["ratio", "ols_std_beta"] > a_rows.loc["log1p_added_lines", "ols_std_beta"]


def rng_solve_rates(frames: dict[str, pd.DataFrame]) -> np.ndarray:
    rng = np.random.default_rng(1)
    return rng.uniform(0.05, 0.95, N_INST)


def test_t3_mechanism_high_ratio_failures_overlap_more() -> None:
    frames = synthetic_frames()
    merged = _merge_with_frames(frames)
    result = t3_mechanism(merged, label="synth")
    assert result.n_unresolved > 0
    hi = result.by_ratio.set_index("tercile")["mean_patch_file_jaccard"]["high"]
    lo = result.by_ratio.set_index("tercile")["mean_patch_file_jaccard"]["low"]
    assert hi > lo
    assert (result.by_ratio["n"] > 0).all()
    assert (result.by_added["n"] > 0).all()


def test_fit_task_irt_returns_tables_and_ranks_combos() -> None:
    frames = synthetic_frames()
    labeled = frames["traj"][frames["traj"]["resolved"].isin([0, 1])]
    task_irt, combo = fit_task_irt(labeled)
    assert {"a_2pl", "b_2pl", "n_labeled", "solve_rate"} <= set(task_irt.columns)
    assert len(task_irt) == N_INST
    assert {"combo", "theta_2pl"} <= set(combo.columns)
    assert len(top_combos(combo)) == 3
    assert top_combos(combo) == combo.sort_values("theta_2pl", ascending=False).head(3)["combo"].tolist()
    # b orders solve_rate (higher b = harder)
    assert float(np.corrcoef(task_irt["b_2pl"], -task_irt["solve_rate"])[0, 1]) > 0.3


def test_build_t1_design_shapes() -> None:
    frames = synthetic_frames()
    merged = _merge_with_frames(frames).head(500)
    X, cols = _build_t1_design(merged, "ratio", use_repo_fe=True)
    assert X.shape[0] == 500
    assert "rung_binary:ratio_std" in cols
    assert "log1p_added_lines" in cols  # size control for the secondary variant
    assert any(c.startswith("repo_") for c in cols)
    assert any(c.startswith("lang_") for c in cols)
    X2, cols2 = _build_t1_design(merged, "ratio", use_repo_fe=False)
    assert not any(c.startswith("repo_") for c in cols2)
    assert X2.shape[1] < X.shape[1]
    # primary variant is itself the size proxy: no duplicate raw log1p(added_lines) column
    _, cols3 = _build_t1_design(merged, "log1p_added_lines", use_repo_fe=False)
    assert "rung_binary:log1p_added_lines_std" in cols3
    assert "log1p_added_lines" not in cols3
