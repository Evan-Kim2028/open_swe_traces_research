#!/usr/bin/env python3
"""Thin CLI over openswe_traces.synth.stage_units."""

from openswe_traces.synth.stage_units import main

def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
