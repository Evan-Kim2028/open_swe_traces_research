# DATA_CATALOG.md — every table, view, and parquet in this project

Maintained by the closure-F data-catalog job (2026-09-19). `scripts/catalog_check.py`
(`uv run python scripts/catalog_check.py`, also `tests/test_catalog.py`) fails if a
parquet under `outputs/derived/`, a view/table in `analytics/schema/*.sql`, or a
`experiments/pipeline/state.db` table is not registered below — and vice versa.

## Registry

The registry is the machine-checked index; every artifact documented below must appear
here, and every entry must have a `### <name>` section in this file (external dumps are
checked by name-in-prose).

| type | name |
|---|---|
| view | traces_raw |
| view | traces |
| view | catalog_columns |
| view | catalog_download |
| view | catalog_by_harness |
| view | catalog_resolved_rates |
| table | trace_summary_by_language |
| table | trace_summary_by_category |
| table | trace_empty_patch_rate |
| view | trace_turns |
| view | trial_summary |
| runtime_table | _config |
| macro | refresh_trace_summaries |
| derived_view | osw_instance_closure_proxies |
| derived_view | osw_instance_rung |
| derived_view | osw_trajectory_frame |
| derived_view | unit_closure_metrics |
| derived_view | harbor_hub_jobs |
| derived_view | harbor_hub_leaderboard_rows |
| derived_view | harbor_hub_row_trials |
| derived_view | harbor_hub_tasks |
| derived_view | harbor_hub_trials |
| parquet | outputs/derived/closure_proxies.parquet |
| parquet | outputs/derived/rung_features.parquet |
| parquet | outputs/derived/trajectory_frame.parquet |
| parquet | outputs/derived/closure_metrics.parquet |
| parquet | outputs/derived/harbor_hub/jobs.parquet |
| parquet | outputs/derived/harbor_hub/leaderboard_rows.parquet |
| parquet | outputs/derived/harbor_hub/row_trials.parquet |
| parquet | outputs/derived/harbor_hub/tasks.parquet |
| parquet | outputs/derived/harbor_hub/trials.parquet |
| sqlite_table | meta |
| sqlite_table | repos |
| sqlite_table | units |
| sqlite_table | steps |
| sqlite_table | trials |
| sqlite_table | events |
| sqlite_table | tokens |
| sqlite_table | sqlite_sequence |
| external | harithoppil__terminal-bench-2-trajectories |
| external | yoonholee__terminalbench-trajectories |

## Conventions, join keys, and naming inconsistencies

### Join keys

| key | tables that carry it |
|---|---|
| `instance_id` | `traces`, `trace_turns`, `trial_summary`, `osw_instance_closure_proxies`, `osw_instance_rung`, `osw_trajectory_frame` |
| `trajectory_id` | `traces`, `trace_turns`, `trial_summary`, `osw_trajectory_frame` |
| `(repo, unit, level, solver, attempt)` | `trials` (state.db); `unit_closure_metrics` carries `(repo, unit)`; `units` carries `(repo, unit)` |
| `task_id` | harbor `tasks`, `trials`; TB2 uses `task_name` instead |
| `trial_id` | harbor `row_trials`, `trials`, TB2 yoonholee |
| `leaderboard_row_id` | harbor `jobs`, `leaderboard_rows`, `trials` |

### Naming inconsistencies (documented here — columns are never renamed)

1. **`trials.level` (state.db) vs `flip` (results.md / `results.parquet`).** `level` is
   the ladder level a single trial was run at (0–6). `flip` is the *measured flip
   point* for a (unit, solver): the lowest level with ≥ 3 attempts and
   passes/attempts ≥ 2/3 over scored trials (C6 policy; `flip_point_for` in
   `src/openswe_traces/results.py`), rendered `L<k>`, or `none` when tried but never
   passed. `units.predicted_flip` is the author's *predicted* flip (`L<k>`), distinct
   from both. The closure-metrics parquet encodes the same flips numerically
   (`flip_*_enc`: level k → k, `none` → max tried + 1).
2. **`solver` `cursor` vs Composer.** In state.db and results.md, `solver = 'cursor'`
   means the Composer 2.5 solver invoked through the cursor-agent CLI
   (`experiments/pipeline/config.yaml`, `solver_order: [cursor, devin]`);
   `'devin'` means Devin swe-2-max. Do not expect a solver literally named "Composer".
   `units.author_backend` ('devin', …) is the *authoring* backend, a different axis.
3. **`trajectory_frame.teacher` vs `traces.teacher_model`.** Same concept (the teacher
   model that produced the trajectory) — short name in the frame, full name in
   `traces`. `harness`/`teacher`/`source` are decomposed from the shard path
   `data/<harness>/<teacher>/<source>/<file>.parquet`.
4. **Ladder labels.** Heuristic rungs in `rung_features.rung` are L0–L6 by the
   `openswe_traces.rungs` classifier (0 = symptom-only … 6 = all tests in text). The
   Harbor authoring ladder (`analytics/research/verifier_rules.md`) previously used
   A0–A4; map `L = A + 2` (L5 ≡ A3, L6 ≡ A4). Nothing exists below L0 by construction.
