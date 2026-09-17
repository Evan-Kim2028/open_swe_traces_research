-- Raw trajectories: reads whatever parquet shards exist on disk (grows as download resumes).
-- {{PARQUET_GLOB}} is substituted by scripts/duckdb_init.py.

CREATE OR REPLACE VIEW traces_raw AS
SELECT
    *,
    filename AS _source_file
FROM read_parquet('{{PARQUET_GLOB}}', union_by_name=true, filename=true);

CREATE OR REPLACE VIEW traces AS
SELECT
    instance_id,
    repo,
    license,
    language,
    trajectory_id,
    resolved,
    len(messages) AS num_messages,
    len(tools) AS num_tools,
    metadata.category AS category,
    try_cast(metadata.reference_patch.num_modified_files AS INTEGER) AS gold_files,
    try_cast(metadata.reference_patch.num_modified_lines AS INTEGER) AS gold_lines,
    metadata.reference_patch.patch AS gold_patch,
    try_cast(metadata.model_patch.num_modified_files AS INTEGER) AS model_files,
    try_cast(metadata.model_patch.num_modified_lines AS INTEGER) AS model_lines,
    metadata.model_patch.patch AS model_patch,
    _source_file,
    regexp_extract(_source_file, '.*/data/([^/]+)/', 1) AS harness,
    regexp_extract(_source_file, '.*/data/[^/]+/([^/]+)/', 1) AS teacher_model,
    regexp_extract(_source_file, '.*/data/[^/]+/[^/]+/([^/]+)/', 1) AS source_dataset,
    messages,
    tools
FROM traces_raw;
