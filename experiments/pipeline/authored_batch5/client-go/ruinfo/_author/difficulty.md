# Difficulty — ruinfo

predicted_flip: L2
details: 5

Missed edges: the -1 sentinel vs a legitimately-zero write;
ScanDetailV2 overriding response size; the three-source KVCPU precedence
with its ms→ns scaling; `internal_others` substring (not prefix) match.

Hardness driver: type-switch bookkeeping where the defaults are
counter-intuitive — a plausible impl gets IsWrite/WriteBytes right for
nonempty writes and still fails the sentinel edge.
