#!/usr/bin/env python3
"""Idempotent download of nvidia/Open-SWE-Traces into traces_data/.

Thin CLI over openswe_traces.download; see `openswe-download --help`.
"""

from openswe_traces.download import main

if __name__ == "__main__":
    main()
