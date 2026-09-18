"""Temporal gold-file features over the corpus, streamed shard-by-shard.

For every trajectory the gold file set is parsed from ``metadata.reference_patch.patch``
(``diff --git a/X b/X`` paths, falling back to ``+++ b/X``) and each assistant tool call is
scanned in order to locate the first call whose arguments mention a gold file and the first
edit-type call touching one. Writes one part per shard to
``outputs/temporal_features_parts/`` (atomic, resume-safe), merged into
``outputs/temporal_features.parquet`` by ``merge_parts`` from ``features``.

Feature definitions (see TEMPORAL_SQL):

  n_turns_total                    number of messages in the trajectory (all roles)
  n_tool_calls                     tool calls across all assistant turns
  turn_first_gold_view             1-based index of the first tool call whose arguments
                                   mention any gold file path or its basename; -1 if never
  turn_first_gold_edit             1-based index of the first edit-type call touching a gold
                                   file; -1 if never (a gold edit always implies a gold view)
  frac_first_gold_view             turn_first_gold_view / n_tool_calls; null if never
  frac_first_gold_edit             turn_first_gold_edit / n_tool_calls; null if never
  n_edits_outside_gold             edit-type calls whose arguments mention no gold file
  n_gold_files                     size of the gold file set
  n_gold_files_touched             gold files mentioned by at least one tool call
  gold_file_recall                 n_gold_files_touched / n_gold_files; null when the gold
                                   set is empty
  frac_calls_after_first_gold_edit (n_tool_calls - turn_first_gold_edit) / n_tool_calls;
                                   null if never

Mentioning is a case-sensitive substring test against the raw tool-call arguments, so the
``/testbed/``-prefixed paths agents type match through the basename. Edit-call detection
reuses the proxy-feature regexes (tool names + bash command patterns).

Examples:
  uv run openswe-temporal --limit 1 --head 5
  uv run openswe-temporal --threads 6 --memory-limit 8GB
  uv run openswe-temporal --merge-only
"""

from __future__ import annotations

import argparse
import sys
import time
import traceback
from pathlib import Path

from rich.console import Console

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .features import (
    BASH_EDIT_RE,
    EDIT_TOOL_NAMES,
    EDIT_TOOL_VIEW_ONLY,
    _sql_str,
    _tuple_sql,
    fmt_duration,
    merge_parts,
    process_file,
    utcnow,
)

console = Console()

PARTS_DIR = ROOT / "outputs" / "temporal_features_parts"
OUT_PATH = ROOT / "outputs" / "temporal_features.parquet"

TEMPORAL_FEATURES = [
    "n_turns_total",
    "n_tool_calls",
    "turn_first_gold_view",
    "turn_first_gold_edit",
    "frac_first_gold_view",
    "frac_first_gold_edit",
    "n_edits_outside_gold",
    "n_gold_files",
    "n_gold_files_touched",
    "gold_file_recall",
    "frac_calls_after_first_gold_edit",
]

