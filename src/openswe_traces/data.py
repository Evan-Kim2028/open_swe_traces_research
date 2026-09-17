"""Repo paths, parquet shard discovery, and the 4GB-capped DuckDB session.

All paths are anchored to the repository root, so every command runs from anywhere in
the checkout. DuckDB connections stream from parquet; the corpus is never loaded whole.
"""

from __future__ import annotations

import argparse
import csv
import glob as globlib
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

from rich.console import Console

import duckdb

ROOT = Path(__file__).resolve().parents[2]
DATA_DIR = ROOT / "traces_data"
DATA_ROOT = DATA_DIR / "data"
DB_PATH = ROOT / "duckdb" / "open_swe.duckdb"
TEMP_DIR = ROOT / "duckdb" / "tmp"
PARQUET_GLOB = str(DATA_ROOT / "*" / "*" / "*" / "*.parquet")
SCHEMA_DIR = ROOT / "analytics" / "schema"

MEMORY_LIMIT = "4GB"
THREADS = 2

QUERY_LOG_DIR = ROOT / "analytics" / "query_log"
INDEX_CSV = QUERY_LOG_DIR / "index.csv"
STREAM_BATCH = 500

console = Console()


@dataclass(frozen=True)
class Shard:
    """One parquet shard, decomposed from data/<harness>/<teacher>/<source>/<file>."""

    path: Path
    rel: Path
    harness: str
    teacher: str
    source: str

    @property
    def label(self) -> str:
        return f"{self.harness}/{self.teacher}/{self.source}/{self.path.name}"


def parse_shard(path: Path | str, root: Path = DATA_ROOT) -> Shard:
    """Split a shard path into (harness, teacher, source); raise on unexpected layouts."""
    path = Path(path)
    rel = path.resolve().relative_to(root)
    if len(rel.parts) != 4:
        raise ValueError(
            f"Unexpected shard path (want data/<harness>/<teacher>/<source>/<file>): {path}"
        )
    return Shard(
        path=path,
        rel=rel,
        harness=rel.parts[0],
        teacher=rel.parts[1],
        source=rel.parts[2],
    )


def list_parquet_files(pattern: str = PARQUET_GLOB) -> list[Path]:
    return sorted(Path(p) for p in globlib.glob(pattern))


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


def init_db(*, refresh_summaries: bool = False) -> Path:
    """Build or refresh the local DuckDB analytics database and views."""
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    (ROOT / "analytics" / "state.local.sql").write_text(state_local_sql())

    con = connect()
    schema_files = sorted(SCHEMA_DIR.glob("*.sql"))
    if not schema_files:
        raise SystemExit(f"No schema files in {SCHEMA_DIR}")

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


def duckdb_init_main() -> None:
    parser = argparse.ArgumentParser(
        description="Build or refresh the local DuckDB analytics database and views."
    )
    parser.add_argument(
        "--refresh-summaries",
        action="store_true",
        help="Rebuild summary tables (slower; run after large download chunks)",
    )
    args = parser.parse_args()
    path = init_db(refresh_summaries=args.refresh_summaries)
    console.print(f"Database: {path}")
    console.print("CLI: duckdb -init analytics/state.sql duckdb/open_swe.duckdb")


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
            [
                started.isoformat(),
                slug,
                sql_file or "",
                row_count,
                elapsed_ms,
                str(log_path.relative_to(ROOT)),
            ]
        )

    console.print(f"\n[dim]Logged → {log_path.relative_to(ROOT)}[/dim]")


def duckdb_query_main() -> None:
    parser = argparse.ArgumentParser(
        description="Run SQL against open_swe.duckdb with streaming results and query logging."
    )
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
