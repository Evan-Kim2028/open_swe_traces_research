# difficulty — raftcodec

Medium. Byte layouts are simple LE but there are six record types plus a
checksumming snapshot and a strict filename round-trip. The hidden traps:
entry Data borrows msg, the lterm tail is best-effort, peer decode must
count FULL ids only, snapshot name must round-trip canonical form. A
solver that writes "an encoder" without reading decode callers will miss
bounds and the no-copy contract.
