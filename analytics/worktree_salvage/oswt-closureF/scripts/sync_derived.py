#!/usr/bin/env python3
"""Copy derived parquets from sibling closure worktrees into outputs/derived/.

See openswe_traces.catalog (sync_derived) for details and --help.
"""

from openswe_traces.catalog import sync_derived_main as main

if __name__ == "__main__":
    main()
