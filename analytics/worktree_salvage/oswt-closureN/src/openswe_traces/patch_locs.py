"""Per-hunk location extraction: gold src hunks per instance, model hunks per trajectory.

One streaming pass over ``traces_data`` shards (projection pushdown: only
``metadata.reference_patch.patch`` / ``metadata.model_patch.patch`` + ids are read).
Each hunk keeps its file path, the ``@@ ... @@ ctx`` enclosing-function context, and
its new-side line range. Gold hunks carry the ``patch_split`` file class so callers
can restrict to src.

Outputs (``outputs/derived/``):
  gold_locs.parquet         instance_id, path, func_ctx, start, end, n_add, n_del, file_class
  model_patch_locs.parquet  trajectory_id, instance_id, path, func_ctx, start, end

Resume-safe: per-shard parts under ``outputs/derived/patch_locs_parts/``; finished
parts are skipped. Merge dedupes gold rows (identical patches repeat across rollouts).

Example:
  uv run python scripts/extract_patch_locs.py
"""

from __future__ import annotations

import argparse
import os
import sys
import time
import traceback
from datetime import UTC, datetime
from pathlib import Path

import pandas as pd
from rich.console import Console

import duckdb

from .data import DATA_ROOT, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .symptom_distance import classify_file, parse_patch

console = Console()

DERIVED_DIR = ROOT / "outputs" / "derived"
PARTS_DIR = DERIVED_DIR / "patch_locs_parts"
GOLD_OUT = DERIVED_DIR / "gold_locs.parquet"
MODEL_OUT = DERIVED_DIR / "model_patch_locs.parquet"

GOLD_COLS = ["instance_id", "path", "func_ctx", "start", "end", "n_add", "n_del", "file_class"]
MODEL_COLS = ["trajectory_id", "instance_id", "path", "func_ctx", "start", "end"]


def utcnow() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%d %H:%M:%S")


def _sql_str(p: Path) -> str:
    return "'" + str(p).replace("'", "''") + "'"


def shard_locs_sql(file_path: Path) -> str:
    return f"""
        SELECT instance_id,
               trajectory_id,
               metadata.reference_patch.patch AS gold_patch,
               metadata.model_patch.patch AS model_patch
        FROM read_parquet({_sql_str(file_path)}, union_by_name=true)
    """


def shard_to_frames(df: pd.DataFrame) -> tuple[pd.DataFrame, pd.DataFrame]:
    """Turn one shard's (instance, trajectory, gold_patch, model_patch) frame into
    exploded hunk-location frames. Gold rows are emitted once per instance per shard."""
    gold_rows: list[tuple] = []
    model_rows: list[tuple] = []
    seen_gold: set[str] = set()
    for inst, traj, gold, model in df.itertuples(index=False):
        if inst not in seen_gold:
            seen_gold.add(inst)
            for h in parse_patch(gold):
                gold_rows.append(
                    (inst, h.path, h.func_ctx, h.start, h.end, h.n_add, h.n_del,
                     classify_file(h.path))
                )
        for h in parse_patch(model):
            model_rows.append((traj, inst, h.path, h.func_ctx, h.start, h.end))
    return (
        pd.DataFrame(gold_rows, columns=GOLD_COLS),
        pd.DataFrame(model_rows, columns=MODEL_COLS),
    )


