"""Build the pair-eligible trajectory set for the within-instance paired analysis.

Inputs (copied into ``outputs/`` from the closure-B / closure-C runs):

  outputs/closure_proxies.parquet   per-instance gold-patch proxies + n_labeled/n_resolved
  outputs/trajectory_frame.parquet  per-trajectory (instance, combo, resolved)

Output: ``outputs/eligible_pairs.parquet`` — one row per trajectory in a mixed
(instance, combo) group, with ``stratum`` = SMALL (added_lines <= 30) / LARGE (>= 150)
/ MID and ``group_id`` = instance_id | combo.

Usage: uv run python scripts/paired_eligible.py
"""

from __future__ import annotations

import argparse
from pathlib import Path

from rich.console import Console

import duckdb

from .data import ROOT, connect_ephemeral

console = Console()

CLOSURE_PROXIES = ROOT / "outputs" / "closure_proxies.parquet"
TRAJECTORY_FRAME = ROOT / "outputs" / "trajectory_frame.parquet"
OUT_PATH = ROOT / "outputs" / "eligible_pairs.parquet"

SMALL_MAX_ADDED = 30
LARGE_MIN_ADDED = 150
MIN_LABELED = 5

ELIGIBLE_SQL = r"""
WITH mixed AS (
    SELECT instance_id, added_lines, n_labeled, n_resolved, repo, language
    FROM read_parquet('{proxies}')
    WHERE n_labeled >= {min_labeled} AND n_resolved > 0 AND n_resolved < n_labeled
),
tf AS (
    SELECT trajectory_id, instance_id, harness, teacher, source, resolved,
           harness || '/' || teacher AS combo
    FROM read_parquet('{frame}')
    WHERE resolved IN (0, 1)
),
g AS (
    SELECT instance_id, combo,
           count(*) FILTER (resolved = 1) AS n_pass,
           count(*) FILTER (resolved = 0) AS n_fail
    FROM tf JOIN mixed USING (instance_id)
    GROUP BY 1, 2
),
elig AS (SELECT instance_id, combo FROM g WHERE n_pass > 0 AND n_fail > 0)
SELECT
    t.trajectory_id, t.instance_id, t.combo, t.harness, t.teacher, t.source,
    t.resolved, m.added_lines, m.n_labeled, m.n_resolved, m.repo, m.language,
    CASE
        WHEN m.added_lines <= {small_max} THEN 'SMALL'
        WHEN m.added_lines >= {large_min} THEN 'LARGE'
        ELSE 'MID'
    END AS stratum,
    t.instance_id || '|' || t.combo AS group_id
FROM tf t
JOIN elig e ON e.instance_id = t.instance_id AND e.combo = t.combo
JOIN mixed m ON m.instance_id = t.instance_id
ORDER BY t.instance_id, t.combo, t.trajectory_id
"""


def build_eligible_pairs(
    con: duckdb.DuckDBPyConnection,
    proxies_path: Path = CLOSURE_PROXIES,
    frame_path: Path = TRAJECTORY_FRAME,
    out_path: Path = OUT_PATH,
    *,
    small_max: int = SMALL_MAX_ADDED,
    large_min: int = LARGE_MIN_ADDED,
    min_labeled: int = MIN_LABELED,
) -> int:
    sql = ELIGIBLE_SQL.format(
        proxies=str(proxies_path).replace("'", "''"),
        frame=str(frame_path).replace("'", "''"),
        small_max=small_max,
        large_min=large_min,
        min_labeled=min_labeled,
    )
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(f"COPY ({sql}) TO '{tmp!s}' (FORMAT PARQUET)")
    tmp.replace(out_path)
    return con.execute(
        f"SELECT count(*) FROM read_parquet('{out_path!s}')"
    ).fetchone()[0]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--proxies", type=Path, default=CLOSURE_PROXIES)
    parser.add_argument("--frame", type=Path, default=TRAJECTORY_FRAME)
    parser.add_argument("--out", type=Path, default=OUT_PATH)
    args = parser.parse_args()

    con = connect_ephemeral()
    try:
        n = build_eligible_pairs(con, args.proxies, args.frame, args.out)
        summary = con.execute(
            f"""
            SELECT stratum, count(*) AS trajectories,
                   count(DISTINCT group_id) AS groups,
                   count(DISTINCT instance_id) AS instances
            FROM read_parquet('{args.out!s}')
            GROUP BY 1 ORDER BY 1
            """
        ).fetchdf()
    finally:
        con.close()
    console.print(f"[green]wrote {n:,} rows → {args.out}[/green]")
    console.print(summary.to_string(index=False))


if __name__ == "__main__":
    main()
