# Proxy features — pre-SFT ranking signal

Generated 2026-09-17 20:33 UTC by `scripts/proxy_features_summary.py` from `outputs/proxy_features.parquet`.

Rows (trajectories) **511,668** across **3 harnesses** × **5 teachers** × **2 sources**; resolved = 1: 131,891, 0: 153,745, -1 (unknown): 226,032. Sections 2-3 use the **285,636** rows with resolved in (0, 1) (positive rate 0.462).

## 1. Means by harness / teacher

`resolved_rate` is computed over rows with resolved in (0, 1) only (`n_known`); the structural means use all rows of the combo (`n`).

| harness | teacher | n | n_known | n_assistant_turns | repeat_call_rate | n_edit_calls | n_test_calls | patch_file_jaccard | resolved_rate |
|---|---|---|---|---|---|---|---|---|---|
| minisweagent | qwen36_27b | 95,291 | 36,952 | 58.5 | 0.059 | 12.3 | 4.107 | 0.489 | 0.386 |
| minisweagent | qwen38_27b | 107,267 | 96,595 | 50.6 | 0.004 | 6.011 | 6.721 | 0.546 | 0.517 |
| openhands | deepseek_v4_flash | 21,208 | 0 | 45.1 | 0.045 | 5.407 | 6.324 | 0.450 | n/a |
| openhands | minimax_m25 | 43,603 | 33,873 | 58.5 | 0.099 | 8.501 | 7.930 | 0.522 | 0.424 |
| openhands | qwen35_122b | 40,463 | 30,957 | 93.3 | 0.086 | 18.4 | 12.4 | 0.440 | 0.318 |
| openhands | qwen36_27b | 59,422 | 0 | 86.6 | 0.079 | 17.2 | 10.7 | 0.456 | n/a |
| sweagent | minimax_m25 | 46,819 | 35,955 | 74.8 | 0.147 | 9.574 | 8.803 | 0.544 | 0.466 |
| sweagent | qwen35_122b | 20,334 | 15,098 | 130.6 | 0.099 | 18.0 | 10.9 | 0.517 | 0.487 |
| sweagent | qwen36_27b | 77,261 | 36,206 | 87.7 | 0.102 | 17.5 | 9.939 | 0.494 | 0.536 |

Excluded: harness/teacher combos with resolved = -1 for **all** rows (no known outcome, so no resolved rate; they contribute no rows to sections 2-3): `openhands/deepseek_v4_flash` (n=21,208), `openhands/qwen36_27b` (n=59,422).

## 2. Pearson correlation with `resolved` (rows with resolved in (0, 1))

`pearson` is the point-biserial correlation with the 0/1 outcome; `auc` is the univariate ROC AUC (0.5 = no signal). `n/a` marks a feature that is constant on this subset.

| feature | n | pearson | auc |
|---|---|---|---|
| patch_file_jaccard | 285,636 | +0.2289 | 0.6368 |
| n_distinct_tool_commands | 285,636 | -0.1760 | 0.3905 |
| n_messages | 285,636 | -0.1722 | 0.3928 |
| n_assistant_turns | 285,636 | -0.1719 | 0.3933 |
| n_tool_calls | 285,636 | -0.1709 | 0.3937 |
| tool_obs_chars | 285,636 | -0.1612 | 0.3964 |
| n_edit_calls | 285,636 | -0.1415 | 0.4162 |
| gold_patch_files | 285,636 | -0.1252 | 0.3751 |
| n_test_calls | 285,636 | -0.0909 | 0.4565 |
| reasoning_chars | 285,636 | -0.0753 | 0.4969 |
| assistant_chars | 285,636 | -0.0730 | 0.4373 |
| gold_patch_lines | 285,636 | -0.0488 | 0.3365 |
| model_patch_files | 285,636 | -0.0182 | 0.4017 |
| repeat_call_rate | 285,636 | -0.0104 | 0.4862 |
| model_patch_lines | 285,636 | -0.0068 | 0.3485 |
| max_consecutive_identical_calls | 285,636 | -0.0040 | 0.4988 |
| ends_with_submit | 285,636 | +0.0025 | 0.5000 |

## 3. Standardized logistic regression (80/20 split grouped by instance_id)

`LogisticRegression` (scikit-learn) on all 17 features, inputs standardized (mean 0, sd 1); the scaler and model are fit on the 80% train split only. The split is grouped by `instance_id`, so trajectories of the same task instance (which can appear under several harness/teacher combos) never cross the train/test boundary. `test_size=0.2`, `random_state=42`. Features are winsorized to the train split's 1st/99th percentile before standardizing (45,865 train values clipped): the raw patch-size tail (max 34,209,809 modified lines, 217 rows above 100k) otherwise puts a -10.28 coefficient on `model_patch_lines` and scores held-out AUC 0.6860.

- Train: 228,483 rows over 31,521 instance_ids (105,711 positives).
- Test (20%): 57,153 rows over 7,881 instance_ids (26,180 positives).
- **AUC (held out): 0.7024** (train 0.7057).

5 largest |std coef| (log-odds change per 1 sd of the feature):

| rank | feature | std coef | direction |
|---|---|---|---|
| 1 | n_tool_calls | +0.7782 | higher → more likely resolved |
| 2 | n_assistant_turns | -0.6764 | higher → less likely resolved |
| 3 | model_patch_lines | -0.4294 | higher → less likely resolved |
| 4 | patch_file_jaccard | +0.3558 | higher → more likely resolved |
| 5 | gold_patch_lines | -0.3363 | higher → less likely resolved |

## Notes

- `model_patch_lines` / `gold_patch_lines` come from patch metadata and have a heavy tail. Winsorizing (section 3) improves the held-out AUC and keeps the coefficients interpretable, but treat the model patch as noisy: it is self-reported by the model and missing or degenerate for some runs.
- `ends_with_submit` is near-constant (~1.0: published trajectories are truncated at each harness's terminal `submit`/`finish` action), so it carries almost no signal; `reasoning_chars` is near-zero for most harnesses.
- The count features (`n_messages`, `n_assistant_turns`, `n_tool_calls`, `n_distinct_tool_commands`) are near-collinear (pairwise r ≈ 0.95-0.99), so their individual coefficients split into offsetting pairs (e.g. `n_tool_calls` positive with `n_assistant_turns` negative); read them jointly as "longer trajectories resolve less", not as independent effects.
- Edit/test counts are lexical: they match command patterns in tool arguments (`$.command`), so they miss edits expressed through other fields and can be triggered by heredoc bodies that contain test command text.
- Correlations are pooled over harnesses, teachers and sources and are associational; harness composition (e.g. a low openhands resolved rate) can confound pooled numbers.
- Reproduce: `uv run python scripts/proxy_features.py` (per-shard parts under `outputs/proxy_features_parts/`, merged into `outputs/proxy_features.parquet`), then `uv run python scripts/proxy_features_summary.py`.
