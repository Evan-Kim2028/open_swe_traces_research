from __future__ import annotations

import json
from pathlib import Path

from openswe_traces.external.harbor_hub import (
    assemble,
    parse_leaderboard_row,
    parse_trial_row,
    task_id_from_name,
)

FIXTURES = Path(__file__).parent / "fixtures" / "harbor_hub"


def test_parse_trial_row_fields():
    page = json.loads((FIXTURES / "job_trials_page.json").read_text())
    trial = page["items"][0]
    rec = parse_trial_row(
        trial,
        bench_version="4.0",
        leaderboard_id="lb",
        leaderboard_row_id="row1",
        harvested_at="2026-09-19T00:00:00+00:00",
    )
    assert rec["trial_id"] == "ad18aec0-8084-4a71-944d-271b25076ecc"
    assert rec["job_id"] == "0f01715e-2f98-40e3-836f-1ac4fdce39a4"
    assert rec["task_id"] == "live-database-cutover"
    assert rec["agent"] == "codex"
    assert rec["model"] == "gpt-6-astra"
    assert rec["model_provider"] == "openai"
    assert rec["reward"] == 0.0
    assert rec["passed"] is False
    assert rec["status"] == "completed"
    assert rec["attempt_index"] == 1
    assert rec["tokens_in"] == 12076595
    assert rec["tokens_out"] == 209680
    assert rec["cost_usd"] == 27.990277
    assert rec["trajectory_available"] is True
    assert rec["error_class"] is None
    # wall clock: 2026-09-01T16:53:42 -> 18:15:03 = 4881.1 s
    assert abs(rec["wall_seconds"] - 4881.104) < 0.01


def test_parse_trial_row_reward_fallback_and_nulls():
    trial = {
        "id": "t1",
        "job_id": "j1",
        "task_name": "terminal-bench/some-task",
        "reward": None,
        "evals": {"reward": {"metrics": [{"reward": 1.0}]}},
        "status": "completed",
        "archive_path": None,
    }
    rec = parse_trial_row(
        trial,
        bench_version="4.0",
        leaderboard_id="lb",
        leaderboard_row_id=None,
        harvested_at="h",
    )
    assert rec["reward"] == 1.0
    assert rec["passed"] is True
    assert rec["trajectory_available"] is False
    assert rec["wall_seconds"] is None
    assert rec["cost_usd"] is None


def test_parse_leaderboard_row_new_schema():
    board = json.loads((FIXTURES / "leaderboard_read_4-0-0.json").read_text())
    rec = parse_leaderboard_row(
        board["rows"][0], bench_version="4.0", leaderboard_id=board["leaderboard"]["id"]
    )
    assert rec["agent"] == "Codex"
    assert rec["model"] == "GPT-6 Astra"
    assert rec["agent_org"] == "OpenAI"
    assert rec["rank"] == 1
    assert rec["accuracy"] == 58.18
    assert rec["n_trials"] == 330
    assert rec["reasoning_effort"] == "max"


def test_parse_leaderboard_row_2_0_schema():
    row = json.loads((FIXTURES / "leaderboard_row_2-0.json").read_text())
    rec = parse_leaderboard_row(row, bench_version="2.0", leaderboard_id="lb20")
    assert rec["agent"] == "NexAU-AHE"
    assert rec["model"] == "GPT-5.5"
    assert rec["accuracy"] == 84.7191011236
    assert rec["n_trials"] == 0


def test_task_id_from_name():
    assert task_id_from_name("terminal-bench/foo-bar") == "foo-bar"
    assert task_id_from_name("bare-task") == "bare-task"
    assert task_id_from_name(None) is None


def test_assemble_from_raw_cache(tmp_path):
    """Round-trip: seed a raw cache, assemble, read the parquet back."""
    board = json.loads((FIXTURES / "leaderboard_read_4-0-0.json").read_text())
    row_id = board["rows"][0]["id"]
    trials = json.loads((FIXTURES / "job_trials_page.json").read_text())["items"]
    job_id = trials[0]["job_id"]

    raw = tmp_path / "raw"
    (raw / "row_trials").mkdir(parents=True)
    (raw / "jobs").mkdir(parents=True)
    (raw / "boards.json").write_text(json.dumps({"4.0": board["leaderboard"]}))
    (raw / "board_4.0.json").write_text(json.dumps(board))
    (raw / "board_4.0.trial_jobs.json").write_text(
        json.dumps({t["id"]: job_id for t in trials})
    )
    (raw / "row_trials" / f"{row_id}.json").write_text(
        json.dumps([{"trial_id": t["id"], "created_at": "x"} for t in trials])
    )
    (raw / "jobs" / f"{job_id}.overview.json").write_text(
        json.dumps(
            {
                "name": trials[0]["job_name"],
                "started_at": "2026-08-31T04:00:37+00:00",
                "n_total_trials": 3,
                "evals": {
                    "rows": [
                        {
                            "key": {
                                "agent": "codex",
                                "model": "gpt-6-astra",
                                "provider": "openai",
                                "agent_version": "0.151.0",
                            }
                        }
                    ]
                },
            }
        )
    )
    (raw / "jobs" / f"{job_id}.trials.json").write_text(json.dumps(trials))

    outs = assemble(tmp_path, log=lambda m: None)
    import pandas as pd

    tdf = pd.read_parquet(outs["trials"])
    assert len(tdf) == 3
    assert set(tdf["bench_version"]) == {"4.0"}
    assert tdf["task_id"].tolist() == [
        "live-database-cutover",
        "cargo-flight-dispatch",
        "legacy-utility-triage",
    ]
    jdf = pd.read_parquet(outs["jobs"])
    assert len(jdf) == 1
    assert jdf.iloc[0]["agent"] == "codex"
    assert jdf.iloc[0]["model"] == "gpt-6-astra"
    assert jdf.iloc[0]["n_trials"] == 3
    kdf = pd.read_parquet(outs["tasks"])
    assert len(kdf) == 3
    assert kdf["n_jobs"].tolist() == [1, 1, 1]
    rdf = pd.read_parquet(outs["leaderboard_rows"])
    assert len(rdf) == 2
    assert rdf.iloc[0]["agent"] == "Codex"
