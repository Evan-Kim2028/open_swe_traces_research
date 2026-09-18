# Task difficulty via IRT — model-independent difficulty and combo abilities

Generated 2026-09-18 00:27 UTC by `scripts/fit_irt.py` from `outputs/proxy_features.parquet`; models in `outputs/task_irt.parquet` and `outputs/combo_ability.parquet`; `outputs/task_difficulty.parquet` gains an `irt_bucket` column.

Fit: **280,116** labeled rollouts of **36,015** instances (>= 3 labeled rollouts each; 42,413 instances total) across **7** labeled harness/teacher combos. `P(resolved) = sigmoid(a_task * (theta_combo - b_task))`, full-batch Adam (2000 steps, lr 0.05); priors theta ~ N(0, 1), b ~ N(0, 1), log a ~ N(0, 0.5). The priors anchor the ability scale's location as well as its shrinkage: the likelihood alone is invariant to shifting theta and b by a common constant, so only differences within each set are meaningful.

## 1. Combo abilities (theta) vs observed resolved rates

| rank | combo | theta_2pl | theta_1pl | observed rate | n labeled |
|---|---|---|---|---|---|
| 1 | minisweagent/qwen38_27b | +0.180 | +0.179 | 0.538 | 91,595 |
| 2 | sweagent/qwen36_27b | +0.059 | +0.070 | 0.536 | 36,184 |
| 3 | sweagent/qwen35_122b | -0.170 | -0.198 | 0.487 | 15,054 |
| 4 | sweagent/minimax_m25 | -0.246 | -0.291 | 0.467 | 35,788 |
| 5 | openhands/minimax_m25 | -0.541 | -0.562 | 0.425 | 33,784 |
| 6 | minisweagent/qwen36_27b | -0.605 | -0.675 | 0.386 | 36,888 |
| 7 | openhands/qwen35_122b | -1.050 | -1.123 | 0.318 | 30,823 |

Rank agreement with the observed resolved rate: Spearman(theta_2pl, rate) +1.0000, Spearman(theta_1pl, rate) +1.0000; the two ability scales correlate at +1.0000. The ordering matches the raw rates, but the scale separates combos the rates compress: minisweagent/qwen36_27b and sweagent/qwen36_27b — same teacher, different harness — differ by 0.150 in raw rate but 0.664 in theta, because rates on mostly-hard tasks saturate while theta keeps the log-odds gap. Task mix differs across combos too: at the extremes, minisweagent/qwen36_27b ran the hardest slice and openhands/minimax_m25 the easiest, 0.12 b-units apart on the (arbitrary) b origin — theta is what puts those different mixes on one scale.

## 2. b_2pl vs solve_rate

Spearman(b_2pl, -solve_rate) +0.9507; Spearman(b_1pl, -solve_rate) +0.9447; Spearman(b_2pl, b_1pl) +0.9965. b is on the combo-ability logit scale: b = k means a combo of ability k solves the task with P = 0.5, and one more unit of combo ability multiplies the task's solve odds by exp(a_task) — so b is difficulty net of which combos attempted the task, which the raw rate is not.

| series | p05 | p25 | median | p75 | p95 |
|---|---|---|---|---|---|
| b_2pl | -1.694 | -1.144 | +0.112 | +1.149 | +1.353 |
| solve_rate | 0.000 | 0.000 | 0.375 | 0.889 | 1.000 |

## 3. Bucket disagreement: b_bucket vs pass-rate bucket

On the fitted set, `irt_bucket` matches the pass-rate `difficulty_bucket` for **86.5%** of instances. The b cuts are quantile-matched to the pass-rate bucket shares (so bucket sizes approximately match — b is piecewise-constant, tasks with identical outcome patterns share an identical estimate, so a cut inside a tie block moves the whole block): all_pass <= -1.213, easy <= -0.517, mid <= +0.191, hard <= +0.894 (ascending b = all_pass to all_fail). `delta` = pass-rank minus b-rank on the hardness scale (positive = the pass-rate bucket looks easier).

| pass \ b | all_fail | hard | mid | easy | all_pass | pass total |
|---|---|---|---|---|---|---|
| all_fail | 13,645 | 560 | 0 | 0 | 0 | 14,205 |
| hard | 568 | 2,719 | 218 | 0 | 0 | 3,505 |
| mid | 0 | 246 | 2,954 | 402 | 0 | 3,602 |
| easy | 0 | 0 | 402 | 4,769 | 1,236 | 6,407 |
| all_pass | 0 | 0 | 0 | 1,240 | 7,056 | 8,296 |

| group | instances | share | mean solve_rate | mean mix theta | weak-combo share | strong-combo share | median combos |
|---|---|---|---|---|---|---|---|
| pass bucket easier than b bucket (delta > 0) | 2,456 | 6.8% | 0.680 | -0.038 | 16.1% | 72.0% | 3 |
| pass bucket harder than b bucket (delta < 0) | 2,416 | 6.7% | 0.585 | -0.402 | 26.4% | 22.8% | 4 |
| agree (delta = 0) | 31,143 | 86.5% | 0.413 | -0.190 | 23.8% | 52.4% | 3 |
| all fitted instances | 36,015 | 100.0% | 0.443 | -0.194 | 23.4% | 51.7% | 3 |

