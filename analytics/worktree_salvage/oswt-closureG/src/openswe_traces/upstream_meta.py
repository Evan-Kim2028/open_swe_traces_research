"""Upstream test metadata for corpus sources, downloaded into ``traces_external/``.

The corpus's ``hf_dataset_name`` column names the SWE-task dataset each instance came
from. For each public source we download only the metadata columns — FAIL_TO_PASS,
PASS_TO_PASS, test patch, problem statement, created_at — and write
``traces_external/<owner__name>/meta.parquet``. No trajectories are fetched.

- ``nebius/SWE-rebench-V2``: single parquet on the hub; downloaded (resumable) then
  column-projected.
- ``AweAI-Team/Scale-SWE``: single JSONL; downloaded (resumable) then parsed
  line-by-line. ``f2p_script`` is the fail-to-pass test script and is stored as
  ``test_patch``. The file has no ``created_at``; the column is kept null.

Example:
  uv run python scripts/upstream_meta.py
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import pandas as pd
from rich.console import Console

from .data import ROOT, connect_ephemeral
from .features import _sql_str, utcnow
from .patch_split import OUT_PARQUET as SPLIT_PARQUET

console = Console()

EXTERNAL_DIR = ROOT / "traces_external"

SOURCES: dict[str, dict] = {
    "nebius/SWE-rebench-V2": {
        "kind": "parquet",
        "file": "data/train-00000-of-00001.parquet",
    },
    "AweAI-Team/Scale-SWE": {
        "kind": "jsonl",
        "file": "processed_to_upload.jsonl",
    },
}

META_COLUMNS = [
    "instance_id",
    "repo",
    "language",
    "created_at",
    "problem_statement",
    "test_patch",
    "fail_to_pass",
    "pass_to_pass",
]


def source_dir(source: str) -> Path:
    return EXTERNAL_DIR / source.replace("/", "__")


def _parse_list_field(value) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        return [str(v) for v in value]
    try:
        parsed = json.loads(value)
    except (TypeError, json.JSONDecodeError):
        return []
    if isinstance(parsed, list):
        return [str(v) for v in parsed]
    return []


def _rebench_meta(raw_path: Path, out_path: Path) -> int:
    con = connect_ephemeral()
    try:
        con.execute(
            f"""
            COPY (
                SELECT
                    instance_id,
                    repo,
                    language,
                    created_at,
                    problem_statement,
                    test_patch,
                    FAIL_TO_PASS AS fail_to_pass,
                    PASS_TO_PASS AS pass_to_pass
                FROM read_parquet({_sql_str(raw_path)})
            ) TO {_sql_str(out_path)} (FORMAT PARQUET)
            """
        )
        return con.execute(
            f"SELECT count(*) FROM read_parquet({_sql_str(out_path)})"
        ).fetchone()[0]
    finally:
        con.close()


def _scaleswe_meta(raw_path: Path, out_path: Path) -> int:
    rows = []
    with raw_path.open() as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            r = json.loads(line)
            user = r.get("user") or ""
            repo = r.get("repo") or ""
            rows.append(
                {
                    "instance_id": r.get("instance_id"),
                    "repo": f"{user}/{repo}" if user and repo else r.get("github_url"),
                    "language": r.get("language"),
                    "created_at": None,
                    "problem_statement": r.get("problem_statement"),
                    "test_patch": r.get("f2p_script") or r.get("f2p_patch"),
                    "fail_to_pass": _parse_list_field(r.get("FAIL_TO_PASS")),
                    "pass_to_pass": _parse_list_field(r.get("PASS_TO_PASS")),
                }
            )
    df = pd.DataFrame(rows, columns=META_COLUMNS)
    df.to_parquet(out_path, index=False)
    return len(df)


def fetch_source(source: str, *, force: bool = False) -> Path:
    """Download one source's metadata columns to traces_external/<src>/meta.parquet."""
    if source not in SOURCES:
        raise SystemExit(f"no fetch recipe for {source}")
    spec = SOURCES[source]
    out_dir = source_dir(source)
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / "meta.parquet"
    if out_path.exists() and not force:
        console.print(f"[{utcnow()}] {source}: meta.parquet exists, skipping")
        return out_path

    from huggingface_hub import hf_hub_download

    console.print(f"[{utcnow()}] {source}: downloading {spec['file']}")
    raw = Path(
        hf_hub_download(
            repo_id=source,
            filename=spec["file"],
            repo_type="dataset",
            local_dir=out_dir,
        )
    )
    if spec["kind"] == "parquet":
        n = _rebench_meta(raw, out_path)
    else:
        n = _scaleswe_meta(raw, out_path)
    console.print(f"[{utcnow()}] {source}: {n:,} rows → {out_path.relative_to(ROOT)}")
    return out_path


def join_coverage(split_path: Path = SPLIT_PARQUET) -> pd.DataFrame:
    """Per source: corpus instances and the share matched by upstream metadata."""
    split = pd.read_parquet(split_path, columns=["instance_id", "hf_dataset_name"])
    inst = split.drop_duplicates("instance_id")
    rows = []
    for source, sub in inst.groupby("hf_dataset_name"):
        meta_path = source_dir(str(source)) / "meta.parquet"
        if not meta_path.exists():
            rows.append(
                {
                    "source": source,
                    "corpus_instances": len(sub),
                    "meta_rows": 0,
                    "matched": 0,
                    "coverage": 0.0,
                    "status": "no meta.parquet",
                }
            )
            continue
        meta = pd.read_parquet(meta_path, columns=["instance_id"])
        meta_ids = set(meta["instance_id"])
        matched = int(sub["instance_id"].isin(meta_ids).sum())
        rows.append(
            {
                "source": source,
                "corpus_instances": len(sub),
                "meta_rows": len(meta),
                "matched": matched,
                "coverage": matched / len(sub),
                "status": "ok",
            }
        )
    return pd.DataFrame(rows)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", nargs="*", default=None, help="limit to sources")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()

    wanted = args.source or list(SOURCES)
    for source in wanted:
        fetch_source(source, force=args.force)

    if SPLIT_PARQUET.exists():
        cov = join_coverage()
        console.print(cov.to_string(index=False))
    else:
        console.print(f"[dim]{SPLIT_PARQUET} missing — coverage report skipped[/dim]")
    sys.exit(0)


if __name__ == "__main__":
    main()
