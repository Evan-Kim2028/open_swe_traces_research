#!/usr/bin/env python3
"""Extract turn-level features using DuckDB streaming COPY only.

Thin CLI over openswe_traces.features.extract_turn_sample.
"""

from openswe_traces.features import main_turn_sample as main

if __name__ == "__main__":
    main()
