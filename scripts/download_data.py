#!/usr/bin/env python3
"""Download the full nvidia/Open-SWE-Traces dataset to traces_data/."""

from __future__ import annotations

import argparse
from pathlib import Path

from huggingface_hub import snapshot_download
from rich.console import Console

console = Console()

REPO_ID = "nvidia/Open-SWE-Traces"
DEFAULT_DIR = Path(__file__).resolve().parent.parent / "traces_data"


def download(local_dir: Path = DEFAULT_DIR) -> Path:
    local_dir.mkdir(parents=True, exist_ok=True)
    console.print(f"[bold]Downloading[/bold] {REPO_ID}")
    console.print(f"Destination: {local_dir}")
    console.print("Expected size: ~43 GB (511,668 trajectories)\n")

    path = snapshot_download(
        repo_id=REPO_ID,
        repo_type="dataset",
        local_dir=str(local_dir),
    )
    console.print(f"\n[green]Done.[/green] Files at: {path}")
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
        "--no-resume",
        action="store_true",
        help="Start fresh instead of resuming partial downloads",
    )
    args = parser.parse_args()
    download(args.output, resume=not args.no_resume)


if __name__ == "__main__":
    main()
