# Difficulty — tagparse

predicted_flip: L2
details: 12

Missed edges: strict order for first three but silent drop for out-of-position duplicates;
last-match signature peel; one-space continuation strip + concatenating repeats; verbatim
signature append; zero-signature triple condition; signature fields excluded from the
source-match; pushback depth of one.

Hardness driver: a state machine whose doc comments describe upstream behaviour but whose
exact tolerance/drop choices must be reconstructed faithfully.
