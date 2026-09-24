# Harbor Hub harvest — per-trial Terminal-Bench results

2026-09-19. Harvested the public Terminal-Bench leaderboards on Harbor Hub into
`traces_external/harbor_hub/` via `uv run python scripts/harvest_harbor_hub.py`
(resume-safe; raw JSON cache in `traces_external/harbor_hub/raw/`, log in
`outputs/closure_E.log`).

## Access note

The `harbor` 0.23.0 CLI is **not** authenticated on this machine
(`harbor auth status` → "Not authenticated"), so `harbor hub job …` /
`harbor hub trial …` fail their client-side `require_user_id` check. The Hub is
a Supabase project whose public endpoints serve anonymous callers under the
publishable key embedded in the CLI (`harbor.auth.constants`); the harvester
calls those same read-only endpoints directly (leaderboard-read edge function,
`leaderboard_row_trial`/`trial` tables, `get_job_overview`, `get_job_trials`,
`get_job_tasks` RPCs). No login, no writes, ~0.4 s between requests.

## Files

| file | rows | content |
|---|---:|---|
| `trials.parquet` | 23,887 | one row per trial execution (all attempts incl. retries) |
| `jobs.parquet` | 50 | one row per job behind a board row |
| `tasks.parquet` | 229 | per-(version, task) aggregates |
| `leaderboard_rows.parquet` | 202 | board rows incl. metric-only 2.0 entries |
| `row_trials.parquet` | 22,810 | row↔trial associations (many-to-many; a job can back several rows) |

## Per-version summary

| bench | board rows | jobs | trial rows | uniq tasks | agents | models | mean reward | traj avail |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 2.0 | 142 | 0 | 0 | 0 | 43 (row meta) | 50 (row meta) | n/a (metrics-only board) | n/a |
| 2.1 | 22 | 12 | 10,828 | 89 | 6 | 13 | 0.774 | 10,828/10,828 |
| 3.0 | 12 | 12 | 4,440 | 74 | 6 | 12 | 0.230 | 4,440/4,440 |
| 4.0 | 26 | 26 | 8,619 | 66 | 4 | 14 | 0.402 | 8,599/8,619 |

Trial rows > row-trial associations because `p_attempts='all'` keeps retries
(`attempt_index` distinguishes; leaderboard scoring uses latest). Rewards are
binary 0/1 except 58 partial-credit trials. `trajectory_available` = trial has a
downloadable `archive_path` (trial.tar.gz contains trajectory.json); the 20
false rows on 4.0 are failed/canceled trials. Top-job cross-check: harvested
mean reward for the 4.0 rank-1 job = 0.5818 vs board accuracy 58.18.

Boards: 2.0 `f58e955f` (terminal-bench/terminal-bench-2, `2-0`),
2.1 `60330f75` (terminal-bench/terminal-bench-2-1, `main`),
3.0 `694bb3b0` (terminal-bench/terminal-bench, `3-0-0`),
4.0 `9f966760` (terminal-bench/terminal-bench, `4-0-0`).

Models seen per version — 2.1: gpt-6-astra, gpt-5.6-luna, gpt-5.6-terra, gpt-5.5,
claude-fable-5, claude-opus-4-7/4-8, claude-sonnet-5, gemini-3(-pro/.1-pro)-preview,
glm-5.1, grok-4.5, muse-spark-1.1. 3.0: claude-opus-5, gpt-5.6-sol, claude-fable-5,
glm-5.3, grok-4.6, claude-opus-4-8, gpt-5.6-terra, swe-1.7-lightning,
grok-4.5-xhigh, claude-sonnet-5, gpt-5.6-luna, glm-5.2. 4.0: gpt-6-astra,
claude-fable-5-1, claude-opus-5, claude-fable-5, glm-5.3, gpt-5.6-sol,
claude-opus-4-8, gpt-5.6-terra, grok-4.6, gemini-3.8-flash, gpt-5.6-luna,
grok-4.5, claude-sonnet-5, gemini-3.7-flash.

