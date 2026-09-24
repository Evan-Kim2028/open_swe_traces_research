# Difficulty — reqframe

predicted_flip: L2
details: 13

Missed edges: EOF-before-first-packet decoding as an empty request; the host field's optional
`host=` prefix on decode; control-byte rejection on BOTH directions including 0x7f; the
no-args-decoder flush requirement; empty param slots being dropped; empty request being a single
flush rather than an empty command line.

Hardness driver: two small wire formats with several asymmetric empty/missing choices each.
