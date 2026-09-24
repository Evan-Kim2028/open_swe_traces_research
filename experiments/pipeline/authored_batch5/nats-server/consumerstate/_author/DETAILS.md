# Details — consumerstate

1. Consumer-state record: byte0 `magic` (22), byte1 version (encoder writes
   2), then uvarint AckFloor.Consumer, AckFloor.Stream, Delivered.Consumer,
   Delivered.Stream, uvarint len(Pending). Inferable: doc — constants
   visible, layout implied by decode order.
2. If Pending non-empty: `PutVarint(mints)` where mints = `time.Now().
   Round(time.Second).Unix()`; then per entry `Uvarint(seq - AckFloor.
   Stream)`, `Uvarint(p.Sequence - AckFloor.Consumer)`, `Varint(mints -
   Timestamp/time.Second)` — pending seqs are DELTAS against the ack floor
   and timestamps are second-resolution offsets from mints (so a pending
   timestamp up to ~1s in the future encodes as varint -1 and must still
   decode). Inferable: partially — delta scheme visible in decode; the -1
   edge is a known regression (norace test).
3. `Uvarint(len(Redelivered))` is always written even when Pending empty;
   redelivered entries are `Uvarint(seq - AckFloor.Stream)`, `Uvarint
   (count)`. Inferable: partially.
4. Encoder preallocates a worst-case buffer (`seqsHdrSize` + per-entry
   `MaxVarintLen64` estimates) and returns `buf[:n]`. Inferable: no —
   sizing is internal.
5. `checkConsumerHeader`: len>=2, hdr[0]==magic, version in {1,2} →
   version; otherwise `errCorruptState` or an "unsupported version" error.
   Inferable: yes — error vars visible.
6. `decodeConsumerState` reads fields strictly sequentially; ANY failed
   varint poisons the cursor (bi=-1) and yields `errCorruptState` at the
   next checkpoint. Inferable: partially.
7. Version-1 compat: Delivered.Consumer/Stream are adjusted UP by
   AckFloor-1 when the floor > 1 (v1 stored next-to-deliver). Version-1
   pending entries have no delivered seq (stays 0) and their timestamp
   decodes as `(ts + mints) * Second`; v2 uses `(mints - ts) * Second`.
   Inferable: no — silent wire-format compat.
8. Corruption guards: AckFloor.Stream or Delivered.Stream with the top
   bit (1<<63) set → `errCorruptState`; a pending stream seq that resolves
   to 0 → `errCorruptState`; redelivered entries with seq==0 or count==0
   are SKIPPED (a zero seq once panicked the server). Inferable: doc —
   regression test exists.
9. Stream snapshot frame: `streamStateMagic` + version byte
   (`streamStateVersion` or `streamStateVersionSources`), then uvarint
   Msgs, Bytes, FirstSeq, LastSeq, Failed. IsEncodedStreamState checks
   only magic+version+min length. Inferable: yes.
10. With-sources version: after Failed, uvarint numSources then per source
    a uvarint-len-prefixed name, uvarint seq, uvarint-len-prefixed ident;
    sources precede deleted blocks because deleted blocks consume the rest
    of the buffer. Inferable: partially — ordering constraint in comment.
11. Deleted blocks: uvarint numDeleted > 0, then per block a magic byte —
    `seqSetMagic` → `avl.Decode` (dense bitmap), `runLengthMagic` →
    `DeleteRange{First, Num}` uvarint pair; unknown magic →
    `ErrCorruptStreamState`. Any uvarint failure → `ErrCorruptStreamState`.
    Inferable: partially.
12. `DeleteRange.State` → (First, First+Num-1, Num); `Range` iterates
    First..First+Num-1. `DeleteSlice.State` → (ds[0], ds[len-1], len), all
    zeros when empty; `Range` iterates slice order (not contiguous).
    `DeleteBlocks.NumDeleted` sums each block's State num. Inferable: yes.
13. `uvarintLen(v)` = ceil(bits.Len64(v|1)/7) — v=0 sizes 1 byte.
    `runLengthEncodeLen` = 1 + uvarintLen(first) + uvarintLen(num);
    `appendRunLength` emits runLengthMagic then both uvarints. Inferable:
    doc — comment says "exactly matching what appendRunLength writes".
