#!/usr/bin/env python3
"""Build or refresh the local DuckDB analytics database and views.

Thin CLI over openswe_traces.data.init_db.
"""

from openswe_traces.data import duckdb_init_main as main

if __name__ == "__main__":
    main()
