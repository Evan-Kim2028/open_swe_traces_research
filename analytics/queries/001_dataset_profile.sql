-- Profile: how much data is loaded and overall resolve mix.
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql

SELECT * FROM catalog_download;

SELECT resolved, count(*) AS n, round(100.0 * count(*) / sum(count(*)) OVER (), 1) AS pct
FROM traces
GROUP BY 1
ORDER BY 1;

SELECT language, count(*) AS n
FROM traces
GROUP BY 1
ORDER BY n DESC
LIMIT 15;
