"""Build ``duckdb/analysis.duckdb`` — the one persistent analysis database.

Today's analysis jobs (closure B/C/G/H) each re-streamed all 212 corpus shards
(42 GB, ~13 min) to rebuild nearly the same per-trajectory and per-instance
frames, then did group-bys and fixed-effect design matrices in pandas. This
module kills that: it loads the *derived* parquets (already produced once by the
sibling closure jobs, copied into ``outputs/derived/`` by ``sync_derived``) plus
the external dumps (``traces_external/``) into one DuckDB file with PRIMARY KEYs
on the documented grains. Analysis queries then read tables, never the corpus.

Layout (all column names keep their catalog names, see
``analytics/schema/DATA_CATALOG.md``):

- ``instance``  — wide per-instance table: closure proxies (closureB) joined with
  rung features (closureC) and gold-patch file-class split (closureG); PK
  ``instance_id``.
- ``trajectory`` — wide per-trajectory table: trajectory frame (closureC) joined
  with the per-trajectory behaviour features (``extract_trajectory_features``,
  this worktree) and pair-eligibility markers (closureH); PK ``trajectory_id``.
- ``paired_features`` / ``eligible_pairs`` — closureH's pair-eligible subset
  (PK ``trajectory_id``); ``llm_labels`` (PK ``trajectory_id``).
- ``closure_metrics`` — authored-unit closure metrics (closureA); PK ``(repo, unit)``.
- ``harbor_hub_*`` — Harbor Hub harvest (closureE), copied into
  ``outputs/derived/harbor_hub/``; PKs on ``job_id`` / ``row_id`` / ``(row_id,
  trial_id)`` / ``(bench_version, task_id)`` / ``trial_id``.
- ``tb2_traj_yoonholee`` / ``tb2_traj_harithoppil`` — HF Terminal-Bench 2 dumps,
  read in place from ``traces_external/`` (no natural key; a trial can repeat).
- ``scale_swe_meta`` / ``rebench_v2_meta`` — upstream HF task metadata (closureG),
  read in place from ``traces_external/``; PK ``instance_id``.
- ``_build_meta`` — one row per build (timestamp, elapsed, size).

Idempotent: every table is DROPped and rebuilt from its parquet sources; rerunning
converges to the same state. Sources are the files in ``outputs/derived/`` (copied
from sibling worktrees by ``scripts/sync_derived.py``) plus the external dumps.

Usage:
  uv run python scripts/sync_derived.py       # copy derived parquets once
  uv run python scripts/build_analysis_db.py  # rebuild duckdb/analysis.duckdb
  uv run python scripts/duckdb_query.py -f analytics/queries/102_...sql --db analysis
"""

from __future__ import annotations

import argparse
import time
from datetime import UTC, datetime
from pathlib import Path

from rich.console import Console

import duckdb

from .data import ANALYSIS_DB_PATH, ROOT

console = Console()

DERIVED_DIR = ROOT / "outputs" / "derived"
TRACES_EXTERNAL = ROOT / "traces_external"


def _q(path: Path | str) -> str:
    """Single-quote a path for inline SQL (DuckDB single-quote escaping)."""
    return str(path).replace("'", "''")


def _table_sql(con: duckdb.DuckDBPyConnection, select_sql: str) -> list[tuple[str, str]]:
    return [(r[0], r[1]) for r in con.execute(f"DESCRIBE {select_sql}").fetchall()]


def _create_from(
    con: duckdb.DuckDBPyConnection,
    name: str,
    select_sql: str,
    pk_columns: tuple[str, ...],
) -> int:
    """DROP + CREATE (with PRIMARY KEY) + INSERT FROM select; returns row count."""
    cols = _table_sql(con, select_sql)
    col_defs = ", ".join(f'"{c}" {t}' for c, t in cols)
    pk_clause = ""
    if pk_columns:
        pk = ", ".join(f'"{c}"' for c in pk_columns)
        pk_clause = f", PRIMARY KEY ({pk})"
    con.execute(f'DROP TABLE IF EXISTS "{name}"')
    con.execute(f'CREATE TABLE "{name}" ({col_defs}{pk_clause})')
    col_list = ", ".join(f'"{c}"' for c, _ in cols)
    con.execute(f'INSERT INTO "{name}" ({col_list}) {select_sql}')
    n = con.execute(f'SELECT count(*) FROM "{name}"').fetchone()[0]
    console.print(f"[green]built[/green] {name}: {n:,} rows (PK {pk_columns})")
    return int(n)


