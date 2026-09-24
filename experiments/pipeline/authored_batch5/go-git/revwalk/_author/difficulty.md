# Difficulty — revwalk

predicted_flip: L3
details: 12

Missed edges: boundary-paint termination; late-arriving havePaint
dropping a queued "new" commit; haves pre-marking trees; asymmetric
missing-object tolerance; deferred missing-parent validation; shallow
leaf boundaries; tag unwrap seeds; any-parent-unchanged tree diff;
submodule exclusion; discovery order; no-haves fast path.

Hardness driver: a naive reachable-set walk passes shallow-clone tests
and only fails on incremental-fetch object counts — the paint/flag
machinery exists purely for correctness at the boundary.