5. **`resolved` encoding.** `traces.resolved` is `1`/`0`/`-1` (resolved / failed /
   unknown). `trial_summary.outcome_label` renders the same field as
   `'success'`/`'failed'`/`'unknown'`; `closure_proxies.n_resolved` counts only
   `resolved = 1` over labeled rollouts.
6. **`task_id` vs `task_name`.** Harbor Hub uses opaque `task_id` UUIDs plus a
   human-readable `task_name`; the TB2 dumps use `task_name` only.

---

## Part 1 — Raw corpus views (DuckDB, `analytics/schema/002_views_raw.sql`)

Built by `uv run python scripts/duckdb_init.py`. Views re-read parquet on every query,
so new shards appear as the download grows. Upstream: `traces_data/` (HF
`nvidia/Open-SWE-Traces`, 212 shards, 511,668 trajectories, 42,413 instances as of
2026-09-19). `{{PARQUET_GLOB}}` is substituted by `scripts/duckdb_init.py` →
`openswe_traces.data.init_db`.

### traces_raw

Grain: one row per trajectory row in the raw corpus parquet shards. Key:
`trajectory_id` (plus `instance_id`, `repo`, `resolved`). Produced by `002_views_raw.sql`
(`CREATE OR REPLACE VIEW traces_raw`).

| column | type | meaning | unit |
|---|---|---|---|
| * (all raw columns) | — | every column of the underlying HF shard: `instance_id`, `repo`, `license`, `language`, `trajectory_id`, `resolved`, `metadata` (struct), `messages` (list), `tools` (list), … | — |
| `_source_file` | VARCHAR | absolute path of the shard the row came from (`filename=true`) | — |

### traces

Grain: one row per trajectory (trial). Keys: `trajectory_id`, `instance_id`, `repo`.
Produced by `002_views_raw.sql`. Upstream: `traces_raw`. Date produced: recreated on
every `duckdb_init.py` run (2026-09-19).

| column | type | meaning | unit |
|---|---|---|---|
| `instance_id` | VARCHAR | Open-SWE-Traces task instance id | — |
| `repo` | VARCHAR | repository name (e.g. `minisweagent/qwen36_27b/scale-swe` is a harness path, not a repo) — actual repo of the task | — |
| `license` | VARCHAR | corpus license field | — |
| `language` | VARCHAR | primary language of the repo | — |
| `trajectory_id` | VARCHAR | unique trajectory id | — |
| `resolved` | INTEGER | 1 resolved / 0 failed / -1 unknown | — |
| `num_messages` | INTEGER | `len(messages)` — number of messages in the trajectory | count |
| `num_tools` | INTEGER | `len(tools)` — number of tool definitions | count |
| `category` | VARCHAR | `metadata.category` (e.g. harness category) | — |
| `gold_files` | INTEGER | `metadata.reference_patch.num_modified_files` | files |
| `gold_lines` | INTEGER | `metadata.reference_patch.num_modified_lines` | lines |
| `gold_patch` | VARCHAR | reference (gold) patch text | — |
| `model_files` | INTEGER | `metadata.model_patch.num_modified_files` | files |
| `model_lines` | INTEGER | `metadata.model_patch.num_modified_lines` | lines |
| `model_patch` | VARCHAR | model patch text | — |
| `_source_file` | VARCHAR | shard path the row came from | — |
| `harness` | VARCHAR | regexp from `_source_file` (`data/<harness>/…`) | — |
| `teacher_model` | VARCHAR | regexp from `_source_file` (`data/<harness>/<teacher>/…`) | — |
| `source_dataset` | VARCHAR | regexp from `_source_file` (`data/<harness>/<teacher>/<source>/…`) | — |
| `messages` | LIST | full message list (role/content/tool_calls) | — |
| `tools` | LIST | tool definitions | — |

Caveats: `harness`/`teacher_model`/`source_dataset` are path-derived; rows whose shard
path does not match `data/<h>/<t>/<s>/` get NULLs. `gold_*`/`model_*` use `try_cast`,
so malformed metadata yields NULL rather than an error.

---

## Part 2 — Catalog views (`analytics/schema/003_views_catalog.sql`)

Lightweight inventory views over `traces`.

### catalog_columns

Grain: one row per column of `traces_raw`. Produced by `003_views_catalog.sql`
(`DESCRIBE SELECT * FROM traces_raw`). Columns: `column_name`, `column_type`, `null`,
`key`, `default`, `extra` (DuckDB `DESCRIBE` output; `extra` contains e.g.
`COLLATE NOCASE` or `auto_increment` for the `_source_file` row). Caveat: it describes
the *union* schema of whatever shards exist — grows as downloads land.

### catalog_download

Grain: one row (download-level summary). Columns: `parquet_files` (INTEGER, distinct
shard files), `trajectory_rows` (INTEGER, row count), `example_file` (VARCHAR, min
`_source_file`).

### catalog_by_harness

Grain: one row per `(harness, teacher_model, source_dataset, language, resolved)`.
Columns: `harness`, `teacher_model`, `source_dataset`, `language`, `resolved` (VARCHAR /
INTEGER as in `traces`), `n` (INTEGER, trajectory count), `avg_messages` (DOUBLE,
`round(avg(num_messages),1)`). Ordered by `n DESC`.