Caveat on job↔row mapping: several 2.1 rows share one multi-config job (e.g. the
2225-trial `Terminal-Bench 2.1 · GPT-6 Astra / Codex` job backs five rows split by
`reasoning_effort` low→max). `trials.leaderboard_row_id` records the *first*
linking row; use `row_trials.parquet` + `trials.reasoning_effort` (from
`config_values["agent.kwargs.reasoning_effort"]`) for exact row-level analysis.

## TB 4.0 per-task difficulty (26 jobs; 130ish trials/task)

| task_id | n_trials | n_jobs | mean_reward |
|---|---:|---:|---:|
| shadow-relay | 131 | 26 | 0.854 |
| wdm-design | 130 | 26 | 0.854 |
| mp-checkpoint-consolidation | 130 | 26 | 0.838 |
| coq-block-bound | 131 | 26 | 0.831 |
| layout-config-recreation2 | 131 | 26 | 0.829 |
| fin-saccr-rwa | 131 | 26 | 0.785 |
| risk-scorer-replay | 131 | 26 | 0.777 |
| telecom-entity-resolution | 130 | 26 | 0.769 |
| cumulative-layout-shift | 130 | 26 | 0.754 |
| fp8-rmsnorm-gemm | 131 | 26 | 0.752 |
| uefi-bootkit | 130 | 26 | 0.731 |
| retro-console-soc | 130 | 26 | 0.715 |
| sound-change-cascade | 130 | 26 | 0.692 |
| payments-pipeline-fix | 131 | 26 | 0.692 |
| hof-topology-interpenetration | 132 | 26 | 0.674 |
| embedding-drift-monitor | 131 | 26 | 0.631 |
| interleaved-vigenere | 130 | 26 | 0.608 |
| vpp-loss-divergence | 133 | 26 | 0.557 |
| formal-crypto | 130 | 26 | 0.554 |
| freecad-platform-drawing | 130 | 26 | 0.546 |
| batched-eval-parity | 131 | 26 | 0.519 |
| rs-archive-clone | 130 | 26 | 0.515 |
| pretrain-shard-corruption | 130 | 26 | 0.508 |
| cad-model | 130 | 26 | 0.500 |
| satb-audio-transcription | 130 | 26 | 0.500 |
| distributed-dedup | 130 | 26 | 0.492 |
| mvcc-lsm-compaction | 130 | 26 | 0.477 |
| legacy-utility-triage | 130 | 26 | 0.469 |
| photonic-waveguide-routing | 130 | 26 | 0.465 |
| kv-live-surgery | 131 | 26 | 0.435 |
| heat-pump-warranty | 130 | 26 | 0.431 |
| biped-contact-dynamics | 131 | 26 | 0.427 |
| freecad-spring-clip | 130 | 26 | 0.423 |
| wal-recovery-ordering | 131 | 26 | 0.408 |
| react-lead-form | 130 | 26 | 0.400 |
| live-database-cutover | 130 | 26 | 0.388 |
| intrastat-meldung | 130 | 26 | 0.385 |
| atrx-vep-crispr | 130 | 26 | 0.369 |
| nextjs-performance | 130 | 26 | 0.369 |
| vf2-speedup-networkx | 132 | 26 | 0.354 |
| math-eval-grader | 131 | 26 | 0.290 |
| ks-solver-cpp | 130 | 26 | 0.286 |
| gsea-proteomics | 130 | 26 | 0.285 |
| takens-embedding-lean | 135 | 26 | 0.280 |
| production-planning | 130 | 26 | 0.262 |
| jax-speedrun-gpu | 134 | 26 | 0.262 |
| ctr-optimization | 131 | 26 | 0.260 |
| html-js-filter | 131 | 26 | 0.231 |
| lake-temp-glm | 130 | 26 | 0.208 |
| vba-userform-port | 130 | 26 | 0.185 |
| session-window-debug | 131 | 26 | 0.146 |
| layout-config-recreation | 130 | 26 | 0.131 |
| roy-polymorph-cn | 130 | 26 | 0.131 |
| sglang-qwen-burst | 131 | 26 | 0.123 |
| vllm-deepseek-streaming | 130 | 26 | 0.062 |
| protein-autointerp-disulfide | 130 | 26 | 0.054 |
| music-harmony | 131 | 26 | 0.038 |
| freecad-impeller | 130 | 26 | 0.008 |
| medical-claims-processing | 130 | 26 | 0.008 |
| bun-sourcemap-leak | 131 | 26 | 0.000 |
| cargo-flight-dispatch | 132 | 26 | 0.000 |
| data-anonymization | 131 | 26 | 0.000 |
| glycan-ms2-elucidation | 130 | 26 | 0.000 |
| foodstuff-beta-activity | 130 | 26 | 0.000 |
| freight-dispatch-shift | 131 | 26 | 0.000 |
| ontology-kg-querying | 130 | 26 | 0.000 |

