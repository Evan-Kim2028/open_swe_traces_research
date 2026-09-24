# Difficulty — srvresp

predicted_flip: L2
details: 10

Missed edges: NAK being the clean terminator rather than an error; a bare `ACK <hash>` ending the
response while a status-bearing ACK continues it; unknown status words degrading to a zero status
rather than failing; empty-set encode emitting NAK; single-ack encode dropping later acks.

Hardness driver: terminator semantics and status-defaulting, three plausible choices each.
