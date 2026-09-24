#!/usr/bin/env python3
"""Dose-response Harbor runner (resume-safe, 429-aware).

Thin CLI over ``openswe_traces.synth.dose_response.main_cli``.  Launches
``harbor run`` for a list of task dirs, records every trial into
``experiments/dose_response/trials.parquet``, skips already-recorded
(task, model) attempts, and retries jobs that hit OpenRouter rate limits.

Example (one validation trial):

    uv run python scripts/dose_response_run.py \
        --task-dir experiments/dose_response/tasks/store-s11-r040-L2 \
        --model openrouter/deepseek-v4-flash-0731:free --attempts 1 --limit 1

Composer anchor (do not run from this job):

    uv run python scripts/dose_response_run.py \
        --tasks-glob 'experiments/dose_response/tasks/*-L2' \
        --agent cursor-cli --model cursor/composer-2.5 --attempts 1
"""

from __future__ import annotations

import sys

from openswe_traces.synth.dose_response import main_cli

if __name__ == "__main__":
    sys.exit(main_cli())
