# Difficulty — gattr

predicted_flip: L2

Sharing preservation and the original↔copy bidirectional map through a
recursive traversal are implementation-only; a shallow copy compiles and
passes trivial shapes but fails recursive/shared-graph tests.
