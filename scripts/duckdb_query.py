#!/usr/bin/env python3
"""Run SQL against open_swe.duckdb with streaming results and query logging.

Thin CLI over openswe_traces.data.run_query.
"""

from openswe_traces.data import duckdb_query_main as main

if __name__ == "__main__":
    main()
