# Cheap task difficulty — how few rollouts identify an instance's bucket

Generated 2026-09-17 23:34 UTC by `scripts/cheap_difficulty.py` from `outputs/proxy_features.parquet`, `outputs/task_difficulty.parquet`, and `outputs/prefix_features.parquet` (streamed from `traces_data/`).

Corpus: 511,668 trajectories, 42,413 instances, 285,636 labeled rollouts; 7 labeled harness/teacher combos.

## A. Cross-teacher probe

For every ordered pair of labeled combos: a single random probe rollout (seed 0, per-instance stable draw order) predicts whether the target combo's solve_rate over its >= 3 labeled rollouts exceeds 0.5 (AUC, ties included); probe means over the first k = 1, 2, 3 rollouts of the same order are Spearman-correlated with the target rate. `n` is the number of instances with >= 1 labeled probe rollout and >= 3 labeled target rollouts.

### A1. AUC(k=1 probe rollout) — full pair matrix

| probe \ target | mi/qwen38_27b | mi/qwen36_27b | sw/qwen36_27b | sw/minimax_m25 | op/minimax_m25 | op/qwen35_122b | sw/qwen35_122b |
|---|---|---|---|---|---|---|---|
| mi/qwen38_27b | 0.965 | 0.797 | 0.891 | 0.828 | 0.807 | 0.753 | 0.824 |
| mi/qwen36_27b | 0.834 | 0.927 | 0.857 | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | 0.899 | 0.818 | 0.964 | n/a | n/a | n/a | n/a |
| sw/minimax_m25 | 0.827 | n/a | n/a | 0.945 | 0.872 | 0.803 | 0.855 |
| op/minimax_m25 | 0.797 | n/a | n/a | 0.870 | 0.954 | 0.839 | 0.833 |
| op/qwen35_122b | 0.742 | n/a | n/a | 0.781 | 0.835 | 0.927 | 0.791 |
| sw/qwen35_122b | 0.829 | n/a | n/a | 0.860 | 0.832 | 0.807 | 0.942 |

### A2. Spearman(probe mean k=1, target solve_rate) — full pair matrix

| probe \ target | mi/qwen38_27b | mi/qwen36_27b | sw/qwen36_27b | sw/minimax_m25 | op/minimax_m25 | op/qwen35_122b | sw/qwen35_122b |
|---|---|---|---|---|---|---|---|
| mi/qwen38_27b | +0.945 | +0.756 | +0.810 | +0.692 | +0.646 | +0.540 | +0.687 |
| mi/qwen36_27b | +0.680 | +0.860 | +0.725 | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | +0.811 | +0.795 | +0.948 | n/a | n/a | n/a | n/a |
| sw/minimax_m25 | +0.659 | n/a | n/a | +0.924 | +0.761 | +0.624 | +0.732 |
| op/minimax_m25 | +0.598 | n/a | n/a | +0.761 | +0.935 | +0.720 | +0.685 |
| op/qwen35_122b | +0.511 | n/a | n/a | +0.602 | +0.710 | +0.885 | +0.636 |
| sw/qwen35_122b | +0.669 | n/a | n/a | +0.746 | +0.690 | +0.646 | +0.920 |

Off-diagonal pairs: 26; mean AUC 0.826 (min 0.742, max 0.899), mean Spearman k=1 +0.688. Self-pairs (same combo, different rollouts): mean AUC 0.946, mean Spearman k=1 +0.917.

### A3. All pairs (k = 1, 2, 3 probe rollouts)

