# Difficulty — commitwalk

predicted_flip: L2
details: 11

Missed edges: ErrStop vs io.EOF split; dual seen-set gating; enqueue-time
marking; isValid-vs-isLimit asymmetry (emit vs descend); error latching
for Error(); heap ordering by committer time; checkParent merge split;
tail-hash stop exclusive; ref-seeded all-iter; Close delegation.

Hardness driver: six iterators sharing a Next/ForEach contract — each
variant's distinguishing rule is one line of state management; surface
behavior (order, dedup, termination) is easy to approximate and hard to
get exactly right.
