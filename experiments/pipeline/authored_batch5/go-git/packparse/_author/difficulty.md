# Difficulty — packparse

predicted_flip: L2
details: 15

Missed edges: deferred delta resolution; DFS walk mixing REF+OFS children;
thin-pack placeholder parents; OFS-without-parent rejection; cached linear
depth cap; low-memory dual precondition + parent-chain buffer release;
empty-pack sentinel; grow-hint clamp; hash-present field preservation;
external-ref type/size rewrite; delta-only storage writes.

Hardness driver: the doc comments give the ALGORITHM but the plumbing
(parent resolution order, placeholder path, pool discipline, single-shot
guard) is the graded surface.