TEMPORAL_SQL = r"""
WITH src AS (
    SELECT instance_id, trajectory_id, resolved, messages, metadata
    FROM read_parquet('{file_sql}', union_by_name=true)
),
gold AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(git_paths) > 0 THEN list_distinct(git_paths)
            ELSE list_distinct(plus_paths)
        END AS gold_paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(
                coalesce(metadata.reference_patch.patch, ''), 'diff --git a/([^\s]+) b/', 1
            ) AS git_paths,
            regexp_extract_all(
                coalesce(metadata.reference_patch.patch, ''), '\+\+\+ b/([^\s]+)', 1
            ) AS plus_paths
        FROM src
    )
),
joined AS (
    SELECT t.trajectory_id, t.instance_id, t.resolved, t.messages, g.gold_paths
    FROM src t
    LEFT JOIN gold g ON g.trajectory_id = t.trajectory_id
),
calls_src AS (
    SELECT
        t.trajectory_id,
        t.gold_paths,
        mi,
        ci,
        coalesce(tc.function.arguments, '') AS args_text,
        lower(coalesce(tc.function.name, '')) AS tool_name,
        json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command') AS command_text
    FROM joined t,
         unnest(t.messages) WITH ORDINALITY AS u(m, mi),
         unnest(m.tool_calls) WITH ORDINALITY AS v(tc, ci)
),
calls AS (
    SELECT
        trajectory_id,
        row_number() OVER (PARTITION BY trajectory_id ORDER BY mi, ci) AS call_idx,
        CASE
            WHEN tool_name = '{edit_view_only}'
                THEN (lower(coalesce(command_text, '')) <> 'view')::TINYINT
            WHEN tool_name IN ({edit_tool_names})
                THEN 1::TINYINT
            WHEN regexp_matches(lower(coalesce(command_text, '')), '{bash_edit_re}')
                THEN 1::TINYINT
            ELSE 0::TINYINT
        END AS is_edit,
        list_filter(
            gold_paths,
            p -> contains(args_text, p)
              OR contains(args_text, regexp_extract(p, '[^/]+$'))
        ) AS matched_gold
    FROM calls_src
),
call_feats AS (
    SELECT
        trajectory_id,
        count(*) AS n_tool_calls,
        min(call_idx) FILTER (WHERE len(matched_gold) > 0) AS first_gold_view,
        min(call_idx) FILTER (WHERE is_edit = 1 AND len(matched_gold) > 0) AS first_gold_edit,
        count(*) FILTER (WHERE is_edit = 1 AND len(matched_gold) = 0) AS n_edits_outside_gold,
        list_distinct(flatten(list(matched_gold))) AS gold_touched
    FROM calls
    GROUP BY 1
)
SELECT
    t.trajectory_id,
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    t.instance_id,
    t.resolved::TINYINT AS resolved,
    len(t.messages)::INTEGER AS n_turns_total,
    coalesce(cf.n_tool_calls, 0)::INTEGER AS n_tool_calls,
    coalesce(cf.first_gold_view, -1)::INTEGER AS turn_first_gold_view,
    coalesce(cf.first_gold_edit, -1)::INTEGER AS turn_first_gold_edit,
    CASE
        WHEN cf.first_gold_view IS NULL OR cf.n_tool_calls = 0 THEN NULL
        ELSE cf.first_gold_view::DOUBLE / cf.n_tool_calls
    END AS frac_first_gold_view,
    CASE
        WHEN cf.first_gold_edit IS NULL OR cf.n_tool_calls = 0 THEN NULL
        ELSE cf.first_gold_edit::DOUBLE / cf.n_tool_calls
    END AS frac_first_gold_edit,
    coalesce(cf.n_edits_outside_gold, 0)::INTEGER AS n_edits_outside_gold,
    len(t.gold_paths)::INTEGER AS n_gold_files,
    coalesce(len(cf.gold_touched), 0)::INTEGER AS n_gold_files_touched,
    CASE
        WHEN len(t.gold_paths) = 0 THEN NULL
        ELSE len(cf.gold_touched)::DOUBLE / len(t.gold_paths)
    END AS gold_file_recall,
    CASE
        WHEN cf.first_gold_edit IS NULL OR cf.n_tool_calls = 0 THEN NULL
        ELSE (cf.n_tool_calls - cf.first_gold_edit)::DOUBLE / cf.n_tool_calls
    END AS frac_calls_after_first_gold_edit
FROM joined t
LEFT JOIN call_feats cf ON cf.trajectory_id = t.trajectory_id
"""


def build_temporal_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    return TEMPORAL_SQL.format(
        file_sql=str(file_path).replace("'", "''"),
        harness=harness.replace("'", "''"),
        teacher=teacher.replace("'", "''"),
        source=source.replace("'", "''"),
        edit_view_only=EDIT_TOOL_VIEW_ONLY,
        edit_tool_names=_tuple_sql(EDIT_TOOL_NAMES),
        bash_edit_re=BASH_EDIT_RE.replace("'", "''"),
    )


