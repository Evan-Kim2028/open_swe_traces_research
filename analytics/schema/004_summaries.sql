-- Materialized summaries: fast repeat queries. Refresh after download chunks land.
-- Run: uv run python scripts/duckdb_init.py --refresh-summaries

CREATE OR REPLACE TABLE trace_summary_by_language AS
SELECT
    language,
    resolved,
    count(*) AS n,
    round(avg(num_messages), 1) AS avg_messages,
    round(avg(model_lines), 1) AS avg_model_lines,
    round(avg(gold_lines), 1) AS avg_gold_lines
FROM traces
GROUP BY 1, 2;

CREATE OR REPLACE TABLE trace_summary_by_category AS
SELECT
    category,
    resolved,
    count(*) AS n,
    round(avg(num_messages), 1) AS avg_messages
FROM traces
GROUP BY 1, 2;

CREATE OR REPLACE TABLE trace_empty_patch_rate AS
SELECT
    harness,
    teacher_model,
    count(*) AS n,
    round(
        100.0 * sum(CASE WHEN coalesce(model_patch, '') = '' THEN 1 ELSE 0 END) / count(*),
        2
    ) AS pct_empty_model_patch
FROM traces
GROUP BY 1, 2;
