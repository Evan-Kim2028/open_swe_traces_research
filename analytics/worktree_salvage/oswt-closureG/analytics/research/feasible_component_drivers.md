# Feasible-component drivers: verifier and specification terms

Generated 2026-09-19 18:05 UTC by `scripts/feasible_drivers.py` (no solver calls). Inputs: `outputs/closure_proxies.parquet` (closure-B), `outputs/patch_split.parquet` + `outputs/rung_features.parquet` + `outputs/trajectory_frame.parquet` (closure-C), `traces_external/*/meta.parquet` (upstream metadata). Log: `outputs/closure_G.log`.

Follow-up to the mixture finding in `task_space_framework.md`: size moves the feasible share, not the feasible pass probability. Here: what moves p1 inside the feasible component, and what predicts the infeasible class at fixed size.

## Data and join coverage

Merged frame: **42,413** instances (38,294 with n_labeled >= 3). Verifier terms: `test_added`, `n_test_files`, `n_new_test_funcs` (test-function defs added to the gold patch, per-language regex), `test_only` (patch touches no src lines), and upstream `n_f2p`/`n_p2p`/`test_patch_chars` where joined. Specification terms: `issue_chars` (first user message length), `spec_density = issue_chars / max(1, src_added)`, `has_repro`, `has_expected_actual`, `rung_binary` (heuristic rung >= 2).

| hf_dataset_name | corpus inst | meta rows | matched | coverage |
|---|---|---|---|---|
| AweAI-Team/Scale-SWE | 19,448 | 20,181 | 19,448 | 100.0% |
| nebius/SWE-rebench-V2 | 22,965 | 32,079 | 22,965 | 100.0% |

Both corpus sources are public on HF and join at 100%. Caveat: `test_patch` is a unified diff for SWE-rebench-V2 but the raw fail-to-pass test script (`f2p_script`) for Scale-SWE, and Scale-SWE has no `created_at` (kept null). `test_patch_chars` is therefore a size proxy, not a like-for-like diff measure.

## a. Mixture baseline (per size decile, n_labeled >= 3)

Two-component binomial EM per decile of `log1p(added_lines)`, all parameters free: component 1 is feasible with pass probability p1; component 0 is infeasible with a small residual p0 (rare lucky resolves / label noise); π is the feasible share.

| decile | log1p(added) | n | π feasible | p0 infeasible | p1 feasible pass | raw solve rate |
|---|---|---|---|---|---|---|
| 1 | 0.0–1.9 | 4,477 | 0.62 | 0.09 | 0.86 | 0.63 |
| 2 | 1.9–2.5 | 3,595 | 0.61 | 0.10 | 0.87 | 0.60 |
| 3 | 2.5–2.9 | 3,827 | 0.59 | 0.07 | 0.86 | 0.56 |
| 4 | 2.9–3.3 | 3,792 | 0.57 | 0.07 | 0.85 | 0.53 |
| 5 | 3.3–3.7 | 3,535 | 0.53 | 0.06 | 0.83 | 0.48 |
| 6 | 3.7–4.0 | 3,925 | 0.49 | 0.05 | 0.81 | 0.43 |
| 7 | 4.0–4.4 | 3,662 | 0.43 | 0.04 | 0.80 | 0.38 |
| 8 | 4.4–4.8 | 3,863 | 0.38 | 0.03 | 0.77 | 0.32 |
| 9 | 4.8–5.4 | 3,790 | 0.31 | 0.02 | 0.74 | 0.25 |
| 10 | 5.4–11.1 | 3,828 | 0.24 | 0.02 | 0.74 | 0.20 |

Baseline reproduces the framework-note shape: π falls 0.62 → 0.24 across deciles while p1 stays in [0.74, 0.87].

## b. What predicts the infeasible class at fixed size?

Logistic of `n_resolved == 0` on 38,294 instances (base rate 0.418); grouped 80/20 split by repo (train 30,661, held-out 7,633 unseen-repo instances). Numeric features winsorized 1/99 and standardized; language and hf_dataset_name one-hots (majority level dropped). AUC is held-out.

| feature set | held-out AUC |
|---|---|
| size only | 0.6588 |
| size + verifier | 0.7080 |
| size + spec | 0.6601 |
| full | 0.7087 |
| full − verifier | 0.6727 |
| full − spec | 0.7069 |
| full − size | 0.7094 |

Standardized coefficients of the full model (log-odds per 1 sd):

| rank | feature | std coef |
|---|---|---|
| 1 | log1p(src_added) | +0.569 |
| 2 | log1p(test_patch_chars) | +0.368 |
| 3 | log1p(n_f2p) | +0.313 |
| 4 | log1p(n_p2p) | +0.287 |
| 5 | log1p(n_test_files) | +0.240 |
| 6 | test_only | +0.208 |
| 7 | log1p(issue_chars) | -0.166 |
| 8 | language_rust | +0.156 |
| 9 | rung_binary | -0.105 |
| 10 | log1p(spec_density) | +0.097 |
| 11 | language_java | +0.079 |
| 12 | language_go | -0.057 |
| 13 | has_expected_actual | +0.045 |
| 14 | log1p(src_removed) | +0.044 |
| 15 | log1p(n_src_files) | +0.039 |
| 16 | log1p(n_new_test_funcs) | +0.026 |
| 17 | log1p(test_added) | +0.017 |
| 18 | language_javascript | -0.013 |
| 19 | language_typescript | -0.007 |
| 20 | log1p(test_removed) | +0.007 |

