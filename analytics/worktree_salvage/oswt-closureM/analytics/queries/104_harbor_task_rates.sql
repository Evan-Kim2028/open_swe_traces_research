-- 104 — Harbor Hub per-task rates (closure-E harvest).
--
-- Per (bench_version, task) pass-rate table over the harvested Harbor Hub
-- trials: n_trials, distinct jobs touching the task, mean reward and the scored
-- pass rate (a trial passes when reward >= 1.0 — the harvester's `passed`
-- rule). tasks.parquet already aggregates at task_id grain; this query re-derives
-- it from trials and carries the human-readable task_name (task_id is an opaque
-- UUID). The task_id grain in the source is per (bench_version, task_id).
--
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/104_harbor_task_rates.sql --db analysis

SELECT
    bench_version,
    task_name,
    count(*) AS n_trials,
    count(DISTINCT job_id) AS n_jobs,
    round(avg(reward), 3) AS mean_reward,
    count(*) FILTER (WHERE reward >= 1.0) AS n_passed,
    round(count(*) FILTER (WHERE reward >= 1.0)::DOUBLE / NULLIF(count(*), 0), 3) AS pass_rate
FROM harbor_hub_trials
WHERE is_scored
GROUP BY bench_version, task_name
ORDER BY bench_version, pass_rate DESC, task_name;
