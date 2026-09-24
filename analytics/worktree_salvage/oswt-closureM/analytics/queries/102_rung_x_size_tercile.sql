-- 102 — Rung x size tercile table (framework tests, closure-C, T1 2x3 table).
--
-- Per (rung_binary, added_lines tercile): mean solve_rate and n over labeled
-- trajectories (resolved in (0,1)), for ALL combos and for the top-3
-- teacher/harness combos by IRT ability (minisweagent/qwen38_27b,
-- sweagent/qwen36_27b, sweagent/qwen35_122b — the closure-C run's theta_2pl
-- order). Terciles are computed over unique instances by added_lines rank
-- (closure-C: pd.qcut(rank(method='first'), 3)); ntile(3) is the SQL equivalent.
-- Rung is the heuristic ladder rung of the task text; rung_binary = rung >= 2.
--
-- This is the timing comparison query in analytics/research/analysis_db.md:
-- streamed over the corpus (closure-C) vs this table read from analysis.duckdb.
--
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/102_rung_x_size_tercile.sql --db analysis

WITH labeled AS (
    SELECT
        t.trajectory_id,
        t.instance_id,
        t.resolved,
        t.harness || '/' || t.teacher AS combo,
        i.rung,
        i.added_lines,
        i.ratio,
        i.new_frac,
        i.n_files
    FROM trajectory t
    JOIN instance i USING (instance_id)
    WHERE t.resolved IN (0, 1)
      AND i.added_lines IS NOT NULL
      AND i.ratio IS NOT NULL
      AND i.new_frac IS NOT NULL
      AND i.n_files IS NOT NULL
      AND i.rung IS NOT NULL
),
inst AS (
    SELECT DISTINCT instance_id, added_lines
    FROM labeled
),
terc AS (
    SELECT instance_id, ntile(3) OVER (ORDER BY added_lines) AS tercile
    FROM inst
)
SELECT
    (l.rung >= 2)::INT AS rung_binary,
    t.tercile,
    round(avg(l.resolved), 3) AS mean_solve_rate_all,
    count(*) AS n_all,
    round(avg(l.resolved) FILTER (
        WHERE l.combo IN ('minisweagent/qwen38_27b', 'sweagent/qwen36_27b', 'sweagent/qwen35_122b')
    ), 3) AS mean_solve_rate_top3,
    count(*) FILTER (
        WHERE l.combo IN ('minisweagent/qwen38_27b', 'sweagent/qwen36_27b', 'sweagent/qwen35_122b')
    ) AS n_top3
FROM labeled l
JOIN terc t USING (instance_id)
GROUP BY 1, 2
ORDER BY 1, 2;
