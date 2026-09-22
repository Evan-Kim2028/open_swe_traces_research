# Difficulty — keyflags

predicted_flip: L2
details: 9

Missed edges: the two-bit assertion truth table (exclusive vs both vs none);
`PresumeKNE` reading the previous-KNE bit too; `SetKeyLockedValueExists`
clearing the constraint bit; persistence mask membership; unknown ops as
no-ops.

Hardness driver: the switch is a tangle of coupled bit writes where three
ops touch two flags each and the assert pair is a two-bit quad-state, not
two independent booleans.
