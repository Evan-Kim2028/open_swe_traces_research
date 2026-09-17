#!/usr/bin/env python3
"""Quick sanity check after download (DuckDB streaming, 4GB cap).

Thin CLI over openswe_traces.verify.
"""

from openswe_traces.verify import main

if __name__ == "__main__":
    main()