def _sources() -> dict[str, tuple[str, str, tuple[str, ...]]]:
    """{table: (source SQL select, provenance label, primary key columns)}."""
    d = DERIVED_DIR
    x = TRACES_EXTERNAL

    instance_sql = f"""
    SELECT
        p.instance_id, p.repo, p.language, p.n_rollouts, p.n_labeled, p.n_resolved,
        p.gold_patch_lines, p.gold_patch_files, p.has_gold_patch, p.n_hunks, p.n_files,
        p.added_lines, p.removed_lines, p.n_new_defs, p.internal_refs, p.boundary_refs,
        p.ratio, p.new_frac, p.edit_frac,
        r.rung, r.task_text, r.leakage_symbols, r.content_len, r.n_rows, r.word_count,
        r.has_repro, r.has_expected_actual, r.has_stack_trace, r.has_test_names,
        r.has_signature, r.has_leakage, r.leakage_count, r.has_test_code,
        r.n_test_funcs_in_text, r.n_test_files_in_text, r.patch_has_tests,
        s.hf_dataset_name, s.src_added, s.src_removed, s.test_added, s.test_removed,
        s.doc_added, s.doc_removed, s.config_added, s.config_removed,
        s.n_src_files, s.n_test_files, s.n_doc_files, s.n_config_files,
        s.n_new_test_funcs, s.test_only
    FROM read_parquet('{_q(d / "closure_proxies.parquet")}') p
    JOIN read_parquet('{_q(d / "rung_features.parquet")}') r USING (instance_id)
    LEFT JOIN read_parquet('{_q(d / "patch_split.parquet")}') s USING (instance_id)
    """

    trajectory_sql = f"""
    SELECT
        f.trajectory_id, f.instance_id, f.repo, f.language, f.harness, f.teacher,
        f.source, f.resolved, f.patch_file_jaccard,
        x.category, x.n_turns, x.n_assistant_turns, x.assistant_chars, x.n_tool_errors,
        x.n_tool_calls, x.n_edit_calls, x.n_test_runs, x.n_repro_calls,
        x.turn_of_first_edit, x.turn_of_last_edit, x.first_repro_call, x.last_test_call,
        x.n_files_edited, x.edited_paths, x.n_files_read, x.n_submit_calls,
        x.ends_with_submit, x.n_model_hunks, x.n_gold_hunks, x.patch_hunk_coverage,
        x.extra_hunks, x.extra_hunk_lines, x.gold_changed_lines, x.model_changed_lines,
        x.model_lines_over_gold, x.edited_test_file, x.touched_test_path,
        x.ran_repro_before_edit, x.ran_tests_after_last_edit, x.has_submit_call,
        x.stop_reason, x.submitted_empty_patch,
        e.combo, e.stratum, e.group_id
    FROM read_parquet('{_q(d / "trajectory_frame.parquet")}') f
    JOIN read_parquet('{_q(d / "trajectory_features.parquet")}') x USING (trajectory_id)
    LEFT JOIN read_parquet('{_q(d / "eligible_pairs.parquet")}') e USING (trajectory_id)
    """

    return {
        "instance": (
            instance_sql,
            "closure_proxies + rung_features + patch_split",
            ("instance_id",),
        ),
        "trajectory": (
            trajectory_sql,
            "trajectory_frame + trajectory_features + eligible_pairs",
            ("trajectory_id",),
        ),
        "paired_features": (
            f"SELECT * FROM read_parquet('{_q(d / 'paired_features.parquet')}')",
            "outputs/derived/paired_features.parquet (oswt-closureH)",
            ("trajectory_id",),
        ),
        "eligible_pairs": (
            f"SELECT * FROM read_parquet('{_q(d / 'eligible_pairs.parquet')}')",
            "outputs/derived/eligible_pairs.parquet (oswt-closureH)",
            ("trajectory_id",),
        ),
        "llm_labels": (
            f"SELECT * FROM read_json_auto('{_q(d / 'llm_labels.jsonl')}')",
            "outputs/derived/llm_labels.jsonl (oswt-closureH)",
            ("trajectory_id",),
        ),
        "closure_metrics": (
            f"SELECT * FROM read_parquet('{_q(d / 'closure_metrics.parquet')}')",
            "outputs/derived/closure_metrics.parquet (oswt-closureA)",
            ("repo", "unit"),
        ),
        "harbor_hub_jobs": (
            f"SELECT * FROM read_parquet('{_q(d / 'harbor_hub' / 'jobs.parquet')}')",
            "outputs/derived/harbor_hub/jobs.parquet (oswt-closureE)",
            ("job_id",),
        ),
        "harbor_hub_leaderboard_rows": (
            f"SELECT * FROM read_parquet('{_q(d / 'harbor_hub' / 'leaderboard_rows.parquet')}')",
            "outputs/derived/harbor_hub/leaderboard_rows.parquet (oswt-closureE)",
            ("row_id",),
        ),
        "harbor_hub_row_trials": (
            f"SELECT * FROM read_parquet('{_q(d / 'harbor_hub' / 'row_trials.parquet')}')",
            "outputs/derived/harbor_hub/row_trials.parquet (oswt-closureE)",
            ("row_id", "trial_id"),
        ),
        # task_id repeats across bench versions (229 rows / 162 distinct ids:
        # e.g. coq-block-bound runs on both 3.0 and 4.0), so the PK is the
        # (bench_version, task_id) pair even though the catalog grain is "per task".
        "harbor_hub_tasks": (
            f"SELECT * FROM read_parquet('{_q(d / 'harbor_hub' / 'tasks.parquet')}')",
            "outputs/derived/harbor_hub/tasks.parquet (oswt-closureE)",
            ("bench_version", "task_id"),
        ),
        "harbor_hub_trials": (
            f"SELECT * FROM read_parquet('{_q(d / 'harbor_hub' / 'trials.parquet')}')",
            "outputs/derived/harbor_hub/trials.parquet (oswt-closureE)",
            ("trial_id",),
        ),
        # TB2 HF dumps are read in place from traces_external/ (large; not copied).
        # trial_id is NOT unique in yoonholee (retries: 52,104 rows / 22,599 ids)
        # and harithoppil has no id column, so neither table gets a PRIMARY KEY.
        "tb2_traj_yoonholee": (
            f"SELECT * FROM read_parquet('{_q(x / 'yoonholee__terminalbench-trajectories' / 'data' / 'train-*.parquet')}', union_by_name=true)",
            "traces_external/yoonholee__terminalbench-trajectories (HF dump)",
            (),
        ),
        "tb2_traj_harithoppil": (
            f"SELECT * FROM read_json_auto('{_q(x / 'harithoppil__terminal-bench-2-trajectories' / 'data' / 'leaderboard_trajectories.jsonl')}')",
            "traces_external/harithoppil__terminal-bench-2-trajectories, 'all' config (HF dump)",
            (),
        ),
        "scale_swe_meta": (
            f"SELECT * FROM read_parquet('{_q(x / 'AweAI-Team__Scale-SWE' / 'meta.parquet')}')",
            "traces_external/AweAI-Team__Scale-SWE/meta.parquet (oswt-closureG)",
            ("instance_id",),
        ),
        "rebench_v2_meta": (
            f"SELECT * FROM read_parquet('{_q(x / 'nebius__SWE-rebench-V2' / 'meta.parquet')}')",
            "traces_external/nebius__SWE-rebench-V2/meta.parquet (oswt-closureG)",
            ("instance_id",),
        ),
    }


