import hashlib
import json
import re

import numpy as np
import pandas as pd
import pytest

from openswe_traces.cheap_difficulty import (
    K_VALUES,
    PREFIX_FEATURES,
    WILSON_Z,
    build_prefix_sql,
    interval_bucket,
    pair_matrix,
    pair_probe_stats,
    permuted_probe_arrays,
    point_bucket,
    prefix_sanity,
    simulate_rules,
    simulate_simple,
    simulate_wilson,
    simulation_bucket_table,
    simulation_per_seed,
    simulation_summary,
    wilson_interval,
)
from openswe_traces.data import connect_ephemeral, list_parquet_files, parse_shard
from openswe_traces.features import (
    BASH_EDIT_RE,
    EDIT_TOOL_NAMES,
    EDIT_TOOL_VIEW_ONLY,
    _sql_str,
)

GIT_PATH_RE = re.compile(r"diff --git a/(\S+) b/")
PLUS_PATH_RE = re.compile(r"\+\+\+ b/(\S+)")
BASH_EDIT_PY_RE = re.compile(BASH_EDIT_RE)
EDIT_TOOL_SET = frozenset(EDIT_TOOL_NAMES)


def test_wilson_interval_matches_closed_form() -> None:
    z = WILSON_Z
    for successes, n in ((0, 5), (5, 10), (10, 10), (1, 3), (3, 4)):
        lo, hi = wilson_interval(successes, n)
        p = successes / n
        denom = 1 + z * z / n
        center = (p + z * z / (2 * n)) / denom
        half = z * np.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / denom
        assert lo == pytest.approx(max(0.0, center - half), abs=1e-12)
        assert hi == pytest.approx(min(1.0, center + half), abs=1e-12)


def test_wilson_interval_known_values_and_properties() -> None:
    assert wilson_interval(0, 5) == pytest.approx((0.0, 0.2473), abs=5e-5)
    assert wilson_interval(5, 10) == pytest.approx((0.3122, 0.6878), abs=5e-5)
    assert wilson_interval(0, 0) == (0.0, 1.0)

    for successes, n in ((0, 4), (2, 7), (11, 12)):
        lo, hi = wilson_interval(successes, n)
        assert 0.0 <= lo <= successes / n <= hi <= 1.0

    lo, hi = wilson_interval(3, 6)
    assert lo == pytest.approx(0.5 - (hi - 0.5), abs=1e-12)
    assert wilson_interval(0, 20)[1] < wilson_interval(0, 5)[1]


def test_interval_bucket_boundaries() -> None:
    assert interval_bucket(0.0, 0.0) == "all_fail"
    assert interval_bucket(1.0, 1.0) == "all_pass"
    assert interval_bucket(0.0, 0.282) == "hard"
    assert interval_bucket(0.2, 0.3399) == "hard"
    assert interval_bucket(0.34, 0.66) == "mid"
    assert interval_bucket(0.4, 0.6) == "mid"
    assert interval_bucket(0.6601, 0.9) == "easy"
    assert interval_bucket(0.8, 1.0) == "easy"
    assert interval_bucket(0.3, 0.4) is None
    assert interval_bucket(0.6, 0.7) is None
    assert interval_bucket(0.1, 0.7) is None
    assert interval_bucket(0.0, 0.4) is None
    assert interval_bucket(0.0, 0.34) is None


def test_point_bucket_matches_difficulty_buckets() -> None:
    assert point_bucket(0.0, 5) == "all_fail"
    assert point_bucket(1.0, 5) == "all_pass"
    assert point_bucket(0.2, 5) == "hard"
    assert point_bucket(0.34, 6) == "mid"
    assert point_bucket(0.66, 6) == "mid"
    assert point_bucket(0.80, 6) == "easy"
    assert point_bucket(0.5, 2) == "unknown"


def test_simulate_wilson_stops_on_interval_and_cap() -> None:
    out = simulate_wilson(np.zeros(12, dtype=np.int8))
    assert (out.n_used, out.bucket, out.stop) == (4, "all_fail", "interval")

    out = simulate_wilson(np.ones(12, dtype=np.int8))
    assert (out.n_used, out.bucket, out.stop) == (4, "all_pass", "interval")

    out = simulate_wilson(np.array([1, 1, 0, 0] * 3, dtype=np.int8))
    assert out.n_used == 12
    assert out.bucket == "mid"
    assert out.stop == "cap"

    out = simulate_wilson(np.array([0, 1, 0, 1, 1, 0], dtype=np.int8))
    assert out.n_used == 6
    assert out.stop == "exhausted"
    assert out.bucket == "mid"


