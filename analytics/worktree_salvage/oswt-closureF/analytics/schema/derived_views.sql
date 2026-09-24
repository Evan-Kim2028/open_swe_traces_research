-- DuckDB views over the derived parquets in outputs/derived/ (copied from sibling
-- closure worktrees by scripts/sync_derived.py; provenance: outputs/derived/SOURCES.md).
-- {{DERIVED_DIR}} is substituted by scripts/duckdb_init.py -> openswe_traces.data.init_db.
-- Full per-column documentation: analytics/schema/DATA_CATALOG.md (Part 5).
-- NOTE: DuckDB binds read_parquet at view-creation time; if outputs/derived/ is empty
-- init_db skips this file (with a warning) instead of failing.

CREATE OR REPLACE VIEW osw_instance_closure_proxies AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/closure_proxies.parquet');

COMMENT ON VIEW osw_instance_closure_proxies IS
    'Per-instance closure-structure proxies of the gold patch plus labeled-rollout solve counts; one row per instance_id (outputs/derived/closure_proxies.parquet, oswt-closureB).';
COMMENT ON COLUMN osw_instance_closure_proxies.instance_id IS 'Open-SWE-Traces task instance id (join key)';
COMMENT ON COLUMN osw_instance_closure_proxies.repo IS 'Repository of the task';
COMMENT ON COLUMN osw_instance_closure_proxies.language IS 'Primary language of the repo';
COMMENT ON COLUMN osw_instance_closure_proxies.n_rollouts IS 'Labeled rollouts for the instance across all shards (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.n_labeled IS 'Rollouts with resolved in (0,1) (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.n_resolved IS 'Rollouts with resolved = 1 (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.gold_patch_lines IS 'Lines in the representative gold patch (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.gold_patch_files IS 'Files touched by the gold patch (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.has_gold_patch IS '1 if the gold patch is non-empty (0/1)';
COMMENT ON COLUMN osw_instance_closure_proxies.n_hunks IS '@@ hunks in the gold patch (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.n_files IS 'diff --git headers in the gold patch (files)';
COMMENT ON COLUMN osw_instance_closure_proxies.added_lines IS 'Plus content lines inside hunks (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.removed_lines IS 'Minus content lines inside hunks (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.n_new_defs IS 'Added lines defining a symbol per-language regex (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.internal_refs IS 'References in added lines to names defined elsewhere in the same patch (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.boundary_refs IS 'References in added lines to pre-existing identifiers from context/removed lines (count)';
COMMENT ON COLUMN osw_instance_closure_proxies.ratio IS 'internal_refs / max(1, boundary_refs); 0 when the patch has no new defs and/or no boundary refs';
COMMENT ON COLUMN osw_instance_closure_proxies.new_frac IS 'n_new_defs / max(1, added_lines)';
COMMENT ON COLUMN osw_instance_closure_proxies.edit_frac IS 'Fraction of hunks with both removed and added lines (0..1)';

CREATE OR REPLACE VIEW osw_instance_rung AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/rung_features.parquet');

COMMENT ON VIEW osw_instance_rung IS
    'Per-instance heuristic ladder rung L0-L6 and task-text features (outputs/derived/rung_features.parquet, oswt-closureC); one row per instance_id.';
COMMENT ON COLUMN osw_instance_rung.instance_id IS 'Open-SWE-Traces task instance id (join key)';
COMMENT ON COLUMN osw_instance_rung.rung IS 'Heuristic ladder rung L0-L6 (0 = symptom only, 6 = tests in text)';
COMMENT ON COLUMN osw_instance_rung.task_text IS 'Task text with harness boilerplate stripped';
COMMENT ON COLUMN osw_instance_rung.leakage_symbols IS 'JSON list of gold-patch symbols leaked into the task text';
COMMENT ON COLUMN osw_instance_rung.content_len IS 'Length of the raw task instruction messages[2].content (chars)';
COMMENT ON COLUMN osw_instance_rung.n_rows IS 'Shard rows aggregated for this instance (count)';
COMMENT ON COLUMN osw_instance_rung.word_count IS 'Word count of the stripped task text (count)';
COMMENT ON COLUMN osw_instance_rung.has_repro IS 'Repro/steps-to-reproduce present (0/1)';
COMMENT ON COLUMN osw_instance_rung.has_expected_actual IS 'Expected/actual behavior contract present (0/1)';
COMMENT ON COLUMN osw_instance_rung.has_stack_trace IS 'Stack trace / error text present (0/1)';
COMMENT ON COLUMN osw_instance_rung.has_test_names IS 'Named tests referenced (0/1)';
COMMENT ON COLUMN osw_instance_rung.has_signature IS 'Exported signatures / API stubs present (0/1)';
COMMENT ON COLUMN osw_instance_rung.has_leakage IS 'Leaked symbols detected (0/1)';
COMMENT ON COLUMN osw_instance_rung.leakage_count IS 'Number of leaked symbols (count)';
COMMENT ON COLUMN osw_instance_rung.has_test_code IS 'Substantial test body in the text (0/1)';
COMMENT ON COLUMN osw_instance_rung.n_test_funcs_in_text IS 'Test functions found in the text (count)';
COMMENT ON COLUMN osw_instance_rung.n_test_files_in_text IS 'Test files found in the text (count)';
COMMENT ON COLUMN osw_instance_rung.patch_has_tests IS 'Reference patch touches test files (0/1)';

CREATE OR REPLACE VIEW osw_trajectory_frame AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/trajectory_frame.parquet');

COMMENT ON VIEW osw_trajectory_frame IS
    'Per-trajectory frame with patch-file Jaccard of model vs gold patch (outputs/derived/trajectory_frame.parquet, oswt-closureC); one row per trajectory_id.';
COMMENT ON COLUMN osw_trajectory_frame.trajectory_id IS 'Trajectory id (join key)';
COMMENT ON COLUMN osw_trajectory_frame.instance_id IS 'Open-SWE-Traces task instance id (join key)';
COMMENT ON COLUMN osw_trajectory_frame.repo IS 'Repository of the task';
COMMENT ON COLUMN osw_trajectory_frame.language IS 'Primary language of the repo';
COMMENT ON COLUMN osw_trajectory_frame.harness IS 'Harness from the shard path';
COMMENT ON COLUMN osw_trajectory_frame.teacher IS 'Teacher model short name (= traces.teacher_model)';
COMMENT ON COLUMN osw_trajectory_frame.source IS 'Source dataset from the shard path';
COMMENT ON COLUMN osw_trajectory_frame.resolved IS '1 resolved / 0 failed / -1 unknown';
COMMENT ON COLUMN osw_trajectory_frame.patch_file_jaccard IS 'Jaccard of model-patch vs gold-patch file sets (0..1; no NULLs)';

CREATE OR REPLACE VIEW unit_closure_metrics AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/closure_metrics.parquet');

COMMENT ON VIEW unit_closure_metrics IS
    'Closure metrics per authored unit plus measured flip per solver (outputs/derived/closure_metrics.parquet, oswt-closureA); one row per (repo, unit).';
COMMENT ON COLUMN unit_closure_metrics.repo IS 'Repository (join key to units/trials)';
COMMENT ON COLUMN unit_closure_metrics.unit IS 'Authored unit name (join key)';
COMMENT ON COLUMN unit_closure_metrics.family IS 'Authoring family: single-file / cross-file / sequence';
COMMENT ON COLUMN unit_closure_metrics.predicted_flip IS 'Author-predicted flip level from difficulty.md, NULL if absent';
COMMENT ON COLUMN unit_closure_metrics.control IS 'Unit marked control: true';
COMMENT ON COLUMN unit_closure_metrics.metric_source IS 'authored unit | authored patch rebranded to base tree | metrics reused';
COMMENT ON COLUMN unit_closure_metrics.n_files IS 'Files in the excision patch (count)';
COMMENT ON COLUMN unit_closure_metrics.n_funcs_removed IS 'Functions the excision removes (count)';
COMMENT ON COLUMN unit_closure_metrics.lines_removed IS 'Lines removed by the excision (count)';
COMMENT ON COLUMN unit_closure_metrics.internal_edges IS 'Call references among removed functions (count)';
COMMENT ON COLUMN unit_closure_metrics.boundary_in IS 'Call sites in the remaining tree referencing removed functions (count)';
COMMENT ON COLUMN unit_closure_metrics.boundary_out IS 'Call references from removed bodies to remaining symbols (count)';
COMMENT ON COLUMN unit_closure_metrics.ratio IS 'internal_edges / max(1, boundary_in + boundary_out); 0 when the closure has no boundary edges';
COMMENT ON COLUMN unit_closure_metrics.flip_devin IS 'Measured Devin flip label (0/2/5/none), NULL if no trials';
COMMENT ON COLUMN unit_closure_metrics.flip_devin_enc IS 'Encoded Devin flip: level k -> k, none -> max tried + 1, NULL if no trials';
COMMENT ON COLUMN unit_closure_metrics.flip_cursor IS 'Measured cursor (Composer 2.5) flip label, NULL if no trials';
COMMENT ON COLUMN unit_closure_metrics.flip_cursor_enc IS 'Encoded cursor flip: level k -> k, none -> max tried + 1, NULL if no trials';
COMMENT ON COLUMN unit_closure_metrics.flip_note IS 'Provenance for manual flips (e.g. flip from HANDOFF.md)';

CREATE OR REPLACE VIEW harbor_hub_jobs AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/harbor_hub/jobs.parquet');

COMMENT ON VIEW harbor_hub_jobs IS
    'Harbor Hub leaderboard jobs (outputs/derived/harbor_hub/jobs.parquet, oswt-closureE); one row per job.';
COMMENT ON COLUMN harbor_hub_jobs.bench_version IS 'Terminal-Bench version (e.g. 2.0)';
COMMENT ON COLUMN harbor_hub_jobs.leaderboard_id IS 'Leaderboard id';
COMMENT ON COLUMN harbor_hub_jobs.leaderboard_row_id IS 'Leaderboard row id (join key)';
COMMENT ON COLUMN harbor_hub_jobs.job_id IS 'Job UUID (join key to harbor_hub_trials)';
COMMENT ON COLUMN harbor_hub_jobs.title IS 'Job title';
COMMENT ON COLUMN harbor_hub_jobs.agent IS 'Agent scaffold';
COMMENT ON COLUMN harbor_hub_jobs.agent_version IS 'Agent version';
COMMENT ON COLUMN harbor_hub_jobs.model IS 'Underlying model';
COMMENT ON COLUMN harbor_hub_jobs.model_provider IS 'Model provider';
COMMENT ON COLUMN harbor_hub_jobs.n_tasks IS 'Tasks attempted (count)';
COMMENT ON COLUMN harbor_hub_jobs.n_trials IS 'Trials run (count)';
COMMENT ON COLUMN harbor_hub_jobs.mean_reward IS 'Mean trial reward (0..1)';
COMMENT ON COLUMN harbor_hub_jobs.rank IS 'Leaderboard rank';
COMMENT ON COLUMN harbor_hub_jobs.accuracy IS 'Pass rate (0..1)';
COMMENT ON COLUMN harbor_hub_jobs.cost_usd IS 'Total cost (USD)';
COMMENT ON COLUMN harbor_hub_jobs.tokens_in IS 'Input tokens (tokens)';
COMMENT ON COLUMN harbor_hub_jobs.tokens_out IS 'Output tokens (tokens)';
COMMENT ON COLUMN harbor_hub_jobs.n_errors IS 'Erroring trials (count)';
COMMENT ON COLUMN harbor_hub_jobs.visibility IS 'public/private';
COMMENT ON COLUMN harbor_hub_jobs.owner_org IS 'Owning organization';
COMMENT ON COLUMN harbor_hub_jobs.submitted_at IS 'Submission timestamp (ISO string)';
COMMENT ON COLUMN harbor_hub_jobs.finished_at IS 'Finish timestamp (ISO string)';

CREATE OR REPLACE VIEW harbor_hub_leaderboard_rows AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/harbor_hub/leaderboard_rows.parquet');

COMMENT ON VIEW harbor_hub_leaderboard_rows IS
    'Harbor Hub leaderboard rows (outputs/derived/harbor_hub/leaderboard_rows.parquet, oswt-closureE); one row per submission row.';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.bench_version IS 'Terminal-Bench version';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.leaderboard_id IS 'Leaderboard id';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.row_id IS 'Row UUID (join key, = leaderboard_row_id)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.rank IS 'Rank on the board';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.status IS 'Submission status';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.agent IS 'Agent scaffold';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.model IS 'Model';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.agent_org IS 'Agent org';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.model_org IS 'Model org';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.reasoning_effort IS 'Reasoning effort setting';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.date IS 'Submission date';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.job_url IS 'Link to the job';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.source_url IS 'Source link';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.accuracy IS 'Pass rate (0..1)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.accuracy_ci95_half_width IS 'Half-width of the 95% CI on accuracy';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.n_trials IS 'Trials (count)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.total_tokens IS 'Total tokens (tokens)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.total_cost_usd IS 'Total cost (USD)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.created_at IS 'Row created at (ISO string)';
COMMENT ON COLUMN harbor_hub_leaderboard_rows.updated_at IS 'Row updated at (ISO string)';

CREATE OR REPLACE VIEW harbor_hub_row_trials AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/harbor_hub/row_trials.parquet');

COMMENT ON VIEW harbor_hub_row_trials IS
    'Harbor Hub leaderboard-row to trial association (outputs/derived/harbor_hub/row_trials.parquet, oswt-closureE); one row per (row_id, trial_id).';
COMMENT ON COLUMN harbor_hub_row_trials.bench_version IS 'Terminal-Bench version';
COMMENT ON COLUMN harbor_hub_row_trials.leaderboard_id IS 'Leaderboard id';
COMMENT ON COLUMN harbor_hub_row_trials.row_id IS 'Leaderboard row UUID (join key)';
COMMENT ON COLUMN harbor_hub_row_trials.trial_id IS 'Trial UUID (join key)';

CREATE OR REPLACE VIEW harbor_hub_tasks AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/harbor_hub/tasks.parquet');

COMMENT ON VIEW harbor_hub_tasks IS
    'Harbor Hub per-task aggregates (outputs/derived/harbor_hub/tasks.parquet, oswt-closureE); one row per task_id.';
COMMENT ON COLUMN harbor_hub_tasks.bench_version IS 'Terminal-Bench version';
COMMENT ON COLUMN harbor_hub_tasks.task_id IS 'Task UUID (join key to harbor_hub_trials.task_id)';
COMMENT ON COLUMN harbor_hub_tasks.n_trials IS 'Trials for the task (count)';
COMMENT ON COLUMN harbor_hub_tasks.n_jobs IS 'Jobs touching the task (count)';
COMMENT ON COLUMN harbor_hub_tasks.mean_reward IS 'Mean trial reward (0..1)';
COMMENT ON COLUMN harbor_hub_tasks.n_passed IS 'Passed trials (count)';

CREATE OR REPLACE VIEW harbor_hub_trials AS
SELECT * FROM read_parquet('{{DERIVED_DIR}}/harbor_hub/trials.parquet');

COMMENT ON VIEW harbor_hub_trials IS
    'Harbor Hub per-trial rows (outputs/derived/harbor_hub/trials.parquet, oswt-closureE); one row per trial.';
COMMENT ON COLUMN harbor_hub_trials.bench_version IS 'Terminal-Bench version';
COMMENT ON COLUMN harbor_hub_trials.leaderboard_id IS 'Leaderboard id';
COMMENT ON COLUMN harbor_hub_trials.leaderboard_row_id IS 'Leaderboard row UUID (join key)';
COMMENT ON COLUMN harbor_hub_trials.job_id IS 'Job UUID (join key)';
COMMENT ON COLUMN harbor_hub_trials.job_title IS 'Job title';
COMMENT ON COLUMN harbor_hub_trials.agent IS 'Agent scaffold';
COMMENT ON COLUMN harbor_hub_trials.agent_version IS 'Agent version';
COMMENT ON COLUMN harbor_hub_trials.reasoning_effort IS 'Reasoning effort setting';
COMMENT ON COLUMN harbor_hub_trials.config_json IS 'Job config JSON';
COMMENT ON COLUMN harbor_hub_trials.model IS 'Model';
COMMENT ON COLUMN harbor_hub_trials.model_provider IS 'Model provider';
COMMENT ON COLUMN harbor_hub_trials.task_id IS 'Task UUID (join key)';
COMMENT ON COLUMN harbor_hub_trials.task_name IS 'Human-readable task name';
COMMENT ON COLUMN harbor_hub_trials.trial_id IS 'Trial UUID (join key)';
COMMENT ON COLUMN harbor_hub_trials.trial_name IS 'Trial label';
COMMENT ON COLUMN harbor_hub_trials.attempt_index IS '0-based attempt within the job';
COMMENT ON COLUMN harbor_hub_trials.n_attempts IS 'Total attempts for the trial (count)';
COMMENT ON COLUMN harbor_hub_trials.reward IS 'Trial reward (0..1)';
COMMENT ON COLUMN harbor_hub_trials.passed IS 'Scored pass (boolean)';
COMMENT ON COLUMN harbor_hub_trials.status IS 'Trial status';
COMMENT ON COLUMN harbor_hub_trials.is_scored IS 'Counts toward the leaderboard (boolean)';
COMMENT ON COLUMN harbor_hub_trials.error_class IS 'Error class if errored';
COMMENT ON COLUMN harbor_hub_trials.started_at IS 'Start timestamp (ISO string)';
COMMENT ON COLUMN harbor_hub_trials.finished_at IS 'Finish timestamp (ISO string)';
COMMENT ON COLUMN harbor_hub_trials.wall_seconds IS 'Wall-clock duration (s)';
COMMENT ON COLUMN harbor_hub_trials.tokens_in IS 'Input tokens (tokens)';
COMMENT ON COLUMN harbor_hub_trials.tokens_out IS 'Output tokens (tokens)';
COMMENT ON COLUMN harbor_hub_trials.cache_tokens IS 'Cached tokens (tokens)';
COMMENT ON COLUMN harbor_hub_trials.cost_usd IS 'Cost (USD)';
COMMENT ON COLUMN harbor_hub_trials.trajectory_available IS 'Full trajectory downloadable (boolean)';
COMMENT ON COLUMN harbor_hub_trials.harvested_at IS 'When this row was harvested from Harbor Hub (ISO string)';
