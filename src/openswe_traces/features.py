"""Feature extraction over the corpus, streamed shard-by-shard.

Two instruments live here:

* per-trajectory structural proxy features (``proxy_features``), zero-GPU ranking
  signals written to ``outputs/proxy_features.parquet``;
* turn-level sample extraction (``extract_turn_sample``), a bounded sample of
  messages written to ``outputs/turn_sample.parquet``.

Neither ever loads the 42 GB corpus into memory.
"""

from __future__ import annotations

import argparse
import os
import sys
import time
import traceback
from collections import defaultdict
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from rich.console import Console

import duckdb

from .data import (
    DATA_ROOT,
    PARQUET_GLOB,
    ROOT,
    connect,
    connect_ephemeral,
    list_parquet_files,
    parse_shard,
)

console = Console()

PARTS_DIR = ROOT / "outputs" / "proxy_features_parts"
OUT_PATH = ROOT / "outputs" / "proxy_features.parquet"

EDIT_TOOL_NAMES = (
    "edit",
    "apply_patch",
    "edit_file",
    "create_file",
    "write_file",
    "str_replace",
    "insert",
)
EDIT_TOOL_VIEW_ONLY = "str_replace_editor"
BASH_EDIT_RE = r"\bsed\b[^|;&]*\s-i|cat\s*>|\btee\b|\bapply_patch\b|str_replace_editor|\bedit\b"
BASH_TEST_RE = r"\bpytest\b|\bgo\s+test\b|\bcargo\s+test\b|\bnpm\s+(run\s+)?test\b|\bjest\b|\bphpunit\b|\bmvn\s+test\b"
SUBMIT_RE = r"\bsubmit\b|\bfinish\b|complete_task_and_submit"

PROXY_FEATURES_DOC = """Zero-GPU per-trajectory structural proxy features for ranking traces before SFT.

Streams the corpus one parquet shard at a time (never loads the full 42 GB) and
writes one part file per shard to outputs/proxy_features_parts/. Part writes are
atomic (tmp + rename) and existing parts are skipped, so an interrupted run
resumes where it stopped. `--merge` folds the parts into outputs/proxy_features.parquet.

Feature definitions (see FEATURE_SQL):
  n_messages                        len(messages)
  n_assistant_turns                 messages with role='assistant'
  n_tool_calls                      tool calls across all assistant turns
  assistant_chars                   total len(content) over assistant turns
  reasoning_chars                   total len(reasoning_content) over assistant turns
  tool_obs_chars                    total len(content) over role='tool' observations
  n_distinct_tool_commands          distinct tool call argument strings (md5-compared)
  repeat_call_rate                  1 - distinct / total tool calls (0.0 when no calls)
  max_consecutive_identical_calls   longest run of identical argument strings
  n_edit_calls                      edit-type tool calls, or bash commands matching
                                    sed -i / cat > / tee / apply_patch / str_replace_editor /
                                    edit. str_replace_editor 'view' calls are not edits.
  n_test_calls                      bash commands matching pytest / go test / cargo test /
                                    npm test / jest / phpunit / mvn test
  model_patch_files / _lines        metadata.model_patch.num_modified_*
  gold_patch_files / _lines         metadata.reference_patch.num_modified_*
  patch_file_jaccard                Jaccard of file paths touched in the model patch vs the
                                    reference patch: 'diff --git a/X b/X', else '+++ b/X'.
                                    0.0 when the union is empty.
  ends_with_submit                  last assistant turn matches submit / finish /
                                    complete_task_and_submit

Arguments are matched through the JSON `$.command` field when present, so file
contents carried in edit args do not trigger test/edit patterns.

Examples:
  uv run openswe-features --limit 1 --head 5
  uv run openswe-features --threads 4 --memory-limit 6GB
  uv run openswe-features --merge-only
"""

TURN_SAMPLE_DOC = """Extract turn-level features using DuckDB streaming COPY only.

Phase 1: lightweight column projection (trajectory_id + filename) to locate files.
Phase 2: per-file COPY with unnest — never scans the full corpus for messages.

Example:
  uv run python scripts/extract_turn_sample.py --sample-size 15 --batch-size 3
"""

