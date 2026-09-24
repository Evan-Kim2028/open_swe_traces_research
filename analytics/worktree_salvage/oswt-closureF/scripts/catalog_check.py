#!/usr/bin/env python3
"""Lint the data catalog: every derived parquet, schema view/table, and state.db
table must be documented in analytics/schema/DATA_CATALOG.md (and vice versa).

See openswe_traces.catalog (catalog_check_main) for details and --help.
"""

from openswe_traces.catalog import catalog_check_main as main

if __name__ == "__main__":
    raise SystemExit(main())
