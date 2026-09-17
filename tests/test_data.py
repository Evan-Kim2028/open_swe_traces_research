from pathlib import Path

import pytest

from openswe_traces.data import list_parquet_files, parse_shard


def test_parse_shard_splits_corpus_path() -> None:
    root = Path("/corpus/data")
    path = root / "minisweagent" / "qwen36_27b" / "swe-rebench-v2" / "train-00000-of-00022.parquet"
    shard = parse_shard(path, root=root)
    assert (shard.harness, shard.teacher, shard.source) == (
        "minisweagent",
        "qwen36_27b",
        "swe-rebench-v2",
    )
    assert shard.rel == Path("minisweagent/qwen36_27b/swe-rebench-v2/train-00000-of-00022.parquet")
    assert shard.label == "minisweagent/qwen36_27b/swe-rebench-v2/train-00000-of-00022.parquet"


def test_parse_shard_rejects_wrong_depth() -> None:
    root = Path("/corpus/data")
    with pytest.raises(ValueError, match="Unexpected shard path"):
        parse_shard(root / "flat.parquet", root=root)


def test_parse_shard_rejects_path_outside_root() -> None:
    with pytest.raises(ValueError):
        parse_shard(Path("/elsewhere/x/y/z/file.parquet"), root=Path("/corpus/data"))


def test_corpus_shards_parse_when_present() -> None:
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")
    for path in files[:5]:
        shard = parse_shard(path)
        assert shard.path == path
        assert shard.harness and shard.teacher and shard.source