FEATURE_SQL = r"""
WITH src AS (
    SELECT instance_id, repo, language, trajectory_id, resolved, messages, metadata
    FROM read_parquet('{file_sql}', union_by_name=true)
),
msg AS (
    SELECT
        t.trajectory_id,
        count(*) FILTER (WHERE m.role = 'assistant') AS n_assistant_turns,
        sum(length(coalesce(m.content, ''))) FILTER (WHERE m.role = 'assistant') AS assistant_chars,
        sum(length(coalesce(m.reasoning_content, ''))) FILTER (WHERE m.role = 'assistant') AS reasoning_chars,
        sum(length(coalesce(m.content, ''))) FILTER (WHERE m.role = 'tool') AS tool_obs_chars
    FROM src t, unnest(t.messages) AS u(m)
    GROUP BY 1
),
calls AS (
    SELECT
        t.trajectory_id,
        row_number() OVER (PARTITION BY t.trajectory_id ORDER BY mi, ci) AS call_idx,
        md5(coalesce(tc.function.arguments, '')) AS call_sig,
        CASE
            WHEN lower(coalesce(tc.function.name, '')) = '{edit_view_only}'
                THEN (lower(coalesce(json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), '')) <> 'view')::TINYINT
            WHEN lower(coalesce(tc.function.name, '')) IN ({edit_tool_names})
                THEN 1::TINYINT
            WHEN regexp_matches(
                lower(coalesce(json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), '')),
                '{bash_edit_re}'
            ) THEN 1::TINYINT
            ELSE 0::TINYINT
        END AS is_edit,
        CASE
            WHEN regexp_matches(
                lower(coalesce(json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), '')),
                '{bash_test_re}'
            ) THEN 1::TINYINT
            ELSE 0::TINYINT
        END AS is_test
    FROM src t,
         unnest(t.messages) WITH ORDINALITY AS u(m, mi),
         unnest(m.tool_calls) WITH ORDINALITY AS v(tc, ci)
),
call_feats AS (
    SELECT
        trajectory_id,
        count(*) AS n_tool_calls,
        count(DISTINCT call_sig) AS n_distinct_tool_commands,
        sum(is_edit) AS n_edit_calls,
        sum(is_test) AS n_test_calls
    FROM calls
    GROUP BY 1
),
runs AS (
    SELECT
        trajectory_id,
        sum(CASE WHEN new_run THEN 1 ELSE 0 END) OVER (
            PARTITION BY trajectory_id ORDER BY call_idx
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS run_id
    FROM (
        SELECT
            trajectory_id,
            call_idx,
            (call_sig IS DISTINCT FROM lag(call_sig) OVER (
                PARTITION BY trajectory_id ORDER BY call_idx
            )) AS new_run
        FROM calls
    )
),
consec AS (
    SELECT trajectory_id, max(run_len) AS max_consecutive_identical_calls
    FROM (SELECT trajectory_id, run_id, count(*) AS run_len FROM runs GROUP BY 1, 2)
    GROUP BY 1
),
last_turn AS (
    SELECT
        trajectory_id,
        lower(
            coalesce(list_last(list_filter(messages, m -> m.role = 'assistant')).content, '') || ' ' ||
            coalesce(cast(list_last(list_filter(messages, m -> m.role = 'assistant')).tool_calls AS VARCHAR), '')
        ) AS last_assistant_text
    FROM src
),
ends AS (
    SELECT trajectory_id, regexp_matches(last_assistant_text, '{submit_re}') AS ends_with_submit
    FROM last_turn
),
patches AS (
    SELECT
        t.trajectory_id,
        coalesce(t.metadata.model_patch.patch, '') AS model_patch,
        coalesce(t.metadata.reference_patch.patch, '') AS gold_patch,
        coalesce(t.metadata.model_patch.num_modified_files, 0) AS model_patch_files,
        coalesce(t.metadata.model_patch.num_modified_lines, 0) AS model_patch_lines,
        coalesce(t.metadata.reference_patch.num_modified_files, 0) AS gold_patch_files,
        coalesce(t.metadata.reference_patch.num_modified_lines, 0) AS gold_patch_lines
    FROM src t
),
patch_paths AS (
    SELECT
        trajectory_id,
        CASE WHEN len(git_paths) > 0 THEN list_distinct(git_paths) ELSE list_distinct(plus_paths) END AS model_paths,
        CASE WHEN len(gold_git_paths) > 0 THEN list_distinct(gold_git_paths) ELSE list_distinct(gold_plus_paths) END AS gold_paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(model_patch, 'diff --git a/([^\s]+) b/', 1) AS git_paths,
            regexp_extract_all(model_patch, '\+\+\+ b/([^\s]+)', 1) AS plus_paths,
            regexp_extract_all(gold_patch, 'diff --git a/([^\s]+) b/', 1) AS gold_git_paths,
            regexp_extract_all(gold_patch, '\+\+\+ b/([^\s]+)', 1) AS gold_plus_paths
        FROM patches
    )
),
jaccard AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(list_distinct(list_concat(model_paths, gold_paths))) = 0 THEN 0.0
            ELSE len(list_filter(list_distinct(model_paths), p -> list_contains(gold_paths, p)))::DOUBLE
                 / len(list_distinct(list_concat(model_paths, gold_paths)))
        END AS patch_file_jaccard
    FROM patch_paths
)
SELECT
    t.trajectory_id,
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    t.instance_id,
    t.repo,
    t.language,
    t.metadata.category AS category,
    t.resolved::TINYINT AS resolved,
    len(t.messages)::INTEGER AS n_messages,
    coalesce(msg.n_assistant_turns, 0)::INTEGER AS n_assistant_turns,
    coalesce(cf.n_tool_calls, 0)::INTEGER AS n_tool_calls,
    coalesce(msg.assistant_chars, 0)::BIGINT AS assistant_chars,
    coalesce(msg.reasoning_chars, 0)::BIGINT AS reasoning_chars,
    coalesce(msg.tool_obs_chars, 0)::BIGINT AS tool_obs_chars,
    coalesce(cf.n_distinct_tool_commands, 0)::INTEGER AS n_distinct_tool_commands,
    CASE
        WHEN coalesce(cf.n_tool_calls, 0) = 0 THEN 0.0
        ELSE 1.0 - cf.n_distinct_tool_commands::DOUBLE / cf.n_tool_calls
    END AS repeat_call_rate,
    coalesce(consec.max_consecutive_identical_calls, 0)::INTEGER AS max_consecutive_identical_calls,
    coalesce(cf.n_edit_calls, 0)::INTEGER AS n_edit_calls,
    coalesce(cf.n_test_calls, 0)::INTEGER AS n_test_calls,
    p.model_patch_files::INTEGER AS model_patch_files,
    p.model_patch_lines::INTEGER AS model_patch_lines,
    p.gold_patch_files::INTEGER AS gold_patch_files,
    p.gold_patch_lines::INTEGER AS gold_patch_lines,
    j.patch_file_jaccard,
    e.ends_with_submit
FROM src t
LEFT JOIN msg ON msg.trajectory_id = t.trajectory_id
LEFT JOIN call_feats cf ON cf.trajectory_id = t.trajectory_id
LEFT JOIN consec ON consec.trajectory_id = t.trajectory_id
LEFT JOIN patches p ON p.trajectory_id = t.trajectory_id
LEFT JOIN jaccard j ON j.trajectory_id = t.trajectory_id
LEFT JOIN ends e ON e.trajectory_id = t.trajectory_id
"""

