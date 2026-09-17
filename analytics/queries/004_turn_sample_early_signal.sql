-- Early signal: at turn k, how separable are success vs failure?
-- Requires: uv run python scripts/extract_turn_sample.py --register-duckdb

SELECT
    CASE
        WHEN pct_through <= 0.25 THEN '0-25%'
        WHEN pct_through <= 0.50 THEN '25-50%'
        WHEN pct_through <= 0.75 THEN '50-75%'
        ELSE '75-100%'
    END AS progress_bucket,
    outcome_label,
    count(DISTINCT trajectory_id) AS trials,
    round(avg(cum_edits), 2) AS avg_edits_so_far,
    round(avg(cum_tests), 2) AS avg_tests_so_far,
    round(avg(cum_tool_errors), 2) AS avg_tool_errors
FROM turn_sample
WHERE role = 'assistant'
GROUP BY 1, 2
ORDER BY 1, 2;
