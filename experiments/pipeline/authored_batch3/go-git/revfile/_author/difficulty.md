# Difficulty — revfile

predicted_flip: L2
details: 12

Missed edges: channel closed on error too; empty count is a failure not an empty decode; typed-
nil writer via reflection; hash-id from size not algorithm; trailing-bytes malformed; hasher
reset; exact-length pack checksum read; checksum covers header+id+entries+pack.

Hardness driver: a state-machine codec where most correctness lives in verification ordering
and teardown semantics, not the happy path.
