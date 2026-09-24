#!/usr/bin/env python3
"""Thin CLI over openswe_traces.pipeline.author_build.

usage: author_unit_build.py excise|cheat <spec.json> <src> <out_dir> <work_dir>
"""
import sys
from openswe_traces.pipeline.author_build import main

if __name__ == "__main__":
    sys.exit(main([sys.argv[0], *sys.argv[1:]]))
