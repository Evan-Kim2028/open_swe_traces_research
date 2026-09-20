# Difficulty — binio

predicted_flip: L2
details: 12

Missed edges: increment-before-shift offset formula; overflow bound placement; negative
input non-termination; data-loss-on-EOF in both delim readers; (0,nil) treated as EOF;
8000-byte window with early NUL exit; pooled buffer staleness.

Hardness driver: tiny functions where the doc comment gives the algorithm but the
edge behaviour (EOF, overflow, zero, negative) is the graded surface.
