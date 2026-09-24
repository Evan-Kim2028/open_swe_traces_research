#!/usr/bin/env python3
"""Harvest per-trial Terminal-Bench results from Harbor Hub into parquet.

Thin CLI over openswe_traces.external.harbor_hub.
"""

from openswe_traces.external.harbor_hub import main

if __name__ == "__main__":
    main()