NUMERIC_FEATURES = [
    "n_messages",
    "n_assistant_turns",
    "n_tool_calls",
    "assistant_chars",
    "reasoning_chars",
    "tool_obs_chars",
    "n_distinct_tool_commands",
    "repeat_call_rate",
    "max_consecutive_identical_calls",
    "n_edit_calls",
    "n_test_calls",
    "model_patch_files",
    "model_patch_lines",
    "gold_patch_files",
    "gold_patch_lines",
    "patch_file_jaccard",
    "ends_with_submit",
]


def _sql_str(value: Any) -> str:
    return "'" + str(value).replace("'", "''") + "'"


def _tuple_sql(values: tuple[str, ...]) -> str:
    return ", ".join(_sql_str(v) for v in values)


def build_feature_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    return FEATURE_SQL.format(
        file_sql=str(file_path).replace("'", "''"),
        harness=harness.replace("'", "''"),
        teacher=teacher.replace("'", "''"),
        source=source.replace("'", "''"),
        edit_view_only=EDIT_TOOL_VIEW_ONLY,
        edit_tool_names=_tuple_sql(EDIT_TOOL_NAMES),
        bash_edit_re=BASH_EDIT_RE.replace("'", "''"),
        bash_test_re=BASH_TEST_RE.replace("'", "''"),
        submit_re=SUBMIT_RE.replace("'", "''"),
    )


