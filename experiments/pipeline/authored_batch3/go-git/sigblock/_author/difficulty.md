# Difficulty — sigblock

predicted_flip: L2
details: 10

Missed edges: last-match split position; line-boundary-only recognition; PGP MESSAGE
counting; chained drop of consecutive signature headers; body latch preventing
body-line stripping; tag-only trailing truncation; trailing-space requirement on the
header spellings.

Hardness driver: small pure predicates where each boundary condition is a deliberate
upstream-mirroring choice.
