import numpy as np
import pandas as pd
import pytest

from openswe_traces.data import connect_ephemeral
from openswe_traces.difficulty import (
    BUCKET_ORDER,
    build_task_design,
    build_task_text_sql,
    compute_instance_frame,
    difficulty_buckets,
    learnability_table,
    leave_one_out_solve_rate,
    merge_task_text,
    normalize_language,
    within_task_residual,
)

TRAJECTORY_COLUMNS = [
    "trajectory_id",
    "instance_id",
    "harness",
    "teacher",
    "resolved",
    "repo",
    "language",
    "category",
    "gold_patch_files",
    "gold_patch_lines",
    "assistant_chars",
]


def _traj(
    instance_id: str,
    i: int,
    *,
    harness: str = "minisweagent",
    resolved: int = 0,
    language: str = "python",
    repo: str = "org/repo",
    category: str = "bug-fix",
    assistant_chars: int = 1000,
) -> dict:
    return {
        "trajectory_id": f"{instance_id}-{i}",
        "instance_id": instance_id,
        "harness": harness,
        "teacher": f"model-{harness}",
        "resolved": resolved,
        "repo": repo,
        "language": language,
        "category": category,
        "gold_patch_files": 2,
        "gold_patch_lines": 20,
        "assistant_chars": assistant_chars,
    }


def _frame() -> pd.DataFrame:
    rows: list[dict] = []
    for i, harness in enumerate(["minisweagent", "minisweagent", "openhands", "openhands"]):
        rows.append(_traj("t-all-fail", i, resolved=0, harness=harness))
    rows += [_traj("t-all-pass", i, resolved=1, harness="sweagent") for i in range(3)]
    for i, (harness, resolved) in enumerate(
        [("minisweagent", 1), ("minisweagent", 0), ("openhands", 1), ("openhands", 0)]
    ):
        rows.append(
            _traj("t-mid", i, harness=harness, resolved=resolved, language="ts" if i == 0 else "typescript")
        )
    for i, resolved in enumerate([1, 0, 0, 0, 0]):
        rows.append(_traj("t-hard", i, resolved=resolved))
    for i, resolved in enumerate([1, 1, 0]):
        rows.append(
            _traj("t-easy", i, resolved=resolved, harness="openhands", category="feature-request")
        )
    rows += [_traj("t-small", i, resolved=resolved) for i, resolved in enumerate([1, 0])]
    rows += [_traj("t-empty", i, resolved=-1) for i in range(2)]
    df = pd.DataFrame(rows, columns=TRAJECTORY_COLUMNS)
    df["p_resolved"] = 0.5
    return df


def _task_text() -> pd.DataFrame:
    return pd.DataFrame(
        {
            "instance_id": ["t-all-fail", "t-all-pass", "t-mid", "t-easy", "t-small", "t-empty"],
            "repo": ["org/repo", "org/repo", "org/repo-streamed", "org/repo", "org/repo", "org/repo"],
            "issue_chars": [1000, 1000, 4000, 1000, 1000, 1000],
            "n_issue_variants": [1, 1, 1, 1, 1, 1],
            "n_repo_variants": [1, 1, 1, 1, 1, 1],
            "n_rows": [4, 3, 4, 3, 2, 2],
        }
    )


def test_difficulty_bucket_boundaries() -> None:
    rate = pd.Series([0.0, 1.0, 0.3399, 0.34, 0.5, 0.66, 0.6601, 0.2, np.nan])
    labeled = pd.Series([10, 10, 10, 10, 10, 10, 10, 2, 5])
    got = difficulty_buckets(rate, labeled).tolist()
    assert got == ["all_fail", "all_pass", "hard", "mid", "mid", "mid", "easy", "unknown", "unknown"]


def test_normalize_language() -> None:
    out = normalize_language(pd.Series(["ts", "TS", "javascript", "python", "js"]))
    assert out.tolist() == ["typescript", "typescript", "javascript", "python", "javascript"]


