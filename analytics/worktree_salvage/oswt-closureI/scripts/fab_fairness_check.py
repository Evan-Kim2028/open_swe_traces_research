"""Self-audit fabricated units for L0 fairness (post-Grok-audit checklist).

Greps each unit's hidden suite for literals absent from the L0 information set
(excised tree + bugreport + repro output, the last subsumed by the smoke-suite
source), checks the contract coverage table maps every hidden test, and runs
the B4/B6/B7 mechanical checks.  Fails (exit 1) if any unit fails any check.

Example:

    uv run python scripts/fab_fairness_check.py experiments/dose_response/fab/store-s101-r015
"""

from __future__ import annotations

import sys

from openswe_traces.synth.fab_fairness import main_cli

if __name__ == "__main__":
    sys.exit(main_cli())
