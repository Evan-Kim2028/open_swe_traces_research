"""Tests for openswe_traces.paired on fully synthetic trajectories — no corpus needed."""

import json

import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq

from openswe_traces.data import connect_ephemeral
from openswe_traces.paired import (
    add_patch_features,
    build_paired_sql,
    classify_failure,
    hunk_features,
    is_test_path,
    parse_patch,
)

GOLD_PATCH = """\
diff --git a/pkg/foo.py b/pkg/foo.py
--- a/pkg/foo.py
+++ b/pkg/foo.py
@@ -10,3 +10,4 @@ def f():
 ctx
-old
+new
+newer
 ctx
diff --git a/pkg/bar.py b/pkg/bar.py
--- a/pkg/bar.py
+++ b/pkg/bar.py
@@ -40,2 +40,3 @@ def g():
 a
-b
+c
+d
"""

MODEL_PATCH_SAME = GOLD_PATCH  # identical to gold → coverage 1, no extras

MODEL_PATCH_WRONG_SITE = """\
diff --git a/other/baz.py b/other/baz.py
--- a/other/baz.py
+++ b/other/baz.py
@@ -5,2 +5,3 @@
 a
-b
+c
"""

MODEL_PATCH_PARTIAL = """\
diff --git a/pkg/foo.py b/pkg/foo.py
--- a/pkg/foo.py
+++ b/pkg/foo.py
@@ -10,3 +10,4 @@ def f():
 ctx
-old
+new
+newer
 ctx
"""

MODEL_PATCH_TEST = """\
diff --git a/tests/test_foo.py b/tests/test_foo.py
--- a/tests/test_foo.py
+++ b/tests/test_foo.py
@@ -1,2 +1,3 @@
 a
-b
+c
"""


def _msg(role, content="", tool_calls=None):
    return {
        "role": role,
        "content": content,
        "reasoning_content": None,
        "tool_calls": tool_calls,
    }


def _call(name, **args):
    return {
        "function": {"arguments": json.dumps(args), "name": name},
        "id": "x",
        "type": "function",
    }


def _metadata(model_patch="", gold_patch=GOLD_PATCH):
    return {
        "category": "test",
        "teacher_model": {"name": "t", "enable_thinking": False, "reasoning_effort": ""},
        "reference_patch": {
            "patch": gold_patch,
            "num_modified_files": 2,
            "num_modified_lines": 4,
        },
        "model_patch": {
            "patch": model_patch,
            "num_modified_files": 0,
            "num_modified_lines": 0,
        },
    }


MSG_TYPE = pa.struct(
    [
        ("role", pa.string()),
        ("content", pa.string()),
        ("reasoning_content", pa.string()),
        (
            "tool_calls",
            pa.list_(
                pa.struct(
                    [
                        (
                            "function",
                            pa.struct(
                                [("arguments", pa.string()), ("name", pa.string())]
                            ),
                        ),
                        ("id", pa.string()),
                        ("type", pa.string()),
                    ]
                )
            ),
        ),
    ]
)

META_TYPE = pa.struct(
    [
        ("category", pa.string()),
        (
            "teacher_model",
            pa.struct(
                [
                    ("name", pa.string()),
                    ("enable_thinking", pa.bool_()),
                    ("reasoning_effort", pa.string()),
                ]
            ),
        ),
        (
            "reference_patch",
            pa.struct(
                [
                    ("patch", pa.string()),
                    ("num_modified_files", pa.int64()),
                    ("num_modified_lines", pa.int64()),
                ]
            ),
        ),
        (
            "model_patch",
            pa.struct(
                [
                    ("patch", pa.string()),
                    ("num_modified_files", pa.int64()),
                    ("num_modified_lines", pa.int64()),
                ]
            ),
        ),
    ]
)

SCHEMA = pa.schema(
    [
        ("instance_id", pa.string()),
        ("repo", pa.string()),
        ("license", pa.string()),
        ("language", pa.string()),
        ("trajectory_id", pa.string()),
        ("messages", pa.list_(MSG_TYPE)),
        ("tools", pa.list_(pa.string())),
        ("resolved", pa.int32()),
        ("metadata", META_TYPE),
        ("hf_dataset_name", pa.string()),
    ]
)


def _row(tid, resolved, messages, model_patch):
    return {
        "instance_id": "inst__1",
        "repo": "r",
        "license": "mit",
        "language": "python",
        "trajectory_id": tid,
        "messages": messages,
        "tools": ["bash", "str_replace_editor"],
        "resolved": resolved,
        "metadata": _metadata(model_patch),
        "hf_dataset_name": "synth",
    }


