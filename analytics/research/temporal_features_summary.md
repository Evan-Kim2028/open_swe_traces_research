# Temporal gold-file features — scoring summary

Generated 2026-09-17 21:19 UTC by `scripts/score_traces.py` from `outputs/proxy_features.parquet` + `outputs/temporal_features.parquet`; scores in `outputs/trace_scores.parquet`.

Rows (trajectories) **511,668**; labeled (resolved in (0, 1)) **285,636** (positive rate 0.462); unlabeled (resolved = -1) **226,032**.

## 1. Temporal features vs `resolved` (labeled rows)

`pearson` is the point-biserial correlation with the 0/1 outcome; `auc` is the univariate ROC AUC (0.5 = no signal); `mean` is the feature mean over labeled rows.

| feature | n | mean | pearson | auc |
|---|---|---|---|---|
| n_turns_total | 285,636 | 137.5 | -0.1722 | 0.3928 |
| n_tool_calls | 285,636 | 69.8 | -0.1709 | 0.3937 |
| gold_file_recall | 285,636 | 0.754 | +0.1525 | 0.5848 |
| n_gold_files_touched | 285,636 | 2.809 | -0.1288 | 0.4127 |
| n_gold_files | 285,636 | 4.694 | -0.1253 | 0.3749 |
| turn_first_gold_edit | 285,636 | 20.3 | -0.1219 | 0.4206 |
| turn_first_gold_view | 285,636 | 4.153 | -0.0705 | 0.4484 |
| n_edits_outside_gold | 285,636 | 6.552 | -0.0647 | 0.4676 |
| frac_first_gold_edit | 285,636 | 0.297 | -0.0469 | 0.4743 |
| frac_calls_after_first_gold_edit | 285,636 | 0.510 | -0.0466 | 0.4810 |
| frac_first_gold_view | 285,636 | 0.068 | +0.0341 | 0.5488 |

## 2. Held-out AUC before vs after adding temporal features

Same grouped 80/20 split for both models (grouped by `instance_id`, `test_size=0.2`, `random_state=42`): 228,483 train rows over 31,521 instance_ids, 57,153 test rows over 7,881 instance_ids. Inputs are winsorized to the train split's 1st/99th percentile before standardizing (proxy only: 45,865 clipped train values; union: 74,595).

| features | n features | train AUC | test AUC |
|---|---|---|---|
| proxy only | 17 | 0.7057 | 0.7024 |
| proxy + temporal | 27 | 0.7135 | 0.7115 |

**Δ test AUC (union − proxy) = +0.0091**.

The proxy-only test AUC reproduces the published `analytics/research/proxy_features_summary.md` value exactly (0.7024), so both rows share one split; the Δ is attributable to the added temporal columns.

## 3. Top coefficients (union model, standardized)

10 largest |std coef| from the proxy + temporal fit (log-odds change per 1 sd of the feature); `family` marks whether the feature comes from the proxy or temporal block.

| rank | feature | family | std coef | direction |
|---|---|---|---|---|
| 1 | n_tool_calls | proxy | +0.7507 | higher → more likely resolved |
| 2 | n_assistant_turns | proxy | -0.6313 | higher → less likely resolved |
| 3 | n_edit_calls | proxy | -0.4379 | higher → less likely resolved |
| 4 | model_patch_lines | proxy | -0.4192 | higher → less likely resolved |
| 5 | patch_file_jaccard | proxy | +0.3923 | higher → more likely resolved |
| 6 | n_edits_outside_gold | temporal | +0.3442 | higher → more likely resolved |
| 7 | gold_patch_lines | proxy | -0.3182 | higher → less likely resolved |
| 8 | n_distinct_tool_commands | proxy | -0.2823 | higher → less likely resolved |
| 9 | turn_first_gold_view | temporal | -0.2315 | higher → less likely resolved |
| 10 | frac_first_gold_edit | temporal | -0.2179 | higher → less likely resolved |

## 4. `p_resolved` for the unlabeled combos vs labeled resolved rate