def test_simulate_simple_rule() -> None:
    out = simulate_simple(np.array([1, 1, 0, 0], dtype=np.int8))
    assert (out.n_used, out.stop) == (2, "agree2")
    assert out.bucket == "unknown"  # 2 draws cannot satisfy the >= 3 labeled rule

    out = simulate_simple(np.array([0, 0, 1, 1], dtype=np.int8))
    assert (out.n_used, out.stop, out.bucket) == (2, "agree2", "unknown")

    out = simulate_simple(np.array([1, 0, 0, 0, 1], dtype=np.int8))
    assert (out.n_used, out.stop, out.bucket) == (4, "cap", "hard")

    out = simulate_simple(np.array([1, 0, 1, 0], dtype=np.int8))
    assert (out.n_used, out.stop, out.bucket) == (4, "cap", "mid")

    assert simulate_simple(np.array([0, 0, 1, 1], dtype=np.int8), implied_two=True).bucket == "all_fail"
    assert simulate_simple(np.array([1, 1, 0, 0], dtype=np.int8), implied_two=True).bucket == "all_pass"
    implied = simulate_simple(np.array([1, 0, 0, 0], dtype=np.int8), implied_two=True)
    assert (implied.n_used, implied.bucket) == (4, "hard")


def _synthetic_outcomes() -> tuple[dict[str, np.ndarray], dict[str, str]]:
    outcomes = {
        "i-fail": np.array([0] * 12, dtype=np.int8),
        "i-pass": np.array([1] * 12, dtype=np.int8),
        "i-mid": np.array([1, 0, 1, 0, 0, 1, 1, 0, 1, 0, 1, 0], dtype=np.int8),
        "i-hardish": np.array([0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0], dtype=np.int8),
    }
    full_bucket = {"i-fail": "all_fail", "i-pass": "all_pass", "i-mid": "mid", "i-hardish": "hard"}
    return outcomes, full_bucket


def test_simulate_rules_deterministic_and_complete() -> None:
    outcomes, full_bucket = _synthetic_outcomes()
    runs = simulate_rules(outcomes, full_bucket, seeds=3)
    assert len(runs) == len(outcomes) * len(runs["rule"].unique()) * 3
    assert set(runs["rule"]) == {"wilson-80-cap12", "agree2-else-4", "agree2-else-4-implied"}
    assert runs["n_used"].between(2, 12).all()
    assert set(runs["bucket"]) <= {"all_fail", "hard", "mid", "easy", "all_pass", "unknown"}

    again = simulate_rules(outcomes, full_bucket, seeds=3)
    pd.testing.assert_frame_equal(runs, again)

    fail_rows = runs[(runs["instance_id"] == "i-fail") & (runs["rule"] == "wilson-80-cap12")]
    assert (fail_rows["bucket"] == "all_fail").all()
    assert (fail_rows["n_used"] == 4).all()

    pass_rows = runs[(runs["instance_id"] == "i-pass") & (runs["rule"] == "agree2-else-4")]
    assert (pass_rows["n_used"] == 2).all()
    assert (pass_rows["bucket"] == "unknown").all()


def test_simulation_aggregates() -> None:
    outcomes, full_bucket = _synthetic_outcomes()
    runs = simulate_rules(outcomes, full_bucket, seeds=2)
    per_seed = simulation_per_seed(runs)
    assert len(per_seed) == 2 * len(runs["rule"].unique())
    assert per_seed["accuracy"].between(0, 1).all()
    assert per_seed["unknown_share"].between(0, 1).all()
    assert (per_seed["n_instances"] == len(outcomes)).all()

    summary = simulation_summary(per_seed)
    assert set(summary.index) == {"wilson-80-cap12", "agree2-else-4", "agree2-else-4-implied"}
    assert summary.loc["wilson-80-cap12", "mean_rollouts_mean"] >= 4
    assert summary.loc["wilson-80-cap12", "mean_rollouts_mean"] <= 12

    by_bucket = simulation_bucket_table(runs)
    assert set(by_bucket["full_bucket"]) == {"all_fail", "all_pass", "mid", "hard"}
    wilson_fail = by_bucket[
        (by_bucket["rule"] == "wilson-80-cap12") & (by_bucket["full_bucket"] == "all_fail")
    ].iloc[0]
    assert wilson_fail["accuracy"] == pytest.approx(1.0)