def _write_shard(tmp_path, rows):
    path = tmp_path / "shard.parquet"
    pq.write_table(pa.Table.from_pylist(rows, schema=SCHEMA), path)
    return path


def _extract(tmp_path, rows):
    shard = _write_shard(tmp_path, rows)
    con = connect_ephemeral()
    try:
        con.register("eligible_ids", pd.DataFrame({"trajectory_id": [r["trajectory_id"] for r in rows]}))
        df = con.execute(build_paired_sql(shard, "harness", "teacher", "src")).fetchdf()
    finally:
        con.close()
    return add_patch_features(df).set_index("trajectory_id")


# --- pure-python unit tests ----------------------------------------------------


def test_parse_patch_two_files():
    hunks = parse_patch(GOLD_PATCH)
    assert len(hunks) == 2
    assert hunks[0].path == "pkg/foo.py"
    assert (hunks[0].start, hunks[0].end) == (10, 13)
    assert (hunks[0].n_add, hunks[0].n_del) == (2, 1)
    assert hunks[1].path == "pkg/bar.py"
    assert (hunks[1].n_add, hunks[1].n_del) == (2, 1)


def test_parse_patch_empty():
    assert parse_patch("") == []
    assert parse_patch("not a diff\n") == []


def test_hunk_features_identical():
    f = hunk_features(MODEL_PATCH_SAME, GOLD_PATCH)
    assert f["patch_hunk_coverage"] == 1.0
    assert f["extra_hunks"] == 0
    assert f["n_gold_hunks"] == 2


def test_hunk_features_wrong_site():
    f = hunk_features(MODEL_PATCH_WRONG_SITE, GOLD_PATCH)
    assert f["patch_hunk_coverage"] == 0.0
    assert f["extra_hunks"] == 1
    assert f["model_lines_over_gold"] is not None


def test_hunk_features_partial():
    f = hunk_features(MODEL_PATCH_PARTIAL, GOLD_PATCH)
    assert f["patch_hunk_coverage"] == 0.5
    assert f["extra_hunks"] == 0


def test_hunk_features_no_gold():
    f = hunk_features(MODEL_PATCH_WRONG_SITE, "")
    assert f["patch_hunk_coverage"] is None
    assert f["extra_hunks"] == 1


def test_hunk_features_margin():
    # model hunk 8 lines away from the gold hunk in the same file → covered (margin 10)
    patch = """\
diff --git a/pkg/foo.py b/pkg/foo.py
--- a/pkg/foo.py
+++ b/pkg/foo.py
@@ -19,2 +19,3 @@
 a
-b
+c
"""
    f = hunk_features(patch, GOLD_PATCH)
    assert f["patch_hunk_coverage"] == 0.5  # foo.py hunk covered, bar.py not


def test_is_test_path():
    assert is_test_path("tests/test_foo.py")
    assert is_test_path("pkg/bar_test.go")
    assert is_test_path("a/b/conftest.py")
    assert not is_test_path("pkg/foo.py")
    assert not is_test_path("latest/results.py")


def test_classify_failure_buckets():
    base = {
        "stop_reason": "submit",
        "n_model_hunks": 2,
        "edited_test_file": False,
        "patch_hunk_coverage": 1.0,
        "extra_hunks": 0,
        "extra_hunk_lines": 0,
        "gold_changed_lines": 10,
    }
    assert classify_failure(base) == "complete_but_wrong"
    assert classify_failure(base | {"stop_reason": "no_submit"}) == "timeout"
    assert classify_failure(base | {"n_model_hunks": 0}) == "no_patch"
    assert classify_failure(base | {"edited_test_file": True}) == "test_edit"
    assert (
        classify_failure(base | {"patch_hunk_coverage": 0.0, "extra_hunks": 1})
        == "wrong_site"
    )
    assert classify_failure(base | {"patch_hunk_coverage": 0.5}) == "partial"
    assert (
        classify_failure(base | {"patch_hunk_coverage": 0.5, "extra_hunks": 2})
        == "partial_extra"
    )
    assert (
        classify_failure(
            base | {"extra_hunks": 3, "extra_hunk_lines": 50, "gold_changed_lines": 10}
        )
        == "over_edit"
    )
    # extras present but not large → still complete_but_wrong
    assert (
        classify_failure(base | {"extra_hunks": 1, "extra_hunk_lines": 2})
        == "complete_but_wrong"
    )
    assert classify_failure(base | {"patch_hunk_coverage": None}) == "other"


