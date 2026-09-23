#!/usr/bin/env python3
"""B10: find hidden tests that assert a literal digest of an internal serialisation.

A sha256 of a struct's serialised form is not a black-box property -- it pins the exact
byte layout (field order, omitempty, key names). A solver cannot invert it, so such a test
is unsolvable from the contract AND unsolvable at L5, where the test is visible: seeing the
expected digest tells you nothing about how to produce it. helm-depresolver fails 0/3 at
every rung for exactly this reason.

Assert the property (equal inputs hash equally, different inputs differ) or compare against
a digest the test computes itself from a value it also constructs.
"""
import re
import sys
from collections import defaultdict
from pathlib import Path

# Only an algorithm-prefixed literal is unambiguously a digest oracle; a bare hex string
# is usually a fixture (an address, a key, a test vector).
HEXLIT_RE = re.compile(r"[\"'`](sha(?:256|512|1):[0-9a-f]{32,128})[\"'`]")
COMPUTED_RE = re.compile(r"sha256\.(?:Sum256|New)|hex\.EncodeToString|fmt\.Sprintf\(\"%x")


def main() -> int:
    hits = defaultdict(list)
    for unit in sorted(Path("experiments/dose_response").glob("sweep_*/*/tests/hidden")):
        name = unit.parent.parent.name
        for f in unit.rglob("*_test.go"):
            src = f.read_text(errors="replace")
            for m in HEXLIT_RE.finditer(src):
                line = src[: m.start()].count("\n") + 1
                ctx = src.splitlines()[line - 1].strip()[:110]
                # a literal next to a computed digest is a fixture, not an oracle
                window = src[max(0, m.start() - 600) : m.start() + 600]
                computed = bool(COMPUTED_RE.search(window))
                hits[name.rsplit("-L", 1)[0]].append((f.name, line, m.group(1)[:24], computed, ctx))
    hard = {k: v for k, v in hits.items() if any(not c for _, _, _, c, _ in v)}
    print(f"units with a literal digest in the hidden suite: {len(hits)}")
    print(f"units where at least one is NOT next to a computed digest (B10 risk): {len(hard)}\n")
    for name, rows in sorted(hard.items()):
        print(f"--- {name}")
        for fn, line, dig, computed, ctx in rows[:4]:
            if not computed:
                print(f"    {fn}:{line}  {dig}…  {ctx}")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()
