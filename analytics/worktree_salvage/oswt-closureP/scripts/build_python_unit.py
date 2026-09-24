"""Build the itsdangerous/signer unit (see synth/python_unit.py).

Generates the authored excision/gold/cheat patches from the pinned base
tree, materialises the excised environment/src, and packages signer-L0 and
signer-L2 with affordance.build_affordance_levels. Idempotent: re-running
rewrites the packaged dirs from the same authored content.
"""

from __future__ import annotations

import json
import sys

from openswe_traces.pipeline.config import load_config
from openswe_traces.synth.python_unit import build_python_unit


def main() -> int:
    cfg = load_config()
    out = build_python_unit(cfg)
    print(json.dumps(out, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
