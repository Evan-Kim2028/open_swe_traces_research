#!/usr/bin/env python3
"""Run a SQL file or inline query against open_swe.duckdb and append to the query log."""

from __future__ import annotations

import argparse
import csv
from datetime import UTC, datetime
from pathlib import Path

import duckdb
from rich.console import Console

console = Console()

ROOT = Path(__file__).resolve().parent.parent
DB_PATH = ROOT / "duckdb" / "open_swe.duckdb"
QUERY_LOG_DIR = ROOT / "analytics" / "query_log"
INDEX_CSV = QUERY_LOG_DIR / "index.csv"


def _ensure_index() -> None:
    QUERY_LOG_DIR.mkdir(parents=True, exist_ok=True)
    if not INDEX_CSV.exists():
        with INDEX_CSV.open("w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(
                ["timestamp", "slug", "sql_file", "row_count", "elapsed_ms", "log_path"]
            )


def _strip_sql_comments(sql: str) -> str:
    lines = []
    for line in sql.splitlines():
        if line.lstrip().startswith("--"):
            continue
        lines.append(line)
    return "\n".join(lines)


def run_query(sql: str, *, slug: str, sql_file: str | None = None) -> None:
    if not DB_PATH.exists():
        raise SystemExit("DuckDB not initialized. Run: uv run python scripts/duckdb_init.py")

    _ensure_index()
    started = datetime.now(UTC)
    con = duckdb.connect(str(DB_PATH), read_only=True)
    con.execute(f"SET variable project_root = '{ROOT}'")

    row_count = 0
    try:
        cleaned = _strip_sql_comments(sql)
        statements = [s.strip() for s in cleaned.split(";") if s.strip()]
        for i, statement in enumerate(statements):
            result = con.execute(statement)
            if result.description:
                df = result.fetchdf()
                row_count += len(df)
                if len(statements) > 1:
                    console.print(f"\n[bold]Result {i + 1}/{len(statements)}[/bold]")
                console.print(df.to_string(index=False))
        if row_count == 0 and not statements:
            console.print("[dim](no result set)[/dim]")
    finally:
        con.close()

    elapsed_ms = int((datetime.now(UTC) - started).total_seconds() * 1000)
    stamp = started.strftime("%Y-%m-%d_%H%M%S")
    log_path = QUERY_LOG_DIR / f"{stamp}_{slug}.sql"
    header = f"-- slug: {slug}\n-- sql_file: {sql_file or 'inline'}\n-- rows: {row_count}\n-- elapsed_ms: {elapsed_ms}\n\n"
    log_path.write_text(header + sql.strip() + "\n")

    with INDEX_CSV.open("a", newline="") as f:
        csv.writer(f).writerow(
            [started.isoformat(), slug, sql_file or "", row_count, elapsed_ms, str(log_path.relative_to(ROOT))]
        )

    console.print(f"\n[dim]Logged → {log_path.relative_to(ROOT)}[/dim]")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sql", nargs="?", help="Inline SQL string")
    parser.add_argument("-f", "--file", type=Path, help="Path to a .sql file under analytics/queries/")
    parser.add_argument("-s", "--slug", default="query", help="Short name for the query log entry")
    args = parser.parse_args()

    if args.file:
        sql_path = args.file if args.file.is_absolute() else (ROOT / args.file)
        sql = sql_path.read_text()
        slug = args.slug if args.slug != "query" else sql_path.stem
        run_query(sql, slug=slug, sql_file=str(sql_path.relative_to(ROOT)))
    elif args.sql:
        run_query(args.sql, slug=args.slug)
    else:
        parser.error("Provide inline SQL or --file")


if __name__ == "__main__":
    main()
