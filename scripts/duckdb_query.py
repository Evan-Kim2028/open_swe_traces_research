#!/usr/bin/env python3
"""Run SQL against open_swe.duckdb with streaming results and query logging."""

from __future__ import annotations

import argparse
import csv
import sys
from datetime import UTC, datetime
from pathlib import Path

from rich.console import Console

sys.path.insert(0, str(Path(__file__).resolve().parent))
from duckdb_session import DB_PATH, ROOT, connect

console = Console()

QUERY_LOG_DIR = ROOT / "analytics" / "query_log"
INDEX_CSV = QUERY_LOG_DIR / "index.csv"
STREAM_BATCH = 500


def _ensure_index() -> None:
    QUERY_LOG_DIR.mkdir(parents=True, exist_ok=True)
    if not INDEX_CSV.exists():
        with INDEX_CSV.open("w", newline="") as f:
            csv.writer(f).writerow(
                ["timestamp", "slug", "sql_file", "row_count", "elapsed_ms", "log_path"]
            )


def _strip_sql_comments(sql: str) -> str:
    return "\n".join(line for line in sql.splitlines() if not line.lstrip().startswith("--"))


def _stream_print(result) -> int:
    """Print result batches without loading full dataframe."""
    if not result.description:
        return 0
    columns = [d[0] for d in result.description]
    console.print(" | ".join(columns))
    console.print("-" * min(120, 8 * len(columns)))
    row_count = 0
    while True:
        rows = result.fetchmany(STREAM_BATCH)
        if not rows:
            break
        for row in rows:
            console.print(" | ".join(str(v) for v in row))
            row_count += 1
    return row_count


def run_query(sql: str, *, slug: str, sql_file: str | None = None) -> None:
    if not DB_PATH.exists():
        raise SystemExit("Run: uv run python scripts/duckdb_init.py")

    _ensure_index()
    started = datetime.now(UTC)
    con = connect(read_only=True)

    row_count = 0
    try:
        statements = [s.strip() for s in _strip_sql_comments(sql).split(";") if s.strip()]
        for i, statement in enumerate(statements):
            result = con.execute(statement)
            if result.description:
                if len(statements) > 1:
                    console.print(f"\n[bold]Result {i + 1}/{len(statements)}[/bold]")
                row_count += _stream_print(result)
        if row_count == 0:
            console.print("[dim](no result set)[/dim]")
    finally:
        con.close()

    elapsed_ms = int((datetime.now(UTC) - started).total_seconds() * 1000)
    stamp = started.strftime("%Y-%m-%d_%H%M%S")
    log_path = QUERY_LOG_DIR / f"{stamp}_{slug}.sql"
    header = (
        f"-- slug: {slug}\n-- sql_file: {sql_file or 'inline'}\n"
        f"-- rows: {row_count}\n-- elapsed_ms: {elapsed_ms}\n\n"
    )
    log_path.write_text(header + sql.strip() + "\n")

    with INDEX_CSV.open("a", newline="") as f:
        csv.writer(f).writerow(
            [started.isoformat(), slug, sql_file or "", row_count, elapsed_ms, str(log_path.relative_to(ROOT))]
        )

    console.print(f"\n[dim]Logged → {log_path.relative_to(ROOT)}[/dim]")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sql", nargs="?", help="Inline SQL string")
    parser.add_argument("-f", "--file", type=Path, help="Path to a .sql file")
    parser.add_argument("-s", "--slug", default="query", help="Short name for query log")
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
