# Difficulty — deadlock

predicted_flip: L2
details: 7

Missed edges: the reported `KeyHash` belongs to the STORED edge that reaches
back to the source (deepest hit in DFS order), not the new wait; a rejected
`Detect` leaves no edge; `register` dedups; `CleanUpWaitFor` deletes the map
entry on empty; `Expire` compares strictly on the source key.

Hardness driver: recursion + which hash is reported — a plausible-but-wrong
impl returns the incoming keyHash and passes the 2-cycle case anyway.
