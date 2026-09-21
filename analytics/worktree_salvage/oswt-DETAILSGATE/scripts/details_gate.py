#!/usr/bin/env python3
"""CLI for openswe_traces.details_gate -- gate DETAILS.md commitments before
the verifier turns each line into a graded assertion.

  uv run python scripts/details_gate.py                      # all discovered units
  uv run python scripts/details_gate.py <path> [...]         # explicit _author dirs or roots
  uv run python scripts/details_gate.py --workers 4 --out outputs/details_gate.jsonl
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from openswe_traces.details_gate import main

if __name__ == "__main__":
    raise SystemExit(main())
