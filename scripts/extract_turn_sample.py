#!/usr/bin/env python3
"""Extract turn-level features using DuckDB streaming COPY only.

Phase 1: lightweight column projection (trajectory_id + filename) to locate files.
Phase 2: per-file COPY with unnest — never scans the full corpus for messages.

Example:
  uv run python scripts/extract_turn_sample.py --sample-size 15 --batch-size 3
"""

from __future__ import annotations

import argparse
import sys
from collections import defaultdict
from pathlib import Path

from rich.console import Console

sys.path.insert(0, str(Path(__file__).resolve().parent))
from duckdb_session import PARQUET_GLOB, connect

console = Console()

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUT = ROOT / "outputs" / "turn_sample.parquet"

TURN_EXTRACT_BODY = """
WITH src AS (
    SELECT
        *,
        filename AS _source_file
    FROM read_parquet('{file_sql}', union_by_name=true, filename=true)
    WHERE trajectory_id IN (SELECT unnest(?::VARCHAR[]))
),
turns AS (
    SELECT
        trajectory_id,
        instance_id,
        resolved,
        CASE resolved WHEN 1 THEN 'success' WHEN 0 THEN 'failed' ELSE 'unknown' END
            AS outcome_label,
        regexp_extract(_source_file, '.*/data/([^/]+)/', 1) AS harness,
        regexp_extract(_source_file, '.*/data/[^/]+/([^/]+)/', 1) AS teacher_model,
        language,
        metadata.category AS category,
        turn_idx::INTEGER AS turn_idx,
        len(messages)::INTEGER AS total_turns,
        round(turn_idx::DOUBLE / len(messages), 4)::FLOAT AS pct_through,
        msg.role AS role,
        coalesce(length(msg.content), 0)::INTEGER AS content_len,
        (
            msg.role = 'assistant'
            AND json_extract(to_json(msg), '$.tool_calls') IS NOT NULL
        ) AS has_tool_call,
        json_extract_string(to_json(msg), '$.tool_calls[0].function.name') AS tool_names,
        left(json_extract_string(to_json(msg), '$.tool_calls[0].function.arguments'), 120)
            AS bash_command_prefix,
        (
            msg.role = 'assistant'
            AND regexp_matches(
                coalesce(json_extract_string(to_json(msg), '$.tool_calls'), ''),
                '(sed |str_replace|patch|write_file|apply_patch)',
                'i'
            )
        ) AS is_edit_command,
        (
            msg.role = 'assistant'
            AND regexp_matches(
                coalesce(json_extract_string(to_json(msg), '$.tool_calls'), ''),
                '(pytest|npm test|go test|make test|tox|cargo test)',
                'i'
            )
        ) AS is_test_command,
        (
            msg.role = 'tool'
            AND NOT regexp_matches(coalesce(msg.content, ''), 'returncode.: 0', 'i')
            AND regexp_matches(coalesce(msg.content, ''), '(error|traceback)', 'i')
        ) AS is_tool_error_turn
    FROM src,
         unnest(messages) WITH ORDINALITY AS u(msg, turn_idx)
)
SELECT
    trajectory_id,
    instance_id,
    resolved::TINYINT AS resolved,
    outcome_label,
    harness,
    teacher_model,
    language,
    category,
    turn_idx,
    total_turns,
    pct_through,
    role,
    content_len,
    has_tool_call,
    tool_names,
    bash_command_prefix,
    is_edit_command,
    is_test_command,
    sum(CASE WHEN role = 'assistant' THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_assistant,
    sum(CASE WHEN role = 'tool' THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_tool,
    sum(CASE WHEN has_tool_call THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_tool_calls,
    sum(CASE WHEN is_edit_command THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_edits,
    sum(CASE WHEN is_test_command THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_tests,
    sum(CASE WHEN is_tool_error_turn THEN 1 ELSE 0 END)
        OVER (PARTITION BY trajectory_id ORDER BY turn_idx)::INTEGER AS cum_tool_errors
FROM turns
"""


def sample_trajectory_ids(n: int, seed: int) -> list[str]:
    con = connect(read_only=True)
    per_bucket = max(1, n // 3)
    ids: list[str] = []
    for resolved in (1, 0, -1):
        rows = con.execute(
            """
            SELECT trajectory_id FROM trial_summary
            WHERE resolved = ?
            ORDER BY hash(trajectory_id || ?)
            LIMIT ?
            """,
            [resolved, str(seed), per_bucket],
        ).fetchall()
        ids.extend(r[0] for r in rows)
    con.close()
    return ids[:n]


def lookup_files(con, ids: list[str]) -> dict[str, list[str]]:
    """Map parquet file → trajectory_ids (reads only trajectory_id + filename columns)."""
    glob_sql = PARQUET_GLOB.replace("'", "''")
    rows = con.execute(
        f"""
        SELECT trajectory_id, filename
        FROM read_parquet('{glob_sql}', union_by_name=true, filename=true)
        WHERE trajectory_id IN (SELECT unnest(?::VARCHAR[]))
        """,
        [ids],
    ).fetchall()
    by_file: dict[str, list[str]] = defaultdict(list)
    for tid, fname in rows:
        by_file[str(fname)].append(str(tid))
    return by_file


def insert_file_batch(con, file_path: str, ids: list[str], *, first: bool) -> None:
    file_sql = file_path.replace("'", "''")
    body = TURN_EXTRACT_BODY.format(file_sql=file_sql)
    if first:
        con.execute(f"CREATE OR REPLACE TEMP TABLE turn_staging AS {body}", [ids])
    else:
        con.execute(f"INSERT INTO turn_staging {body}", [ids])


def export_staging(con, out_path: Path) -> None:
    out_sql = str(out_path).replace("'", "''")
    con.execute(f"COPY turn_staging TO '{out_sql}' (FORMAT PARQUET)")


def register_view(parquet_path: Path) -> None:
    con = connect()
    p = str(parquet_path).replace("'", "''")
    con.execute(f"CREATE OR REPLACE VIEW turn_sample AS SELECT * FROM read_parquet('{p}');")
    n, trials = con.execute(
        "SELECT count(*), count(DISTINCT trajectory_id) FROM turn_sample"
    ).fetchone()
    con.close()
    console.print(f"[green]Registered turn_sample[/green] — {n:,} turns, {trials:,} trials")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sample-size", type=int, default=30)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUT)
    parser.add_argument("--register-duckdb", action="store_true")
    args = parser.parse_args()

    ids = sample_trajectory_ids(args.sample_size, args.seed)
    console.print(f"Target {len(ids)} trials | DuckDB streaming | memory_limit=4GB")

    con = connect()
    by_file = lookup_files(con, ids)
    console.print(f"Located trials across {len(by_file)} parquet files")
    con.close()

    if not by_file:
        raise SystemExit("No matching trajectories found in parquet files")

    args.output.parent.mkdir(parents=True, exist_ok=True)
    if args.output.exists():
        args.output.unlink()

    con = connect()
    try:
        for i, (file_path, file_ids) in enumerate(by_file.items()):
            insert_file_batch(con, file_path, file_ids, first=(i == 0))
            console.print(f"  file {i + 1}/{len(by_file)}: {len(file_ids)} trials")
        export_staging(con, args.output)
    finally:
        con.close()

    out_sql = str(args.output).replace("'", "''")
    total = connect(read_only=True).execute(
        f"SELECT count(*) FROM read_parquet('{out_sql}')"
    ).fetchone()[0]
    console.print(f"[green]Wrote[/green] {total:,} turn rows → {args.output}")

    if args.register_duckdb:
        register_view(args.output.resolve())


if __name__ == "__main__":
    main()
