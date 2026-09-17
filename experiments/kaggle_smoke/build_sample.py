#!/usr/bin/env python3
"""Sample N traces from one harness/teacher into a compact JSONL for Kaggle SFT.

Thin CLI over openswe_traces.sft.sample; see `openswe-sample --help`.
"""

from openswe_traces.sft.sample import main

if __name__ == "__main__":
    main()