def part_paths(file_path: Path, parts_dir: Path = PARTS_DIR) -> tuple[Path, Path]:
    # resolve() both sides: traces_data may be a symlink into a shared corpus dir
    rel = parse_shard(file_path, root=DATA_ROOT.resolve()).rel
    base = "__".join(rel.with_suffix("").parts) + ".parquet"
    return parts_dir / "gold" / base, parts_dir / "model" / base


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool = False,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int, int]:
    """Extract hunk locs for one shard. Returns (status, gold_rows, model_rows)."""
    gold_part, model_part = part_paths(file_path, parts_dir)
    if (
        gold_part.exists()
        and model_part.exists()
        and gold_part.stat().st_size > 0
        and not force
    ):
        return "skip", 0, 0

    df = con.execute(shard_locs_sql(file_path)).fetchdf()
    gold_df, model_df = shard_to_frames(df)

    for part, frame in ((gold_part, gold_df), (model_part, model_df)):
        tmp = Path(f"{part}.{os.getpid()}.tmp")
        tmp.parent.mkdir(parents=True, exist_ok=True)
        try:
            frame.to_parquet(tmp, index=False)
        except Exception:
            tmp.unlink(missing_ok=True)
            raise
        os.replace(tmp, part)
    return "done", len(gold_df), len(model_df)


def merge_parts(parts_dir: Path = PARTS_DIR) -> tuple[int, int]:
    """Merge per-shard parts into gold_locs.parquet + model_patch_locs.parquet."""
    DERIVED_DIR.mkdir(parents=True, exist_ok=True)
    con = connect_ephemeral()
    gold_glob = _sql_str(parts_dir / "gold" / "*.parquet")
    model_glob = _sql_str(parts_dir / "model" / "*.parquet")
    con.execute(
        f"""
        COPY (
            SELECT DISTINCT * FROM read_parquet({gold_glob}, union_by_name=true)
        ) TO {_sql_str(GOLD_OUT)} (FORMAT PARQUET)
        """
    )
    con.execute(
        f"""
        COPY (
            SELECT DISTINCT * FROM read_parquet({model_glob}, union_by_name=true)
        ) TO {_sql_str(MODEL_OUT)} (FORMAT PARQUET)
        """
    )
    g = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(GOLD_OUT)})").fetchone()[0]
    m = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(MODEL_OUT)})").fetchone()[0]
    return g, m


def extract_patch_locs(args: argparse.Namespace) -> int:
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    if args.file:
        files = [Path(f).resolve() for f in args.file]
    else:
        files = list_parquet_files(args.data_glob)
    if args.limit:
        files = files[: args.limit]
    if not files:
        raise SystemExit(f"No parquet files matched {args.data_glob}")

    con = connect_ephemeral()
    console.print(f"[{utcnow()}] shards to process: {len(files):,}")

    done = skipped = failed = 0
    started = time.monotonic()
    for i, file_path in enumerate(files, start=1):
        try:
            label = parse_shard(file_path).label
        except ValueError:
            label = file_path.name
        t0 = time.monotonic()
        try:
            status, ng, nm = process_shard(con, file_path, force=args.force)
        except Exception as exc:  # noqa: BLE001 — keep going; shard stays unprocessed
            failed += 1
            console.print(f"[{utcnow()}] [{i}/{len(files)}] FAILED {label}: {exc}")
            console.print(traceback.format_exc(limit=2))
            continue
        if status == "skip":
            skipped += 1
            continue
        done += 1
        el = time.monotonic() - t0
        rate = (time.monotonic() - started) / max(done, 1)
        eta = rate * (len(files) - i)
        console.print(
            f"[{utcnow()}] [{i}/{len(files)}] done {label} "
            f"gold={ng:,} model={nm:,} in {el:.1f}s (eta {eta / 60:.0f}m)"
        )

    console.print(
        f"[{utcnow()}] extraction finished: done={done} skipped={skipped} failed={failed}"
    )
    if args.merge or not args.no_merge:
        g, m = merge_parts()
        console.print(f"[{utcnow()}] merged: gold_locs={g:,} rows, model_patch_locs={m:,} rows")
    return 0 if failed == 0 else 1


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--data-glob", default=str(DATA_ROOT / "*" / "*" / "*" / "*.parquet"))
    ap.add_argument("--file", action="append", help="process only this shard (repeatable)")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--force", action="store_true")
    ap.add_argument("--no-merge", action="store_true")
    ap.add_argument("--merge", action="store_true")
    return extract_patch_locs(ap.parse_args())


if __name__ == "__main__":
    sys.exit(main())
