# Contract — wsframe

WebSocket (RFC 6455) frame codec: header construction, masking,
close-message bodies, control classification, and buffered reads.
Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **Header layout.** Byte 0 carries the frame type (only on first
   fragments), the final bit, and the compression bit; byte 1 carries
   the mask bit; lengths encode inline up to 125, via the 126 marker
   with a 2-byte big-endian length, or the 127 marker with an 8-byte
   big-endian length. Covered by `TestDetail01`.
2. **Masking key.** With masking, a 4-byte key is generated, written
   after the length field, and returned to the caller. Covered by
   `TestDetail02`.
3. **Pooled headers.** The header constructor returns a pooled maximum-
   size buffer sliced to the actual header length. Covered by
   `TestDetail03`.
4. **Contiguous masking.** The multi-buffer mask continues the key
   position across buffers as if they were one stream. Covered by
   `TestDetail04`.
5. **Streaming unmask.** The reader unmask honours the running key
   position across calls, stored mod 4. Covered by `TestDetail05`.
6. **Control frames.** Opcodes at or above the close opcode are
   control frames, including unassigned ones. Covered by
   `TestDetail06`.
7. **Close status validity.** Codes below 1000, at or above 5000, the
   named reserved codes, and the reserved range 1016–2999 are invalid;
   other assigned and 3000–4999 codes pass. Covered by `TestDetail07`.
8. **Close message.** The body is a big-endian u16 status plus text,
   truncated to the max control payload with an ellipsis tail. Covered
   by `TestDetail08`.
9. **Buffered get.** Reads are zero-copy while the buffer suffices and
   otherwise copy the tail and pull the rest from the reader, advancing
   the position only past consumed buffer bytes. Covered by
   `TestDetail09`.
10. **Message-size cap.** The max message size is eight times the max
    payload capped at 64MB, substituting the default payload size for
    non-positive values. Covered by `TestDetail10`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — RFC layout asserted byte-wise |
| TestDetail02 | 2 | partially — key placement/return asserted; fallback internals not pinned |
| TestDetail03 | 3 | partially — pooled cap asserted |
| TestDetail04 | 4 | doc — cross-buffer position asserted |
| TestDetail05 | 5 | no — shape: cross-call mkpos continuation |
| TestDetail06 | 6 | doc — >=8 boundary asserted both sides |
| TestDetail07 | 7 | doc — named codes and ranges asserted |
| TestDetail08 | 8 | partially — truncation length and ellipsis asserted |
| TestDetail09 | 9 | no — shape: zero-copy vs read-through and pos semantics |
| TestDetail10 | 10 | partially — multiple, cap, and default substitution asserted |
