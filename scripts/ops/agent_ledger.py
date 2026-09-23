#!/usr/bin/env python3
"""Moved to openswe_traces.reports.agent_ledger. This path keeps old commands and `import agent_ledger` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from openswe_traces.reports import agent_ledger as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