def test_compute_instance_frame_counts_and_buckets() -> None:
    inst = compute_instance_frame(_frame(), _task_text()).set_index("instance_id")

    expected_counts = {
        "t-all-fail": (4, 4, 0),
        "t-all-pass": (3, 3, 3),
        "t-mid": (4, 4, 2),
        "t-hard": (5, 5, 1),
        "t-easy": (3, 3, 2),
        "t-small": (2, 2, 1),
        "t-empty": (2, 0, 0),
    }
    for name, (n_roll, n_lab, n_res) in expected_counts.items():
        assert inst.loc[name, "n_rollouts"] == n_roll
        assert inst.loc[name, "n_labeled"] == n_lab
        assert inst.loc[name, "n_resolved"] == n_res

    assert pd.isna(inst.loc["t-empty", "solve_rate"])
    assert inst.loc["t-mid", "solve_rate"] == pytest.approx(0.5)
    assert inst.loc["t-easy", "solve_rate"] == pytest.approx(2 / 3)

    assert inst["difficulty_bucket"].to_dict() == {
        "t-all-fail": "all_fail",
        "t-all-pass": "all_pass",
        "t-mid": "mid",
        "t-hard": "hard",
        "t-easy": "easy",
        "t-small": "unknown",
        "t-empty": "unknown",
    }
    assert set(inst["difficulty_bucket"]) <= set(BUCKET_ORDER)


def test_compute_instance_frame_harness_columns() -> None:
    inst = compute_instance_frame(_frame(), _task_text()).set_index("instance_id")

    for harness in ("minisweagent", "openhands", "sweagent"):
        assert f"solve_rate_{harness}" in inst.columns
        assert f"frac_rollouts_{harness}" in inst.columns

    assert inst.loc["t-all-fail", "solve_rate_minisweagent"] == 0.0
    assert inst.loc["t-all-fail", "solve_rate_openhands"] == 0.0
    assert pd.isna(inst.loc["t-all-fail", "solve_rate_sweagent"])
    assert inst.loc["t-mid", "solve_rate_minisweagent"] == pytest.approx(0.5)
    assert inst.loc["t-mid", "solve_rate_openhands"] == pytest.approx(0.5)
    assert inst.loc["t-all-pass", "solve_rate_sweagent"] == 1.0

    fracs = inst[[f"frac_rollouts_{h}" for h in ("minisweagent", "openhands", "sweagent")]]
    assert fracs.sum(axis=1).to_numpy() == pytest.approx(np.ones(len(inst)))
    assert inst.loc["t-all-fail", "frac_rollouts_minisweagent"] == pytest.approx(0.5)


def test_compute_instance_frame_identity_and_task_text() -> None:
    inst = compute_instance_frame(_frame(), _task_text()).set_index("instance_id")

    assert inst.loc["t-mid", "language"] == "typescript"
    assert inst.loc["t-mid", "repo"] == "org/repo-streamed"
    assert inst.loc["t-mid", "issue_chars"] == 4000
    assert inst.loc["t-hard", "repo"] == "org/repo"
    assert pd.isna(inst.loc["t-hard", "issue_chars"])
    assert inst.loc["t-hard", "category"] == "bug-fix"
    assert inst.loc["t-hard", "gold_patch_files"] == 2
    assert inst.loc["t-hard", "gold_patch_lines"] == 20
    assert inst.index.name == "instance_id"


def test_leave_one_out_and_residual() -> None:
    df = _frame()
    loo = leave_one_out_solve_rate(df)
    assert loo.index.equals(df.index)

    mid_labeled = (df["instance_id"] == "t-mid") & df["resolved"].isin([0, 1])
    own = df.loc[mid_labeled, "resolved"].to_numpy(dtype=float)
    assert loo[mid_labeled].to_numpy() == pytest.approx((2 - own) / 3)

    small = df["instance_id"] == "t-small"
    assert loo[small & (df["resolved"] == 1)].iloc[0] == pytest.approx(0.0)
    assert loo[small & (df["resolved"] == 0)].iloc[0] == pytest.approx(1.0)
    assert loo[df["instance_id"] == "t-empty"].isna().all()

    resid = within_task_residual(df)
    assert resid[small & (df["resolved"] == 1)].iloc[0] == pytest.approx(1.0)
    assert resid[small & (df["resolved"] == 0)].iloc[0] == pytest.approx(-1.0)
    assert resid[df["instance_id"] == "t-empty"].isna().all()


def test_leave_one_out_single_labeled_instance_is_undefined() -> None:
    df = pd.DataFrame([_traj("solo", 0, resolved=1), _traj("solo", 1, resolved=-1)])
    assert leave_one_out_solve_rate(df).isna().all()


