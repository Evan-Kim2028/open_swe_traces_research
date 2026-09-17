"""Quick sanity check after download (DuckDB streaming, 4GB cap)."""

from __future__ import annotations

from rich.console import Console
from rich.table import Table

from .data import DATA_DIR, PARQUET_GLOB, connect_ephemeral

console = Console()

METRICS = ["rows", "unique_instances", "unique_trajectories", "resolved", "unresolved", "unknown"]


def verify() -> dict[str, int] | None:
    """Print corpus row counts; return them, or None when no parquet files exist yet."""
    if not DATA_DIR.exists():
        console.print(f"[red]Missing {DATA_DIR}[/red] — run: uv run openswe-download")
        raise SystemExit(1)

    parquet_files = list(DATA_DIR.rglob("*.parquet"))
    console.print(f"Parquet files: {len(parquet_files)}")
    if not parquet_files:
        console.print("[yellow]No parquet files yet.[/yellow]")
        return None

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
    for label, value in zip(METRICS, summary, strict=True):
        table.add_row(label, f"{value:,}")
    console.print(table)
    return dict(zip(METRICS, summary, strict=True))


def main() -> None:
    verify()
