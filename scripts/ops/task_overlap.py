#!/usr/bin/env python3
"""Moved to openswe_traces.analysis.task_overlap. This path keeps old commands and `import task_overlap` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from openswe_traces.analysis import task_overlap as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