### catalog_resolved_rates

Grain: one row per `(language, harness)`. Columns: `language`, `harness`, `n`
(INTEGER), `pct_resolved`, `pct_failed`, `pct_unknown` (DOUBLE, percent of rows with
`resolved` = 1 / 0 / -1). Caveat: percentages may not sum to 100 when `resolved` is
NULL.

---

## Part 3 — Materialized summary tables (`analytics/schema/004_summaries.sql`)

Materialized snapshots (`CREATE OR REPLACE TABLE`). Rebuilt only with
`uv run python scripts/duckdb_init.py --refresh-summaries` — refresh after large
download chunks.

### trace_summary_by_language

Grain: one row per `(language, resolved)`. Columns: `language`, `resolved` (INTEGER
1/0/-1), `n` (INTEGER), `avg_messages` (DOUBLE), `avg_model_lines` (DOUBLE),
`avg_gold_lines` (DOUBLE).

### trace_summary_by_category

Grain: one row per `(category, resolved)`. Columns: `category`, `resolved`, `n`,
`avg_messages`.

### trace_empty_patch_rate

Grain: one row per `(harness, teacher_model)`. Columns: `harness`, `teacher_model`,
`n` (INTEGER), `pct_empty_model_patch` (DOUBLE, percent of rows with empty/`coalesce`
NULL `model_patch`).

### _config

Grain: one row (runtime metadata). Created by `openswe_traces.data.init_db`
(`CREATE TABLE IF NOT EXISTS _config`), **not** by a schema file. Columns:
`parquet_glob` (VARCHAR), `project_root` (VARCHAR), `updated_at` (TIMESTAMP, set on
each `duckdb_init.py` run). Produced 2026-09-19.

### refresh_trace_summaries

`CREATE OR REPLACE MACRO` stub in `001_macros.sql`; the real refresh is procedural and
lives in `004_summaries.sql`, invoked by `duckdb_init.py --refresh-summaries`. Not a
table.

---

## Part 4 — Turn / trial views (`analytics/schema/005_views_turns.sql`)

### trace_turns

Grain: one row per message (turn) within a trajectory. Keys: `trajectory_id`,
`instance_id`. Produced by `005_views_turns.sql` (`unnest(messages) WITH ORDINALITY`).
Upstream: `traces`.

| column | type | meaning | unit |
|---|---|---|---|
| `trajectory_id` | VARCHAR | trajectory (trial) id | — |
| `instance_id` | VARCHAR | task instance id | — |
| `repo`, `language`, `harness`, `teacher_model`, `resolved`, `category` | — | inherited from `traces` | — |
| `turn_idx` | BIGINT | 1-based ordinal of the message in `messages` | — |
| `role` | VARCHAR | `msg.role` (e.g. `user` / `assistant` / `tool`) | — |
| `content_len` | INTEGER | `coalesce(length(msg.content),0)` | chars |
| `tool_name` | VARCHAR | `json_extract_string(msg,'$.name')` for tool messages | — |
| `has_tool_call` | BOOLEAN | assistant message with non-null `tool_calls` | — |
| `msg` | STRUCT | the full message struct | — |

Caveat: heavy full-table scan — use `WHERE`/`LIMIT` during exploration.

### trial_summary

Grain: one row per trajectory. Keys: `trajectory_id`, `instance_id`. Produced by
`005_views_turns.sql`. Columns: all `traces` columns listed in Part 1 plus
`empty_model_patch` (BOOLEAN, `coalesce(model_patch,'') = ''`) and `outcome_label`
(VARCHAR, `success`/`failed`/`unknown` from `resolved`). Cheap — no message unpacking.

---

## Part 5 — Derived parquets (`outputs/derived/`)

Copied (never symlinked) from sibling worktrees by
`uv run python scripts/sync_derived.py`; provenance + md5s in
`outputs/derived/SOURCES.md`. Registered as DuckDB views by
`analytics/schema/derived_views.sql` (see Part 8). Column types are the parquet
physical types.

### outputs/derived/closure_proxies.parquet

Grain: one row per `instance_id` (one representative gold patch per instance, `max`
kept across shards; identical patches across shards). View: `osw_instance_closure_proxies`.
Produced by `scripts/closure_proxies.py` (oswt-closureB)
→ `openswe_traces.closure_proxies.main_closure_proxies`, one streaming DuckDB pass
(resume-safe parts in `closure_proxies_parts/`). Upstream: `traces_data` shards +
gold patches. Produced 2026-09-19. Rows: 42,413 (38,294 with `n_labeled >= 3`).

