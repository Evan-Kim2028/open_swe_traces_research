#!/usr/bin/env python3
"""Build equal-budget SFT manifests for the curve experiment.

Thin CLI over openswe_traces.sft.manifests; see that module for the arm definitions.
"""

from openswe_traces.sft.manifests import main

if __name__ == "__main__":
    main()