Combo mix moves the disagreement in the expected direction: Spearman(delta, per-instance mean combo theta) +0.3346. Instances whose pass-rate bucket reads easier than the b-bucket were disproportionately attempted by strong combos (mean mix theta -0.038, strong-combo share 72.0% vs 51.7% overall), so their successes overstate ease. The mirror group (pass bucket reads harder) is the opposite but milder: a small weak-combo tilt (26.4% weak-combo share vs 23.4% overall) on tasks with more combos attempted (median 4 vs 3), where a few weak-combo failures drag the rate below what the ability-adjusted difficulty implies.

Largest disagreement cells (pass bucket → b bucket):

| cell | instances | mean solve_rate | mean b_2pl | mean mix theta | weak-combo share |
|---|---|---|---|---|---|
| all_pass → easy | 1,240 | 1.000 | -1.036 | +0.036 | 11.8% |
| easy → all_pass | 1,236 | 0.887 | -1.421 | -0.383 | 21.1% |
| hard → all_fail | 568 | 0.106 | +0.978 | -0.217 | 24.8% |
| all_fail → hard | 560 | 0.000 | +0.807 | -0.420 | 37.5% |
| easy → mid | 402 | 0.679 | -0.373 | +0.008 | 12.2% |
| mid → easy | 402 | 0.611 | -0.611 | -0.402 | 23.7% |
| mid → hard | 246 | 0.393 | +0.224 | -0.078 | 23.7% |
| hard → mid | 218 | 0.324 | +0.115 | -0.461 | 32.8% |

## 4. Discrimination a_2pl

a_2pl is the slope: high-a tasks separate strong from weak combos sharply (outcomes flip over a narrow ability band), low-a tasks are near coin flips for everyone. For tasks with all-fail or all-pass outcomes the likelihood keeps a(a-b) flat, so a is only pinned by its prior — those estimates are weak by construction.

| quantile | a_2pl |
|---|---|
| min | 0.678 |
| p05 | 1.018 |
| p25 | 1.231 |
| median | 1.438 |
| p75 | 1.551 |
| p95 | 1.694 |
| max | 2.136 |

| pass bucket | instances | mean a | median a | share of bucket in high-a decile | share of high-a decile from bucket |
|---|---|---|---|---|---|
| all_fail | 14,205 | 1.452 | 1.480 | 8.0% | 31.5% |
| hard | 3,505 | 1.233 | 1.232 | 1.1% | 1.0% |
| mid | 3,602 | 1.195 | 1.152 | 6.2% | 6.2% |
| easy | 6,407 | 1.315 | 1.326 | 7.6% | 13.5% |
| all_pass | 8,296 | 1.530 | 1.537 | 20.8% | 47.8% |

| median metric | high-a decile | low-a decile | high-a decile (mixed only) | low-a decile (mixed only) |
|---|---|---|---|---|
| n_labeled | 12 | 8 | 9 | 8 |
| solve_rate | 0.923 | 0.500 | 0.223 | 0.500 |
| gold_patch_lines | 17 | 32 | 33 | 30 |
| gold_patch_files | 2 | 3 | 3 | 2 |
| issue_chars | 6,146 | 5,866 | 5,832 | 5,885 |

| Spearman(a_2pl, .) | rho |
|---|---|
| n_labeled | +0.5400 |
| solve_rate | +0.0663 |
| gold_patch_lines | -0.0345 |
| gold_patch_files | -0.0296 |
| issue_chars | -0.0123 |
| abs(b_2pl - median b_2pl) | +0.7029 |

| language | fitted share | share of high-a decile | share of mixed high-a decile |
|---|---|---|---|
| python | 60.4% | 36.8% | 59.6% |
| go | 13.2% | 19.0% | 13.1% |
| typescript | 8.9% | 12.9% | 8.9% |
| javascript | 7.6% | 17.5% | 9.9% |
| rust | 4.2% | 4.4% | 3.1% |

| category | fitted share | share of high-a decile | share of mixed high-a decile |
|---|---|---|---|
| bug-fix | 49.9% | 70.6% | 52.4% |
| feature-request | 30.6% | 16.8% | 30.0% |
| enhancement | 12.7% | 8.3% | 11.7% |
| other | 6.8% | 4.4% | 5.9% |

Reading the tables: the high-a decile is a well-sampled, extreme-outcome phenomenon, not a language or category story. Degenerate tasks supply 79.3% of the top decile against their 62.5% fitted share, with all_pass over-represented (47.8% of the top decile from its 23.0% share) and hard tasks all but absent (1.0%); a is largest where outcomes are most extreme and most observed (median n_labeled 12 vs 8 in the low-a decile). Among mixed tasks, high-a instances are harder (median solve_rate 0.223 vs 0.500) but structurally ordinary: python is 59.6% of the mixed high-a decile vs 60.4% fitted, and gold patch size is flat (33 vs 30 median lines).

