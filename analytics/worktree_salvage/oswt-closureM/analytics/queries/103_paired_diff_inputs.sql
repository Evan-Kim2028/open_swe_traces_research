-- 103 — Paired-difference inputs (within-instance paired analysis, closure-H).
--
-- One row per pair-eligible trajectory (mixed instances, n_labeled >= 5,
-- 0 < n_resolved < n_labeled, group = instance x combo with >= 1 pass and >= 1
-- fail), carrying the eligibility markers (combo / stratum / group_id) and the
-- full behaviour-feature vector. The closure-H bootstrap consumes this frame:
-- within (group_id, feature) it averages fail-vs-pass sibling differences, then
-- resamples instances. Python keeps the bootstrap; this query is the input.
--
-- stratum: SMALL (gold added_lines <= 30) / LARGE (>= 150) / MID; group_id =
-- instance_id | combo (closure-H `paired_eligible`).
--
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/103_paired_diff_inputs.sql --db analysis

SELECT
    t.trajectory_id,
    t.instance_id,
    t.repo,
    t.combo,
    t.group_id,
    t.stratum,
    t.resolved,
    t.n_turns,
    t.n_assistant_turns,
    t.n_tool_calls,
    t.assistant_chars,
    t.n_tool_errors,
    t.n_edit_calls,
    t.n_test_runs,
    t.n_repro_calls,
    t.turn_of_first_edit,
    t.turn_of_last_edit,
    t.first_repro_call,
    t.last_test_call,
    t.n_files_read,
    t.n_files_edited,
    t.n_model_hunks,
    t.n_gold_hunks,
    t.patch_hunk_coverage,
    t.extra_hunks,
    t.extra_hunk_lines,
    t.model_lines_over_gold,
    t.edited_test_file,
    t.touched_test_path,
    t.ran_repro_before_edit,
    t.ran_tests_after_last_edit,
    t.stop_reason,
    t.submitted_empty_patch
FROM trajectory t
WHERE t.group_id IS NOT NULL
  AND t.resolved IN (0, 1)
ORDER BY t.group_id, t.trajectory_id;
