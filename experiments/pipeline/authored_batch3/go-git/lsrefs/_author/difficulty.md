# Difficulty — lsrefs

predicted_flip: L2
details: 13

Missed edges: all-or-nothing prefix validation before writing; the 65536-prefix drop-everything
fallback; peeled folding on encode AND peeled expansion on decode; `unborn` requiring
symref-target; symref-target making the ref symbolic even with a full oid; strict 40/64 hash
parsing where the rest of the tree pads.

Hardness driver: two grammars (args + output) with asymmetric fold/expand semantics.
