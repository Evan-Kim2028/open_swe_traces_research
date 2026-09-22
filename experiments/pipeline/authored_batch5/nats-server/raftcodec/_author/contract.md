# Contract — raftcodec

NRG wire codecs: binary encode/decode for append entries, responses,
votes, peer state, and snapshots. Every commitment below is covered by
a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Append-entry layout.** Leader id, four u64 fields, a u16 entry
   count, per-entry length+type+data framing, and a trailing uvarint
   leader-term. Covered by `TestDetail01`.
2. **Encode rejection.** A leader of the wrong length, more than 65535
   entries, or oversized entry data produce the corresponding package
   errors. Covered by `TestDetail02`.
3. **Buffer reuse.** Encode reuses a caller buffer with sufficient
   capacity and returns the exact encoded length. Covered by
   `TestDetail03`.
4. **Decode validation and aliasing.** Decode requires the base length;
   each entry requires its length prefix, positive length, and in-range
   data; entry data aliases the wire buffer which is retained on the
   entry. Covered by `TestDetail04`.
5. **Leader-term tail.** A well-formed trailing uvarint decodes as the
   leader term; a truncated tail leaves it zero. Covered by
   `TestDetail05`.
6. **Response record.** The response is a fixed 25-byte record with the
   peer in a fixed field (truncated or zero-padded) and a success byte;
   decode rejects any other length. Covered by `TestDetail06`.
7. **Peer state.** Cluster size, peer count, fixed-length ids, and an
   optional domain extension; decode rejects short buffers and missing
   ids. Covered by `TestDetail07`.
8. **Vote request.** A fixed 32-byte record; decode rejects any other
   length and copies the reply subject. Covered by `TestDetail08`.
9. **Snapshot frame.** Nil encodes to nil; otherwise the frame is
   term, index, peerstate-length, peerstate, data, and a checksum over
   the preceding bytes. Covered by `TestDetail09`.
10. **Snapshot filename.** The basename parses strictly: only names
    that round-trip through the canonical format are accepted. Covered
    by `TestDetail10`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — byte layout asserted |
| TestDetail02 | 2 | yes — error identities asserted |
| TestDetail03 | 3 | partially — reuse asserted via pointer equality; exact-length allocation asserted |
| TestDetail04 | 4 | yes — guards, aliasing, and buf retention asserted |
| TestDetail05 | 5 | partially — tail decode and truncated-tail zero asserted |
| TestDetail06 | 6 | partially — width/peer/success/nil-on-badlen asserted; interning not directly asserted |
| TestDetail07 | 7 | partially — layout, ext round-trip, and corruption errors asserted |
| TestDetail08 | 8 | yes — asserted exactly |
| TestDetail09 | 9 | doc — layout and checksum asserted via the visible hash init |
| TestDetail10 | 10 | partially — strict round-trip rejections asserted |
