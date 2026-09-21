-- 101 — Mixture-by-size-bin inputs (feasible-component analysis, closure-G).
--
-- Input to the two-component binomial EM (`em_two_component` in
-- openswe_traces.feasible_drivers): one row per instance with n_labeled >= 3,
-- tagged with its decile of log1p(added_lines). The EM runs in Python per decile
-- on (n_resolved, n_labeled); this query is the per-decile input frame it consumes.
--
-- Deciles are computed over the n_labeled >= 3 instances themselves, exactly like
-- the closure-G `pd.qcut(log1p(clip(added_lines,0)), 10)` — ntile(10) is the SQL
-- equivalent on continuous values (ties are negligible). Python keeps the EM fit;
-- nothing here needs the corpus.
--
-- Run: uv run python scripts/duckdb_query.py -f analytics/queries/101_mixture_size_deciles.sql --db analysis

SELECT
    instance_id,
    ntile(10) OVER (ORDER BY ln(1 + greatest(added_lines, 0))) AS decile,
    round(ln(1 + greatest(added_lines, 0)), 3) AS log1p_added,
    n_labeled,
    n_resolved,
    round(n_resolved::DOUBLE / NULLIF(n_labeled, 0), 3) AS solve_rate
FROM instance
WHERE n_labeled >= 3
ORDER BY decile, log1p_added;
