# Details — wsframe

1. `wsFillFrameHeader`: byte 0 is `frameType | wsFinalBit(if final) |
   wsRsv1Bit(if compressed)` — but frameType bits are written only when
   `first` is set (continuation frames leave them 0). Byte 1 gets `wsMaskBit`
   when masking. Length encoding: `l <= 125` inline (header size 2);
   `l < 65536` → marker 126 + 2-byte big-endian (size 4); otherwise marker
   127 + 8-byte big-endian (size 10). Inferable: doc — RFC 6455 layout.
2. When `useMasking` the 4-byte mask key is generated from crypto/rand with
   a `math/rand/v2` FALLBACK on read error, copied into the header right
   after the length field, and also returned to the caller. Inferable:
   partially — the random fallback is an internal choice.
3. `wsCreateFrameHeader` pulls a `wsMaxFrameHeaderSize` (14) buffer from
   `nbPoolGet` and returns `fh[:n]` — callers get a pooled slice they must
   return via `nbPoolPut`. Inferable: partially — pooling contract visible
   at call sites.
4. `wsMaskBuf` XORs `buf[i] ^= key[i&3]`; `wsMaskBufs` does the same but the
   key position CONTINUES across buffers (`pos` is not reset per buffer).
   Inferable: doc — the "as if contiguous" comment states it.
5. `wsReadInfo.unmask` honours the running key position `r.mkpos` across
   calls (a masked frame may be split across reads); for buffers ≥16 bytes
   it XORs 8-byte lanes at a time built from the rotated key, then handles
   the tail. `mkpos` is stored mod 4. Inferable: no — the lane
   vectorisation and the cross-call position are internal.
6. `wsIsControlFrame` is `frameType >= wsCloseMessage` (8) — so unknown
   opcodes ≥8 count as control too. Inferable: doc — spec-shaped, the
   >=-not-== boundary is a choice.
7. `wsIsValidCloseStatus` rejects: 1005 (NoStatusReceived), 1004, 1006,
   1015 (TLSHandshake), anything <1000 or ≥5000, and the reserved range
   1016–2999 — note that makes MOST currently-assigned codes invalid
   (1000–1015 minus the four named, and 3000–4999 pass). Inferable: doc —
   the reserved-range rule follows RFC, exact table is a choice.
8. `wsCreateCloseMessage` emits `uint16 BE status + body`; if body exceeds
   `wsMaxControlPayloadSize-2` (123) it is cut to `size-5` (120) and
   `"..."` appended; buffer comes from `nbPoolGet`. Inferable: partially —
   truncation-with-ellipsis is stated in the comment, exact offsets are not.
9. `wsGet` returns `buf[pos:pos+needed]` zero-copy when enough bytes remain,
   else copies the tail into a fresh slice and reads the REST from `r` —
   the returned `pos` advances only past consumed buffer bytes (`pos+avail`),
   not the bytes read from `r`. Inferable: no.
10. `wsMaxMessageSize(mpay)` is `mpay * wsMaxMsgPayloadMultiple` (8) capped
    at `wsMaxMsgPayloadLimit` (64MB); `mpay <= 0` substitutes
    `MAX_PAYLOAD_SIZE`. Inferable: partially — constants visible, the
    default substitution is not.
