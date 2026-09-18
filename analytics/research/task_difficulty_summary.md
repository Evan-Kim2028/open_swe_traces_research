# Task difficulty per instance — summary

Generated 2026-09-17 21:42 UTC by `scripts/task_difficulty.py` from `outputs/proxy_features.parquet`, `outputs/temporal_features.parquet`, `outputs/trace_scores.parquet`, and `outputs/task_text.parquet` (streamed from `traces_data/`); per-instance difficulty in `outputs/task_difficulty.parquet`.

Trajectories **511,668**; instances **42,413**; labeled trajectories **285,636** (resolved rate 0.462); solve rates over 39,402 instances with at least one labeled rollout.

Task text: 42,413 instances in `outputs/task_text.parquet`; `issue_chars` present for 42,413/42,413 instances (0 instances had differing first-message lengths across shards — the max is kept; 0 had differing repos).

## 1. Difficulty distribution

Quantiles per instance; `solve_rate` is defined only where at least one rollout is labeled.

| series | n | mean | p10 | p25 | median | p75 | p90 | max |
|---|---|---|---|---|---|---|---|---|
| n_labeled | 42,413 | 6.73 | 1 | 4 | 7 | 9 | 11 | 15 |
| solve_rate | 39,402 | 0.417 | 0.000 | 0.000 | 0.286 | 0.875 | 1.000 | 1.000 |

| n_labeled | instances | share of instances |
|---|---|---|
| 0 | 3,011 | 7.1% |
| 1 | 1,254 | 3.0% |
| 2 | 2,133 | 5.0% |
| 3–5 | 7,410 | 17.5% |
| 6–10 | 22,682 | 53.5% |
| 11–20 | 5,923 | 14.0% |
| 21+ | 0 | 0.0% |

Solve rate is a discrete grid (n_resolved / n_labeled): exactly 0 for 17,053 instances (43.3% of defined) and exactly 1 for 8,703 (22.1%).

| bucket | instances | share of instances | trajectories | share of trajectories | mean n_rollouts | mean solve_rate |
|---|---|---|---|---|---|---|
| all_fail | 14,205 | 33.5% | 171,234 | 33.5% | 12.1 | 0.000 |
| hard | 3,505 | 8.3% | 45,264 | 8.8% | 12.9 | 0.214 |
| mid | 3,602 | 8.5% | 47,026 | 9.2% | 13.1 | 0.505 |
| easy | 6,407 | 15.1% | 84,275 | 16.5% | 13.2 | 0.794 |
| all_pass | 8,296 | 19.6% | 99,132 | 19.4% | 11.9 | 1.000 |
| unknown | 6,398 | 15.1% | 64,737 | 12.7% | 10.1 | 0.140 |

## 2. Task-only predictor of solve_rate

WLS on 39,402 instances (weight = n_labeled, total weight 285,636), grouped 80/20 split by repo: 31,482 train instances over 2,978 repos, 7,920 test instances over 745 repos. Numeric inputs are winsorized to the train split's 1st/99th percentile then standardized; language/category are one-hot with the most frequent level dropped; harness mix drops `minisweagent` as the baseline fraction. R² is the unweighted fit metric; the fit itself is weighted.

| split | R² | Spearman(pred, actual) |
|---|---|---|
| train | 0.1090 | +0.3686 |
| test (held out) | 0.1025 | +0.3542 |

Top standardized coefficients (largest |coef|, solve_rate per 1 sd):

| rank | feature | std coef | direction |
|---|---|---|---|
| 1 | gold_patch_lines | -0.0670 | lower solve_rate |
| 2 | gold_patch_files | -0.0596 | lower solve_rate |
| 3 | language_typescript | -0.0519 | lower solve_rate |
| 4 | language_go | -0.0466 | lower solve_rate |
| 5 | language_rust | -0.0459 | lower solve_rate |
| 6 | language_javascript | -0.0412 | lower solve_rate |
| 7 | language_java | -0.0302 | lower solve_rate |
| 8 | issue_chars | -0.0284 | lower solve_rate |
| 9 | category_feature-request | -0.0252 | lower solve_rate |
| 10 | category_enhancement | -0.0192 | lower solve_rate |
| 11 | frac_rollouts_openhands | +0.0183 | higher solve_rate |
| 12 | language_php | -0.0169 | lower solve_rate |

## 3. Trace classifier decomposition: task difficulty vs trace behavior

The trace classifier is the 27-feature proxy + temporal union from `score.py` under its grouped 80/20 split by instance_id (seed 42); on all 285,636 labeled rows it reproduces the published held-out AUC **0.7115**. The decomposition reuses that split but restricts to the 284,382 labeled trajectories whose instance has at least one other labeled trajectory, so the leave-one-out task rate is defined (227,495 train / 56,887 test).

| predictor | n features | held-out AUC | Δ vs trace alone |
|---|---|---|---|
| task solve_rate alone (leave-one-out) | 1 | 0.9361 | +0.2247 |
| trace features alone (same subset) | 27 | 0.7115 | — |
| task + trace | 28 | 0.9484 | +0.2370 |
| reference: trace features alone, all labeled rows | 27 | 0.7115 | — |