def path_parts(file_path: Path) -> tuple[str, str, str]:
    """(harness, teacher, source) for a shard under traces_data/data/."""
    shard = parse_shard(file_path)
    return shard.harness, shard.teacher, shard.source


def part_path_for(file_path: Path) -> Path:
    rel = file_path.resolve().relative_to(DATA_ROOT)
    return PARTS_DIR / ("__".join(rel.with_suffix("").parts) + ".parquet")


def process_file(con: duckdb.DuckDBPyConnection, file_path: Path, *, force: bool) -> tuple[str, int]:
    """Return (status, rows) with status 'done' | 'skip'. Raises on row-count guard failure (part is removed)."""
    part = part_path_for(file_path)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    harness, teacher, source = path_parts(file_path)
    src_rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(file_path)})").fetchone()[0]

    # PID-scoped tmp so concurrent runs (this pipeline may be launched more than once on the
    # same parts dir) can never write into each other's staging file; the rename is atomic.
    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        con.execute(
            f"COPY ({build_feature_sql(file_path, harness, teacher, source)}) "
            f"TO {_sql_str(tmp)} (FORMAT PARQUET)"
        )
        out_rows, out_tids = con.execute(
            f"SELECT count(*), count(DISTINCT trajectory_id) FROM read_parquet({_sql_str(tmp)})"
        ).fetchone()
        if out_rows != src_rows or out_tids != src_rows:
            raise RuntimeError(
                f"row guard failed: source={src_rows} out={out_rows} "
                f"distinct_trajectories={out_tids}"
            )
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", out_rows


