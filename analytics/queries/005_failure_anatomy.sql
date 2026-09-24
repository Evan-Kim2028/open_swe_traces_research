-- Stopping-rule flags for labeled Open-SWE-Traces rollouts.
-- Built by scripts/failure_anatomy.py (shard stream → outputs/failure_anatomy.parquet).
-- Run after that job:
--   uv run python scripts/duckdb_query.py -f analytics/queries/005_failure_anatomy.sql

SELECT
    resolved,
    count(*) AS n,
    round(100.0 * avg(asserts_done::INT), 1) AS pct_asserts_done,
    round(100.0 * avg(asserts_tests_pass::INT), 1) AS pct_tests_pass,
    round(100.0 * avg(expresses_uncertainty::INT), 1) AS pct_uncertain,
    round(100.0 * avg(mentions_unseen_gap::INT), 1) AS pct_unseen_gap,
    round(100.0 * avg(ran_test_in_tail::INT), 1) AS pct_ran_test,
    round(
        100.0 * avg(CASE WHEN last_test_green THEN 1.0 END),
        1
    ) AS pct_last_test_green_among_classified,
    round(100.0 * avg(model_touches_tests::INT), 1) AS pct_test_path_edit
FROM read_parquet('outputs/failure_anatomy.parquet')
GROUP BY 1
ORDER BY 1;
