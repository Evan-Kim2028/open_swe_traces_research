-- Lightweight catalog views so you don't forget what's available.

CREATE OR REPLACE VIEW catalog_columns AS
SELECT *
FROM (DESCRIBE SELECT * FROM traces_raw);

CREATE OR REPLACE VIEW catalog_download AS
SELECT
    count(DISTINCT _source_file) AS parquet_files,
    count(*) AS trajectory_rows,
    min(_source_file) AS example_file
FROM traces;

CREATE OR REPLACE VIEW catalog_by_harness AS
SELECT
    harness,
    teacher_model,
    source_dataset,
    language,
    resolved,
    count(*) AS n,
    round(avg(num_messages), 1) AS avg_messages
FROM traces
GROUP BY 1, 2, 3, 4, 5
ORDER BY n DESC;

CREATE OR REPLACE VIEW catalog_resolved_rates AS
SELECT
    language,
    harness,
    count(*) AS n,
    round(100.0 * sum(CASE WHEN resolved = 1 THEN 1 ELSE 0 END) / count(*), 1) AS pct_resolved,
    round(100.0 * sum(CASE WHEN resolved = 0 THEN 1 ELSE 0 END) / count(*), 1) AS pct_failed,
    round(100.0 * sum(CASE WHEN resolved = -1 THEN 1 ELSE 0 END) / count(*), 1) AS pct_unknown
FROM traces
GROUP BY 1, 2
ORDER BY n DESC;
