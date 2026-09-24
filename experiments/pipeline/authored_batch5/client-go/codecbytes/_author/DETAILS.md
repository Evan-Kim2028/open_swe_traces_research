# Details — codecbytes

1. `EncodeBytes` emits `[group][marker]` pairs: each group is 8 bytes of data
   zero-padded on the right, each marker is `0xFF - padCount`. Inferable: doc —
   the format and the worked table are in the doc comment.
2. Encoding empty input still produces one group: eight `0x00` bytes then
   marker `0xF7`. Inferable: partially — the always-terminate group is visible
   in the doc table but the general rule is not stated.
3. When the input length is an exact multiple of 8, a full group is emitted
   with marker `0xFF` and then a final all-pad group with marker `0xF7`
   terminates the stream. Inferable: partially.
4. `DecodeBytes` consumes groups until the first marker that implies nonzero
   padding; that group's `realGroupSize` bytes end the value. It returns the
   leftover slice as its first result. Inferable: partially.
5. Decode rejects a marker implying more than 8 pad bytes ("invalid marker
   byte"), a padding byte that is not `0x00` ("invalid padding byte"), and any
   truncated tail ("insufficient bytes to decode value"). Inferable: partially —
   three distinct error cases, wording is a choice.
6. Decode reuses the caller's `buf` (reset to length 0) instead of allocating
   when `buf` is non-nil. Inferable: doc — stated in the DecodeBytes comment.
7. Round-trip: `DecodeBytes(EncodeBytes(nil, data))` returns `data` verbatim
   and an empty leftover, including for `data` of length 0, 7, 8, 9, 16.
   Inferable: yes.
8. `reallocBytes` returns a slice with the same length and contents as `b`
   that can hold `n` more bytes; it reallocates only when `cap(b)` is too
   small. Inferable: no — internal allocator detail.