| probe | target | n | target >0.5 share | AUC k=1 | rho k=1 | rho k=2 | rho k=3 |
|---|---|---|---|---|---|---|---|
| mi/qwen38_27b | mi/qwen38_27b | 24,492 | 0.562 | 0.965 | +0.945 | +0.982 | +1.000 |
| mi/qwen38_27b | mi/qwen36_27b | 8,759 | 0.333 | 0.797 | +0.756 | +0.781 | +0.789 |
| mi/qwen38_27b | sw/qwen36_27b | 8,499 | 0.568 | 0.891 | +0.810 | +0.838 | +0.847 |
| mi/qwen38_27b | sw/minimax_m25 | 8,062 | 0.520 | 0.828 | +0.692 | +0.718 | +0.722 |
| mi/qwen38_27b | op/minimax_m25 | 7,149 | 0.477 | 0.807 | +0.646 | +0.670 | +0.674 |
| mi/qwen38_27b | op/qwen35_122b | 5,601 | 0.333 | 0.753 | +0.540 | +0.558 | +0.565 |
| mi/qwen38_27b | sw/qwen35_122b | 2,392 | 0.536 | 0.824 | +0.687 | +0.707 | +0.718 |
| mi/qwen36_27b | mi/qwen38_27b | 13,408 | 0.597 | 0.834 | +0.680 | +0.739 | +0.775 |
| mi/qwen36_27b | mi/qwen36_27b | 8,783 | 0.332 | 0.927 | +0.860 | +0.946 | +1.000 |
| mi/qwen36_27b | sw/qwen36_27b | 8,371 | 0.567 | 0.857 | +0.725 | +0.782 | +0.821 |
| mi/qwen36_27b | sw/minimax_m25 | 0 | n/a | n/a | n/a | n/a | n/a |
| mi/qwen36_27b | op/minimax_m25 | 0 | n/a | n/a | n/a | n/a | n/a |
| mi/qwen36_27b | op/qwen35_122b | 0 | n/a | n/a | n/a | n/a | n/a |
| mi/qwen36_27b | sw/qwen35_122b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | mi/qwen38_27b | 12,275 | 0.602 | 0.899 | +0.811 | +0.834 | +0.836 |
| sw/qwen36_27b | mi/qwen36_27b | 7,904 | 0.337 | 0.818 | +0.795 | +0.825 | +0.829 |
| sw/qwen36_27b | sw/qwen36_27b | 8,520 | 0.567 | 0.964 | +0.948 | +0.984 | +1.000 |
| sw/qwen36_27b | sw/minimax_m25 | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | op/minimax_m25 | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | op/qwen35_122b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen36_27b | sw/qwen35_122b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/minimax_m25 | mi/qwen38_27b | 7,463 | 0.629 | 0.827 | +0.659 | +0.694 | +0.702 |
| sw/minimax_m25 | mi/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/minimax_m25 | sw/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/minimax_m25 | sw/minimax_m25 | 8,579 | 0.511 | 0.945 | +0.924 | +0.977 | +1.000 |
| sw/minimax_m25 | op/minimax_m25 | 7,231 | 0.471 | 0.872 | +0.761 | +0.794 | +0.800 |
| sw/minimax_m25 | op/qwen35_122b | 5,584 | 0.332 | 0.803 | +0.624 | +0.660 | +0.667 |
| sw/minimax_m25 | sw/qwen35_122b | 2,417 | 0.534 | 0.855 | +0.732 | +0.780 | +0.795 |
| op/minimax_m25 | mi/qwen38_27b | 7,434 | 0.635 | 0.797 | +0.598 | +0.623 | +0.627 |
| op/minimax_m25 | mi/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| op/minimax_m25 | sw/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| op/minimax_m25 | sw/minimax_m25 | 8,097 | 0.520 | 0.870 | +0.761 | +0.791 | +0.795 |
| op/minimax_m25 | op/minimax_m25 | 7,553 | 0.469 | 0.954 | +0.935 | +0.980 | +1.000 |
| op/minimax_m25 | op/qwen35_122b | 5,639 | 0.337 | 0.839 | +0.720 | +0.750 | +0.759 |
| op/minimax_m25 | sw/qwen35_122b | 2,322 | 0.540 | 0.833 | +0.685 | +0.729 | +0.735 |
| op/qwen35_122b | mi/qwen38_27b | 7,098 | 0.633 | 0.742 | +0.511 | +0.545 | +0.556 |
| op/qwen35_122b | mi/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| op/qwen35_122b | sw/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| op/qwen35_122b | sw/minimax_m25 | 7,593 | 0.514 | 0.781 | +0.602 | +0.647 | +0.660 |
| op/qwen35_122b | op/minimax_m25 | 6,875 | 0.474 | 0.835 | +0.710 | +0.756 | +0.771 |
| op/qwen35_122b | op/qwen35_122b | 6,013 | 0.326 | 0.927 | +0.885 | +0.964 | +1.000 |
| op/qwen35_122b | sw/qwen35_122b | 2,380 | 0.532 | 0.791 | +0.636 | +0.691 | +0.710 |
| sw/qwen35_122b | mi/qwen38_27b | 4,119 | 0.641 | 0.829 | +0.669 | +0.691 | +0.692 |
| sw/qwen35_122b | mi/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen35_122b | sw/qwen36_27b | 0 | n/a | n/a | n/a | n/a | n/a |
| sw/qwen35_122b | sw/minimax_m25 | 4,883 | 0.523 | 0.860 | +0.746 | +0.774 | +0.779 |
| sw/qwen35_122b | op/minimax_m25 | 3,844 | 0.492 | 0.832 | +0.690 | +0.713 | +0.718 |
| sw/qwen35_122b | op/qwen35_122b | 3,772 | 0.343 | 0.807 | +0.646 | +0.674 | +0.681 |
| sw/qwen35_122b | sw/qwen35_122b | 2,536 | 0.525 | 0.942 | +0.920 | +0.976 | +1.000 |

