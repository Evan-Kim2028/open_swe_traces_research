# Framework tests on Open-SWE-Traces: interaction, discrimination, mechanism

Generated 2026-09-19 19:06 UTC by `scripts/framework_tests.py` (zero solver cost).

Framework: a task's difficulty decomposes into a *tree term* (how much the surrounding code determines the change; proxied by the closure structure of the gold patch, `ratio` = internal_refs / boundary_refs), an *instruction term* (issue-text rung, binary `L<2` vs `L>=2` per the revised decision in `task_space_framework.md`), and a *capacity term* (environment/horizon, held fixed here). Per the sibling closure job, `log1p(added_lines)` is the dominant difficulty proxy on this corpus (ratio adds nothing on top of size), so every test reports `log1p(added_lines)` as the PRIMARY proxy with `ratio` and `new_frac` as SECONDARY variants in the same table. Each test runs on the top-3 teacher/harness combos by IRT ability (`theta_2pl`) and on all combos, side by side.

Inputs: per-trajectory frame streamed from `traces_data/` (only `patch_file_jaccard`, `resolved`, combo + identity columns); per-instance rung labels from `outputs/rung_features.parquet` (`scripts/rung_mapping.py --stream-only`); per-instance closure proxies from the sibling closure job (`outputs/closure_proxies.parquet`); IRT refit with the committed `openswe_traces.irt` 2PL model (the same model `scripts/fit_irt.py` fits) on the labeled rollouts.

Top-3 combos by theta_2pl:

| combo | theta_2pl |
|---|---|
| minisweagent/qwen38_27b | +0.100 |
| sweagent/qwen36_27b | +0.046 |
| sweagent/qwen35_122b | -0.287 |

T1 uses repo fixed effects; interaction CIs are 200 cluster bootstraps over instances (seed 42).

## T1 interaction — instruction benefit vs difficulty proxy

Per-trajectory logistic `resolved ~ rung_binary * z(proxy) + log1p(added_lines) + log1p(n_files) + language one-hots` with repo fixed effects, grouped 80/20 split by repo (seed 42). The interaction coefficient is the change in the rung benefit per 1 SD of the proxy.

| interaction (rung_binary x) | coef all | CI all | coef top-3 | CI top-3 | n all | n top-3 |
|---|---|---|---|---|---|---|
| size log1p(added_lines) | -0.041 | [-0.075, +0.015] | -0.051 | [-0.108, -0.013] | 334,537 | 147,899 |
| closure ratio | -0.074 | [-0.162, -0.007] | +0.042 | [-0.165, +0.077] | 334,537 | 147,899 |
| new_frac | -0.019 | [-0.078, +0.020] | -0.030 | [-0.089, +0.022] | 334,537 | 147,899 |

### 2x3 tables — rung_binary x tercile -> mean solve_rate

rung_binary x size (added_lines) tercile -> mean solve_rate:

| rung_binary | tercile | mean solve_rate all | n all | mean solve_rate top-3 | n top-3 |
|---|---|---|---|---|---|
| 0 | low | 0.556 | 59,747 | 0.600 | 23,562 |
| 0 | mid | 0.411 | 58,162 | 0.485 | 22,774 |
| 0 | high | 0.243 | 55,421 | 0.306 | 21,736 |
| 1 | low | 0.637 | 59,348 | 0.693 | 27,277 |
| 1 | mid | 0.505 | 53,195 | 0.603 | 27,197 |
| 1 | high | 0.300 | 48,664 | 0.376 | 25,353 |

rung_binary x closure ratio tercile -> mean solve_rate:

| rung_binary | tercile | mean solve_rate all | n all | mean solve_rate top-3 | n top-3 |
|---|---|---|---|---|---|
| 0 | low | 0.496 | 53,665 | 0.541 | 24,812 |
| 0 | mid | 0.376 | 63,628 | 0.424 | 22,182 |
| 0 | high | 0.357 | 56,037 | 0.428 | 21,078 |
| 1 | low | 0.594 | 58,524 | 0.647 | 30,466 |
| 1 | mid | 0.433 | 49,403 | 0.507 | 22,282 |
| 1 | high | 0.434 | 53,280 | 0.511 | 27,079 |

## T2 discrimination — a_2pl / b_2pl vs ratio, new_frac, size

Mixed tasks (n_labeled >= 3, 0 < solve_rate < 1): 14,254 of 38,294 (all) / 5,971 of 30,017 (top-3). OLS beta is per-1-SD of the feature.

| target | feature | rho all | beta all | rho top-3 | beta top-3 | n all | n top-3 |
|---|---|---|---|---|---|---|---|
| a_2pl | ratio | 0.015 | -0.000 | 0.027 | 0.001 | 14,254 | 5,971 |
| a_2pl | new_frac | -0.000 | -0.004 | 0.066 | 0.002 | 14,254 | 5,971 |
| a_2pl | log1p_added_lines | -0.072 | -0.027 | 0.022 | 0.002 | 14,254 | 5,971 |
| b_2pl | ratio | 0.061 | -0.003 | 0.030 | -0.006 | 14,254 | 5,971 |
| b_2pl | new_frac | 0.044 | -0.003 | 0.010 | -0.003 | 14,254 | 5,971 |
| b_2pl | log1p_added_lines | 0.195 | 0.144 | 0.114 | 0.052 | 14,254 | 5,971 |

## T3 mechanism — patch_file_jaccard among unresolved trajectories

Unresolved (`resolved == 0`) trajectories: 184,670 (all) / 71,208 (top-3).

| factor | tercile | mean jaccard all | n all | mean jaccard top-3 | n top-3 |
|---|---|---|---|---|---|
| added_lines (size, primary) | low | 0.574 | 55,095 | 0.621 | 22,423 |
| added_lines (size, primary) | mid | 0.452 | 62,105 | 0.477 | 23,760 |
| added_lines (size, primary) | high | 0.352 | 67,470 | 0.371 | 25,025 |
| ratio (tree term) | low | 0.510 | 65,220 | 0.548 | 25,515 |
| ratio (tree term) | mid | 0.402 | 56,741 | 0.435 | 21,759 |
| ratio (tree term) | high | 0.437 | 62,709 | 0.464 | 23,934 |

## Verdicts

**T1 interaction** (all combos): rung_binary x z(log1p(added_lines)) -0.041 [-0.075, +0.015]; rung_binary x z(ratio) -0.074 [-0.162, -0.007]; rung_binary x z(new_frac) -0.019 [-0.078, +0.020] — the prediction is not confirmed for the primary proxy (CI includes 0); the rung x ratio interaction is the only significant one and is negative — the rung benefit shrinks on tree-heavy tasks, the opposite of the prediction.

**T2 discrimination** (all combos): Spearman(a_2pl, ratio) = +0.015 vs Spearman(a_2pl, log1p(added_lines)) = -0.072 — the prediction (ratio relates to a, separates solvers; size less so) not confirmed.

**T3 mechanism** (all combos): among unresolved trajectories, mean patch_file_jaccard 0.510 (low ratio tercile) -> 0.437 (high) — the prediction (high-ratio failures have HIGH file overlap, right files wrong content) not confirmed.

## Reproduce

```bash
uv run python scripts/rung_mapping.py --stream-only
uv run python scripts/framework_tests.py --skip-stream
uv run pytest tests/test_framework_tests.py
```