| column | type | meaning | unit |
|---|---|---|---|
| `instance_id` | VARCHAR | task instance id — join key | — |
| `repo` | VARCHAR | repo of the task | — |
| `language` | VARCHAR | primary language | — |
| `n_rollouts` | INTEGER | labeled rollouts for the instance across shards | count |
| `n_labeled` | INTEGER | rollouts with `resolved in (0,1)` | count |
| `n_resolved` | INTEGER | rollouts with `resolved = 1` | count |
| `gold_patch_lines` | INTEGER | size of the representative gold patch | lines |
| `gold_patch_files` | INTEGER | files touched by the gold patch | files |
| `has_gold_patch` | INTEGER | 1 if gold patch non-empty | 0/1 |
| `n_hunks` | DOUBLE | `@@` hunks in the patch | count |
| `n_files` | DOUBLE | `diff --git` headers | files |
| `added_lines` | DOUBLE | `+` content lines inside hunks | lines |
| `removed_lines` | DOUBLE | `-` content lines inside hunks | lines |
| `n_new_defs` | DOUBLE | added lines defining a symbol (per-language regex; first match per line) | count |
| `internal_refs` | DOUBLE | occurrences, in added lines, of names defined elsewhere in the same patch (defining line and recursive self-calls excluded) | count |
| `boundary_refs` | DOUBLE | occurrences, in added lines, of identifiers from context/removed lines that are not new defs (keyword stoplist per language; builtins kept) | count |
| `ratio` | DOUBLE | `internal_refs / max(1, boundary_refs)` | — |
| `new_frac` | DOUBLE | `n_new_defs / max(1, added_lines)` | — |
| `edit_frac` | DOUBLE | fraction of hunks with both removed and added lines | 0..1 |

Caveats: `ratio` is 0 for patches with no new defs and/or no boundary refs (20,461 of
42,413 rows) — include `n_new_defs`/`boundary_refs` when interpreting. Unknown
languages get no def regex → `n_new_defs = internal_refs = 0`. Correlations use the
`n_labeled >= 3` subset. `solve_rate = n_resolved / n_labeled` is derived, not stored.

### outputs/derived/rung_features.parquet

Grain: one row per `instance_id`. View: `osw_instance_rung`. Produced by
`scripts/rung_mapping.py --stream-only` (oswt-closureC)
→ `openswe_traces.rung_analysis.main_rung_mapping` (parts in `rung_features_parts/`).
Upstream: `traces_data` task text (`messages[2].content`) + reference patch. Produced
2026-09-19. Rows: 42,413.

| column | type | meaning | unit |
|---|---|---|---|
| `instance_id` | VARCHAR | task instance id — join key | — |
| `rung` | BIGINT | heuristic ladder rung L0–L6 (0 = symptom only … 6 = multiple test files/bodies in text) | level |
| `task_text` | VARCHAR | task text with harness boilerplate stripped | — |
| `leakage_symbols` | VARCHAR | JSON list of gold-patch symbols appearing in the task text | — |
| `content_len` | BIGINT | length of `messages[2].content` | chars |
| `n_rows` | BIGINT | shard rows aggregated for this instance | count |
| `word_count` | BIGINT | word count of the stripped task text | count |
| `has_repro` | BIGINT | repro/steps-to-reproduce present | 0/1 |
| `has_expected_actual` | BIGINT | expected/actual-behavior contract present | 0/1 |
| `has_stack_trace` | BIGINT | stack trace / error text present | 0/1 |
| `has_test_names` | BIGINT | named tests referenced | 0/1 |
| `has_signature` | BIGINT | exported signatures / API stubs present | 0/1 |
| `has_leakage` | BIGINT | leaked symbols detected | 0/1 |
| `leakage_count` | BIGINT | number of leaked symbols | count |
| `has_test_code` | BIGINT | substantial test body in the text | 0/1 |
| `n_test_funcs_in_text` | BIGINT | test functions found in the text | count |
| `n_test_files_in_text` | BIGINT | test files found in the text | count |
| `patch_has_tests` | BIGINT | reference patch touches test files | 0/1 |

Caveats: purely heuristic (regex) — no model calls; `task_text` assumes the user
instruction is `messages[2]`. The framework tests use `rung_binary = rung >= 2`.
`rung` here is the corpus-task heuristic; it is **not** the Harbor authored-ladder
level of `state.db.trials.level` (both are 0–6 but measured differently).

### outputs/derived/trajectory_frame.parquet

Grain: one row per trajectory. Keys: `trajectory_id`, `instance_id`. View:
`osw_trajectory_frame`. Produced by `scripts/framework_tests.py` (oswt-closureC)
→ `openswe_traces.framework_tests.main_framework_tests` (parts in
`trajectory_frame_parts/`). Upstream: `traces_data` (`metadata.model_patch` /
`metadata.reference_patch`, `resolved`). Produced 2026-09-19. Rows: 511,668.

| column | type | meaning | unit |
|---|---|---|---|
| `trajectory_id` | VARCHAR | trajectory id — join key | — |
| `instance_id` | VARCHAR | task instance id — join key | — |
| `repo` | VARCHAR | repo of the task | — |
| `language` | VARCHAR | primary language | — |
| `harness` | VARCHAR | harness (from shard path) | — |
| `teacher` | VARCHAR | teacher model (short name; = `traces.teacher_model`) | — |
| `source` | VARCHAR | source dataset (from shard path) | — |
| `resolved` | TINYINT | 1 / 0 / -1 | — |
| `patch_file_jaccard` | DOUBLE | Jaccard of model-patch file set vs gold-patch file set (0..1; no NULLs, 0 when disjoint) | 0..1 |

Caveat: `teacher` is the short name — join to `traces` on `instance_id` +
`teacher_model` if full names are needed.