def merge_parts(
    con: duckdb.DuckDBPyConnection, out_path: Path = OUT_PATH
) -> tuple[int, int, int] | None:
    parts = sorted(PARTS_DIR.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = str(PARTS_DIR / "*.parquet").replace("'", "''")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(
        f"COPY (SELECT * FROM read_parquet('{glob_sql}') ORDER BY harness, teacher, source, trajectory_id) "
        f"TO {_sql_str(tmp)} (FORMAT PARQUET)"
    )
    rows, tids = con.execute(
        f"SELECT count(*), count(DISTINCT trajectory_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), rows, tids


def print_head_and_sanity(con: duckdb.DuckDBPyConnection, path: Path, head_rows: int) -> None:
    path_sql = _sql_str(path)
    head_cols = [
        "harness",
        "teacher",
        "n_messages",
        "n_tool_calls",
        "repeat_call_rate",
        "max_consecutive_identical_calls",
        "n_edit_calls",
        "n_test_calls",
        "model_patch_files",
        "patch_file_jaccard",
        "ends_with_submit",
        "resolved",
    ]
    head = con.execute(
        f"SELECT trajectory_id, {', '.join(head_cols)} FROM read_parquet({path_sql}) LIMIT {head_rows}"
    ).fetchdf()
    console.print(f"\n[bold]head({head_rows})[/bold] {path.name}")
    console.print(head.to_string(index=False, max_colwidth=28))

    rows, tids, files = con.execute(
        f"SELECT count(*), count(DISTINCT trajectory_id), count(DISTINCT source || teacher || harness) FROM read_parquet({path_sql})"
    ).fetchone()
    console.print(f"\n[bold]sanity[/bold] rows={rows:,} distinct_trajectory_id={tids:,} shards={files:,}")
    if rows != tids:
        console.print("[red]WARNING[/red] trajectory_id is not unique within the merged output")

    checks = con.execute(
        f"""
        SELECT
            count(*) FILTER (WHERE resolved = 1) AS r1,
            count(*) FILTER (WHERE resolved = 0) AS r0,
            count(*) FILTER (WHERE resolved = -1) AS r_unknown,
            count(*) FILTER (WHERE n_tool_calls = 0) AS zero_call_trajectories,
            count(*) FILTER (WHERE repeat_call_rate < 0 OR repeat_call_rate > 1) AS bad_repeat_rate,
            count(*) FILTER (WHERE patch_file_jaccard < 0 OR patch_file_jaccard > 1) AS bad_jaccard,
            count(*) FILTER (WHERE ends_with_submit) AS submitted,
            avg(n_messages) AS avg_messages,
            avg(n_tool_calls) AS avg_calls,
            avg(repeat_call_rate) AS avg_repeat_call_rate,
            avg(n_edit_calls) AS avg_edit_calls,
            avg(n_test_calls) AS avg_test_calls,
            avg(patch_file_jaccard) AS avg_jaccard,
            avg(CASE WHEN resolved IN (0, 1) THEN resolved::DOUBLE END) AS resolved_rate_known
        FROM read_parquet({path_sql})
        """
    ).fetchdf().iloc[0]
    console.print(
        f"resolved: 1={int(checks.r1):,} 0={int(checks.r0):,} -1={int(checks.r_unknown):,} "
        f"(resolved rate over known: {checks.resolved_rate_known:.3f})"
    )
    console.print(
        f"means: messages={checks.avg_messages:.1f} calls={checks.avg_calls:.1f} "
        f"repeat_call_rate={checks.avg_repeat_call_rate:.3f} edits={checks.avg_edit_calls:.1f} "
        f"tests={checks.avg_test_calls:.1f} jaccard={checks.avg_jaccard:.3f} "
        f"ends_with_submit={int(checks.submitted):,}"
    )
    console.print(
        f"guards: zero_call_trajectories={int(checks.zero_call_trajectories):,} "
        f"bad_repeat_rate={int(checks.bad_repeat_rate):,} bad_jaccard={int(checks.bad_jaccard):,}"
    )

    columns = con.execute(f"DESCRIBE SELECT * FROM read_parquet({path_sql})").fetchdf()
    console.print(f"columns: {len(columns)} → {', '.join(columns.column_name.tolist())}")


def proxy_features(args: argparse.Namespace) -> int:
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)

    if args.file:
        files = [Path(f).resolve() for f in args.file]
    else:
        files = list_parquet_files(args.data_glob)
    if args.limit:
        files = files[: args.limit]
    if not files:
        raise SystemExit(f"No parquet files matched {args.data_glob}")

    con = connect_ephemeral()
    con.execute(f"SET memory_limit='{args.memory_limit}'")
    con.execute(f"SET threads={args.threads}")

    done = skipped = failed = 0
    elapsed_total = 0.0
    started = time.monotonic()
    failures: list[str] = []

    try:
        for i, file_path in enumerate(files, start=1):
            shard = parse_shard(file_path)
            label = shard.label
            t0 = time.monotonic()
            try:
                status, rows = process_file(con, file_path, force=args.force)
            except Exception as exc:  # noqa: BLE001 (keep going; the shard stays unprocessed)
                failed += 1
                failures.append(label)
                console.print(f"[{utcnow()}] [{i}/{len(files)}] [red]FAILED[/red] {label}: {exc}")
                console.print(f"[dim]{traceback.format_exc(limit=2)}[/dim]")
                continue
            elapsed = time.monotonic() - t0
            if status == "skip":
                skipped += 1
                console.print(f"[{utcnow()}] [{i}/{len(files)}] skipped (part exists) {label}")
                continue
            done += 1
            elapsed_total += elapsed
            rate = elapsed_total / max(done, 1)
            eta = rate * (len(files) - i)
            console.print(
                f"[{utcnow()}] [{i}/{len(files)}] [green]done[/green] {label} "
                f"rows={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )

        if not args.no_merge:
            merged = merge_parts(con)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, tids = merged
                processed_all = not args.file and not args.limit
                suffix = (
                    ""
                    if processed_all
                    else " (partial run: rerun without --file/--limit to cover the corpus)"
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {OUT_PATH.relative_to(ROOT)}: "
                    f"{rows:,} rows, {tids:,} distinct trajectories{suffix}"
                )
                if args.head:
                    print_head_and_sanity(con, OUT_PATH, args.head)
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def main_proxy_features() -> None:
    parser = argparse.ArgumentParser(
        description=PROXY_FEATURES_DOC,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument("--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: the corpus)")
    parser.add_argument("--force", action="store_true", help="Recompute parts that already exist")
    parser.add_argument("--no-merge", action="store_true", help="Leave shard parts unmerged")
    parser.add_argument("--merge-only", action="store_true", help="Skip processing; merge existing parts")
    parser.add_argument("--head", type=int, default=0, help="Print first N merged rows + sanity summary")
    parser.add_argument("--threads", type=int, default=4, help="DuckDB threads for this batch job")
    parser.add_argument("--memory-limit", default="6GB", help="DuckDB memory limit for this batch job")
    args = parser.parse_args()

    if args.merge_only:
        con = connect_ephemeral()
        con.execute(f"SET memory_limit='{args.memory_limit}'")
        con.execute(f"SET threads={args.threads}")
        try:
            merged = merge_parts(con)
            if merged is None:
                raise SystemExit(f"No parts found in {PARTS_DIR}")
            n_parts, rows, tids = merged
            console.print(
                f"[{utcnow()}] merged {n_parts} parts → {OUT_PATH.relative_to(ROOT)}: "
                f"{rows:,} rows, {tids:,} trajectories"
            )
            if args.head:
                print_head_and_sanity(con, OUT_PATH, args.head)
        finally:
            con.close()
        return

    sys.exit(proxy_features(args))


def utcnow() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def fmt_duration(seconds: float) -> str:
    seconds = int(seconds)
    if seconds < 60:
        return f"{seconds}s"
    if seconds < 3600:
        return f"{seconds // 60}m{seconds % 60:02d}s"
    return f"{seconds // 3600}h{(seconds % 3600) // 60:02d}m"


DEFAULT_TURN_OUT = ROOT / "outputs" / "turn_sample.parquet"

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


def extract_turn_sample(
    *,
    sample_size: int = 30,
    seed: int = 42,
    output: Path = DEFAULT_TURN_OUT,
    register_duckdb: bool = False,
) -> Path:
    ids = sample_trajectory_ids(sample_size, seed)
    console.print(f"Target {len(ids)} trials | DuckDB streaming | memory_limit=4GB")

    con = connect()
    by_file = lookup_files(con, ids)
    console.print(f"Located trials across {len(by_file)} parquet files")
    con.close()

    if not by_file:
        raise SystemExit("No matching trajectories found in parquet files")

    output.parent.mkdir(parents=True, exist_ok=True)
    if output.exists():
        output.unlink()

    con = connect()
    try:
        for i, (file_path, file_ids) in enumerate(by_file.items()):
            insert_file_batch(con, file_path, file_ids, first=(i == 0))
            console.print(f"  file {i + 1}/{len(by_file)}: {len(file_ids)} trials")
        export_staging(con, output)
    finally:
        con.close()

    out_sql = str(output).replace("'", "''")
    total = connect(read_only=True).execute(
        f"SELECT count(*) FROM read_parquet('{out_sql}')"
    ).fetchone()[0]
    console.print(f"[green]Wrote[/green] {total:,} turn rows → {output}")

    if register_duckdb:
        register_view(output.resolve())
    return output


def main_turn_sample() -> None:
    parser = argparse.ArgumentParser(
        description=TURN_SAMPLE_DOC,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--sample-size", type=int, default=30)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--output", type=Path, default=DEFAULT_TURN_OUT)
    parser.add_argument("--register-duckdb", action="store_true")
    args = parser.parse_args()

    extract_turn_sample(
        sample_size=args.sample_size,
        seed=args.seed,
        output=args.output,
        register_duckdb=args.register_duckdb,
    )
