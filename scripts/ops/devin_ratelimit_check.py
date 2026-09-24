#!/usr/bin/env python3
"""Moved to openswe_traces.ops.devin_ratelimit_check. This path keeps old commands and `import devin_ratelimit_check` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from openswe_traces.ops import devin_ratelimit_check as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
