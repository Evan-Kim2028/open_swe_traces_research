-- Trial-level EDA (no message unpack — fast).
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/003_trial_eda.sql

SELECT count(*) AS trials,
       count(DISTINCT instance_id) AS unique_tasks,
       round(100.0 * avg(CASE WHEN resolved = 1 THEN 1.0 ELSE 0.0 END), 1) AS pct_resolved
FROM trial_summary;

SELECT outcome_label, harness, count(*) AS n, round(avg(num_messages), 1) AS avg_turns
FROM trial_summary
GROUP BY 1, 2
ORDER BY n DESC;

SELECT category, outcome_label, count(*) AS n
FROM trial_summary
GROUP BY 1, 2
ORDER BY n DESC
LIMIT 15;
