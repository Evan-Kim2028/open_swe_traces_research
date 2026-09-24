#!/usr/bin/env python3
"""Before/after timing for the rung x size tercile query (102) — streamed vs DB.

The "before" is what today's analysis jobs did: scan the corpus parquets
(here: projection-pushed read of the 212 shards for the trajectory frame
columns + join the rung/closure parquets) and group in the engine. The "after"
reads the same table from `duckdb/analysis.duckdb` (built once by
`scripts/build_analysis_db.py`). Both run the same query shape; only the data
source differs. Results land in the query log; the numbers are reported in
`analytics/research/analysis_db.md`.

Usage:
  uv run python scripts/time_rung_tercile.py
"""

from __future__ import annotations

import time

from rich.console import Console

import duckdb
from openswe_traces.data import ANALYSIS_DB_PATH, DERIVED_DIR, PARQUET_GLOB, ROOT

console = Console()

STREAMED_SQL = """
WITH frame AS (
    SELECT trajectory_id, instance_id, resolved,
           regexp_extract(filename, '/data/([^/]+)/([^/]+)/', 1) AS harness,
           regexp_extract(filename, '/data/[^/]+/([^/]+)/', 1) AS teacher
    FROM read_parquet('{glob}', union_by_name=true, filename=true)
),
labeled AS (
    SELECT t.trajectory_id, t.instance_id, t.resolved, t.harness || '/' || t.teacher AS combo,
           r.rung, c.added_lines, c.ratio, c.new_frac, c.n_files
    FROM frame t
    JOIN read_parquet('{derived}/rung_features.parquet') r ON r.instance_id = t.instance_id
    JOIN read_parquet('{derived}/closure_proxies.parquet') c ON c.instance_id = t.instance_id
    WHERE t.resolved IN (0, 1)
      AND c.added_lines IS NOT NULL AND c.ratio IS NOT NULL AND c.new_frac IS NOT NULL
      AND c.n_files IS NOT NULL AND r.rung IS NOT NULL
),
inst AS (SELECT DISTINCT instance_id, added_lines FROM labeled),
terc AS (SELECT instance_id, ntile(3) OVER (ORDER BY added_lines) AS tercile FROM inst)
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
ORDER BY 1, 2
"""

DB_SQL = (ROOT / "analytics" / "queries" / "102_rung_x_size_tercile.sql").read_text()


def run_timed(con: duckdb.DuckDBPyConnection, label: str, sql: str) -> tuple[float, list[tuple]]:
    t0 = time.monotonic()
    rows = con.execute(sql).fetchall()
    elapsed = time.monotonic() - t0
    console.print(f"[{label}] {elapsed:.1f}s — {len(rows)} rows")
    return elapsed, rows


def main() -> None:
    con = duckdb.connect()
    try:
        con.execute("SET memory_limit='4GB'")
        con.execute("SET threads=4")
        con.execute("SET preserve_insertion_order=false")

        console.print("Streamed baseline (corpus scan):")
        streamed_sql = STREAMED_SQL.format(glob=PARQUET_GLOB, derived=DERIVED_DIR)
        t_streamed, rows_streamed = run_timed(con, "streamed", streamed_sql)

        console.print("From analysis.duckdb:")
        con_db = duckdb.connect(str(ANALYSIS_DB_PATH), read_only=True)
        try:
            con_db.execute("SET memory_limit='4GB'")
            con_db.execute("SET threads=4")
            t_db, rows_db = run_timed(con_db, "db", DB_SQL)
        finally:
            con_db.close()

        console.print(
            f"\nspeedup {t_streamed / max(t_db, 1e-9):.1f}x "
            f"({t_streamed:.1f}s streamed -> {t_db:.2f}s from DB)"
        )
        # Tercile boundaries are tie-sensitive (qcut-on-rank vs ntile), so cell
        # counts can move a few instances across a boundary; means to 2dp and the
        # total labeled n must agree.
        means_match = all(
            abs(a[2] - b[2]) < 0.005 for a, b in zip(rows_streamed, rows_db)
        )
        totals_match = sum(r[3] for r in rows_streamed) == sum(r[3] for r in rows_db)
        if means_match and totals_match:
            console.print("[green]means (2dp) + total n identical[/green] "
                          "(cell n / 3rd-decimal means may differ by tercile tie placement)")
        else:
            console.print("[red]WARNING: streamed and DB results differ[/red]")
            console.print(f"streamed: {rows_streamed}")
            console.print(f"db:       {rows_db}")
    finally:
        con.close()


if __name__ == "__main__":
    main()