Repo fixed effects (FWL-demeaned linear probability model, all data; coefficients are per-1-sd of the within-repo-demeaned feature):

| feature | repo-FE LPM coef |
|---|---|
| log1p(src_added) | +0.1014 |
| log1p(test_patch_chars) | +0.0561 |
| log1p(n_f2p) | +0.0436 |
| test_only | +0.0343 |
| log1p(n_p2p) | +0.0281 |
| log1p(issue_chars) | -0.0195 |
| log1p(n_test_files) | +0.0193 |
| rung_binary | -0.0127 |
| log1p(n_new_test_funcs) | +0.0118 |
| log1p(test_added) | +0.0098 |
| has_repro | -0.0063 |
| has_expected_actual | +0.0052 |
| log1p(test_removed) | +0.0051 |
| log1p(n_src_files) | -0.0034 |
| log1p(src_removed) | -0.0025 |

## c. What moves p1 inside the feasible component?

WLS of solve_rate on 21,057 feasible instances (n_resolved >= 1, n_labeled >= 5; weight = n_labeled), grouped 80/20 by repo. Held-out R² by block.

| feature set | held-out R² |
|---|---|
| size only | 0.0652 |
| size + verifier | 0.1002 |
| size + spec | 0.0788 |
| full | 0.1285 |
| full − verifier | 0.1064 |
| full − spec | 0.1270 |
| full − size | 0.1270 |

Standardized coefficients of the full model (solve_rate per 1 sd):

| rank | feature | std coef |
|---|---|---|
| 1 | log1p(src_added) | -0.0530 |
| 2 | language_typescript | -0.0382 |
| 3 | log1p(test_patch_chars) | -0.0368 |
| 4 | language_javascript | -0.0225 |
| 5 | test_only | -0.0205 |
| 6 | language_rust | -0.0189 |
| 7 | language_go | -0.0177 |
| 8 | log1p(test_removed) | -0.0161 |
| 9 | log1p(n_p2p) | -0.0160 |
| 10 | language_java | -0.0091 |
| 11 | rung_binary | +0.0086 |
| 12 | has_repro | +0.0079 |
| 13 | log1p(n_f2p) | -0.0079 |
| 14 | log1p(spec_density) | +0.0071 |
| 15 | log1p(n_test_files) | +0.0067 |
| 16 | log1p(n_new_test_funcs) | +0.0065 |
| 17 | has_expected_actual | -0.0065 |
| 18 | log1p(issue_chars) | -0.0064 |
| 19 | log1p(n_src_files) | -0.0058 |
| 20 | language_php | +0.0052 |

## d. Per-combo feasible difficulty

Per-(instance, combo) solve rates from `trajectory_frame.parquet` over labeled rollouts; feasible set per combo = >= 3 labeled rollouts in that combo and >= 1 resolved. Combo ordering below is by observed rate — the IRT fit's rank agreement with observed rate is Spearman +1.0 (`irt_summary.md`), so observed-rate order = theta order.

| combo | n labeled rollouts | observed rate |
|---|---|---|
| sweagent/qwen36_27b | 36,206 | 0.536 |
| minisweagent/qwen38_27b | 96,595 | 0.517 |
| sweagent/qwen35_122b | 15,098 | 0.487 |
| sweagent/minimax_m25 | 35,955 | 0.466 |
| openhands/minimax_m25 | 33,873 | 0.424 |
| minisweagent/qwen36_27b | 85,853 | 0.375 |
| openhands/qwen35_122b | 30,957 | 0.318 |

| combo | n feasible | R² size | R² full | Δ | coef n_f2p | coef spec_density |
|---|---|---|---|---|---|---|
| sweagent/qwen36_27b | 5,223 | 0.007 | 0.041 | 0.034 | -0.014 | 0.005 |
| minisweagent/qwen38_27b | 14,961 | 0.021 | 0.048 | 0.027 | -0.011 | -0.018 |
| sweagent/qwen35_122b | 1,512 | 0.022 | 0.043 | 0.021 | -0.004 | -0.068 |
| sweagent/minimax_m25 | 4,952 | 0.018 | 0.031 | 0.012 | -0.002 | -0.054 |
| openhands/minimax_m25 | 3,982 | 0.006 | 0.025 | 0.019 | -0.002 | 0.008 |
| minisweagent/qwen36_27b | 10,110 | 0.028 | 0.088 | 0.060 | -0.009 | 0.074 |
| openhands/qwen35_122b | 2,525 | 0.023 | 0.084 | 0.061 | -0.003 | 0.006 |

## e. Verdicts

**Does the verifier term move p1?** Verifier block (test lines/files/new test funcs/test_only + upstream n_f2p/n_p2p/test_patch_chars): removing it from the feasible WLS changes held-out R² 0.1285 → 0.1064 (Δ -0.0221); log1p(n_f2p) standardized coef -0.0079. Per-combo ΔR² (full − size): sweagent/qwen36_27b 0.034, minisweagent/qwen38_27b 0.027, sweagent/qwen35_122b 0.021, sweagent/minimax_m25 0.012, openhands/minimax_m25 0.019, minisweagent/qwen36_27b 0.060, openhands/qwen35_122b 0.061.

**Does specification density move p1?** Removing the spec block changes held-out R² 0.1285 → 0.1270 (Δ -0.0015); log1p(spec_density) coef +0.0071.

**Does either predict the feasibility gate beyond size?** Held-out AUC of all_fail: size 0.6588 → +verifier 0.7080, +spec 0.6601, full 0.7087.

