#!/usr/bin/env python3
"""Idempotent download of nvidia/Open-SWE-Traces into traces_data/.

Safe to rerun: huggingface_hub snapshot_download skips files already on disk.
Progress is written to traces_data/.download_status.json after each run.
"""

from __future__ import annotations

import argparse
import json
from datetime import UTC, datetime
from pathlib import Path

from huggingface_hub import snapshot_download
from rich.console import Console
from rich.table import Table

console = Console()

REPO_ID = "nvidia/Open-SWE-Traces"
DEFAULT_DIR = Path(__file__).resolve().parent.parent / "traces_data"
STATUS_FILE = ".download_status.json"
EXPECTED_FILES = 215
EXPECTED_ROWS = 511_668
EXPECTED_BYTES = 42_600_000_000


def _parquet_paths(data_dir: Path) -> list[Path]:
    return sorted(data_dir.rglob("*.parquet"))


def read_status(data_dir: Path) -> dict:
    path = data_dir / STATUS_FILE
    if path.exists():
        return json.loads(path.read_text())
    return {}


def write_status(data_dir: Path, *, complete: bool, error: str | None = None) -> dict:
    parquet_files = _parquet_paths(data_dir)
    total_bytes = sum(p.stat().st_size for p in parquet_files)
    status = {
        "repo_id": REPO_ID,
        "updated_at": datetime.now(UTC).isoformat(),
        "complete": complete,
        "parquet_files": len(parquet_files),
        "expected_parquet_files_approx": EXPECTED_FILES,
        "bytes_downloaded": total_bytes,
        "expected_bytes_approx": EXPECTED_BYTES,
        "error": error,
    }
    data_dir.mkdir(parents=True, exist_ok=True)
    (data_dir / STATUS_FILE).write_text(json.dumps(status, indent=2) + "\n")
    return status


def print_status(data_dir: Path) -> dict:
    status = write_status(data_dir, complete=read_status(data_dir).get("complete", False))
    table = Table(title="Open-SWE-Traces download status")
    table.add_column("Metric")
    table.add_column("Value", justify="right")
    rows = [
        ("Parquet files", f"{status['parquet_files']} / ~{EXPECTED_FILES}"),
        (
            "Size",
            f"{status['bytes_downloaded'] / 1e9:.2f} GB / ~{EXPECTED_BYTES / 1e9:.1f} GB",
        ),
        ("Complete flag", str(status["complete"])),
        ("Updated", status["updated_at"]),
    ]
    for metric, value in rows:
        table.add_row(metric, value)
    console.print(table)
    if status["parquet_files"] >= EXPECTED_FILES - 5 and status["bytes_downloaded"] > EXPECTED_BYTES * 0.95:
        console.print("[green]Download looks complete.[/green] Run: uv run python scripts/verify_data.py")
    elif status["parquet_files"] > 0:
        console.print("[yellow]Partial download — rerun the same command to resume.[/yellow]")
    else:
        console.print("[dim]No parquet files yet.[/dim]")
    return status


def download(data_dir: Path = DEFAULT_DIR) -> Path:
    data_dir.mkdir(parents=True, exist_ok=True)
    before = write_status(data_dir, complete=False)
    console.print(f"[bold]Downloading[/bold] {REPO_ID}")
    console.print(f"Destination: {data_dir}")
    console.print(
        f"Resume: {before['parquet_files']} parquet files "
        f"({before['bytes_downloaded'] / 1e9:.2f} GB) already on disk\n"
    )

    try:
        path = snapshot_download(
            repo_id=REPO_ID,
            repo_type="dataset",
            local_dir=str(data_dir),
        )
    except Exception as exc:
        write_status(data_dir, complete=False, error=str(exc))
        raise

    after = write_status(data_dir, complete=True)
    console.print(
        f"\n[green]Done.[/green] {after['parquet_files']} parquet files, "
        f"{after['bytes_downloaded'] / 1e9:.2f} GB"
    )
    console.print(f"Files at: {path}")
    return Path(path)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=DEFAULT_DIR,
        help=f"Local directory (default: {DEFAULT_DIR})",
    )
    parser.add_argument(
        "--status",
        action="store_true",
        help="Print download progress and exit (no download)",
    )
    args = parser.parse_args()
    if args.status:
        print_status(args.output)
        return
    download(args.output)


if __name__ == "__main__":
    main()