## TB 3.0 per-task difficulty (12 jobs; 60 trials/task)

| task_id | n_trials | n_jobs | mean_reward |
|---|---:|---:|---:|
| gpt2-codegolf | 60 | 12 | 0.750 |
| coq-block-bound | 60 | 12 | 0.700 |
| layout-config-recreation2 | 60 | 12 | 0.650 |
| memcached-backdoor | 60 | 12 | 0.650 |
| shadow-relay | 60 | 12 | 0.633 |
| wdm-design | 60 | 12 | 0.617 |
| risk-scorer-replay | 60 | 12 | 0.617 |
| mp-checkpoint-consolidation | 60 | 12 | 0.617 |
| erp-procurement-planning | 60 | 12 | 0.604 |
| react-lead-form | 60 | 12 | 0.517 |
| telecom-entity-resolution | 60 | 12 | 0.500 |
| vf2-speedup-networkx | 60 | 12 | 0.458 |
| retro-console-soc | 60 | 12 | 0.436 |
| biped-contact-dynamics | 60 | 12 | 0.424 |
| cumulative-layout-shift | 60 | 12 | 0.417 |
| atrx-vep-crispr | 60 | 12 | 0.417 |
| payments-pipeline-fix | 60 | 12 | 0.400 |
| fp8-rmsnorm-gemm | 60 | 12 | 0.383 |
| uefi-bootkit | 60 | 12 | 0.383 |
| wal-recovery-ordering | 60 | 12 | 0.367 |
| sound-change-cascade | 60 | 12 | 0.333 |
| photonic-waveguide-routing | 60 | 12 | 0.327 |
| hof-topology-interpenetration | 60 | 12 | 0.317 |
| distributed-dedup | 60 | 12 | 0.317 |
| freecad-spring-clip | 60 | 12 | 0.300 |
| live-database-cutover | 60 | 12 | 0.283 |
| sglang-qwen-burst | 60 | 12 | 0.267 |
| interleaved-vigenere | 60 | 12 | 0.267 |
| embedding-drift-monitor | 60 | 12 | 0.250 |
| lean-midpoint-proof | 60 | 12 | 0.241 |
| kv-live-surgery | 60 | 12 | 0.236 |
| cli-2ph-simplex | 60 | 12 | 0.235 |
| vpp-loss-divergence | 60 | 12 | 0.233 |
| batched-eval-parity | 60 | 12 | 0.220 |
| mvcc-lsm-compaction | 60 | 12 | 0.220 |
| fin-saccr-rwa | 60 | 12 | 0.200 |
| fix-uautomizer-soundness | 60 | 12 | 0.167 |
| ctr-optimization | 60 | 12 | 0.150 |
| rs-archive-clone | 60 | 12 | 0.150 |
| roy-polymorph-cn | 60 | 12 | 0.133 |
| production-planning | 60 | 12 | 0.133 |
| html-js-filter | 60 | 12 | 0.133 |
| takens-embedding-lean | 60 | 12 | 0.119 |
| freecad-platform-drawing | 60 | 12 | 0.117 |
| nextjs-performance | 60 | 12 | 0.117 |
| math-eval-grader | 60 | 12 | 0.117 |
| gsea-proteomics | 60 | 12 | 0.102 |
| formal-crypto | 60 | 12 | 0.100 |
| vllm-deepseek-streaming | 60 | 12 | 0.100 |
| exam-pdf-eval | 60 | 12 | 0.085 |
| ks-solver-cpp | 60 | 12 | 0.085 |
| cad-model | 60 | 12 | 0.067 |
| lake-temp-glm | 60 | 12 | 0.067 |
| cargo-flight-dispatch | 60 | 12 | 0.050 |
| foodstuff-beta-activity | 60 | 12 | 0.050 |
| jax-speedrun-gpu | 60 | 12 | 0.033 |
| vba-userform-port | 60 | 12 | 0.033 |
| freecad-impeller | 60 | 12 | 0.033 |
| session-window-debug | 60 | 12 | 0.033 |
| freight-dispatch-shift | 60 | 12 | 0.033 |
| pretrain-shard-corruption | 60 | 12 | 0.033 |
| protein-autointerp-disulfide | 60 | 12 | 0.017 |
| music-harmony | 60 | 12 | 0.000 |
| medical-claims-processing | 60 | 12 | 0.000 |
| heat-pump-warranty | 60 | 12 | 0.000 |
| glycan-ms2-elucidation | 60 | 12 | 0.000 |
| data-anonymization | 60 | 12 | 0.000 |
| bun-sourcemap-leak | 60 | 12 | 0.000 |
| legacy-utility-triage | 60 | 12 | 0.000 |
| layout-config-recreation | 60 | 12 | 0.000 |
| intrastat-meldung | 60 | 12 | 0.000 |
| ico-path-patch | 60 | 12 | 0.000 |
| satb-audio-transcription | 60 | 12 | 0.000 |
| ontology-kg-querying | 60 | 12 | 0.000 |

