#!/usr/bin/env python3
"""Quick sanity check after download (DuckDB streaming, 4GB cap)."""

from __future__ import annotations

import sys
from pathlib import Path

from rich.console import Console
from rich.table import Table

sys.path.insert(0, str(Path(__file__).resolve().parent))
from duckdb_session import PARQUET_GLOB, connect_ephemeral

console = Console()

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "traces_data"


def main() -> None:
    if not DATA_DIR.exists():
        console.print(f"[red]Missing {DATA_DIR}[/red] — run scripts/download_data.py first")
        raise SystemExit(1)

    parquet_files = list(DATA_DIR.rglob("*.parquet"))
    console.print(f"Parquet files: {len(parquet_files)}")
    if not parquet_files:
        console.print("[yellow]No parquet files yet.[/yellow]")
        raise SystemExit(0)

    con = connect_ephemeral()
    summary = con.execute(
        f"""
        SELECT
            count(*) AS rows,
            count(DISTINCT instance_id) AS unique_instances,
            count(DISTINCT trajectory_id) AS unique_trajectories,
            sum(CASE WHEN resolved = 1 THEN 1 ELSE 0 END) AS resolved,
            sum(CASE WHEN resolved = 0 THEN 1 ELSE 0 END) AS unresolved,
            sum(CASE WHEN resolved = -1 THEN 1 ELSE 0 END) AS unknown
        FROM read_parquet('{PARQUET_GLOB}', union_by_name=true)
        """
    ).fetchone()
    con.close()

    table = Table(title="Open-SWE-Traces summary")
    table.add_column("Metric")
    table.add_column("Value", justify="right")
    for label, value in zip(
        ["rows", "unique_instances", "unique_trajectories", "resolved", "unresolved", "unknown"],
        summary,
        strict=True,
    ):
        table.add_row(label, f"{value:,}")
    console.print(table)


if __name__ == "__main__":
    main()
