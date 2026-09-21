"""Pre-registered analysis for the detail-count dial (thin CLI).

Runs the per-detail logistic (cluster-robust by unit), the geometric
pass-rate test, and the within-unit dispersion check over trials.parquet.
With no trials yet, prints the calibration simulation that proves the
estimator recovers p^d.

    uv run python scripts/detail_dial_analyze.py --simulate
    uv run python scripts/detail_dial_analyze.py
"""

from __future__ import annotations

import sys

from openswe_traces.synth.detail_dial_analysis import main_cli

if __name__ == "__main__":
    sys.exit(main_cli())
