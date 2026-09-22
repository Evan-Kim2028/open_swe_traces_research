# Difficulty — mvcccodec

predicted_flip: L2
details: 6

Missed edges: uvarint length framing vs fixed-width LE numbers in one
stream; the sticky-error latch on marshalHelper; the 10MiB ReadSlice cap;
`MvccKey.Raw` panics on malformed input rather than returning an error;
`op` is written as its concrete enum width (4 bytes), not uvarint.

Hardness driver: the wire format is invisible in signatures — any
self-consistent codec round-trips but only the exact layout survives
cross-checks.
