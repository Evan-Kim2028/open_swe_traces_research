-- Refresh helper for summary tables (called from duckdb_init.py --refresh-summaries).

CREATE OR REPLACE MACRO refresh_trace_summaries() AS TABLE
    SELECT 1 AS refreshed;

-- Real refresh is procedural; implemented as explicit statements in 004_summaries.sql
-- and invoked by duckdb_init when --refresh-summaries is passed.
