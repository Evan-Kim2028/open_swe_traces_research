"""Synthetic-frame tests for the curve manifest builder (no corpus required)."""

from __future__ import annotations

import random

import pandas as pd

from openswe_traces.sft.manifests import (
    ARMS,
    ELIGIBLE_BUCKETS,
    gold_paths_from_patch,
    language_shares,
    select_arm,
    select_eval,
    select_eval_small,
    step_masks,
)

BUDGET = 2_000_000
LANG_WEIGHTS = {"python": 400, "go": 200, "rust": 100}


def make_candidates(seed: int = 7) -> pd.DataFrame:
    rng = random.Random(seed)
    rows = []
    iid = 0
    for lang, n in LANG_WEIGHTS.items():
        for k in range(n):
            iid += 1
            rows.append(
                {
                    "trajectory_id": f"{lang}-{k:04d}",
                    "instance_id": f"inst-{iid:05d}",
                    "harness": rng.choice(["minisweagent", "sweagent"]),
                    "teacher": rng.choice(["t1", "t2"]),
                    "language": lang,
                    "resolved": rng.choice([0, 1, 1]),
                    "n_assistant_turns": rng.randint(5, 40),
                    "supervised_chars": rng.randint(4_000, 12_000),
                    "p_resolved": round(rng.random(), 4),
                    "difficulty_bucket": rng.choice(list(ELIGIBLE_BUCKETS)),
                }
            )
    df = pd.DataFrame(rows)
    sib = df.iloc[::7].copy()
    sib["trajectory_id"] = sib["trajectory_id"] + "-sib"
    sib["p_resolved"] = (sib["p_resolved"] + 0.5).clip(upper=0.999)
    return pd.concat([df, sib], ignore_index=True)


def one_language(**overrides) -> pd.DataFrame:
    row = {
        "trajectory_id": "a",
        "instance_id": "i1",
        "harness": "h",
        "teacher": "t",
        "language": "python",
        "resolved": 1,
        "n_assistant_turns": 20,
        "supervised_chars": 1000,
        "p_resolved": 0.5,
        "difficulty_bucket": "hard",
    }
    rows = []
    for value in overrides.values():
        rows.append({**row, **value})
    return pd.DataFrame(rows)


def test_equal_budget_within_5pct():
    cand = make_candidates()
    shares = language_shares(cand)
    totals = {}
    for rule in ARMS:
        picked = select_arm(cand, rule, BUDGET, shares, seed=0)
        totals[rule] = int(picked["supervised_chars"].sum())
        assert 0.95 * BUDGET <= totals[rule] <= 1.05 * BUDGET, (rule, totals[rule])
    assert max(totals.values()) - min(totals.values()) <= 0.05 * BUDGET, totals


def test_language_stratification_matches_corpus_shares():
    cand = make_candidates()
    shares = language_shares(cand)
    for rule in ARMS:
        picked = select_arm(cand, rule, BUDGET, shares, seed=0)
        total = picked["supervised_chars"].sum()
        got = picked.groupby("language")["supervised_chars"].sum() / total
        for lang, share in shares.items():
            assert abs(float(got[lang]) - share) < 0.05, (rule, lang, float(got[lang]), share)


def test_masked_arm_uses_the_random_traces():
    cand = make_candidates()
    shares = language_shares(cand)
    rand = select_arm(cand, "random", BUDGET, shares, seed=0)
    masked = select_arm(cand, "random_masked", BUDGET, shares, seed=0)
    assert set(rand["trajectory_id"]) == set(masked["trajectory_id"])


def test_within_task_rules_pick_extremes_and_tie_break_by_turns():
    df = one_language(
        a={"trajectory_id": "a", "n_assistant_turns": 30, "p_resolved": 0.9},
        b={"trajectory_id": "b", "n_assistant_turns": 10, "p_resolved": 0.9},
        c={"trajectory_id": "c", "n_assistant_turns": 5, "p_resolved": 0.1},
        d={"trajectory_id": "d", "n_assistant_turns": 1, "p_resolved": 0.99, "resolved": -1},
    )
    top = select_arm(df, "top_within_task", 1000, {"python": 1.0}, seed=0)
    assert list(top["trajectory_id"]) == ["b"]
    bottom = select_arm(df, "bottom_within_task", 1000, {"python": 1.0}, seed=0)
    assert list(bottom["trajectory_id"]) == ["c"]


def test_within_task_rules_keep_one_representative_per_teacher():
    df = one_language(
        a={"trajectory_id": "a", "teacher": "t1", "p_resolved": 0.9},
        b={"trajectory_id": "b", "teacher": "t2", "p_resolved": 0.8},
        c={"trajectory_id": "c", "teacher": "t2", "p_resolved": 0.2, "resolved": 0},
        d={"trajectory_id": "d", "teacher": "t1", "p_resolved": 0.1, "resolved": 0},
    )
    top = select_arm(df, "top_within_task", 2000, {"python": 1.0}, seed=0)
    assert sorted(top["trajectory_id"]) == ["a", "b"]
    bottom = select_arm(df, "bottom_within_task", 2000, {"python": 1.0}, seed=0)
    assert sorted(bottom["trajectory_id"]) == ["c", "d"]


