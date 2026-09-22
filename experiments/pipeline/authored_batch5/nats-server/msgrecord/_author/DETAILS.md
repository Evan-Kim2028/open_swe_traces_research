# Details — msgrecord

1. Record layout (msgHdrSize=22): LE u32 `rl` (bit 31 = `hbit` =
  has-headers; remaining = total record len), u64 `seq` (bit 63 =
  `ebit` = erased → decoded as seq=0), u64 `ts`, u16 `slen`, then
  `data = subj | [u32 hlen | hdr] | msg | u64 checksum`.
  `emptyRecordLen = 30`. Inferable: partially — layout is internal,
  but the constants are visible.
2. Sanity gate BEFORE slicing: `dlen<0 || shlen>dlen-8 ||
  dlen>rl || rl>len(buf) || rl>32MB` → errBadMsg. Headered records
  reserve 4 extra bytes for the hlen field (`shlen = slen+4`).
  Inferable: partially.
3. Checksum (when `hh != nil`): highwayhash-64 over `hdr[4:20]` +
  subject + (hdr after its 4-byte len, when hbit) or msg — compared
  to the LAST 8 bytes of data; mismatch → errBadMsg("invalid
  checksum"). Inferable: partially — the byte ranges are precise and
  non-obvious.
4. `ebit` on seq → `sm.seq = 0` (erased record), ts still decoded.
  Inferable: partially.
5. Headered slicing: `sm.buf` holds `data[slen+4 : end]`; `sm.hdr` is
  a CAPACITY-LIMITED subslice `buf[0:hlen:hlen]`; `sm.msg` =
  `buf[hlen:end]`. Unheadered: `sm.msg = buf[0:end-slen]` — every
  boundary is bounds-checked into a distinct errBadMsg detail.
  Inferable: partially.
6. `doCopy`: true appends into `sm.buf` (backing cache reuse-safe) and
  copies subj via `string()`; false aliases via `bytesToString` and
  `sm.buf = data[...]` — callers promise not to escape.
  `sm.clear()` reuses a passed StoreMsg. Inferable: yes — comment
  documents the lifetime contract.
7. `fileStoreMsgSizeRaw`: 22 + slen + mlen + 8, plus 4+hlen when
  headers. `isFileStoreMsgTooLarge` rejects `rl&hbit != 0` (a length
  with the header bit set is impossible) or `rl > 32MB`.
  Inferable: partially — the hbit-in-length check is subtle.
8. `errBadMsg.Error()` prints basename + optional `: detail` — detail
  text distinguishes which check failed. Inferable: no (exact
  strings), shape yes.