### outputs/derived/closure_metrics.parquet

Grain: one row per authored unit `(repo, unit)` with a base tree on disk. View:
`unit_closure_metrics`. Produced by `scripts/closure_metrics.py` (oswt-closureA).
Upstream: `experiments/pipeline/authored/<repo>/<unit>/_author/` (`excision.patch`,
`difficulty.md`), `experiments/pipeline/closure_A/trees.json` / `repos.yaml` base
trees, `experiments/pipeline/state.db` + HANDOFF.md (measured flips), Go helper
`tools/goclosure/main.go` (go/ast parsing). Produced 2026-09-19 (log DONE 18:10:00Z).
Rows: 38 (17 units computed + `connarray-cv` reuse row + 20 authored-patch-rebranded
helm/kops rows).

| column | type | meaning | unit |
|---|---|---|---|
| `repo` | VARCHAR | repo — join key to `units`/`trials` | — |
| `unit` | VARCHAR | unit name — join key | — |
| `family` | VARCHAR | `single-file` / `cross-file` / `sequence` (authoring family) | — |
| `predicted_flip` | DOUBLE | author-predicted flip level from `difficulty.md` (`predicted_flip: L<k>`), NULL if absent | level |
| `control` | BOOLEAN | unit marked `control: true` | — |
| `metric_source` | VARCHAR | `authored unit` / `authored patch rebranded to base tree` / `same authored unit as … (metrics reused)` | — |
| `n_files` | BIGINT | files in the excision patch | files |
| `n_funcs_removed` | BIGINT | functions the excision removes | count |
| `lines_removed` | BIGINT | lines removed by the excision | lines |
| `internal_edges` | BIGINT | call references among removed functions | count |
| `boundary_in` | BIGINT | call sites in the remaining tree referencing removed functions | count |
| `boundary_out` | BIGINT | call references from removed bodies to remaining symbols | count |
| `ratio` | DOUBLE | `internal_edges / max(1, boundary_in + boundary_out)` | — |
| `flip_devin` | VARCHAR | measured Devin flip label (`0`/`2`/`5`/`none`), NULL if no trials | — |
| `flip_devin_enc` | DOUBLE | encoded: level k → k, `none` → max tried + 1, NULL if no trials | level |
| `flip_cursor` | VARCHAR | measured cursor (Composer 2.5) flip label, NULL if no trials | — |
| `flip_cursor_enc` | DOUBLE | encoded as `flip_devin_enc` | level |
| `flip_note` | VARCHAR | provenance for manual flips (e.g. "devin flip from HANDOFF.md") | — |

Caveats: only units whose repo had a base tree in the closureA worktree are present
(gin/client-go/go-github; helm/kops via patch rebranding; goa absent; nats-server
seqset/subjecttree skipped). Measured flips come from state.db + HANDOFF.md, not from
a live solver. `ratio` is 0 when the closure has no boundary edges. Spearman analyses
are tiny-n (7 metrics × 38 rows).

### outputs/derived/harbor_hub/jobs.parquet

Grain: one row per Harbor Hub leaderboard job (a submitted agent×model run). View:
`harbor_hub_jobs`. Produced by `scripts/harvest_harbor_hub.py` (oswt-closureE)
→ `openswe_traces.external.harbor_hub.main`; raw JSON cached in
`traces_external/harbor_hub/raw/`. Upstream: Harbor Hub Supabase public REST endpoints
(leaderboard tables + `get_job_overview` RPC). Produced 2026-09-19. Rows: 50.

| column | type | meaning | unit |
|---|---|---|---|
| `bench_version` | VARCHAR | Terminal-Bench version (e.g. `2.0`) | — |
| `leaderboard_id` | VARCHAR | leaderboard id | — |
| `leaderboard_row_id` | VARCHAR | leaderboard row id — join key | — |
| `job_id` | VARCHAR | job UUID — join key to `trials` | — |
| `title` | VARCHAR | job title | — |
| `agent` | VARCHAR | agent scaffold | — |
| `agent_version` | VARCHAR | agent version | — |
| `model` | VARCHAR | underlying model | — |
| `model_provider` | VARCHAR | model provider | — |
| `n_tasks` | BIGINT | tasks attempted | count |
| `n_trials` | BIGINT | trials run | count |
| `mean_reward` | DOUBLE | mean trial reward | 0..1 |
| `rank` | BIGINT | leaderboard rank | — |
| `accuracy` | DOUBLE | pass rate | 0..1 |
| `cost_usd` | DOUBLE | total cost | USD |
| `tokens_in` | BIGINT | input tokens | tokens |
| `tokens_out` | BIGINT | output tokens | tokens |
| `n_errors` | BIGINT | erroring trials | count |
| `visibility` | VARCHAR | public/private | — |
| `owner_org` | VARCHAR | owning org | — |
| `submitted_at` | VARCHAR | submission timestamp (ISO string) | — |
| `finished_at` | VARCHAR | finish timestamp | — |

### outputs/derived/harbor_hub/leaderboard_rows.parquet

Grain: one row per leaderboard row (agent×model submission). View:
`harbor_hub_leaderboard_rows`. Same producer/upstream as `jobs`. Rows: 202.

