"""Shared DuckDB connection settings for this project."""

from __future__ import annotations

from pathlib import Path

import duckdb

ROOT = Path(__file__).resolve().parent.parent
DB_PATH = ROOT / "duckdb" / "open_swe.duckdb"
TEMP_DIR = ROOT / "duckdb" / "tmp"
PARQUET_GLOB = str(ROOT / "traces_data" / "data" / "*" / "*" / "*" / "*.parquet")

MEMORY_LIMIT = "4GB"
THREADS = 2


def configure_connection(con: duckdb.DuckDBPyConnection) -> None:
    TEMP_DIR.mkdir(parents=True, exist_ok=True)
    con.execute(f"SET memory_limit='{MEMORY_LIMIT}'")
    con.execute(f"SET threads={THREADS}")
    con.execute("SET preserve_insertion_order=false")
    con.execute(f"SET temp_directory='{TEMP_DIR}'")


def connect(*, read_only: bool = False) -> duckdb.DuckDBPyConnection:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    con = duckdb.connect(str(DB_PATH), read_only=read_only)
    configure_connection(con)
    return con


def connect_ephemeral() -> duckdb.DuckDBPyConnection:
    con = duckdb.connect()
    configure_connection(con)
    return con


def state_local_sql() -> str:
    return "\n".join(
        [
            f"SET memory_limit='{MEMORY_LIMIT}';",
            f"SET threads={THREADS};",
            "SET preserve_insertion_order=false;",
            f"SET temp_directory='{TEMP_DIR}';",
            f"SET variable project_root = '{ROOT}';",
            f"SET variable parquet_glob = '{PARQUET_GLOB}';",
        ]
    ) + "\n"
