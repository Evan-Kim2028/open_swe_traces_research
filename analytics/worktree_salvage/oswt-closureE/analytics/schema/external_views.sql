-- External Terminal-Bench sources: Harbor Hub harvest + the two TB 2.0 HF dumps.
-- {{PROJECT_ROOT}} is substituted by scripts/duckdb_init.py.
-- Column naming follows the Hub harvest: task_id (bare task slug), agent, model,
-- reward (double), passed (bool), tokens_in/out, cost_usd, wall_seconds.

-- Harbor Hub per-trial results (scripts/harvest_harbor_hub.py output).
CREATE OR REPLACE VIEW tb_hub_trials AS
SELECT *
FROM read_parquet('{{PROJECT_ROOT}}/traces_external/harbor_hub/trials.parquet');

CREATE OR REPLACE VIEW tb_hub_jobs AS
SELECT *
FROM read_parquet('{{PROJECT_ROOT}}/traces_external/harbor_hub/jobs.parquet');

CREATE OR REPLACE VIEW tb_hub_tasks AS
SELECT *
FROM read_parquet('{{PROJECT_ROOT}}/traces_external/harbor_hub/tasks.parquet');

-- TB 2.0 trajectory dump: yoonholee/terminalbench-trajectories (52,104 trials).
-- steps is a JSON blob of the full message trace; cost is in cents.
CREATE OR REPLACE VIEW tb2_traj_yoonholee AS
SELECT
    '2.0' AS bench_version,
    task_name AS task_id,
    trial_id,
    trial_name,
    agent,
    model,
    try_cast(reward AS DOUBLE) AS reward,
    try_cast(reward AS DOUBLE) >= 1.0 AS passed,
    duration_seconds AS wall_seconds,
    input_tokens AS tokens_in,
    output_tokens AS tokens_out,
    cache_tokens,
    cost_cents / 100.0 AS cost_usd,
    try_cast(started_at AS TIMESTAMP) AS started_at,
    try_cast(ended_at AS TIMESTAMP) AS finished_at,
    steps IS NOT NULL AS trajectory_available
FROM read_parquet(
    '{{PROJECT_ROOT}}/traces_external/yoonholee__terminalbench-trajectories/data/*.parquet',
    union_by_name = true
);

-- TB 2.0 trajectory dump: harithoppil/terminal-bench-2-trajectories (3,723 trials).
-- The _pass/_ml files are filtered subsets of leaderboard_trajectories.jsonl.
CREATE OR REPLACE VIEW tb2_traj_harithoppil AS
SELECT
    '2.0' AS bench_version,
    task_name AS task_id,
    NULL AS trial_id,
    agent,
    model,
    try_cast(reward AS DOUBLE) AS reward,
    try_cast(reward AS DOUBLE) >= 1.0 AS passed,
    elapsed_seconds AS wall_seconds,
    NULL AS tokens_in,
    NULL AS tokens_out,
    NULL AS cost_usd,
    NULL AS started_at,
    response IS NOT NULL AND len(response) > 0 AS trajectory_available,
    is_ml_related
FROM read_json(
    '{{PROJECT_ROOT}}/traces_external/harithoppil__terminal-bench-2-trajectories/data/leaderboard_trajectories.jsonl',
    format = 'newline_delimited'
);
