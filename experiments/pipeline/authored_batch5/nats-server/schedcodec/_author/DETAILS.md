# Details — schedcodec

1. `encode` wire format: `b[0]` = version byte 1, then LE uint64
  `count`, then LE uint64 `highSeq` stamp (headerLen = 17). Per entry:
  LE uint16 subject length, subject bytes (truncated at MaxUint16),
  `binary.AppendVarint` signed ts, `binary.AppendUvarint` seq.
  Inferable: partially — field order/widths visible in the test that
  hand-builds headers, but ts is signed-varint while seq is uvarint.
2. `decode` returns `io.ErrShortBuffer` when `len(b) < headerLen`,
  `ErrMsgScheduleInvalidVersion` when `b[0] != 1`, and
  `io.ErrUnexpectedEOF` for every truncation past the header —
  subject len, subject bytes, ts varint (`tn <= 0`), seq varint.
  It calls `ms.init(seq, subj, ts)` per entry (populates schedules,
  seqToSubj, AND the ttls wheel) and returns the stamp. Inferable:
  yes — TestFileStoreMessageScheduleDecodeRejectsMalformed pins each
  truncation to its exact error.
3. `parseMsgSchedule("")` returns zero-time, false, true — empty is a
  valid no-op. Inferable: partially.
4. `@at <RFC3339>` → one-shot at that instant; `@every <dur>` →
  repeating, dur must parse AND be ≥ 1s. NEITHER accepts a non-nil
  `loc` (time zone unsupported → valid=false). Inferable: doc —
  the `@at`/`@every` subtests exercise both.
5. Predefined aliases expand to 6-field cron: `@yearly`/`@annually` →
  `0 0 0 1 1 *`, `@monthly` → `0 0 0 1 * *`, `@weekly` → `0 0 0 * * 0`,
  `@daily`/`@midnight` → `0 0 0 * * *`, `@hourly` → `0 0 * * * *`.
  Anything else goes to `parseCron` as-is; parse failure →
  valid=false. Inferable: partially — the table is internal.
6. Past-fire catch-up: if the computed `next` is before now (e.g. a
  repeating schedule after restart), it is skipped forward — `@every`
  adds one interval to `now.Round(Second)`; the cron path re-parses
  from the bumped ts — and still fires only once (returns
  repeating=true). Inferable: partially — restart semantics.
7. `@every` next is computed from `time.Unix(0, ts).UTC().Round(Second)
  + dur`, i.e. anchored to the supplied timestamp, not wall clock —
  unless that lands in the past. Inferable: partially.
