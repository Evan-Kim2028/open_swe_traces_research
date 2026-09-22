# Difficulty — mergebase

predicted_flip: L2
details: 10

Missed edges: reachable-older early return; index-as-both-filters trick;
round-based candidate eviction; dedupe-before-walk; seen-set descent
limiter; hash-vs-pointer equality; error propagation from walkers.

Hardness driver: the two-phase design (index one side, filter-walk the
other) is an upstream algorithm — a naive both-sides-compare passes easy
cases and fails criss-cross fixtures.
