#!/usr/bin/env python3
"""Compare certified-hard L0 failure anatomy against Open-SWE-Traces.

Thin CLI over openswe_traces.failure_anatomy.
"""

from openswe_traces.failure_anatomy import main_failure_anatomy as main

if __name__ == "__main__":
    main()