def _probe_targets() -> tuple[dict[str, np.ndarray], dict[str, np.ndarray]]:
    probe = {
        "i1": np.array([0], dtype=np.int8),
        "i2": np.array([1], dtype=np.int8),
        "i3": np.array([1], dtype=np.int8),
        "i4": np.array([0], dtype=np.int8),
        "i5": np.array([1], dtype=np.int8),  # target has only 2 labels -> excluded
    }
    target = {
        "i1": np.array([0, 0, 0], dtype=np.int8),
        "i2": np.array([1, 1, 1], dtype=np.int8),
        "i3": np.array([1, 0, 1], dtype=np.int8),
        "i4": np.array([0, 0, 1], dtype=np.int8),
        "i5": np.array([0, 1], dtype=np.int8),
    }
    return probe, target


def test_pair_probe_stats() -> None:
    probe, target = _probe_targets()
    stats = pair_probe_stats(probe, target, ("h", "p"), ("h", "t"))
    assert stats.n_instances == 4
    assert stats.target_pos_share == pytest.approx(0.5)
    assert stats.auc_k1 == pytest.approx(1.0)
    assert stats.rho[1] == pytest.approx(0.8944271909999159)
    assert stats.rho[2] == pytest.approx(0.8944271909999159)

    empty = pair_probe_stats({}, target, ("h", "p"), ("h", "t"))
    assert empty.n_instances == 0
    assert np.isnan(empty.auc_k1)


def test_probe_permutations_are_seeded_and_nested() -> None:
    arrays = {f"s{i}": np.array([1, 0, 1, 1, 0, 0, 1], dtype=np.int8) for i in range(5)}
    first = permuted_probe_arrays(arrays, seed=0)
    second = permuted_probe_arrays(arrays, seed=0)
    for key, arr in arrays.items():
        np.testing.assert_array_equal(first[key], second[key])
        assert sorted(first[key].tolist()) == sorted(arr.tolist())

    other = permuted_probe_arrays(arrays, seed=1)
    assert any(first[key].tolist() != other[key].tolist() for key in arrays)


def test_pair_matrix() -> None:
    probe, target = _probe_targets()
    pairs = [pair_probe_stats(probe, target, ("a", "x"), ("b", "y"))]
    matrix = pair_matrix(pairs, [("a", "x"), ("b", "y")], "auc")
    assert matrix[0][0] == "a/x"
    assert matrix[0][1] == "n/a"
    assert matrix[0][2] == f"{pairs[0].auc_k1:.3f}"


def _tool_call(name: str, arguments: str) -> dict:
    return {"id": name, "type": "function", "function": {"name": name, "arguments": arguments}}


def _bash(command: str) -> dict:
    return _tool_call("bash", json.dumps({"command": command}))


def _is_edit(call: dict) -> bool:
    fn = call.get("function") or {}
    name = (fn.get("name") or "").lower()
    args = fn.get("arguments") or ""
    try:
        command = (json.loads(args) or {}).get("command")
    except (json.JSONDecodeError, TypeError, AttributeError):
        command = None
    command = command.lower() if isinstance(command, str) else ""
    if name == EDIT_TOOL_VIEW_ONLY:
        return command != "view"
    if name in EDIT_TOOL_SET:
        return True
    return bool(BASH_EDIT_PY_RE.search(command))