### A4. Cheapest teacher as the probe

Probe = single rollout of `minisweagent/qwen36_27b`; target = leave-this-combo-out solve_rate over all other combos (>= 3 labeled elsewhere).

| probe | target | n | target >0.5 share | AUC k=1 | rho k=1 | rho k=2 | rho k=3 |
|---|---|---|---|---|---|---|---|
| minisweagent/qwen36_27b | all other combos | 15,898 | 0.542 | 0.8446 | +0.7217 | +0.7843 | +0.8214 |

## B. Partial-rollout probe

Prefix features at k tool calls: gold file viewed/edited, n edit calls, n distinct commands, repeat rate, assistant chars, any tool observation matching `error|traceback|failed` (case-insensitive). One streaming pass over the corpus: 511,668 rows / 511,668 distinct trajectories, mean 73.4 tool calls (median 69; 0.0% with zero calls). Sanity: 0 non-monotone prefix rows, 0 gold-edit-without-view, 0 bad repeat rates.

| k | share with >=k calls | gold viewed | gold edited | edits | distinct cmds | repeat rate | assistant chars | any error |
|---|---|---|---|---|---|---|---|---|
| 5 | 100.0% | 0.833 | 0.004 | 0.05 | 5.00 | 0.000 | 304 | 0.802 |
| 10 | 100.0% | 0.951 | 0.060 | 0.36 | 9.97 | 0.003 | 636 | 0.919 |
| 20 | 99.1% | 0.983 | 0.384 | 1.96 | 19.58 | 0.019 | 1,544 | 0.972 |
| 40 | 85.2% | 0.991 | 0.771 | 5.95 | 36.89 | 0.045 | 3,622 | 0.993 |

### B1. Prefix-only classifier of `resolved`

Standardized (winsorized 1st/99th, mean 0 / sd 1) logistic regression per k, grouped 80/20 split by instance_id (`random_state=42`): 285,636 labeled rows, 228,483 train / 57,153 test. References on the same split: all 27 full-trace features AUC 0.7115 (published 0.7115), leave-one-out task rate AUC 0.9361 (published 0.9361).

| predictor | held-out AUC | Δ vs full trace | signal captured vs full trace |
|---|---|---|---|
| 5 | 0.5574 | -0.1541 | 27.1% |
| 10 | 0.5857 | -0.1258 | 40.5% |
| 20 | 0.5973 | -0.1142 | 46.0% |
| 40 | 0.5986 | -0.1129 | 46.6% |
| full trace (27 features) | 0.7115 | — | 100.0% |
| task solve_rate alone (leave-one-out, 1 feature) | 0.9361 | — | — |

Best prefix k = 40 (AUC 0.5986).

### B2. Prefix-only classifier of the instance bucket (mid vs degenerate)

Instance-level: features are means over the instance's rollouts; positive class = `mid`, negative = `all_fail` or `all_pass` (hard/easy/unknown dropped). 26,103 instances, mid share 0.138; grouped 80/20 by instance_id.

| k | held-out AUC (mid vs degenerate) |
|---|---|
| 5 | 0.5272 |
| 10 | 0.5198 |
| 20 | 0.5155 |
| 40 | 0.5362 |

## C. Adaptive sampling simulation

Instances with >= 6 labeled rollouts; rollouts drawn in random order (5 seeds). Wilson rule: stop when the 80% Wilson interval for solve_rate fits entirely inside one bucket (all_fail/hard/mid/easy/all_pass ranges as in `difficulty.py`), cap 12. Simple rule: stop after 2 agreeing outcomes, else 4. The final bucket is `difficulty_buckets(point rate, n drawn)` — the interval only decides when to stop; `agree2-else-4` keeps that rule (a 2-draw stop cannot satisfy the >= 3 labeled-rollout requirement and lands in `unknown`), while `agree2-else-4-implied` reads two agreeing failures as `all_fail` and two agreeing successes as `all_pass`. Metrics are mean ± sd across seeds.

| rule | mean rollouts | bucket accuracy | accuracy (defined only) | unknown share | cap share | agree2 share |
|---|---|---|---|---|---|---|
| wilson-80-cap12 | 5.38 ± 0.01 | 0.885 ± 0.003 | 0.885 ± 0.003 | 0.000 ± 0.000 | 0.036 | 0.000 |
| agree2-else-4 | 2.34 ± 0.00 | 0.113 ± 0.002 | 0.671 ± 0.008 | 0.832 ± 0.002 | 0.168 | 0.832 |
| agree2-else-4-implied | 2.34 ± 0.00 | 0.696 ± 0.002 | 0.696 ± 0.002 | 0.000 ± 0.000 | 0.168 | 0.832 |

### C1. By full-data bucket (pooled over seeds)

