# Difficulty — reqsource

predicted_flip: L1
details: 6

Missed edges: nil-or-both-empty collapses to `"unknown"` even when the
internal flag is set; explicit==type dedups out of the label;
`IsInternalRequest` is a bare-prefix match (`"internalx"` is internal).

Hardness driver: mostly-readable composition rules with a few guard-order
choices; the dedup and unknown-fallback are the non-obvious bits.
