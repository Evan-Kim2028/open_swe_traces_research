# Difficulty — cgenc

predicted_flip: L2
details: 12

Missed edges: absolute offsets incl. header+terminator; cumulative fanout; octopus parent2 =
list-index|marker not position; last-extra-edge marker; GDA2 overflow row-index|marker with
u64 spill; generation<<34 packing; overflow staging slice aliasing; empty index still writing
mandatory chunks.

Hardness driver: bit-packing and indirection conventions that look similar but differ in
exactly one position each.
