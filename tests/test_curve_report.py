"""Synthetic-fixture tests for the curve run report (no real out dirs required)."""

from __future__ import annotations

import json
import math
from pathlib import Path

from openswe_traces.curve_report import (
    ARMS,
    common_slope,
    efficiency_table,
    eval_table,
    figure,
    load_manifests,
    load_run,
    loss_table,
    markdown_tables,
    pairwise_table,
    parse_encode_lines,
    slope_table,
    sup_tokens_seen,
)

WINDOWS = [1000, 2000, 4000, 5000]
STEPS = [10, 20, 40, 50]
SLOPE = -0.05  # CE = ce0 + SLOPE * ln(windows)


def write_arm(root: Path, arm: str, ce0: float, *, n_evals: int = 4) -> None:
    out = root / "arms" / arm / "out"
    out.mkdir(parents=True)
    evals = [
        {
            "step": STEPS[i],
            "windows_seen": WINDOWS[i],
            "eval_ce": ce0 + SLOPE * math.log(WINDOWS[i]),
            "event": "final_full" if i == 3 else "windows",
        }
        for i in range(n_evals)
    ]
    log = [
        {
            "step": step,
            "windows_seen": step * 100,
            "loss": 1.0 - step / 100,
            "tok_per_s": 300.0,
            "sup_tok_per_s": 50.0,
            "elapsed_s": step * 10,
            "mem_gb": 11.5,
        }
        for step in STEPS
    ]
    metrics = {
        "model": "test-model",
        "manifest": arm,
        "world_size": 2,
        "steps_done": 50,
        "windows_seen": 5000,
        "elapsed_s": 500,
        "tok_per_s": 600.0,
        "mem_gb": 11.5,
        "stop_reason": "stop_after_seconds",
        "n_traces": 40,
        "log": log,
        "eval": evals,
    }
    (out / "metrics.json").write_text(json.dumps(metrics))
    entries = [
        {"data": f"[rank 0] encoded 19/20 traces from {arm}.jsonl in 1s\n"},
        {"data": f"[rank 1] encoded 18/20 traces from {arm}.jsonl in 1s\n"},
    ]
    (out / f"openswe-curve-{arm}.log").write_text(json.dumps(entries))


def write_summary(root: Path) -> None:
    data = root / "data"
    data.mkdir(parents=True, exist_ok=True)
    arms = {
        arm: {
            "n_traces": 40,
            "n_tasks": 7,
            "supervised_chars": 2_000_000,
            "per_bucket": {"easy": 10, "hard": 10, "mid": 20},
            "per_language": {"python": 40},
            "per_teacher": {"t1": 40},
        }
        for arm in ARMS
    }
    (data / "manifests_summary.json").write_text(
        json.dumps({"budget_chars": 2_000_000, "arms": arms})
    )


def make_runs(tmp_path: Path, ce0: dict[str, float] | None = None) -> dict[str, dict]:
    ce0 = ce0 or {arm: 1.0 + 0.01 * i for i, arm in enumerate(ARMS)}
    for arm in ARMS:
        write_arm(tmp_path, arm, ce0[arm])
    write_summary(tmp_path)
    return {arm: load_run(arm, tmp_path) for arm in ARMS}


def test_load_run_missing_arm_returns_none(tmp_path):
    assert load_run("random", tmp_path) is None


def test_parse_encode_lines(tmp_path):
    runs = make_runs(tmp_path)
    hits, dropped = parse_encode_lines(
        tmp_path / "arms" / "random" / "out" / "openswe-curve-random.log"
    )
    assert hits == [(0, 19, 20), (1, 18, 20)]
    assert dropped == 3
    assert runs["random"]["dropped"] == 3


def test_eval_table_alignment(tmp_path):
    table = eval_table(make_runs(tmp_path))
    assert list(table["windows_seen"]) == [1000, 2000, 4000, 5000]
    assert set(table.columns) == {"event", "windows_seen", *ARMS}
    assert abs(table["random"][2] - (1.0 + SLOPE * math.log(4000))) < 1e-12


def test_slope_table_recovers_exact_slope(tmp_path):
    table = slope_table(make_runs(tmp_path))
    for arm in ARMS:
        assert abs(table.loc[arm, "b_per_ln"] - SLOPE) < 1e-9
        assert abs(table.loc[arm, "drop_per_doubling"] - (-SLOPE * math.log(2))) < 1e-9
        assert table.loc[arm, "resid_sd"] < 1e-9


def test_common_slope_pools_to_zero_residual_on_exact_lines(tmp_path):
    noise = common_slope(make_runs(tmp_path))
    assert abs(noise["slope_per_ln"] - SLOPE) < 1e-9
    assert noise["resid_sd"] < 1e-9
    assert noise["dof"] == 4 * 3 - 4


def test_loss_table_picks_every_20_steps(tmp_path):
    table = loss_table(make_runs(tmp_path))
    assert list(table.index) == [20, 40, 60, 80, 100, 120, 140, 160, 180]
    assert abs(table["random"].loc[20] - 0.8) < 1e-12
    assert math.isnan(table["random"].loc[60])


def test_sup_tokens_seen_is_global_and_cumulative(tmp_path):
    runs = make_runs(tmp_path)
    # per-rank rate 50 * elapsed 200 * world 2 -> 20k global supervised tokens by step 20
    assert sup_tokens_seen(runs["random"], 20) == 20_000.0
    eff = efficiency_table(runs)
    row = eff[(eff["arm"] == "random") & (eff["step"] == 20)].iloc[0]
    assert row["sup_tokens_seen"] == 20_000.0


def test_pairwise_deltas(tmp_path):
    ce0 = {
        "random": 1.0,
        "top_within_task": 1.05,
        "bottom_within_task": 0.98,
        "random_masked": 1.03,
    }
    table = pairwise_table(make_runs(tmp_path, ce0))
    assert abs(table.loc["top-random"].iloc[0] - 0.05) < 1e-12
    assert abs(table.loc["random-bottom"].iloc[0] - 0.02) < 1e-12
    assert abs(table.loc["masked-random"].iloc[0] - 0.03) < 1e-12


def test_eval_table_uses_shortest_arm(tmp_path):
    ce0 = {arm: 1.0 for arm in ARMS}
    for arm in ARMS:
        write_arm(tmp_path, arm, ce0[arm], n_evals=3 if arm == "random" else 4)
    runs = {arm: load_run(arm, tmp_path) for arm in ARMS}
    assert len(eval_table(runs)) == 3


def test_figure_written(tmp_path):
    path = figure(make_runs(tmp_path), tmp_path / "fig.png")
    assert path.exists() and path.stat().st_size > 0


def test_markdown_tables_contains_every_section(tmp_path):
    runs = make_runs(tmp_path)
    text = markdown_tables(runs, load_manifests(tmp_path))
    for section in (
        "### Run config",
        "### Train loss",
        "### Eval CE",
        "### Slopes",
        "### Pairwise",
    ):
        assert section in text
    assert "traces (manifest)" in text
    assert "windows_seen" in text