## 5. Held-out log-likelihood: 1PL vs 2PL vs pass-rate baseline

20% random split of rollouts (seed 42): 220,246 train rollouts / 33,880 fitted tasks; 52,525 test rollouts of fitted tasks scored against the train-fitted models (and their task's train solve rate, Jeffreys-smoothed as (resolved + 0.5) / (n + 1)). Smoothing breaks the ties among 0/1-rate tasks and keeps degenerate rates out of the log-likelihood clip; the baseline still sees no combo identity, which is where the IRT models gain.

| predictor | mean LL / rollout | total LL | AUC |
|---|---|---|---|
| 1PL (a = 1) | -0.3874 | -20,347 | 0.9465 |
| 2PL (free a) | -0.3407 | -17,894 | 0.9495 |
| pass-rate baseline (task train rate, smoothed) | -0.3273 | -17,192 | 0.9352 |

The aggregate LL favors the smoothed pass rate, but that comes entirely from the degenerate tasks it predicts sharply: on the 29,998 degenerate test rollouts (all_fail/all_pass) it scores -0.0776 vs -0.1691 (2PL) and -0.2484 (1PL), while on the 22,527 mixed rollouts (hard/mid/easy — the ones manifest eligibility selects) both IRT models win decisively: -0.5691 (2PL) and -0.5725 (1PL) vs -0.6598. The models also rank better overall (AUC 0.9495 vs 0.9352) and the 2PL beats the 1PL, so both the ability scale and the per-task slope carry information the rate misses.

| test rollouts | n | 1PL LL | 2PL LL | pass-rate LL |
|---|---|---|---|---|
| degenerate (all_fail/all_pass) | 29,998 | -0.2484 | -0.1691 | -0.0776 |
| mixed (hard/mid/easy) | 22,527 | -0.5725 | -0.5691 | -0.6598 |

## 6. Recommendation for manifest bucketing

**Use `irt_bucket` for the hard/mid/easy eligibility of manifests; keep the pass-rate `difficulty_bucket` only as the fallback for instances with fewer than 3 labeled rollouts (which stay `unknown`).** In order of strength: (1) on the mixed held-out rollouts — exactly the tasks manifest eligibility selects — the IRT models beat the task's own rate by +0.0907 (2PL) and +0.0874 (1PL) nats/rollout, the 2PL beats the 1PL, and the models rank better overall (AUC 0.9495 vs 0.9352); the smoothed rate's aggregate LL edge comes only from degenerate tasks that eligibility drops; (2) where the buckets disagree (13.5% of fitted instances), the pass-rate side is the biased one — the group whose pass bucket reads easier consists of 72.0% strong-combo attempts (vs 51.7% overall), exactly the composition that inflates an observed rate; (3) `irt_bucket` keeps the pass-rate bucket sizes (quantile-matched up to b tie blocks, so budget allocation and eval balancing carry over largely unchanged), and the cuts (-1.213, -0.517, +0.191, +0.894) are reusable across refits while the corpus mix is stable. The costs to accept: the bucket is defined only for tasks with >= 3 labeled rollouts, it comes from a global fit (rebuild order matters), and it moves a task's label whenever its attempted-combo mix changes even if the task text does not.

## Notes

- Fitting detail: full-batch Adam on NLL + priors, 2000 steps at lr 0.05; objective 108,790.5 (2PL) vs 120,028.6 (1PL); loss history: 0:194,162, 100:108,792, 200:108,790, 300:108,790, 400:108,790, 500:108,790, 600:108,790, 700:108,790, 800:108,790, 900:108,790, 1000:108,790, 1100:108,790, 1200:108,790, 1300:108,790, 1400:108,790, 1500:108,790, 1600:108,790, 1700:108,790, 1800:108,790, 1900:108,790, 2000:108,790.
- theta and b are jointly shift-invariant: the likelihood alone cannot fix the ability scale's location, so the N(0, 1) priors anchor it; only differences within the theta set and within the b set are meaningful.
- b_2pl is piecewise-constant: 7,494 distinct values over 36,015 instances (largest tie block 2,078). Tasks with identical outcome-pattern × combo-mix signatures have the same likelihood and the same penalized MLE, so the quantile cuts reproduce the pass-rate bucket sizes only up to the tie block that straddles each cut.
- a_2pl for all-fail/all-pass tasks is weakly identified (the likelihood is flat along the a(a-b) ridge); the prior on log a is what pins it. Discrimination comparisons should be read on the mixed buckets as well as overall.
- `n_labeled` counts rollouts with `resolved in (0, 1)`; `solve_rate` is n_resolved / n_labeled over the instance's labeled rollouts.
- `irt_bucket` is the b_2pl quantile bucket (`unknown` below 3 labeled rollouts). Rebuilding `outputs/task_difficulty.parquet` with `scripts/task_difficulty.py` drops the column — rerun `scripts/fit_irt.py` after it.
- Reproduce: `uv run python scripts/fit_irt.py` (about a minute; `--steps`, `--lr`, `--no-bucket` available).
