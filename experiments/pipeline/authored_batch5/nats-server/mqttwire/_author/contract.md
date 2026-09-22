# Contract — mqttwire

MQTT wire codec primitives: the reader cursor with partial-packet
carry-over, the writer, and the fixed-header validators. Every
commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Variable-length integer.** Seven bits per byte LSB-first with a
   continuation bit; a terminated value returns complete, a buffer
   exhausted mid-value returns incomplete without error, and more than
   four continuation bytes is malformed. Covered by `TestDetail01`.
2. **Packet length.** A complete packet returns its declared length;
   an incomplete packet (short payload or unterminated length) stashes
   the whole packet bytes into the pending buffer for carry-over. The
   maximum-length check counts the whole packet including fixed-header
   bytes and on violation returns the max-payload error without
   touching the pending buffer. Covered by `TestDetail02`.
3. **Reset carry-over.** Resetting prepends any pending partial packet
   before the new buffer and clears the cursor state. Covered by
   `TestDetail03`.
4. **Field reads.** Byte-string reads are uint16-length-prefixed; a
   zero length consumes only the prefix; the no-copy form aliases the
   read buffer while the copy form detaches. Short reads produce errors
   naming the field with EOF-family wording. Covered by `TestDetail04`.
5. **Fixed-header flags.** CONNECT, the PUB-ack types, PINGREQ and
   DISCONNECT require flags of zero; PUBREL, SUBSCRIBE and UNSUBSCRIBE
   require 0x2; PUBLISH and unrecognized packet types accept any flags.
   Covered by `TestDetail05`.
6. **Remaining length.** CONNECT/PUBLISH/SUBSCRIBE/UNSUBSCRIBE accept
   any length; the PUB-ack types require exactly 2; PINGREQ/DISCONNECT
   require 0; unknown types pass. Covered by `TestDetail06`.
7. **Packet identifier.** A packet identifier is a big-endian uint16
   and zero is rejected with the zero-identifier error. Covered by
   `TestDetail07`.
8. **Flag helpers.** QoS is the two-bit field at bits 1-2; retain is
   bit 0. Covered by `TestDetail08`.
9. **Writer layout.** Uint16 writes big-endian; strings and byte
   fields carry a uint16 length prefix; varints emit LSB-first
   continuation bytes with at least one byte for zero; the constructor
   pre-grows to the requested capacity. Covered by `TestDetail09`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — spec varints plus the 4-byte cap and complete-flag contract |
| TestDetail02 | 2 | no — asserted as observable carry-over/whole-packet/maxviolation shapes, no literals |
| TestDetail03 | 3 | partially — prepend-then-clear observed through reader state |
| TestDetail04 | 4 | partially — zero-length, alias-vs-copy, and field-named error shapes |
| TestDetail05 | 5 | doc — spec table including the unknown-type default arm |
| TestDetail06 | 6 | doc — same |
| TestDetail07 | 7 | doc |
| TestDetail08 | 8 | doc |
| TestDetail09 | 9 | doc |