| rule | full bucket | instance-draws | mean rollouts | accuracy |
|---|---|---|---|---|
| agree2-else-4 | all_fail | 49,075 | 2.00 | 0.000 |
| agree2-else-4 | all_pass | 34,370 | 2.00 | 0.000 |
| agree2-else-4 | easy | 28,940 | 2.70 | 0.256 |
| agree2-else-4 | hard | 15,185 | 2.72 | 0.256 |
| agree2-else-4 | mid | 15,455 | 3.10 | 0.315 |
| agree2-else-4-implied | all_fail | 49,075 | 2.00 | 1.000 |
| agree2-else-4-implied | all_pass | 34,370 | 2.00 | 1.000 |
| agree2-else-4-implied | easy | 28,940 | 2.70 | 0.256 |
| agree2-else-4-implied | hard | 15,185 | 2.72 | 0.256 |
| agree2-else-4-implied | mid | 15,455 | 3.10 | 0.315 |
| wilson-80-cap12 | all_fail | 49,075 | 4.00 | 1.000 |
| wilson-80-cap12 | all_pass | 34,370 | 4.00 | 1.000 |
| wilson-80-cap12 | easy | 28,940 | 6.82 | 0.651 |
| wilson-80-cap12 | hard | 15,185 | 6.85 | 0.666 |
| wilson-80-cap12 | mid | 15,455 | 8.65 | 0.918 |

### C2. Per-seed detail

| rule | seed | mean rollouts | accuracy | defined accuracy | unknown share |
|---|---|---|---|---|---|
| agree2-else-4 | 0 | 2.33 | 0.111 | 0.672 | 0.834 |
| agree2-else-4 | 1 | 2.34 | 0.115 | 0.678 | 0.830 |
| agree2-else-4 | 2 | 2.34 | 0.115 | 0.672 | 0.828 |
| agree2-else-4 | 3 | 2.33 | 0.113 | 0.676 | 0.833 |
| agree2-else-4 | 4 | 2.33 | 0.110 | 0.658 | 0.833 |
| agree2-else-4-implied | 0 | 2.33 | 0.695 | 0.695 | 0.000 |
| agree2-else-4-implied | 1 | 2.34 | 0.698 | 0.698 | 0.000 |
| agree2-else-4-implied | 2 | 2.34 | 0.699 | 0.699 | 0.000 |
| agree2-else-4-implied | 3 | 2.33 | 0.697 | 0.697 | 0.000 |
| agree2-else-4-implied | 4 | 2.33 | 0.694 | 0.694 | 0.000 |
| wilson-80-cap12 | 0 | 5.37 | 0.883 | 0.883 | 0.000 |
| wilson-80-cap12 | 1 | 5.37 | 0.884 | 0.884 | 0.000 |
| wilson-80-cap12 | 2 | 5.40 | 0.890 | 0.890 | 0.000 |
| wilson-80-cap12 | 3 | 5.38 | 0.885 | 0.885 | 0.000 |
| wilson-80-cap12 | 4 | 5.37 | 0.883 | 0.883 | 0.000 |

## Notes

- Analysis B streams the corpus shard by shard (DuckDB, 6 GB cap); parts under `outputs/prefix_features_parts/` are atomic and skipped on rerun, so an interrupted pass resumes where it stopped.
- Prefix semantics: a feature at k uses tool calls 1..k (and the assistant turn containing call k); trajectories with fewer than k calls repeat their full-trajectory values; observed errors count tool observations whose position is at or before call k.
- In `difficulty.py` the extremes are point buckets (`all_fail` is exactly 0, `all_pass` exactly 1), so a Wilson interval can never sit inside them; the extremes are certified through the `hard` / `easy` ranges instead (0/4 fits inside `hard`, 4/4 inside `easy`), and the final assignment falls back to the point estimate. Mid-rate tasks exhaust the cap.
- The simple rule stops at 2 agreeing outcomes; `difficulty.py` requires >= 3 labeled rollouts for a bucket, so under the canonical rule those stops are `unknown` (hence the low raw accuracy of `agree2-else-4`) — the `-implied` variant shows the same rule with the two agreeing draws read as the extreme buckets.
- `any_error` is a loose substring test (`error|traceback|failed`, case-insensitive), so it also fires on benign mentions (e.g. pytest's `0 failed`); it saturates quickly (0.80 of trajectories by call 5, 0.99 by call 40) and carries little signal.
- `assistant_chars` understates combos whose assistant text lives in reasoning fields (e.g. `openhands/deepseek_v4_flash`).
- Reproduce: `uv run python scripts/cheap_difficulty.py --stream-only` (corpus pass, ~76 min, resume-safe), then `uv run python scripts/cheap_difficulty.py --skip-stream`.
