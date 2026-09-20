# Difficulty — fieldmap

predicted_flip: L3
details: 6
Commitments: two translation directions over one table, unmapped passthrough, exact-match semantics, and the naming trap — `HumanPath` yields the OLD api spelling, `InternalPath` the NEW one.

Hardness driver: the unit is small but the direction is genuinely counterintuitive ("human" = legacy spelling); a solver that wires the obvious direction writes a mirror-image implementation that passes the trivial cases and fails everything mapped.
