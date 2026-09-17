#!/usr/bin/env python3
"""Build or refresh the local DuckDB analytics database and views."""

from __future__ import annotations

import argparse
from pathlib import Path

import duckdb
from rich.console import Console

console = Console()

ROOT = Path(__file__).resolve().parent.parent
DB_PATH = ROOT / "duckdb" / "open_swe.duckdb"
SCHEMA_DIR = ROOT / "analytics" / "schema"
PARQUET_GLOB = str(ROOT / "traces_data" / "data" / "*" / "*" / "*" / "*.parquet")


def _write_state_local() -> None:
    path = ROOT / "analytics" / "state.local.sql"
    path.write_text(
        "\n".join(
            [
                f"SET variable project_root = '{ROOT}';",
                f"SET variable parquet_glob = '{PARQUET_GLOB}';",
            ]
        )
        + "\n"
    )


def init_db(*, refresh_summaries: bool = False) -> Path:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    _write_state_local()
    con = duckdb.connect(str(DB_PATH))

    schema_files = sorted(SCHEMA_DIR.glob("*.sql"))
    if not schema_files:
        raise SystemExit(f"No schema files in {SCHEMA_DIR}")

    # Ensure _config exists before views reference it.
    con.execute(
        """
        CREATE TABLE IF NOT EXISTS _config (
            parquet_glob VARCHAR,
            project_root VARCHAR,
            updated_at TIMESTAMP
        );
        """
    )
    con.execute(
        """
        DELETE FROM _config;
        INSERT INTO _config (parquet_glob, project_root, updated_at)
        VALUES (?, ?, current_timestamp);
        """,
        [PARQUET_GLOB, str(ROOT)],
    )

    for sql_file in schema_files:
        if sql_file.name == "004_summaries.sql" and not refresh_summaries:
            console.print(f"Skipping {sql_file.name} (pass --refresh-summaries to build)")
            continue
        console.print(f"Applying {sql_file.name}")
        sql = sql_file.read_text().replace("{{PARQUET_GLOB}}", PARQUET_GLOB.replace("'", "''"))
        con.execute(sql)

    row_count = con.execute("SELECT count(*) FROM traces_raw").fetchone()[0]
    console.print(f"[green]Ready.[/green] traces_raw rows: {row_count:,}")
    con.close()
    return DB_PATH


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--refresh-summaries",
        action="store_true",
        help="Rebuild summary tables (slower; run after large download chunks)",
    )
    args = parser.parse_args()
    path = init_db(refresh_summaries=args.refresh_summaries)
    console.print(f"Database: {path}")
    console.print("CLI: duckdb -init analytics/state.sql duckdb/open_swe.duckdb")


if __name__ == "__main__":
    main()
