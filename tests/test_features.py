import pytest

from openswe_traces.data import connect_ephemeral, list_parquet_files, parse_shard
from openswe_traces.features import NUMERIC_FEATURES, build_feature_sql

COUNT_COLUMNS = [
    "n_messages",
    "n_assistant_turns",
    "n_tool_calls",
    "n_distinct_tool_commands",
    "max_consecutive_identical_calls",
    "n_edit_calls",
    "n_test_calls",
    "model_patch_files",
    "model_patch_lines",
    "gold_patch_files",
    "gold_patch_lines",
]


def test_proxy_features_first_50_rows() -> None:
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")

    shard = parse_shard(files[0])
    con = connect_ephemeral()
    try:
        sql = build_feature_sql(shard.path, shard.harness, shard.teacher, shard.source)
        df = con.execute(f"SELECT * FROM ({sql}) LIMIT 50").fetchdf()
    finally:
        con.close()

    assert len(df) == 50
    assert set(NUMERIC_FEATURES) <= set(df.columns)
    assert set(COUNT_COLUMNS) <= set(df.columns)
    assert df[COUNT_COLUMNS].notna().all().all()
    assert (df[COUNT_COLUMNS] >= 0).all().all()
    assert set(df["resolved"].unique()) <= {-1, 0, 1}