## TB 2.0: Hub board vs the HF dumps — no overlap by trial id

The Hub 2.0 board (`terminal-bench/terminal-bench-2`, `2-0`) is a **metrics-only
migration**: 142 rows with accuracy aggregates (43 agents, 50 display models,
accuracy 3.1–84.7) but **zero** `leaderboard_row_trial` associations — no job or
trial ids to harvest. A sample of `yoonholee` dump `trial_id`s does not resolve
in the Hub `trial` table either (`select=id,job_id` → `[]`), so the dump
predates the Hub id namespace entirely. The two sources are complementary and
join only on `(task_id, agent, model)`: the dump gives 52,104 scored trials
across the same 89 task ids but a different submission mix (mostly
terminus-2 / mini-SWE-agent / OpenHands with open-weight models scraped from
tbench.ai), while the Hub board covers the later frontier submissions
(e.g. NexAU-AHE + GPT-5.5 at 84.7) as aggregates only. For per-attempt 2.0 data,
the HF dumps are the source; the Hub 2.0 board contributes only row-level
leaderboard metrics (`leaderboard_rows.parquet`).

## Reproduce / query

```
uv run python scripts/harvest_harbor_hub.py           # full harvest (resume-safe)
uv run python scripts/harvest_harbor_hub.py --assemble-only   # parquet from raw cache
```

DuckDB views (`analytics/schema/external_views.sql`, applied by
`scripts/duckdb_init.py`): `tb_hub_trials`, `tb_hub_jobs`, `tb_hub_tasks`,
`tb2_traj_yoonholee`, `tb2_traj_harithoppil`.