Leave-one-out task rate alone scores AUC 0.9383 over all 284,382 defined rows (no fitting, so held-out by construction); the standardized leave-one-out coefficient in the combined model is +2.6709 (log-odds per 1 sd).

### Within-task residual (resolved − leave-one-out solve_rate)

Linear fit of the residual on the 27 trace features on the same split (284,382 rows; 227,495 train / 56,887 test): held-out R² 0.0623, Spearman(pred, residual) +0.2274.

Top feature associations with the residual (Spearman, all rows):

| rank | feature | Spearman with residual | direction | n |
|---|---|---|---|---|
| 1 | patch_file_jaccard | +0.1524 | beats siblings | 284,382 |
| 2 | reasoning_chars | +0.1391 | beats siblings | 284,382 |
| 3 | n_edit_calls | -0.1041 | loses to siblings | 284,382 |
| 4 | repeat_call_rate | -0.1019 | loses to siblings | 284,382 |
| 5 | n_assistant_turns | -0.0858 | loses to siblings | 284,382 |
| 6 | model_patch_lines | -0.0852 | loses to siblings | 284,382 |
| 7 | frac_calls_after_first_gold_edit | -0.0803 | loses to siblings | 284,382 |
| 8 | n_messages | -0.0752 | loses to siblings | 284,382 |

### Classifier score vs difficulty buckets (labeled rows)

| bucket | n labeled | actual solve rate | mean p_resolved | gap |
|---|---|---|---|---|
| all_fail | 99,625 | 0.000 | 0.395 | +0.395 |
| hard | 29,168 | 0.206 | 0.427 | +0.221 |
| mid | 30,854 | 0.506 | 0.449 | -0.058 |
| easy | 55,360 | 0.801 | 0.506 | -0.295 |
| all_pass | 65,109 | 1.000 | 0.551 | -0.449 |
| unknown | 5,520 | 0.144 | 0.431 | +0.287 |

## 4. Learnability strip: dropping all_pass + all_fail tasks

Dropping both degenerate buckets removes **270,366** trajectories (52.8%) and **521.0M** estimated tokens (50.5% of assistant chars; tokens ≈ chars / 4).

| harness/teacher | n | dropped n | dropped % | chars (M) | dropped chars (M) | dropped chars % | dropped tokens (M) |
|---|---|---|---|---|---|---|---|
| minisweagent/qwen36_27b | 95,291 | 50,160 | 52.6% | 719.3 | 367.4 | 51.1% | 91.8 |
| minisweagent/qwen38_27b | 107,267 | 58,660 | 54.7% | 763.4 | 410.5 | 53.8% | 102.6 |
| openhands/deepseek_v4_flash | 21,208 | 12,455 | 58.7% | 0.5 | 0.2 | 53.7% | 0.1 |
| openhands/minimax_m25 | 43,603 | 22,472 | 51.5% | 123.6 | 61.5 | 49.7% | 15.4 |
| openhands/qwen35_122b | 40,463 | 19,788 | 48.9% | 391.2 | 187.6 | 47.9% | 46.9 |
| openhands/qwen36_27b | 59,422 | 31,815 | 53.5% | 476.0 | 248.3 | 52.2% | 62.1 |
| sweagent/minimax_m25 | 46,819 | 23,484 | 50.2% | 197.5 | 96.1 | 48.7% | 24.0 |
| sweagent/qwen35_122b | 20,334 | 9,650 | 47.5% | 710.4 | 323.1 | 45.5% | 80.8 |
| sweagent/qwen36_27b | 77,261 | 41,882 | 54.2% | 746.9 | 389.3 | 52.1% | 97.3 |
| TOTAL | 511,668 | 270,366 | 52.8% | 4,129.0 | 2,084.0 | 50.5% | 521.0 |

## Notes

- `difficulty_bucket` needs at least 3 labeled rollouts; the boundaries are all_fail (solve_rate == 0), hard (< 0.34), mid (0.34–0.66), easy (> 0.66), all_pass (== 1); everything else is `unknown`.
- `n_labeled` counts rollouts with `resolved in (0, 1)`; rollouts with `resolved == -1` never enter solve_rate.
- The leave-one-out task rate for a trajectory is the solve rate of its instance excluding the trajectory itself; it is defined only for labeled trajectories whose instance has at least one other labeled trajectory.
- The leave-one-out task rate is near-deterministic on degenerate tasks (all_fail / all_pass), so its standalone AUC is a ceiling on task-level information rather than a deployable signal for unseen instances; the within-task residual section is the behavior-only view on mixed tasks.
- `assistant_chars` is near zero for `openhands/deepseek_v4_flash` (median 0; its assistant text lives in reasoning fields), so that combo's token column understates its true size.
- `issue_chars` is `len(messages[2].content)` (the first user message, i.e. the PR description) from one streaming pass over the corpus; where shards disagree the maximum is kept.
- `language` is normalized (`ts` → `typescript`, `js` → `javascript`); instances whose harnesses disagree on the language label take the modal value.
- Reproduce: `uv run python scripts/task_difficulty.py` (add `--stream-only` to run just the corpus pass, `--skip-stream` to reuse `outputs/task_text.parquet`).
