import json
import re

import pandas as pd
import pytest

from openswe_traces.data import connect_ephemeral, list_parquet_files, parse_shard
from openswe_traces.features import BASH_EDIT_RE, EDIT_TOOL_NAMES, EDIT_TOOL_VIEW_ONLY
from openswe_traces.temporal_features import TEMPORAL_FEATURES, build_temporal_sql

GIT_PATH_RE = re.compile(r"diff --git a/(\S+) b/")
PLUS_PATH_RE = re.compile(r"\+\+\+ b/(\S+)")
BASH_EDIT_PY_RE = re.compile(BASH_EDIT_RE)
EDIT_TOOL_SET = frozenset(EDIT_TOOL_NAMES)


def _gold_paths(metadata) -> list[str]:
    patch = ((metadata or {}).get("reference_patch") or {}).get("patch") or ""
    git_paths = list(dict.fromkeys(GIT_PATH_RE.findall(patch)))
    if git_paths:
        return git_paths
    return list(dict.fromkeys(PLUS_PATH_RE.findall(patch)))


def _is_edit(call) -> bool:
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


def _reference_features(messages, metadata) -> dict:
    gold = _gold_paths(metadata)
    n_calls = 0
    first_view = first_edit = None
    n_edits_outside = 0
    touched: set[str] = set()
    for message in messages:
        calls = message.get("tool_calls")
        if calls is None:
            continue
        for call in calls:
            n_calls += 1
            args_text = (call.get("function") or {}).get("arguments") or ""
            matched = [p for p in gold if p in args_text or p.rsplit("/", 1)[-1] in args_text]
            if matched:
                touched.update(matched)
                if first_view is None:
                    first_view = n_calls
            if _is_edit(call):
                if matched:
                    if first_edit is None:
                        first_edit = n_calls
                else:
                    n_edits_outside += 1
    return {
        "n_turns_total": len(messages),
        "n_tool_calls": n_calls,
        "turn_first_gold_view": first_view if first_view is not None else -1,
        "turn_first_gold_edit": first_edit if first_edit is not None else -1,
        "frac_first_gold_view": (first_view / n_calls) if first_view is not None else None,
        "frac_first_gold_edit": (first_edit / n_calls) if first_edit is not None else None,
        "n_edits_outside_gold": n_edits_outside,
        "n_gold_files": len(gold),
        "n_gold_files_touched": len(touched),
        "gold_file_recall": (len(touched) / len(gold)) if gold else None,
        "frac_calls_after_first_gold_edit": (
            (n_calls - first_edit) / n_calls if first_edit is not None else None
        ),
    }


def _first_shard_query(con, limit: int, *, ordered: bool = False):
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")
    shard = parse_shard(files[0])
    sql = build_temporal_sql(shard.path, shard.harness, shard.teacher, shard.source)
    order = "ORDER BY trajectory_id " if ordered else ""
    return con.execute(f"SELECT * FROM ({sql}) {order}LIMIT {limit}").fetchdf()


def test_temporal_features_first_50_rows() -> None:
    con = connect_ephemeral()
    try:
        df = _first_shard_query(con, 50)
    finally:
        con.close()

    assert len(df) == 50
    assert set(TEMPORAL_FEATURES) <= set(df.columns)
    assert set(df["resolved"].unique()) <= {-1, 0, 1}
    assert (df["n_turns_total"] >= 1).all()
    assert (df["n_tool_calls"] >= 0).all()

    for first, frac in (
        ("turn_first_gold_view", "frac_first_gold_view"),
        ("turn_first_gold_edit", "frac_first_gold_edit"),
    ):
        assert (df[first] >= -1).all()
        assert (df[first] <= df["n_tool_calls"]).all()
        sentinel = df[first] == -1
        assert df.loc[sentinel, frac].isna().all()
        assert df.loc[~sentinel, frac].notna().all()
        values = df[frac].dropna()
        assert ((values >= 0) & (values <= 1)).all()

    edited = df[df["turn_first_gold_edit"] != -1]
    assert (edited["turn_first_gold_view"] != -1).all()
    assert (edited["turn_first_gold_edit"] >= edited["turn_first_gold_view"]).all()

    assert (df["n_edits_outside_gold"] >= 0).all()
    assert (df["n_gold_files"] >= 0).all()
    assert (df["n_gold_files_touched"] >= 0).all()
    assert (df["n_gold_files_touched"] <= df["n_gold_files"]).all()
    assert df.loc[df["n_gold_files"] == 0, "gold_file_recall"].isna().all()
    recall = df["gold_file_recall"].dropna()
    assert ((recall >= 0) & (recall <= 1)).all()
    after = df["frac_calls_after_first_gold_edit"]
    assert after[df["turn_first_gold_edit"] == -1].isna().all()
    assert ((after.dropna() >= 0) & (after.dropna() <= 1)).all()


def test_temporal_features_match_python_reference() -> None:
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")
    shard = parse_shard(files[0])
    path_sql = str(shard.path).replace("'", "''")

    con = connect_ephemeral()
    try:
        out = _first_shard_query(con, 10, ordered=True)
        src = con.execute(
            f"SELECT trajectory_id, messages, metadata FROM read_parquet('{path_sql}') "
            f"ORDER BY trajectory_id LIMIT 10"
        ).fetchdf()
    finally:
        con.close()

    expected = {
        row["trajectory_id"]: _reference_features(row["messages"], row["metadata"])
        for _, row in src.iterrows()
    }
    assert set(out["trajectory_id"]) == set(expected)

    for _, row in out.iterrows():
        ref = expected[row["trajectory_id"]]
        for feature in TEMPORAL_FEATURES:
            got, want = row[feature], ref[feature]
            if want is None:
                assert pd.isna(got), (row["trajectory_id"], feature, got)
            else:
                assert float(got) == pytest.approx(float(want), abs=1e-9), (
                    row["trajectory_id"],
                    feature,
                    got,
                    want,
                )