| column | type | meaning | unit |
|---|---|---|---|
| `bench_version` | VARCHAR | Terminal-Bench version | — |
| `leaderboard_id` | VARCHAR | leaderboard id | — |
| `row_id` | VARCHAR | row UUID — join key (`leaderboard_row_id`) | — |
| `rank` | BIGINT | rank on the board | — |
| `status` | VARCHAR | submission status | — |
| `agent` | VARCHAR | agent scaffold | — |
| `model` | VARCHAR | model | — |
| `agent_org` | VARCHAR | agent org | — |
| `model_org` | VARCHAR | model org | — |
| `reasoning_effort` | VARCHAR | reasoning effort setting | — |
| `date` | VARCHAR | submission date | — |
| `job_url` | VARCHAR | link to the job | — |
| `source_url` | VARCHAR | source link | — |
| `accuracy` | DOUBLE | pass rate | 0..1 |
| `accuracy_ci95_half_width` | DOUBLE | half-width of the 95% CI on accuracy | — |
| `n_trials` | BIGINT | trials | count |
| `total_tokens` | DOUBLE | total tokens | tokens |
| `total_cost_usd` | DOUBLE | total cost | USD |
| `created_at` | VARCHAR | row created at | — |
| `updated_at` | VARCHAR | row updated at | — |

### outputs/derived/harbor_hub/row_trials.parquet

Grain: one row per (leaderboard row, trial) association. Keys: `row_id`, `trial_id`.
View: `harbor_hub_row_trials`. Rows: 22,810. Columns: `bench_version` (VARCHAR),
`leaderboard_id` (VARCHAR), `row_id` (VARCHAR), `trial_id` (VARCHAR). Pure join table.

### outputs/derived/harbor_hub/tasks.parquet

Grain: one row per task. Key: `task_id`. View: `harbor_hub_tasks`. Rows: 229.
Columns: `bench_version` (VARCHAR), `task_id` (VARCHAR), `n_trials` (BIGINT),
`n_jobs` (BIGINT), `mean_reward` (DOUBLE, 0..1), `n_passed` (BIGINT). Caveat:
`task_id` is an opaque UUID; join to `trials.task_id`; the human-readable name lives
in `trials.task_name`.

### outputs/derived/harbor_hub/trials.parquet

Grain: one row per trial (one agent attempt at one task). Keys: `trial_id`,
`task_id`, `job_id`, `leaderboard_row_id`. View: `harbor_hub_trials`. Rows: 23,887.

| column | type | meaning | unit |
|---|---|---|---|
| `bench_version` | VARCHAR | Terminal-Bench version | — |
| `leaderboard_id` | VARCHAR | leaderboard id | — |
| `leaderboard_row_id` | VARCHAR | row UUID — join key | — |
| `job_id` | VARCHAR | job UUID — join key | — |
| `job_title` | VARCHAR | job title | — |
| `agent` | VARCHAR | agent scaffold | — |
| `agent_version` | VARCHAR | agent version | — |
| `reasoning_effort` | VARCHAR | reasoning effort setting | — |
| `config_json` | VARCHAR | job config JSON | — |
| `model` | VARCHAR | model | — |
| `model_provider` | VARCHAR | model provider | — |
| `task_id` | VARCHAR | task UUID — join key | — |
| `task_name` | VARCHAR | human-readable task name | — |
| `trial_id` | VARCHAR | trial UUID — join key | — |
| `trial_name` | VARCHAR | trial label | — |
| `attempt_index` | BIGINT | 0-based attempt within the job | — |
| `n_attempts` | BIGINT | total attempts for the trial | count |
| `reward` | DOUBLE | trial reward | 0..1 |
| `passed` | BOOLEAN | reward ≥ threshold (scored pass) | — |
| `status` | VARCHAR | trial status | — |
| `is_scored` | BOOLEAN | whether the trial counts toward the leaderboard | — |
| `error_class` | VARCHAR | error class if errored | — |
| `started_at` | VARCHAR | start timestamp | — |
| `finished_at` | VARCHAR | finish timestamp | — |
| `wall_seconds` | DOUBLE | wall-clock duration | s |
| `tokens_in` | DOUBLE | input tokens | tokens |
| `tokens_out` | DOUBLE | output tokens | tokens |
| `cache_tokens` | DOUBLE | cached tokens | tokens |
| `cost_usd` | DOUBLE | cost | USD |
| `trajectory_available` | BOOLEAN | full trajectory downloadable | — |
| `harvested_at` | VARCHAR | when this row was harvested from Harbor Hub | — |

Caveat: snapshot at harvest time (2026-09-19); the live leaderboard moves. Timestamps
are ISO strings (VARCHAR), not typed timestamps.

---

## Part 6 — `experiments/pipeline/state.db` (SQLite)

The overnight-pipeline state machine (schema in `src/openswe_traces/pipeline/state.py`,
`PipelineStore`). Regenerated by the pipeline; gitignored. All `ts`/`updated_at`
values are ISO-8601 UTC strings. Statuses: `pending`/`running`/`done`/`failed`/
`skipped`/`rejected`/`paused`; stages: `prepare`, `author`, `verifier`, `package`,
`solve`, `aggregate`. Current row counts (2026-09-19): units 6, trials 12, steps 12.

