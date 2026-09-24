#!/usr/bin/env python3
"""Moved to openswe_traces.analysis.classify_gaps. This path keeps old commands and `import classify_gaps` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from openswe_traces.analysis import classify_gaps as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
