#!/usr/bin/env python3
"""Zero-GPU per-trajectory structural proxy features for ranking traces before SFT.

Thin CLI over openswe_traces.features; see `openswe-features --help`.
"""

from openswe_traces.features import main_proxy_features as main

if __name__ == "__main__":
    main()