### meta

Grain: one row per key. Columns: `key` (TEXT PK), `value` (TEXT). Key/value store
(empty in the current DB). Produced by `PipelineStore._init`.

### repos

Grain: one row per pipeline repo. Key: `name`. Columns: `name` (TEXT PK), `status`
(TEXT), `error` (TEXT), `updated_at` (TEXT). Tracks `prepare` stage per repo.

### units

Grain: one row per authored unit. Key: `(repo, unit)`. Columns: `repo` (TEXT),
`unit` (TEXT), `status` (TEXT), `family` (TEXT, `single-file`/`cross-file`/
`sequence`), `closure_json` (TEXT, closure summary), `n_files` (INTEGER),
`n_lines` (INTEGER), `rejected_rule` (TEXT, e.g. `A3`), `updated_at` (TEXT),
`predicted_flip` (INTEGER, author's predicted flip `L<k>`), `is_control` (INTEGER
0/1), `author_backend` (TEXT, e.g. `devin`). See naming note 1/2: `predicted_flip`
≠ measured `flip`; `author_backend` ≠ solver.

### steps

Grain: one row per (repo, unit, stage) step. Key: `(repo, unit, stage)`. Columns:
`repo` (TEXT), `unit` (TEXT, `''` for repo-level stages), `stage` (TEXT), `status`
(TEXT), `payload_json` (TEXT), `error` (TEXT), `updated_at` (TEXT).

### trials

Grain: one row per solver trial at a ladder level. Key: `(repo, unit, level, solver,
attempt)` (natural). Columns: `id` (INTEGER PK), `repo` (TEXT), `unit` (TEXT),
`level` (INTEGER 0–6 — the ladder level the trial ran at; see naming note 1),
`solver` (TEXT — `cursor` = Composer 2.5, `devin` = Devin swe-2-max; note 2),
`attempt` (INTEGER), `reward` (REAL 0..1), `tokens_in` (INTEGER), `tokens_out`
(INTEGER), `audit_class` (TEXT — `clean`, `contaminated`, `checksum`, `hacked`,
`infra`, timeout class; majority rolled up per unit), `wall_minutes` (REAL),
`job_dir` (TEXT), `excluded` (INTEGER 0/1), `timeout` (INTEGER 0/1). Caveats:
`excluded`/`timeout`/`audit_class` filter which trials count toward the flip point
(`aggregate.py` excludes `contaminated`/`checksum`/`hacked`/timeout/infra); the
current DB has 12 rows all `clean`, reward 1.0.

### events

Grain: one row per pipeline event. Columns: `id` (INTEGER PK), `ts` (TEXT), `kind`
(TEXT), `message` (TEXT). Append-only event log.

### tokens

Grain: one row per token-accounting entry. Columns: `id` (INTEGER PK), `source`
(TEXT), `tokens_in` (INTEGER), `tokens_out` (INTEGER), `ts` (TEXT).

### sqlite_sequence

SQLite-internal AUTOINCREMENT counter table (schema `(name, seq)`); do not join on it.

---

## Part 7 — External dumps (`traces_external/`)

Located in the main checkout `/home/evan/Documents/open_swe_traces_research/
traces_external/` (this worktree does not carry a copy). **Not copied** into
`outputs/derived/` because of size (≈ 280 MB combined); read in place. No DuckDB
views are registered for them (files are outside this repo); query with
`read_parquet('…')` / `read_json` directly.

### harithoppil__terminal-bench-2-trajectories

HF dataset snapshot (`data/*.jsonl`, JSONL, one row per (task_name, model, agent)
trial). Producer: upstream HF export; README in the dump dir. Harvested as of
2026-09-19. Four configs, all with the same schema:

| file | rows |
|---|---|
| `data/leaderboard_trajectories.jsonl` (all) | 3,723 |
| `data/leaderboard_trajectories_pass.jsonl` (passed only) | 2,388 |
| `data/leaderboard_trajectories_ml.jsonl` (ML tasks) | 672 |
| `data/leaderboard_trajectories_ml_pass.jsonl` (ML, passed) | 310 |

| column | type | meaning | unit |
|---|---|---|---|
| `task_name` | string | task identifier (89 unique) — join key | — |
| `model` | string | model (e.g. Claude-Opus-4.6) | — |
| `agent` | string | agent scaffold | — |
| `prompt` | string | full task instruction | — |
| `response` | string | agent output (trajectory.json / stdout.txt) | — |
| `reward` | float64 | 1.0 pass / 0.0 fail | 0/1 |
| `elapsed_seconds` | float64 | time to complete | s |

Caveat: `response` is the agent's final output text, not a structured step log; pass
splits are redundant subsets of `all`.

### yoonholee__terminalbench-trajectories

HF dataset snapshot (`data/train-0000{0,1}-of-00002.parquet`, one row per trial,
52,104 rows total across both files; ~212 MB). Producer: upstream HF export
(README in the dump dir; scraped from tbench.ai). Harvested as of 2026-09-19.

