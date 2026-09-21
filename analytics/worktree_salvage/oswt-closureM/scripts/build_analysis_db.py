#!/usr/bin/env python3
"""CLI: build duckdb/analysis.duckdb, the one persistent analysis database.

See openswe_traces.analysis_db (build_analysis_db_main) for details and --help.
"""

from openswe_traces.analysis_db import build_analysis_db_main as main

if __name__ == "__main__":
    main()
