#!/usr/bin/env python3
"""Quick sanity check after download: file counts, parquet row totals, schema sample."""

from __future__ import annotations

from pathlib import Path

import duckdb
from rich.console import Console
from rich.table import Table

console = Console()

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "traces_data"
PARQUET_GLOB = str(DATA_DIR / "data" / "**" / "*.parquet")


def main() -> None:
    if not DATA_DIR.exists():
        console.print(f"[red]Missing {DATA_DIR}[/red] — run scripts/download_data.py first")
        raise SystemExit(1)

    con = duckdb.connect()
    parquet_files = list(DATA_DIR.rglob("*.parquet"))
    console.print(f"Parquet files: {len(parquet_files)}")

    if not parquet_files:
        console.print("[yellow]No parquet files yet — download may still be running.[/yellow]")
        raise SystemExit(0)

    summary = con.sql(
        f"""
        SELECT
            count(*) AS rows,
            count(DISTINCT instance_id) AS unique_instances,
            count(DISTINCT trajectory_id) AS unique_trajectories,
            sum(CASE WHEN resolved = 1 THEN 1 ELSE 0 END) AS resolved,
            sum(CASE WHEN resolved = 0 THEN 1 ELSE 0 END) AS unresolved,
            sum(CASE WHEN resolved = -1 THEN 1 ELSE 0 END) AS unknown
        FROM read_parquet('{PARQUET_GLOB}', hive_partitioning=false, union_by_name=true)
        """
    ).fetchone()

    table = Table(title="Open-SWE-Traces summary")
    table.add_column("Metric")
    table.add_column("Value", justify="right")
    labels = [
        "rows",
        "unique_instances",
        "unique_trajectories",
        "resolved",
        "unresolved",
        "unknown",
    ]
    for label, value in zip(labels, summary, strict=True):
        table.add_row(label, f"{value:,}")
    console.print(table)

    sample = con.sql(
        f"""
        SELECT instance_id, repo, language, resolved,
               len(messages) AS num_messages,
               metadata.category AS category
        FROM read_parquet('{PARQUET_GLOB}', hive_partitioning=false, union_by_name=true)
        LIMIT 3
        """
    ).fetchdf()
    console.print("\nSample rows:")
    console.print(sample.to_string(index=False))


if __name__ == "__main__":
    main()
