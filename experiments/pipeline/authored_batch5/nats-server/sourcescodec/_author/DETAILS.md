# Details — sourcescodec

1. `sources.db` layout: byte 0 = version 1; LE u64 `count`; LE u64
  `highSeq` stamp. Per source: uvarint len + name bytes, uvarint seq,
  uvarint len + ident bytes. `writeSourcesState` pre-sizes the buffer
  exactly (uvarintLen accounting) and uses `highSeq = LastSeq+1` — the
  stamp means "sources valid up TO this seq". Inferable: partially —
  the +1 off-by-one is an internal contract.
2. `decodeSourcesState`: `<17` → io.ErrShortBuffer; `b[0]!=1` →
  errSourcesInvalidVersion; every per-entry field (name uvarint/bytes,
  seq uvarint, ident uvarint/bytes) checked with `io.ErrUnexpectedEOF`
  — including the uvarint-declared length overrunning the remaining
  buffer. Allocates `fs.sources` only when count>0. Inferable: yes —
  TestFileStoreSourcesDecodeRejectsMalformed pins each case.
3. `streamAndSeq` dispatch: header starting with the `$JS.ACK` prefix
  → legacy decode; else space-split. `nFields != 2 && nFields <= 3`
  rejects 1- and 3-field headers — the v2 header needs ≥4 fields
  (name seq src dest [orig ident]). Inferable: partially — the
  2-vs-≥4 arity split is internal.
4. v2 iname reconstruction: `iname = fields[0]+" "+fields[2]+" "+
  fields[3]` (stream name + source + dest re-joined), seq from
  `fields[1]` via parseAckReplyNum, `ident = fields[5]` copied (stored
  on sourceInfo) when ≥6 fields. Returns `_EMPTY_`s on mismatch.
  Inferable: partially — field positions are a wire contract.
5. `genSourceHeader` writes `"<iname-part0> <seq> <iname-part1>
  <iname-part2> <orig> [ident]"` where seq is token 5 (v1) or 7 (v2)
  of the inbound `$JS.ACK` reply, defaulting to "1" when the reply is
  not an ack — and ident only appended when non-empty. Inferable:
  partially — the token offsets mirror ackReplyInfo.
6. `streamAndSeqFromAckReply` extracts `(stream, iname, sseq)` from
  the raw `$JS.ACK.<domain>.<acc>.<stream>.<consumer>.<del>.<sseq>…`
  subject — v1/v2 token layouts both handled, empty on malformed.
  Inferable: partially.