| column | type | meaning | unit |
|---|---|---|---|
| `task_name` | string | task identifier — join key (≈ `task_name` in harithoppil) | — |
| `agent` | string | agent scaffold (26 distinct) | — |
| `model` | string | underlying LLM | — |
| `reward` | int64 | 1 solved / 0 not | 0/1 |
| `duration_seconds` | float64 | wall-clock duration (NULL for some agents) | s |
| `input_tokens` | float64 | input tokens (NULL for some agents) | tokens |
| `output_tokens` | float64 | output tokens | tokens |
| `cache_tokens` | float64 | cached tokens | tokens |
| `cost_cents` | float64 | cost | cents |
| `trial_name` | string | human-readable trial name | — |
| `trial_id` | string | trial UUID — join key | — |
| `started_at` | string | start timestamp | — |
| `ended_at` | string | end timestamp | — |
| `steps` | string | JSON-serialized list of step objects `{src, msg, tools, obs}` | — |

Caveats: `steps` is a JSON string (parse with `json_extract`/`from_json`); `model`
uses `model@provider` format (e.g. `claude-opus-4-6@anthropic`) unlike harithoppil's
bare names; `duration_seconds`/token columns are NULL for some agents. Only 34,462 of
52,104 trials carry non-empty steps.

---

## Part 8 — DuckDB views over derived parquets (`analytics/schema/derived_views.sql`)

Registered by `uv run python scripts/duckdb_init.py` (skipped with a warning when
`outputs/derived/` has no parquets). Each view is a thin `SELECT * FROM
read_parquet('<{{DERIVED_DIR}}>/<file>')` with a `COMMENT ON VIEW` and `COMMENT ON
COLUMN` for every column (DuckDB ≥ 1.2 supports comments). Views:

| view | reads |
|---|---|
| `osw_instance_closure_proxies` | `outputs/derived/closure_proxies.parquet` |
| `osw_instance_rung` | `outputs/derived/rung_features.parquet` |
| `osw_trajectory_frame` | `outputs/derived/trajectory_frame.parquet` |
| `unit_closure_metrics` | `outputs/derived/closure_metrics.parquet` |
| `harbor_hub_jobs` | `outputs/derived/harbor_hub/jobs.parquet` |
| `harbor_hub_leaderboard_rows` | `outputs/derived/harbor_hub/leaderboard_rows.parquet` |
| `harbor_hub_row_trials` | `outputs/derived/harbor_hub/row_trials.parquet` |
| `harbor_hub_tasks` | `outputs/derived/harbor_hub/tasks.parquet` |
| `harbor_hub_trials` | `outputs/derived/harbor_hub/trials.parquet` |

Caveat: views bind at creation time — DuckDB raises "No files found" if a referenced
parquet is missing, so `duckdb_init.py` skips this file (with a warning) until
`uv run python scripts/sync_derived.py` has populated `outputs/derived/`.

### osw_instance_closure_proxies

Thin view over `outputs/derived/closure_proxies.parquet`. Grain: one row per
`instance_id`. All columns and caveats: section
`### outputs/derived/closure_proxies.parquet` in Part 5. Join on `instance_id`.

### osw_instance_rung

Thin view over `outputs/derived/rung_features.parquet`. Grain: one row per
`instance_id`. All columns and caveats: section
`### outputs/derived/rung_features.parquet` in Part 5. Join on `instance_id`.

### osw_trajectory_frame

Thin view over `outputs/derived/trajectory_frame.parquet`. Grain: one row per
`trajectory_id`. All columns and caveats: section
`### outputs/derived/trajectory_frame.parquet` in Part 5. Join on `trajectory_id` or
`instance_id`.

### unit_closure_metrics

Thin view over `outputs/derived/closure_metrics.parquet`. Grain: one row per
authored unit `(repo, unit)`. All columns and caveats: section
`### outputs/derived/closure_metrics.parquet` in Part 5. Join on `(repo, unit)`.

### harbor_hub_jobs

Thin view over `outputs/derived/harbor_hub/jobs.parquet`. Grain: one row per Harbor
Hub leaderboard job. Columns: section `### outputs/derived/harbor_hub/jobs.parquet`
in Part 5. Join on `job_id` / `leaderboard_row_id`.

### harbor_hub_leaderboard_rows

Thin view over `outputs/derived/harbor_hub/leaderboard_rows.parquet`. Grain: one row
per leaderboard row. Columns: section
`### outputs/derived/harbor_hub/leaderboard_rows.parquet` in Part 5. Join on `row_id`.

### harbor_hub_row_trials

Thin view over `outputs/derived/harbor_hub/row_trials.parquet`. Grain: one row per
(row, trial) association. Columns: section
`### outputs/derived/harbor_hub/row_trials.parquet` in Part 5. Join on `row_id` /
`trial_id`.

### harbor_hub_tasks

Thin view over `outputs/derived/harbor_hub/tasks.parquet`. Grain: one row per task.
Columns: section `### outputs/derived/harbor_hub/tasks.parquet` in Part 5. Join on
`task_id`.

### harbor_hub_trials

Thin view over `outputs/derived/harbor_hub/trials.parquet`. Grain: one row per trial.
Columns: section `### outputs/derived/harbor_hub/trials.parquet` in Part 5. Join on
`trial_id` / `task_id` / `job_id` / `leaderboard_row_id`.