def _missing_sources(sources: dict[str, tuple[str, str, tuple[str, ...]]]) -> list[str]:
    """Check every referenced file exists before touching the DB."""
    missing: list[str] = []
    for table, (_, provenance, _) in sources.items():
        # provenance is human-readable; re-derive the paths we actually probe.
        pass
    for path in [
        DERIVED_DIR / "closure_proxies.parquet",
        DERIVED_DIR / "rung_features.parquet",
        DERIVED_DIR / "patch_split.parquet",
        DERIVED_DIR / "trajectory_frame.parquet",
        DERIVED_DIR / "trajectory_features.parquet",
        DERIVED_DIR / "eligible_pairs.parquet",
        DERIVED_DIR / "paired_features.parquet",
        DERIVED_DIR / "llm_labels.jsonl",
        DERIVED_DIR / "closure_metrics.parquet",
        TRACES_EXTERNAL / "yoonholee__terminalbench-trajectories" / "data",
        TRACES_EXTERNAL / "harithoppil__terminal-bench-2-trajectories" / "data",
        TRACES_EXTERNAL / "AweAI-Team__Scale-SWE" / "meta.parquet",
        TRACES_EXTERNAL / "nebius__SWE-rebench-V2" / "meta.parquet",
    ]:
        if not path.exists():
            missing.append(str(path))
    for rel in ("jobs", "leaderboard_rows", "row_trials", "tasks", "trials"):
        if not (DERIVED_DIR / "harbor_hub" / f"{rel}.parquet").exists():
            missing.append(str(DERIVED_DIR / "harbor_hub" / f"{rel}.parquet"))
    return missing