| harness | teacher | n | n labeled | labeled resolved rate | mean p (labeled) | mean p (all) | share p > 0.5 |
|---|---|---|---|---|---|---|---|
| minisweagent | qwen36_27b | 95,291 | 36,952 | 0.386 | 0.470 | 0.488 | 0.498 |
| minisweagent | qwen38_27b | 107,267 | 96,595 | 0.517 | 0.496 | 0.489 | 0.520 |
| openhands | deepseek_v4_flash | 21,208 | 0 | n/a | n/a | 0.432 | 0.374 |
| openhands | minimax_m25 | 43,603 | 33,873 | 0.424 | 0.449 | 0.445 | 0.409 |
| openhands | qwen35_122b | 40,463 | 30,957 | 0.318 | 0.390 | 0.383 | 0.256 |
| openhands | qwen36_27b | 59,422 | 0 | n/a | n/a | 0.409 | 0.323 |
| sweagent | minimax_m25 | 46,819 | 35,955 | 0.466 | 0.473 | 0.467 | 0.440 |
| sweagent | qwen35_122b | 20,334 | 15,098 | 0.487 | 0.431 | 0.421 | 0.327 |
| sweagent | qwen36_27b | 77,261 | 36,206 | 0.536 | 0.437 | 0.427 | 0.361 |

Quantiles of `p_resolved` for the two fully unlabeled combos, next to the labeled rows as the reference distribution:

Per-combo calibration varies: mean `p_resolved` over labeled rows is close to the actual resolved rate for most combos (e.g. `minisweagent/qwen38_27b` 0.496 vs 0.517, `sweagent/minimax_m25` 0.473 vs 0.466) but overshoots for `minisweagent/qwen36_27b` (0.470 vs 0.386) and undershoots for `sweagent/qwen36_27b` (0.437 vs 0.536); the unlabeled scores inherit that pooled-model bias.

| combo | n | p10 | p25 | median | p75 | p90 | share p > 0.5 | actual |
|---|---|---|---|---|---|---|---|---|
| openhands/deepseek_v4_flash (unlabeled) | 21,208 | 0.184 | 0.330 | 0.451 | 0.551 | 0.647 | 0.374 | n/a (resolved unknown) |
| openhands/qwen36_27b (unlabeled) | 59,422 | 0.162 | 0.301 | 0.428 | 0.531 | 0.628 | 0.323 | n/a (resolved unknown) |
| all labeled rows | 285,636 | 0.220 | 0.349 | 0.477 | 0.591 | 0.685 | 0.447 | 0.462 (actual resolved rate) |

Overall mean `p_resolved`: 0.451 over all 511,668 rows, 0.462 over the 285,636 labeled rows (actual resolved rate 0.462).

## Notes

- `turn_first_gold_view` / `turn_first_gold_edit` are 1-based tool-call indices within the trajectory (-1 = never); the matching `frac_*` columns divide them by `n_tool_calls`.
- Null temporal fractions (no gold file ever mentioned/edited, or an empty gold set) are imputed with 0.0 for modeling (67,918 rows have at least one such null); the `turn_first_gold_view` / `turn_first_gold_edit` sentinel (-1) and `n_gold_files` keep the "never" information as their own feature values.
- `is_imputed` in `outputs/trace_scores.parquet` marks rows with an unknown label (`resolved == -1`) — it is a label flag, not a feature-imputation flag.
- `n_turns_total` (temporal) equals `n_messages` (proxy) for 100.0% of rows, so the union carries one duplicated feature; its coefficient splits with `n_messages` and is not interpretable on its own. `n_tool_calls` appears in both blocks but is counted once (proxy block) in the model.
- Mentions are case-sensitive substring tests on raw tool-call arguments, so the `/testbed/`-prefixed paths agents type match through the basename; heredoc bodies that merely quote a gold file path also count as a mention.
- Reproduce: `uv run python scripts/temporal_features.py --threads 6 --memory-limit 8GB` (parts under `outputs/temporal_features_parts/`, merged into `outputs/temporal_features.parquet`), then `uv run python scripts/score_traces.py`.
