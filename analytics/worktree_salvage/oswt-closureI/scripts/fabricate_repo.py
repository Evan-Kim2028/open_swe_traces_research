#!/usr/bin/env python3
"""Fabricate a self-contained Go repo unit with a target closure ratio.

Thin CLI over ``openswe_traces.synth.fabricate.main_cli``.  Writes the unit
(module tree, excised tree, hidden suite, ``_author/`` artifacts) under
``--out`` and proves it locally (A1 gold passes, A3 cheat fails, module green),
then appends one JSONL row to ``--log``.

Example:

    uv run python scripts/fabricate_repo.py --seed 7 --n-funcs 24 --ratio 0.8 \\
        --out experiments/fabricated/store-s7-r080
"""

from __future__ import annotations

import sys

from openswe_traces.synth.fabricate import main_cli

if __name__ == "__main__":
    sys.exit(main_cli())