def _reference_prefix(messages: list[dict], metadata: dict, k: int) -> dict:
    patch = ((metadata or {}).get("reference_patch") or {}).get("patch") or ""
    gold = list(dict.fromkeys(GIT_PATH_RE.findall(patch)))
    if not gold:
        gold = list(dict.fromkeys(PLUS_PATH_RE.findall(patch)))

    calls = []
    error_positions = []
    cum_calls = 0
    cum_assistant = 0
    assistant_after: dict[int, int] = {}
    for mi, message in enumerate(messages, start=1):
        role = message.get("role")
        if role == "assistant":
            for call in message.get("tool_calls") or []:
                args = (call.get("function") or {}).get("arguments") or ""
                cum_calls += 1
                calls.append(
                    {
                        "idx": cum_calls,
                        "mi": mi,
                        "args": args,
                        "is_edit": _is_edit(call),
                        "sees_gold": any(
                            path in args or path.rsplit("/", 1)[-1] in args for path in gold
                        ),
                    }
                )
            cum_assistant += len(message.get("content") or "")
            assistant_after[mi] = cum_assistant
        if role == "tool" and re.search(
            r"error|traceback|failed", message.get("content") or "", re.IGNORECASE
        ):
            error_positions.append(cum_calls)

    prefix = [call for call in calls if call["idx"] <= k]
    n = len(prefix)
    distinct = len({hashlib.md5(call["args"].encode()).hexdigest() for call in prefix})
    return {
        "n_calls": n,
        "gold_view": int(any(call["sees_gold"] for call in prefix)),
        "gold_edit": int(any(call["sees_gold"] and call["is_edit"] for call in prefix)),
        "n_edit": sum(call["is_edit"] for call in prefix),
        "n_distinct": distinct,
        "repeat_rate": 0.0 if n == 0 else 1.0 - distinct / n,
        "assistant_chars": assistant_after[prefix[-1]["mi"]] if prefix else 0,
        "any_error": any(position <= k for position in error_positions),
    }


def _synthetic_shard() -> pd.DataFrame:
    gold_patch = "diff --git a/src/foo.py b/src/foo.py\n@@ -1 +1 @@\n-old\n+new\n"
    rich = [
        {"role": "system", "content": "sys", "tool_calls": None},
        {"role": "user", "content": "fix it", "tool_calls": None},
        {"role": "assistant", "content": "a" * 10, "tool_calls": [_bash("ls")]},
        {"role": "tool", "content": "listing", "tool_calls": None},
        {"role": "assistant", "content": "b" * 20, "tool_calls": [_bash("cat src/foo.py")]},
        {"role": "tool", "content": "An Error occurred", "tool_calls": None},
        {
            "role": "assistant",
            "content": "c" * 30,
            "tool_calls": [_bash("apply_patch src/foo.py")],
        },
        {"role": "tool", "content": "ok", "tool_calls": None},
        {
            "role": "assistant",
            "content": "d" * 40,
            "tool_calls": [_bash("apply_patch src/foo.py")],
        },
        {"role": "tool", "content": "ok", "tool_calls": None},
        {"role": "assistant", "content": "e" * 50, "tool_calls": [_bash("pytest -q")]},
        {"role": "tool", "content": "FAILED test_x", "tool_calls": None},
        {"role": "assistant", "content": "f" * 60, "tool_calls": [_bash("ls")]},
        {"role": "tool", "content": "listing", "tool_calls": None},
        {"role": "assistant", "content": "done", "tool_calls": None},
    ]
    quiet = [
        {"role": "system", "content": "sys", "tool_calls": None},
        {"role": "user", "content": "nothing", "tool_calls": None},
    ]
    return pd.DataFrame(
        {
            "instance_id": ["t-rich", "t-quiet"],
            "repo": ["org/repo", "org/repo"],
            "trajectory_id": ["traj-rich", "traj-quiet"],
            "resolved": [1, -1],
            "messages": [rich, quiet],
            "metadata": [
                {"reference_patch": {"patch": gold_patch}},
                {"reference_patch": {"patch": gold_patch}},
            ],
        }
    )


