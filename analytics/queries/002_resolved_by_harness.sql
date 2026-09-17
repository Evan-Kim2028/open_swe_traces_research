-- Compare resolve rates across harness / teacher model slices.
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/002_resolved_by_harness.sql

SELECT *
FROM catalog_resolved_rates
ORDER BY n DESC;
