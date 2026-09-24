"""Analysis DB: trajectory-feature extraction (1-shard fixture) and DB build."""

from __future__ import annotations

import json
from pathlib import Path

import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq
import pytest

import duckdb
from openswe_traces import analysis_db, data, trajectory_features

# -------------------------------------------------------------------------------------
# fixtures
# -------------------------------------------------------------------------------------

def write_shard(path: Path, rows: list[dict]) -> None:
    """Write a corpus-shaped shard parquet (data/<h>/<t>/<s>/<file>.parquet)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    table = pa.Table.from_pylist(rows)
    pq.write_table(table, path)


def write_parquet(path: Path, rows: list[dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    pq.write_table(pa.Table.from_pylist(rows), path)


def corpus_rows() -> list[dict]:
    """Two trajectories: one resolved with edits + submit, one failed without."""
    gold = (
        "diff --git a/src/foo.py b/src/foo.py\n"
        "--- a/src/foo.py\n"
        "+++ b/src/foo.py\n"
        "@@ -1,3 +1,5 @@\n"
        " line1\n"
        "+new\n"
        " line2\n"
        " line3\n"
    )
    model = (
        "diff --git a/src/foo.py b/src/foo.py\n"
        "--- a/src/foo.py\n"
        "+++ b/src/foo.py\n"
        "@@ -1,3 +1,5 @@\n"
        " line1\n"
        "+new\n"
        " line2\n"
        " line3\n"
    )

    def meta(category: str, patch: str) -> dict:
        return {
            "category": category,
            "teacher_model": {"name": "Qwen3.6-27B", "enable_thinking": False, "reasoning_effort": None},
            "reference_patch": {"patch": gold, "num_modified_files": 1, "num_modified_lines": 2},
            "model_patch": {"patch": patch, "num_modified_files": 1, "num_modified_lines": 2},
        }

    def call(name: str, arguments: str) -> dict:
        return {"function": {"name": name, "arguments": arguments}, "id": "c1", "type": "function"}

    msgs_pass = [
        {"role": "system", "content": "You are helpful.", "reasoning_content": None, "tool_calls": None},
        {"role": "user", "content": "fix foo", "reasoning_content": None, "tool_calls": None},
        {
            "role": "assistant",
            "content": "reproduce",
            "reasoning_content": None,
            "tool_calls": [call("bash", json.dumps({"command": "python test_foo.py"}))],
        },
        {
            "role": "tool",
            "content": json.dumps({"returncode": 1, "output": "Traceback (most recent call last)"}),
            "reasoning_content": None,
            "tool_calls": None,
        },
        {
            "role": "assistant",
            "content": "edit",
            "reasoning_content": None,
            "tool_calls": [
                call("str_replace_editor", json.dumps({"path": "src/foo.py", "old_string": "a", "new_string": "b"}))
            ],
        },
        {"role": "tool", "content": json.dumps({"returncode": 0}), "reasoning_content": None, "tool_calls": None},
        {"role": "assistant", "content": "submit", "reasoning_content": None, "tool_calls": None},
    ]
    msgs_fail = [
        {"role": "system", "content": "You are helpful.", "reasoning_content": None, "tool_calls": None},
        {"role": "user", "content": "fix foo", "reasoning_content": None, "tool_calls": None},
        {"role": "assistant", "content": "give up", "reasoning_content": None, "tool_calls": None},
    ]
    return [
        {
            "instance_id": "repo__foo-1",
            "repo": "repo",
            "license": "MIT",
            "language": "python",
            "trajectory_id": "traj-0001",
            "messages": msgs_pass,
            "tools": ["bash"],
            "resolved": 1,
            "metadata": meta("other", model),
            "hf_dataset_name": "AweAI-Team/Scale-SWE",
        },
        {
            "instance_id": "repo__foo-1",
            "repo": "repo",
            "license": "MIT",
            "language": "python",
            "trajectory_id": "traj-0002",
            "messages": msgs_fail,
            "tools": ["bash"],
            "resolved": 0,
            "metadata": meta("other", ""),
            "hf_dataset_name": "AweAI-Team/Scale-SWE",
        },
    ]


@pytest.fixture()
def corpus_shard(tmp_path: Path) -> Path:
    p = tmp_path / "data" / "minisweagent" / "qwen36_27b" / "scale-swe" / "train-00000-of-00001.parquet"
    write_shard(p, corpus_rows())
    return p


def build_derived_fixtures(tmp_path: Path, shard_path: Path) -> tuple[Path, Path]:
    """Tiny derived + external fixture trees for build_analysis_db; returns (derived, external)."""
    derived = tmp_path / "derived"
    external = tmp_path / "external"
    write_parquet(
        derived / "closure_proxies.parquet",
        [
            {
                "instance_id": "repo__foo-1", "repo": "repo", "language": "python",
                "n_rollouts": 2, "n_labeled": 2, "n_resolved": 1,
                "gold_patch_lines": 2, "gold_patch_files": 1, "has_gold_patch": 1,
                "n_hunks": 1.0, "n_files": 1.0, "added_lines": 1.0, "removed_lines": 0.0,
                "n_new_defs": 0.0, "internal_refs": 0.0, "boundary_refs": 1.0,
                "ratio": 0.0, "new_frac": 0.0, "edit_frac": 0.0,
            }
        ],
    )
    write_parquet(
        derived / "rung_features.parquet",
        [
            {
                "instance_id": "repo__foo-1", "rung": 2, "task_text": "fix foo",
                "leakage_symbols": "[]", "content_len": 10, "n_rows": 1, "word_count": 2,
                "has_repro": 0, "has_expected_actual": 0, "has_stack_trace": 0,
                "has_test_names": 1, "has_signature": 0, "has_leakage": 0, "leakage_count": 0,
                "has_test_code": 0, "n_test_funcs_in_text": 0, "n_test_files_in_text": 0,
                "patch_has_tests": 0,
            }
        ],
    )
    write_parquet(
        derived / "patch_split.parquet",
        [
            {
                "instance_id": "repo__foo-1", "repo": "repo", "language": "python",
                "hf_dataset_name": "AweAI-Team/Scale-SWE", "n_rollouts": 2, "n_labeled": 2,
                "n_resolved": 1, "has_gold_patch": 1, "src_added": 1, "src_removed": 0,
                "test_added": 0, "test_removed": 0, "doc_added": 0, "doc_removed": 0,
                "config_added": 0, "config_removed": 0, "n_src_files": 1, "n_test_files": 0,
                "n_doc_files": 0, "n_config_files": 0, "n_new_test_funcs": 0, "test_only": 0,
            }
        ],
    )
    write_parquet(
        derived / "trajectory_frame.parquet",
        [
            {
                "trajectory_id": "traj-0001", "instance_id": "repo__foo-1", "repo": "repo",
                "language": "python", "harness": "minisweagent", "teacher": "qwen36_27b",
                "source": "scale-swe", "resolved": 1, "patch_file_jaccard": 1.0,
            },
            {
                "trajectory_id": "traj-0002", "instance_id": "repo__foo-1", "repo": "repo",
                "language": "python", "harness": "minisweagent", "teacher": "qwen36_27b",
                "source": "scale-swe", "resolved": 0, "patch_file_jaccard": 0.0,
            },
        ],
    )
    write_parquet(
        derived / "eligible_pairs.parquet",
        [
            {
                "trajectory_id": "traj-0001", "instance_id": "repo__foo-1",
                "combo": "minisweagent/qwen36_27b", "harness": "minisweagent",
                "teacher": "qwen36_27b", "source": "scale-swe", "resolved": 1,
                "added_lines": 1.0, "n_labeled": 5, "n_resolved": 1, "repo": "repo",
                "language": "python", "stratum": "SMALL", "group_id": "repo__foo-1|minisweagent/qwen36_27b",
            }
        ],
    )
    feats_pass = {
        "trajectory_id": "traj-0001", "harness": "minisweagent",
        "teacher": "qwen36_27b", "source": "scale-swe", "instance_id": "repo__foo-1",
        "repo": "repo", "language": "python", "category": "other", "resolved": 1,
        "n_turns": 7, "n_assistant_turns": 3, "assistant_chars": 50,
        "n_tool_errors": 1, "n_tool_calls": 2, "n_edit_calls": 1,
        "n_test_runs": 0, "n_repro_calls": 1, "turn_of_first_edit": 2,
        "turn_of_last_edit": 2, "first_repro_call": 1, "last_test_call": -1,
        "n_files_edited": 1, "edited_paths": ["src/foo.py"], "n_files_read": 0,
        "n_submit_calls": 0, "ends_with_submit": True, "n_model_hunks": 1.0,
        "n_gold_hunks": 1.0, "patch_hunk_coverage": 1.0, "extra_hunks": 0.0,
        "extra_hunk_lines": 0.0, "gold_changed_lines": 2.0, "model_changed_lines": 2.0,
        "model_lines_over_gold": 1.0, "edited_test_file": False,
        "touched_test_path": False, "ran_repro_before_edit": True,
        "ran_tests_after_last_edit": False, "has_submit_call": False,
        "stop_reason": "submit", "submitted_empty_patch": False,
    }
    feats_fail = {
        "trajectory_id": "traj-0002", "harness": "minisweagent",
        "teacher": "qwen36_27b", "source": "scale-swe", "instance_id": "repo__foo-1",
        "repo": "repo", "language": "python", "category": "other", "resolved": 0,
        "n_turns": 3, "n_assistant_turns": 1, "assistant_chars": 20,
        "n_tool_errors": 0, "n_tool_calls": 0, "n_edit_calls": 0,
        "n_test_runs": 0, "n_repro_calls": 0, "turn_of_first_edit": -1,
        "turn_of_last_edit": -1, "first_repro_call": -1, "last_test_call": -1,
        "n_files_edited": 0, "edited_paths": [], "n_files_read": 0,
        "n_submit_calls": 0, "ends_with_submit": False, "n_model_hunks": 0.0,
        "n_gold_hunks": 1.0, "patch_hunk_coverage": None, "extra_hunks": 0.0,
        "extra_hunk_lines": 0.0, "gold_changed_lines": 2.0, "model_changed_lines": 0.0,
        "model_lines_over_gold": 0.0, "edited_test_file": False,
        "touched_test_path": False, "ran_repro_before_edit": False,
        "ran_tests_after_last_edit": False, "has_submit_call": False,
        "stop_reason": "no_submit", "submitted_empty_patch": False,
    }
    write_parquet(derived / "paired_features.parquet", [feats_pass])
    write_parquet(derived / "trajectory_features.parquet", [feats_pass, feats_fail])
    (derived / "llm_labels.jsonl").write_text(
        json.dumps(
            {
                "trajectory_id": "traj-0001", "instance_id": "repo__foo-1",
                "combo": "minisweagent/qwen36_27b", "rule_bucket": "complete_but_wrong",
                "model": "nvidia/nemotron-3-super-120b-a12b:free", "label": "other",
                "evidence": "fixture",
            }
        )
        + "\n"
    )
    write_parquet(
        derived / "closure_metrics.parquet",
        [
            {
                "repo": "repo", "unit": "unit-a", "family": "single-file",
                "predicted_flip": 2.0, "control": False, "metric_source": "authored unit",
                "n_files": 1, "n_funcs_removed": 1, "lines_removed": 3,
                "internal_edges": 0, "boundary_in": 1, "boundary_out": 0, "ratio": 1.0,
                "flip_devin": "2", "flip_devin_enc": 2.0, "flip_cursor": None,
                "flip_cursor_enc": None, "flip_note": None,
            }
        ],
    )
    hub = derived / "harbor_hub"
    write_parquet(hub / "jobs.parquet", [{"job_id": "j1", "bench_version": "2.1", "title": "job", "n_trials": 1}])
    write_parquet(
        hub / "leaderboard_rows.parquet",
        [{"row_id": "r1", "bench_version": "2.1", "agent": "a", "model": "m", "n_trials": 1}],
    )
    write_parquet(hub / "row_trials.parquet", [{"row_id": "r1", "trial_id": "t1", "bench_version": "2.1"}])
    # Same task_id on two bench versions: proves the composite PK (bench_version, task_id).
    write_parquet(
        hub / "tasks.parquet",
        [
            {"bench_version": "3.0", "task_id": "task-1", "n_trials": 2, "n_jobs": 1, "mean_reward": 0.5, "n_passed": 1},
            {"bench_version": "4.0", "task_id": "task-1", "n_trials": 2, "n_jobs": 1, "mean_reward": 0.5, "n_passed": 1},
        ],
    )
    write_parquet(
        hub / "trials.parquet",
        [
            {
                "trial_id": "t1", "bench_version": "2.1", "job_id": "j1",
                "leaderboard_row_id": "r1", "task_id": "task-1", "task_name": "task",
                "reward": 1.0, "passed": True, "is_scored": True, "status": "completed",
            },
            {
                "trial_id": "t2", "bench_version": "2.1", "job_id": "j1",
                "leaderboard_row_id": "r1", "task_id": "task-1", "task_name": "task",
                "reward": 0.0, "passed": False, "is_scored": True, "status": "completed",
            },
        ],
    )
    # yoonholee: trial_id is NOT unique (retries) -> table has no PK.
    yoon = external / "yoonholee__terminalbench-trajectories" / "data"
    write_parquet(
        yoon / "train-00000-of-00002.parquet",
        [
            {"task_name": "t", "agent": "a", "model": "m", "reward": 1, "trial_id": "dup", "steps": None},
            {"task_name": "t", "agent": "a", "model": "m", "reward": 0, "trial_id": "dup", "steps": None},
        ],
    )
    har = external / "harithoppil__terminal-bench-2-trajectories" / "data"
    har.mkdir(parents=True, exist_ok=True)
    (har / "leaderboard_trajectories.jsonl").write_text(
        json.dumps({"task_name": "t", "model": "m", "agent": "a", "prompt": "", "response": "", "reward": 1.0})
        + "\n"
        + json.dumps({"task_name": "t", "model": "m", "agent": "a", "prompt": "", "response": "", "reward": 0.0})
        + "\n"
    )
    write_parquet(
        external / "AweAI-Team__Scale-SWE" / "meta.parquet",
        [
            {
                "instance_id": "repo__foo-1", "repo": "repo", "language": "python",
                "created_at": None, "problem_statement": "fix foo", "test_patch": "",
                "fail_to_pass": ["test_a"], "pass_to_pass": [],
            }
        ],
    )
    write_parquet(
        external / "nebius__SWE-rebench-V2" / "meta.parquet",
        [
            {
                "instance_id": "repo__foo-1", "repo": "repo", "language": "python",
                "created_at": "2024-01-01", "problem_statement": "fix foo", "test_patch": "",
                "fail_to_pass": [], "pass_to_pass": [],
            }
        ],
    )
    return derived, external


# -------------------------------------------------------------------------------------
# pure functions
# -------------------------------------------------------------------------------------

def test_parse_patch_hunk_features() -> None:
    gold = (
        "diff --git a/src/a.py b/src/a.py\n--- a/src/a.py\n+++ b/src/a.py\n"
        "@@ -1,3 +1,5 @@\n line1\n+new\n line2\n"
    )
    model = gold  # identical patch -> full coverage, no extras
    feats = trajectory_features.hunk_features(model, gold)
    assert feats["n_gold_hunks"] == 1
    assert feats["n_model_hunks"] == 1
    assert feats["patch_hunk_coverage"] == 1.0
    assert feats["extra_hunks"] == 0
    assert feats["model_lines_over_gold"] == 1.0

    off_target = (
        "diff --git a/src/b.py b/src/b.py\n--- a/src/b.py\n+++ b/src/b.py\n"
        "@@ -10,2 +10,2 @@\n old\n+other\n"
    )
    feats2 = trajectory_features.hunk_features(off_target, gold)
    assert feats2["patch_hunk_coverage"] == 0.0
    assert feats2["extra_hunks"] == 1


# -------------------------------------------------------------------------------------
# 1-shard extraction
# -------------------------------------------------------------------------------------

def test_process_shard_extracts_all_trajectories(corpus_shard: Path, tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setattr(
        trajectory_features, "parse_shard", lambda p: data.parse_shard(p, root=corpus_shard.parents[3])
    )
    con = duckdb.connect()
    try:
        status, rows = trajectory_features.process_shard(con, corpus_shard, parts_dir=tmp_path)
    finally:
        con.close()
    assert status == "done"
    assert rows == 2  # all trajectories, not just eligible ones

    part = next(tmp_path.glob("*.parquet"))
    df = pd.read_parquet(part)
    assert len(df) == 2
    passed = df[df["trajectory_id"] == "traj-0001"].iloc[0]
    assert passed["n_turns"] == 7
    assert passed["n_edit_calls"] == 1
    assert passed["n_repro_calls"] == 1
    assert passed["n_test_runs"] == 0  # 'python test_foo.py' is a repro, not a BASH_TEST_RE match
    assert passed["n_tool_errors"] == 1
    assert passed["turn_of_first_edit"] == 2
    assert bool(passed["ran_repro_before_edit"]) is True
    assert passed["patch_hunk_coverage"] == 1.0
    assert passed["stop_reason"] == "submit"
    failed = df[df["trajectory_id"] == "traj-0002"].iloc[0]
    assert failed["n_turns"] == 3
    assert failed["n_tool_calls"] == 0

    # Resume safety: rerunning with parts present skips.
    con = duckdb.connect()
    try:
        status2, _ = trajectory_features.process_shard(con, corpus_shard, parts_dir=tmp_path)
    finally:
        con.close()
    assert status2 == "skip"


# -------------------------------------------------------------------------------------
# DB build
# -------------------------------------------------------------------------------------

@pytest.fixture()
def analysis_fixture(tmp_path: Path, corpus_shard: Path) -> tuple[Path, Path, Path]:
    derived, external = build_derived_fixtures(tmp_path, corpus_shard)
    db_path = tmp_path / "analysis.duckdb"
    return db_path, derived, external


def _constraint(con: duckdb.DuckDBPyConnection, table: str) -> list[str]:
    rows = con.execute(
        f"SELECT constraint_type, constraint_column_names FROM duckdb_constraints() WHERE table_name = '{table}'"
    ).fetchall()
    return sorted({tuple(r[1]) for r in rows if "PRIMARY KEY" in r[0]})


def test_build_analysis_db(analysis_fixture: tuple[Path, Path, Path]) -> None:
    db_path, derived, external = analysis_fixture
    counts = analysis_db.build_analysis_db(db_path, derived_dir=derived, traces_external=external)

    assert counts == {
        "closure_metrics": 1,
        "eligible_pairs": 1,
        "harbor_hub_jobs": 1,
        "harbor_hub_leaderboard_rows": 1,
        "harbor_hub_row_trials": 1,
        "harbor_hub_tasks": 2,
        "harbor_hub_trials": 2,
        "instance": 1,
        "llm_labels": 1,
        "paired_features": 1,
        "rebench_v2_meta": 1,
        "scale_swe_meta": 1,
        "tb2_traj_harithoppil": 2,
        "tb2_traj_yoonholee": 2,
        "trajectory": 2,
    }

    con = duckdb.connect(str(db_path), read_only=True)
    try:
        assert _constraint(con, "instance") == [("instance_id",)]
        assert _constraint(con, "trajectory") == [("trajectory_id",)]
        assert _constraint(con, "harbor_hub_tasks") == [("bench_version", "task_id")]
        assert _constraint(con, "closure_metrics") == [("repo", "unit")]
        assert _constraint(con, "tb2_traj_yoonholee") == []  # no natural key

        # instance wide: proxies + rung + patch_split joined on instance_id.
        row = con.execute("SELECT added_lines, rung, src_added FROM instance").fetchone()
        assert row == (1.0, 2, 1)

        # trajectory wide: frame + features + eligibility joined on trajectory_id.
        row = con.execute(
            "SELECT patch_file_jaccard, n_turns, combo, stratum, group_id "
            "FROM trajectory WHERE trajectory_id = 'traj-0001'"
        ).fetchone()
        assert row[0] == 1.0
        assert row[1] == 7
        assert row[2] == "minisweagent/qwen36_27b"
        assert row[3] == "SMALL"
        assert row[4] == "repo__foo-1|minisweagent/qwen36_27b"

        # _build_meta recorded.
        assert con.execute('SELECT n_tables FROM "_build_meta"').fetchone()[0] == 15

        # The 4 analysis queries run against the fixture DB.
        for q in ("101_mixture_size_deciles.sql", "102_rung_x_size_tercile.sql",
                  "103_paired_diff_inputs.sql", "104_harbor_task_rates.sql"):
            sql = (Path(__file__).parents[1] / "analytics" / "queries" / q).read_text()
            con.execute(sql)  # must not raise
    finally:
        con.close()


def test_build_analysis_db_idempotent(analysis_fixture: tuple[Path, Path, Path]) -> None:
    db_path, derived, external = analysis_fixture
    first = analysis_db.build_analysis_db(db_path, derived_dir=derived, traces_external=external)
    second = analysis_db.build_analysis_db(db_path, derived_dir=derived, traces_external=external)
    assert first == second
