import json
from pathlib import Path

import pytest

from openswe_traces.data import list_parquet_files
from openswe_traces.sft.sample import SRC_GLOB, build_sample


def test_build_sample_makes_tool_call_arguments_dicts(tmp_path: Path) -> None:
    sources = list_parquet_files(SRC_GLOB)
    if not sources:
        pytest.skip("minisweagent/qwen36_27b source not downloaded")

    out = build_sample(5, seed=0, src=str(sources[0]), out_dir=tmp_path)
    rows = [json.loads(line) for line in out.read_text().splitlines()]

    assert len(rows) == 5
    n_tool_calls = 0
    for row in rows:
        assert row["messages"]
        for msg in row["messages"]:
            for call in msg.get("tool_calls", []):
                assert isinstance(call["function"]["arguments"], dict)
                n_tool_calls += 1
    assert n_tool_calls > 0