def print_head_and_sanity(con: duckdb.DuckDBPyConnection, path: Path, head_rows: int) -> None:
    path_sql = _sql_str(path)
    head_cols = [
        "harness",
        "teacher",
        "n_turns_total",
        "n_tool_calls",
        "turn_first_gold_view",
        "turn_first_gold_edit",
        "frac_first_gold_view",
        "n_edits_outside_gold",
        "n_gold_files",
        "n_gold_files_touched",
        "gold_file_recall",
        "frac_calls_after_first_gold_edit",
        "resolved",
    ]
    head = con.execute(
        f"SELECT trajectory_id, {', '.join(head_cols)} FROM read_parquet({path_sql}) LIMIT {head_rows}"
    ).fetchdf()
    console.print(f"\n[bold]head({head_rows})[/bold] {path.name}")
    console.print(head.to_string(index=False, max_colwidth=28))

    rows, tids, files = con.execute(
        f"SELECT count(*), count(DISTINCT trajectory_id), "
        f"count(DISTINCT source || teacher || harness) FROM read_parquet({path_sql})"
    ).fetchone()
    console.print(
        f"\n[bold]sanity[/bold] rows={rows:,} distinct_trajectory_id={tids:,} shards={files:,}"
    )
    if rows != tids:
        console.print("[red]WARNING[/red] trajectory_id is not unique within the merged output")

    checks = (
        con.execute(
            f"""
        SELECT
            count(*) FILTER (WHERE resolved = 1) AS r1,
            count(*) FILTER (WHERE resolved = 0) AS r0,
            count(*) FILTER (WHERE resolved = -1) AS r_unknown,
            count(*) FILTER (WHERE n_tool_calls = 0) AS zero_call_trajectories,
            count(*) FILTER (
                WHERE turn_first_gold_view < -1 OR turn_first_gold_view > n_tool_calls
            ) AS bad_first_view,
            count(*) FILTER (
                WHERE turn_first_gold_edit < -1 OR turn_first_gold_edit > n_tool_calls
            ) AS bad_first_edit,
            count(*) FILTER (
                WHERE frac_first_gold_view IS NOT NULL
                  AND (frac_first_gold_view < 0 OR frac_first_gold_view > 1)
            ) AS bad_frac_view,
            count(*) FILTER (
                WHERE frac_first_gold_edit IS NOT NULL
                  AND (frac_first_gold_edit < 0 OR frac_first_gold_edit > 1)
            ) AS bad_frac_edit,
            count(*) FILTER (
                WHERE (turn_first_gold_view = -1) <> (frac_first_gold_view IS NULL)
            ) AS bad_view_sentinel,
            count(*) FILTER (
                WHERE (turn_first_gold_edit = -1) <> (frac_first_gold_edit IS NULL)
            ) AS bad_edit_sentinel,
            count(*) FILTER (
                WHERE turn_first_gold_edit <> -1
                  AND (turn_first_gold_view = -1 OR turn_first_gold_edit < turn_first_gold_view)
            ) AS edit_before_view,
            count(*) FILTER (WHERE n_gold_files = 0) AS empty_gold_sets,
            count(*) FILTER (WHERE n_gold_files_touched > n_gold_files) AS bad_touch_count,
            count(*) FILTER (
                WHERE (n_gold_files = 0) <> (gold_file_recall IS NULL)
            ) AS bad_recall_sentinel,
            avg(n_turns_total) AS avg_turns,
            avg(n_tool_calls) AS avg_calls,
            avg(turn_first_gold_view) AS avg_first_view,
            avg(turn_first_gold_edit) AS avg_first_edit,
            avg(frac_first_gold_view) AS avg_frac_view,
            avg(frac_first_gold_edit) AS avg_frac_edit,
            avg(n_edits_outside_gold) AS avg_edits_outside_gold,
            avg(n_gold_files) AS avg_gold_files,
            avg(gold_file_recall) AS avg_recall,
            avg(CASE WHEN resolved IN (0, 1) THEN resolved::DOUBLE END) AS resolved_rate_known
        FROM read_parquet({path_sql})
        """
        )
        .fetchdf()
        .iloc[0]
    )
    console.print(
        f"resolved: 1={int(checks.r1):,} 0={int(checks.r0):,} -1={int(checks.r_unknown):,} "
        f"(resolved rate over known: {checks.resolved_rate_known:.3f})"
    )
    console.print(
        f"means: turns={checks.avg_turns:.1f} calls={checks.avg_calls:.1f} "
        f"first_gold_view={checks.avg_first_view:.1f} first_gold_edit={checks.avg_first_edit:.1f} "
        f"frac_view={checks.avg_frac_view:.3f} frac_edit={checks.avg_frac_edit:.3f} "
        f"edits_outside_gold={checks.avg_edits_outside_gold:.1f} "
        f"gold_files={checks.avg_gold_files:.1f} recall={checks.avg_recall:.3f}"
    )
    console.print(
        f"guards: zero_call_trajectories={int(checks.zero_call_trajectories):,} "
        f"bad_first_view={int(checks.bad_first_view):,} bad_first_edit={int(checks.bad_first_edit):,} "
        f"bad_frac_view={int(checks.bad_frac_view):,} bad_frac_edit={int(checks.bad_frac_edit):,} "
        f"bad_view_sentinel={int(checks.bad_view_sentinel):,} "
        f"bad_edit_sentinel={int(checks.bad_edit_sentinel):,} "
        f"edit_before_view={int(checks.edit_before_view):,} "
        f"empty_gold_sets={int(checks.empty_gold_sets):,} "
        f"bad_touch_count={int(checks.bad_touch_count):,} "
        f"bad_recall_sentinel={int(checks.bad_recall_sentinel):,}"
    )

    columns = con.execute(f"DESCRIBE SELECT * FROM read_parquet({path_sql})").fetchdf()
    console.print(f"columns: {len(columns)} → {', '.join(columns.column_name.tolist())}")


def temporal_features(args: argparse.Namespace) -> int:
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
                status, rows = process_file(
                    con,
                    file_path,
                    force=args.force,
                    parts_dir=PARTS_DIR,
                    sql_builder=build_temporal_sql,
                )
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
            merged = merge_parts(con, parts_dir=PARTS_DIR, out_path=OUT_PATH)
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


def main_temporal_features() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument(
        "--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: the corpus)"
    )
    parser.add_argument("--force", action="store_true", help="Recompute parts that already exist")
    parser.add_argument("--no-merge", action="store_true", help="Leave shard parts unmerged")
    parser.add_argument(
        "--merge-only", action="store_true", help="Skip processing; merge existing parts"
    )
    parser.add_argument(
        "--head", type=int, default=0, help="Print first N merged rows + sanity summary"
    )
    parser.add_argument("--threads", type=int, default=4, help="DuckDB threads for this batch job")
    parser.add_argument(
        "--memory-limit", default="6GB", help="DuckDB memory limit for this batch job"
    )
    args = parser.parse_args()

    if args.merge_only:
        con = connect_ephemeral()
        con.execute(f"SET memory_limit='{args.memory_limit}'")
        con.execute(f"SET threads={args.threads}")
        try:
            merged = merge_parts(con, parts_dir=PARTS_DIR, out_path=OUT_PATH)
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

    sys.exit(temporal_features(args))