def test_random_arm_covers_both_teachers_before_duplicates():
    df = one_language(
        a={"trajectory_id": "a", "teacher": "t1"},
        b={"trajectory_id": "b", "teacher": "t2"},
        c={"trajectory_id": "c", "teacher": "t1"},
        d={"trajectory_id": "d", "teacher": "t2"},
    )
    picked = select_arm(df, "random", 2000, {"python": 1.0}, seed=0)
    assert len(picked) == 2
    assert picked["teacher"].nunique() == 2


def test_bucket_weights_prefer_mid_tasks():
    rows = [
        {
            "trajectory_id": f"{bucket}-{k:03d}",
            "instance_id": f"{bucket}-{k:03d}",
            "harness": "h",
            "teacher": "t",
            "language": "python",
            "resolved": 1,
            "n_assistant_turns": 10,
            "supervised_chars": 1_000_000,
            "p_resolved": 0.5,
            "difficulty_bucket": bucket,
        }
        for bucket in ELIGIBLE_BUCKETS
        for k in range(60)
    ]
    cand = pd.DataFrame(rows)
    picked = select_arm(cand, "random", 30_000_000, {"python": 1.0}, seed=0)
    counts = picked["difficulty_bucket"].value_counts()
    assert len(picked) == 30
    assert counts["mid"] > counts["hard"] and counts["mid"] > counts["easy"]


def test_eval_small_is_balanced_subset_of_full_eval():
    cand = make_candidates()
    shares = language_shares(cand)
    picks = {rule: select_arm(cand, rule, BUDGET, shares, seed=0) for rule in ARMS}
    used = set().union(*(set(df["instance_id"]) for df in picks.values()))
    ev = select_eval(cand, used, n=30, seed=0)
    small = select_eval_small(ev, n=12, seed=0)
    assert len(small) == 12
    assert set(small["trajectory_id"]) <= set(ev["trajectory_id"])
    assert set(small["difficulty_bucket"]) == set(ELIGIBLE_BUCKETS)
    assert small["trajectory_id"].is_unique


def test_eval_excludes_manifest_instances_one_per_instance_and_mixes_buckets():
    cand = make_candidates()
    shares = language_shares(cand)
    picks = {rule: select_arm(cand, rule, BUDGET, shares, seed=0) for rule in ARMS}
    used = set().union(*(set(df["instance_id"]) for df in picks.values()))
    ev = select_eval(cand, used, n=30, seed=0)
    assert len(ev) == 30
    assert ev["instance_id"].nunique() == 30
    assert not set(ev["instance_id"]) & used
    assert set(ev["difficulty_bucket"]) == set(ELIGIBLE_BUCKETS)


def test_gold_paths_from_patch():
    patch = "diff --git a/x/y.py b/x/y.py\n--- a/x/y.py\n+++ b/x/y.py\n"
    assert gold_paths_from_patch(patch) == ["x/y.py"]
    assert gold_paths_from_patch("+++ b/only/z.ts\n") == ["only/z.ts"]
    assert gold_paths_from_patch(None) == []


def test_step_masks_flags_repeat_error_and_outside_gold_edits():
    gold = ["src/app.py"]
    messages = [
        {"role": "system", "content": "sys"},
        {"role": "user", "content": "fix it"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [
                {"function": {"name": "bash", "arguments": '{"command": "pytest tests"}'}}
            ],
        },
        {"role": "tool", "content": "all good"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [
                {"function": {"name": "bash", "arguments": '{"command": "pytest tests"}'}}
            ],
        },
        {"role": "tool", "content": "fine"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [{"function": {"name": "bash", "arguments": '{"command": "ls"}'}}],
        },
        {"role": "tool", "content": "Traceback (most recent call last):"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [
                {
                    "function": {
                        "name": "bash",
                        "arguments": '{"command": "sed -i s/a/b/ src/other.py"}',
                    }
                }
            ],
        },
        {"role": "tool", "content": "done"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [
                {
                    "function": {
                        "name": "bash",
                        "arguments": '{"command": "sed -i s/a/b/ src/app.py"}',
                    }
                }
            ],
        },
        {"role": "tool", "content": "ok"},
        {
            "role": "assistant",
            "content": "",
            "tool_calls": [
                {"function": {"name": "bash", "arguments": '{"command": "go test ./..."}'}}
            ],
        },
        {"role": "tool", "content": "FAILED"},
    ]
    assert step_masks(messages, gold) == [False, True, True, True, False, True]
