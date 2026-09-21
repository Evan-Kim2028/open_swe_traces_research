import pandas as pd
import pytest

from openswe_traces.data import connect_ephemeral, list_parquet_files, parse_shard
from openswe_traces.failure_anatomy import (
    FIX_COMPLETE_RE,
    TESTS_PASS_RE,
    UNCERTAIN_RE,
    UNSEEN_GAP_RE,
    build_anatomy_sql,
    classify_last_text,
    classify_test_obs,
    pick_q3_sample,
    wilson,
)


def test_wilson_bounds_on_known_values() -> None:
    p, lo, hi = wilson(0, 61)
    assert p == 0.0
    assert 0.0 <= lo <= hi <= 1.0
    p, lo, hi = wilson(46, 61)
    assert abs(p - 46 / 61) < 1e-12
    assert lo < p < hi


def test_last_text_flags_on_anatomy_examples() -> None:
    fail_claim = (
        "All 65 existing unit tests pass. Implementation is complete. "
        "Now I'll submit the patch."
    )
    flags = classify_last_text(fail_claim)
    assert flags["asserts_tests_pass"]
    assert flags["asserts_fix_complete"]
    assert not flags["expresses_uncertainty"]
    assert not flags["mentions_unseen_gap"]

    uncertain = "I'm not sure this is complete; tests may still fail on hidden cases."
    flags = classify_last_text(uncertain)
    assert flags["expresses_uncertainty"]

    # SWE-bench-aware mention is not the unseen-gap flag.
    hidden = "The hidden test patch will update this test. All 12 existing tests pass."
    flags = classify_last_text(hidden)
    assert flags["mentions_hidden_tests"]
    assert flags["asserts_tests_pass"]
    assert not flags["mentions_unseen_gap"]

    gap = "I cannot run the hidden tests and they might fail."
    flags = classify_last_text(gap)
    assert flags["mentions_unseen_gap"]


def test_complete_task_and_submit_is_not_a_pass_claim() -> None:
    text = 'echo COMPLETE_TASK_AND_SUBMIT_FINAL_OUTPUT && cat patch.txt'
    flags = classify_last_text(text)
    assert not flags["asserts_tests_pass"]
    assert not flags["asserts_fix_complete"]


def test_classify_test_obs_prefers_exit_code() -> None:
    mini_green = '{"returncode": 0, "output": "..... [100%] ALL CHECKS PASSED"}'
    assert classify_test_obs(mini_green) is True
    mini_red = '{"returncode": 1, "output": "FAILED test_foo.py::test_bar"}'
    assert classify_test_obs(mini_red) is False
    mixed = "1 failed, 10 passed [The command completed with exit code 1.]"
    assert classify_test_obs(mixed) is False
    jest_green = "Test Suites: 5 passed, 5 total Tests: 52 passed, 52 total exit code 0"
    assert classify_test_obs(jest_green) is True
    assert classify_test_obs("no test output here") is None


def test_regexes_are_re2_safe() -> None:
    for pattern in (TESTS_PASS_RE, FIX_COMPLETE_RE, UNCERTAIN_RE, UNSEEN_GAP_RE):
        assert "(?" not in pattern.replace("(?i)", "")


def test_q3_sample_mixes_fail_to_pass_size() -> None:
    rows = []
    for lang in ("go", "python", "rust"):
        for n_ftp, n in ((1, 40), (3, 40), (12, 40)):
            for i in range(n):
                rows.append(
                    {
                        "instance_id": f"{lang}-{n_ftp}-{i}",
                        "language": lang,
                        "repo": "r",
                        "source": "swe-rebench-v2",
                        "n_labeled": 6,
                        "n_fail": 6,
                        "n_ok": 0,
                        "solve_rate": 0.0,
                        "n_ftp": n_ftp,
                    }
                )
    inst = pd.DataFrame(rows)
    sample = pick_q3_sample(inst, n=40, seed=42)
    assert len(sample) == 40
    assert set(sample["language"]) >= {"go", "python"}
    ftp_counts = sample["n_ftp"].value_counts()
    assert ftp_counts.min() >= 8, ftp_counts.to_dict()


def test_anatomy_sql_first_shard_labeled_rows() -> None:
    files = list_parquet_files()
    if not files:
        pytest.skip("traces_data corpus not downloaded")
    shard = parse_shard(files[0])
    con = connect_ephemeral()
    try:
        sql = build_anatomy_sql(shard.path, shard.harness, shard.teacher, shard.source)
        labeled = con.execute(
            f"SELECT count(*) FROM read_parquet('{shard.path}') WHERE resolved IN (0, 1)"
        ).fetchone()[0]
        if labeled == 0:
            pytest.skip("first shard has no labeled rows")
        df = con.execute(f"SELECT * FROM ({sql}) LIMIT 20").fetchdf()
    finally:
        con.close()
    assert len(df) == min(20, labeled)
    assert df["resolved"].isin([0, 1]).all()
    assert df["asserts_done"].isin([True, False]).all()
    assert "last_test_green" in df.columns
    assert "patch_file_jaccard" in df.columns
    assert ((df["patch_file_jaccard"] >= 0) & (df["patch_file_jaccard"] <= 1)).all()
