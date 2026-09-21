#!/usr/bin/env python3
"""Detail-dial Harbor runner (resume-safe, 429-aware).

Thin CLI over ``openswe_traces.synth.detail_dial.main_cli``.  Launches
``harbor run`` for the packaged L0 task dirs, records every trial into
``experiments/detail_dial/trials.parquet`` (including the per-detail vector
from each trial's ``verifier/details.json``), and skips already-recorded
(task, model) attempts.

Example (one validation trial):

    uv run python scripts/detail_dial_run.py \
        --task-dir experiments/detail_dial/tasks/store-s11-d1-vlow-L0 \
        --model openrouter/deepseek-v4-flash-0731:free --attempts 1 --limit 1

Planned launch (from analytics/research/detail_count_dial.md):

    uv run python scripts/detail_dial_run.py \
        --tasks-glob 'experiments/detail_dial/tasks/*-L0' \
        --agent cursor-cli --model cursor/composer-2.5 --attempts 1
"""

from __future__ import annotations

import sys

from openswe_traces.synth.detail_dial import main_cli

if __name__ == "__main__":
    sys.exit(main_cli())
