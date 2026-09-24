# Difficulty — packenc

predicted_flip: L2
details: 14

Missed edges: object-format-aware trailer hash; selector/recovery split;
base-before-delta recursion; sentinel-based write tracking (>1 vs 1);
negative-distance OFS guard; single delta kind per pack; type-tagged varint
header layout; Original→saved-metadata→delta fallback chain with panics.

Hardness driver: the reader half of the same format stays visible, so the
layout is derivable — the graded surface is the write-side bookkeeping
(sentinels, cycle recovery, hash tee) that no sibling shows.