def build_analysis_db(
    db_path: Path = ANALYSIS_DB_PATH,
    *,
    derived_dir: Path | None = None,
    traces_external: Path | None = None,
) -> dict[str, int]:
    """Rebuild the analysis database; returns {table: row count}.

    ``derived_dir``/``traces_external`` override the repo defaults (used by tests
    to point at a tiny fixture tree). Idempotent: rerunning drops and rebuilds
    every table from the same sources.
    """
    global DERIVED_DIR, TRACES_EXTERNAL
    if derived_dir is not None:
        DERIVED_DIR = Path(derived_dir)
    if traces_external is not None:
        TRACES_EXTERNAL = Path(traces_external)

    sources = _sources()
    missing = _missing_sources(sources)
    if missing:
        raise SystemExit(
            "missing source files (run `uv run python scripts/sync_derived.py` "
            "and `uv run python scripts/extract_trajectory_features.py` first):\n  "
            + "\n  ".join(missing)
        )

    db_path.parent.mkdir(parents=True, exist_ok=True)
    con = duckdb.connect(str(db_path))
    try:
        con.execute("SET memory_limit='4GB'")
        con.execute("SET threads=4")
        started = time.monotonic()
        counts: dict[str, int] = {}
        for table in sorted(sources):
            select_sql, _, pk = sources[table]
            counts[table] = _create_from(con, table, select_sql, pk)
        elapsed = time.monotonic() - started

        con.execute(
            """
            DROP TABLE IF EXISTS "_build_meta";
            CREATE TABLE "_build_meta" (
                built_at TIMESTAMP, elapsed_seconds DOUBLE, n_tables INTEGER
            );
            """
        )
        con.execute(
            'INSERT INTO "_build_meta" VALUES (?, ?, ?)',
            [datetime.now(UTC), elapsed, len(counts)],
        )
        console.print(
            f"[green]analysis DB ready[/green] {db_path} "
            f"({db_path.stat().st_size / 1e6:.1f} MB, {elapsed:.1f}s, "
            f"{len(counts)} tables)"
        )
        return counts
    finally:
        con.close()


def build_analysis_db_main() -> None:
    parser = argparse.ArgumentParser(
        description="Rebuild duckdb/analysis.duckdb from outputs/derived/ and traces_external/."
    )
    parser.add_argument(
        "--db", type=Path, default=ANALYSIS_DB_PATH, help="Output database path"
    )
    args = parser.parse_args()
    counts = build_analysis_db(args.db)
    for name, n in counts.items():
        console.print(f"  {name}: {n:,}")


if __name__ == "__main__":
    build_analysis_db_main()
