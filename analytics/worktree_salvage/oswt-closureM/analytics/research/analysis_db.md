# One persistent analysis database — no more re-scanning the corpus

2026-09-19. Built by `scripts/build_analysis_db.py` (idempotent) into
`duckdb/analysis.duckdb` (gitignored). Before this job, analysis jobs B/C/G/H
each streamed all 212 shards (42 GB, ~13 min per the job brief; the closure-C
frame pass actually logged ~30 min at ~9 s/shard) to rebuild nearly the same
per-trajectory and per-instance frames, then grouped and fit in pandas. The
corpus scan was not the cost; repeating it — and modelling outside the database
— was. Now the shared frames are tables, built once.

## Table list and row counts

Build: `uv run python scripts/sync_derived.py && uv run python scripts/extract_trajectory_features.py && uv run python scripts/build_analysis_db.py`.
Build wall time: **52.0 s** (15 tables, DROP + CREATE + INSERT FROM parquet).
Size on disk: **4,044.9 MB** (`duckdb/analysis.duckdb`; dominated by the
yoonholee `steps` JSON text, ~2 GB uncompressed). The one-time per-trajectory
extraction (`extract_trajectory_features.py`) took **34 m 34 s** over the 212
shards (210 processed, 2 resumed from parts, 0 failed) and writes
`outputs/derived/trajectory_features.parquet` (511,668 rows).

| table | rows | PK | source |
|---|---:|---|---|
| instance | 42,413 | instance_id | closure_proxies + rung_features + patch_split (wide join) |
| trajectory | 511,668 | trajectory_id | trajectory_frame + trajectory_features + eligible_pairs (wide join) |
| paired_features | 41,362 | trajectory_id | outputs/derived/paired_features.parquet (closure-H, eligible subset) |
| eligible_pairs | 41,362 | trajectory_id | outputs/derived/eligible_pairs.parquet (closure-H) |
| llm_labels | 293 | trajectory_id | outputs/derived/llm_labels.jsonl (closure-H) |
| closure_metrics | 38 | (repo, unit) | outputs/derived/closure_metrics.parquet (closure-A) |
| harbor_hub_jobs | 50 | job_id | outputs/derived/harbor_hub/jobs.parquet (closure-E) |
| harbor_hub_leaderboard_rows | 202 | row_id | outputs/derived/harbor_hub/leaderboard_rows.parquet |
| harbor_hub_row_trials | 22,810 | (row_id, trial_id) | outputs/derived/harbor_hub/row_trials.parquet |
| harbor_hub_tasks | 229 | (bench_version, task_id) | outputs/derived/harbor_hub/tasks.parquet |
| harbor_hub_trials | 23,887 | trial_id | outputs/derived/harbor_hub/trials.parquet |
| tb2_traj_yoonholee | 52,104 | — (trial_id repeats) | traces_external/yoonholee (read in place, ~212 MB) |
| tb2_traj_harithoppil | 3,723 | — (no id column) | traces_external/harithoppil 'all' config (read in place) |
| scale_swe_meta | 20,181 | instance_id | traces_external/AweAI-Team__Scale-SWE/meta.parquet |
| rebench_v2_meta | 32,079 | instance_id | traces_external/nebius__SWE-rebench-V2/meta.parquet |

Note: `harbor_hub_tasks.task_id` is *not* unique across bench versions (229 rows /
162 ids — e.g. `coq-block-bound` runs on 3.0 and 4.0), so the PK is
`(bench_version, task_id)`; `yoonholee.trial_id` repeats across attempts
(52,104 rows / 22,599 ids), so that table has no PK. Full column docs:
`analytics/schema/DATA_CATALOG.md` (Part 9).

## The queries that moved into SQL (`analytics/queries/1xx_*.sql`)

| query | produces | rows | notes |
|---|---|---|---|
| 101_mixture_size_deciles.sql | mixture-by-size-bin inputs | 38,294 | per-instance (decile, n_labeled, n_resolved) for the two-component binomial EM (closure-G); Python keeps only the EM fit |
| 102_rung_x_size_tercile.sql | rung × size tercile table | 6 | rung_binary × added_lines tercile → mean solve_rate / n, all + top-3 combos (closure-C T1 2×3 table) |
| 103_paired_diff_inputs.sql | paired-difference inputs | 41,362 | per eligible trajectory: group_id/stratum + behaviour features for the fail-vs-pass bootstrap (closure-H); Python keeps only the bootstrap |
| 104_harbor_task_rates.sql | harbor-hub per-task rates | 229 | per (bench_version, task_name): n_trials, n_jobs, mean_reward, n_passed, pass_rate (closure-E) |

All runnable with `uv run python scripts/duckdb_query.py -f analytics/queries/1xx_*.sql --db analysis`.
Query-log timings (includes console streaming of the result rows): 101 = 12.5 s
(38 K rows printed), 102 = 0.14 s, 103 = 39 s (41 K rows printed), 104 = 0.12 s.

## Before/after: the rung × size tercile table (102)

Reproduce: `uv run python scripts/time_rung_tercile.py` (both runs are logged to
`analytics/query_log/`; the streamed SQL mirrors 102 against the raw parquets).

| variant | engine time |
|---|---|
| **streamed** — read the 212 corpus shards (projection-pushed, warm page cache) + join rung/closure parquets + groupby | 0.1–0.4 s |
| **from `analysis.duckdb`** — same query shape against the `trajectory`/`instance` tables | 0.08–0.14 s |

Results agree: means match to 2 dp in every cell and the labeled total is
identical (334,537) in both variants; cell counts and the 3rd decimal can differ
by a few instances because tercile boundaries are tie-sensitive
(`pd.qcut(rank(method='first'), 3)` in closure-C vs `ntile(3)` here).

The sub-second streamed number is the *cheapest possible* re-read of the corpus
— the query only needs three thin columns per row, so projection pushdown makes
the scan itself cheap. What the DB actually eliminates is the **repeated
extraction**: today each of B/C/G/H re-streamed the corpus and re-parsed
messages/patches to rebuild its frame (≈13–30 min per job), where the frame
columns and behaviour features are now built **once** (34 m 34 s extraction +
52 s DB build) and every query reads tables.

## Costs and caveats

- The DB file is ~4 GB (gitignored); the yoonholee `steps` text dominates. If
  the TB2 step text is never needed, drop `tb2_traj_yoonholee` to shrink it.
- `trajectory` joins `trajectory_features` inner — rebuild both from the same
  corpus snapshot; a partial extraction merge is detected by the row-count guard
  in `extract_trajectory_features.py` (row guard per shard) but the wide-table
  build itself does not re-validate 511,668 = 511,668. Run the extraction to
  completion before `build_analysis_db.py`.
- IRT outputs (closure-C `outputs/task_irt.parquet` etc.) do not exist on disk
  in any worktree (only `analytics/research/irt_summary.md`), so no IRT table is
  loaded; the fit is reproduced by `scripts/fit_irt.py`.
- `catalog_check.py` lints the analysis tables against `analysis.duckdb`
  (`analysis_table` registry type); `tests/test_analysis_db.py` covers the
  extraction on a 1-shard fixture and the build on a tiny fixture tree.
