#!/usr/bin/env python3
"""Thin CLI: openswe-synth --repo <path> --rung 5 --hops 4 --sites 2 --decoys 1."""

from openswe_traces.synth.difficulty import main

if __name__ == "__main__":
    raise SystemExit(main())