def test_task_text_sql(tmp_path) -> None:
    messages_a = [{"role": "system", "content": "sys"}, {"role": "user", "content": "hello"}]
    messages_b = [
        {"role": "system", "content": "sys"},
        {"role": "user", "content": "x" * 40},
        {"role": "assistant", "content": "reply"},
    ]
    messages_short = [{"role": "system", "content": "sys"}]
    src = pd.DataFrame(
        {
            "instance_id": ["i1", "i2", "i1", "i3"],
            "repo": ["r1", "r2", "r1", "r3"],
            "messages": [messages_a, messages_b, messages_a, messages_short],
        }
    )
    path = tmp_path / "shard.parquet"
    src.to_parquet(path)

    con = connect_ephemeral()
    try:
        out = con.execute(build_task_text_sql(path)).fetchdf().set_index("instance_id")
    finally:
        con.close()

    assert out.loc["i1", "issue_chars"] == len("hello")
    assert out.loc["i1", "n_rows"] == 2
    assert out.loc["i1", "n_issue_variants"] == 1
    assert out.loc["i1", "repo"] == "r1"
    assert out.loc["i2", "issue_chars"] == 40
    assert pd.isna(out.loc["i3", "issue_chars"])


def test_merge_task_text(tmp_path) -> None:
    parts = tmp_path / "parts"
    parts.mkdir()
    pd.DataFrame(
        {
            "instance_id": ["i1", "i2"],
            "repo": ["r1", "r2"],
            "issue_chars": [100, 200],
            "n_issue_variants": [1, 1],
            "n_repo_variants": [1, 1],
            "n_rows": [3, 1],
        }
    ).to_parquet(parts / "a.parquet")
    pd.DataFrame(
        {
            "instance_id": ["i1"],
            "repo": ["r1"],
            "issue_chars": [150],
            "n_issue_variants": [2],
            "n_repo_variants": [1],
            "n_rows": [2],
        }
    ).to_parquet(parts / "b.parquet")

    out_path = tmp_path / "task_text.parquet"
    con = connect_ephemeral()
    try:
        result = merge_task_text(con, parts_dir=parts, out_path=out_path)
        merged = (
            con.execute(f"SELECT * FROM read_parquet('{out_path}')").fetchdf().set_index("instance_id")
        )
    finally:
        con.close()

    assert result == (2, 2, 2)
    assert merged.loc["i1", "issue_chars"] == 150
    assert merged.loc["i1", "n_rows"] == 5
    assert merged.loc["i1", "n_issue_variants"] == 2
    assert merged.loc["i2", "issue_chars"] == 200


def test_learnability_table() -> None:
    df = _frame()
    inst = compute_instance_frame(df, _task_text())
    learn = learnability_table(df, inst)

    total = learn.iloc[-1]
    assert total["harness"] == "TOTAL"
    assert total["n_rollouts"] == len(df)
    assert total["n_dropped"] == 7  # t-all-fail (4) + t-all-pass (3)
    assert total["dropped_share"] == pytest.approx(7 / len(df))
    assert total["chars"] == pytest.approx(1000 * len(df))
    assert total["chars_dropped"] == pytest.approx(7000)
    assert total["tokens_dropped"] == pytest.approx(1750)

    by_harness = learn.set_index("harness")
    assert by_harness.loc["minisweagent", "n_dropped"] == 2
    assert by_harness.loc["openhands", "n_dropped"] == 2
    assert by_harness.loc["sweagent", "n_dropped"] == 3
    assert learn.iloc[:-1]["n_dropped"].sum() == total["n_dropped"]


def test_build_task_design() -> None:
    inst = compute_instance_frame(_frame(), _task_text())
    frame = inst[inst["solve_rate"].notna() & inst["issue_chars"].notna()].reset_index(drop=True)
    harnesses = ["minisweagent", "openhands", "sweagent"]
    X, features, baseline = build_task_design(frame, harnesses)

    assert baseline == "minisweagent"
    assert "frac_rollouts_openhands" in features
    assert "frac_rollouts_minisweagent" not in features
    assert "gold_patch_files" in features
    assert "gold_patch_lines" in features
    assert "issue_chars" in features
    assert "language_typescript" in features
    assert "language_python" not in features
    assert "category_feature-request" in features
    assert "category_bug-fix" not in features
    assert X.notna().all().all()
    assert len(X) == len(frame)
