# Difficulty — priorityqueue

predicted_flip: L2
details: 6

Missed edges: `Take(n >= Len)` leaks raw heap-array order instead of sorted
order (invisible for <3 remaining items); the stale doc comment inverts the
priority direction; `Take(0)`/`Take(-1)` return nil rather than an empty
slice; `clean` mutates the heap in place.

Hardness driver: a sorted-slice implementation passes every sorted-order
example but fails the bulk-take ordering — the surface rewards reading the
code, not the signature.