def test_prefix_sql_against_python_reference(tmp_path) -> None:
    src = _synthetic_shard()
    path = tmp_path / "shard.parquet"
    src.to_parquet(path)

    con = connect_ephemeral()
    try:
        sql = build_prefix_sql(path, "minisweagent", "model", "src")
        out = con.execute(f"SELECT * FROM ({sql}) ORDER BY trajectory_id").fetchdf()
    finally:
        con.close()

    assert set(out["trajectory_id"]) == {"traj-rich", "traj-quiet"}
    assert set(PREFIX_FEATURES) <= set(out.columns)
    assert out.loc[out["trajectory_id"] == "traj-rich", "n_calls_total"].iloc[0] == 6

    rich = src.loc[src["trajectory_id"] == "traj-rich"].iloc[0]
    for k in K_VALUES:
        expected = _reference_prefix(rich["messages"], rich["metadata"], k)
        row = out[out["trajectory_id"] == "traj-rich"].iloc[0]
        assert row[f"n_calls_{k}"] == expected["n_calls"]
        assert row[f"gold_view_{k}"] == expected["gold_view"]
        assert row[f"gold_edit_{k}"] == expected["gold_edit"]
        assert row[f"n_edit_{k}"] == expected["n_edit"]
        assert row[f"n_distinct_{k}"] == expected["n_distinct"]
        assert row[f"repeat_rate_{k}"] == pytest.approx(expected["repeat_rate"], abs=1e-9)
        assert row[f"assistant_chars_{k}"] == expected["assistant_chars"]
        assert bool(row[f"any_error_{k}"]) == expected["any_error"]

    quiet = out[out["trajectory_id"] == "traj-quiet"].iloc[0]
    for k in K_VALUES:
        assert quiet[f"n_calls_{k}"] == 0
        assert quiet[f"gold_view_{k}"] == 0
        assert quiet[f"repeat_rate_{k}"] == 0.0
        assert quiet[f"assistant_chars_{k}"] == 0
        assert not bool(quiet[f"any_error_{k}"])
    assert quiet["n_calls_total"] == 0


def test_prefix_sanity_on_synthetic_frame(tmp_path) -> None:
    src = _synthetic_shard()
    path = tmp_path / "shard.parquet"
    src.to_parquet(path)
    derived = tmp_path / "derived.parquet"
    con = connect_ephemeral()
    try:
        sql = build_prefix_sql(path, "minisweagent", "model", "src")
        con.execute(f"COPY ({sql}) TO '{derived}' (FORMAT PARQUET)")
        sanity = prefix_sanity(con, derived)
    finally:
        con.close()
    assert sanity["non_monotone"] == 0
    assert int(sanity["checks"]["n_rows"]) == 2
    assert int(sanity["checks"]["bad_repeat_rate"]) == 0
    assert int(sanity["checks"]["edit_without_view"]) == 0


def _first_shard_query(con, limit: int):
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")
    shard = parse_shard(files[0])
    sql = build_prefix_sql(shard.path, shard.harness, shard.teacher, shard.source)
    return con.execute(f"SELECT * FROM ({sql}) ORDER BY trajectory_id LIMIT {limit}").fetchdf(), shard


def test_prefix_sql_matches_reference_on_corpus_sample() -> None:
    con = connect_ephemeral()
    try:
        out, shard = _first_shard_query(con, 5)
        src = con.execute(
            f"SELECT trajectory_id, messages, metadata "
            f"FROM read_parquet({_sql_str(shard.path)}) ORDER BY trajectory_id LIMIT 5"
        ).fetchdf()
    finally:
        con.close()

    expected = {
        row["trajectory_id"]: {
            k: _reference_prefix(row["messages"], row["metadata"], k) for k in K_VALUES
        }
        for _, row in src.iterrows()
    }
    assert set(out["trajectory_id"]) == set(expected)
    for _, row in out.iterrows():
        ref = expected[row["trajectory_id"]]
        for k in K_VALUES:
            want = ref[k]
            assert row[f"n_calls_{k}"] == want["n_calls"], (row["trajectory_id"], k)
            assert row[f"gold_view_{k}"] == want["gold_view"], (row["trajectory_id"], k)
            assert row[f"gold_edit_{k}"] == want["gold_edit"], (row["trajectory_id"], k)
            assert row[f"n_edit_{k}"] == want["n_edit"], (row["trajectory_id"], k)
            assert row[f"n_distinct_{k}"] == want["n_distinct"], (row["trajectory_id"], k)
            assert row[f"repeat_rate_{k}"] == pytest.approx(want["repeat_rate"], abs=1e-9)
            assert row[f"assistant_chars_{k}"] == want["assistant_chars"], (row["trajectory_id"], k)
            assert bool(row[f"any_error_{k}"]) == want["any_error"], (row["trajectory_id"], k)