# --- end-to-end SQL on a synthetic shard ----------------------------------------


def _pass_traj():
    return _row(
        "t-pass",
        1,
        [
            _msg("user", "fix the bug"),
            _msg("assistant", "let me reproduce", [_call("bash", command="python repro.py")]),
            _msg("tool", '{"returncode": 1, "output": "boom"}'),
            _msg(
                "assistant",
                "view file",
                [_call("str_replace_editor", command="view", path="/repo/pkg/foo.py")],
            ),
            _msg("tool", "file contents"),
            _msg(
                "assistant",
                "edit",
                [
                    _call(
                        "str_replace_editor",
                        command="str_replace",
                        path="/repo/pkg/foo.py",
                        old_str="old",
                        new_str="new",
                    )
                ],
            ),
            _msg("tool", "edited"),
            _msg("assistant", "run tests", [_call("bash", command="pytest tests/ -x")]),
            _msg("tool", "ok"),
            _msg("assistant", "submit", [_call("submit")]),
        ],
        MODEL_PATCH_SAME,
    )


def _fail_wrong_site_traj():
    return _row(
        "t-fail-wrong",
        0,
        [
            _msg("user", "fix the bug"),
            _msg("assistant", "read", [_call("bash", command="cat /repo/other/baz.py")]),
            _msg("tool", "contents"),
            _msg(
                "assistant",
                "edit",
                [
                    _call(
                        "str_replace_editor",
                        command="str_replace",
                        path="/repo/other/baz.py",
                        old_str="a",
                        new_str="b",
                    )
                ],
            ),
            _msg("tool", "edited"),
            _msg("assistant", "done, submitting", [_call("submit")]),
        ],
        MODEL_PATCH_WRONG_SITE,
    )


def _fail_timeout_traj():
    return _row(
        "t-fail-timeout",
        0,
        [
            _msg("user", "fix the bug"),
            _msg("assistant", "view", [_call("bash", command="cat /repo/pkg/foo.py")]),
            _msg("tool", "contents"),
            _msg(
                "assistant",
                "edit",
                [
                    _call(
                        "str_replace_editor",
                        command="str_replace",
                        path="/repo/pkg/foo.py",
                        old_str="x",
                        new_str="y",
                    )
                ],
            ),
            _msg("tool", "edited"),
            _msg("assistant", "still working"),
        ],
        MODEL_PATCH_PARTIAL,
    )


def test_end_to_end_features(tmp_path):
    df = _extract(
        tmp_path, [_pass_traj(), _fail_wrong_site_traj(), _fail_timeout_traj()]
    )
    assert set(df.index) == {"t-pass", "t-fail-wrong", "t-fail-timeout"}

    p = df.loc["t-pass"]
    assert p["n_turns"] == 10
    assert p["n_tool_calls"] == 5  # bash(repro), view, str_replace, pytest, submit
    assert p["ran_repro_before_edit"]
    assert p["ran_tests_after_last_edit"]
    assert p["n_test_runs"] == 1
    assert p["n_tool_errors"] == 1
    assert p["n_files_read"] == 1
    assert p["n_files_edited"] == 1
    assert p["stop_reason"] == "submit"
    assert p["patch_hunk_coverage"] == 1.0
    assert p["extra_hunks"] == 0
    assert p["turn_of_first_edit"] == 3
    assert p["turn_of_last_edit"] == 3

    w = df.loc["t-fail-wrong"]
    assert not w["ran_repro_before_edit"]
    assert not w["ran_tests_after_last_edit"]
    assert w["patch_hunk_coverage"] == 0.0
    assert w["extra_hunks"] == 1
    assert w["stop_reason"] == "submit"
    assert classify_failure(w) == "wrong_site"

    t = df.loc["t-fail-timeout"]
    assert t["stop_reason"] == "no_submit"
    assert t["patch_hunk_coverage"] == 0.5
    assert classify_failure(t) == "timeout"


def test_empty_patch_submit(tmp_path):
    row = _row(
        "t-empty",
        0,
        [
            _msg("user", "fix"),
            _msg("assistant", "nothing to do, submit", [_call("submit")]),
        ],
        "",
    )
    df = _extract(tmp_path, [row])
    r = df.loc["t-empty"]
    assert r["n_model_hunks"] == 0
    assert r["submitted_empty_patch"]
    assert classify_failure(r) == "no_patch"
