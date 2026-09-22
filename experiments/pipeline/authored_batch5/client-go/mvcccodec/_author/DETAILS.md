# Details — mvcccodec

1. `mvccLock.MarshalBinary` writes, in order: `startTS`, `primary`,
   `value`, `op`, `ttl`, `forUpdateTS`, `txnSize`, `minCommitTS` — numbers
   little-endian fixed-width, byte slices as `uvarint length || bytes`.
   `UnmarshalBinary` reads the same order. Inferable: no — field order and
   width choices are the format.
2. `mvccValue` marshals `valueType` (int64), `startTS`, `commitTS`,
   `value` in that order. Inferable: no.
3. `marshalHelper` latches the FIRST error (`mh.err`) and every later
   write/read is a no-op; `MarshalBinary`/`UnmarshalBinary` return that
   error. Inferable: partially — the sticky-error pattern is visible in
   the struct but the check placement is a choice.
4. `WriteSlice` frames with `binary.PutUvarint`; `ReadSlice` enforces a
   `10MiB` cap (`"too large slice, maybe something wrong"`) and reads
   exactly `sz` bytes via `io.ReadFull`. Inferable: no — the cap constant
   is arbitrary.
5. `writeFull` loops `Write` until all bytes are out. Inferable: yes.
6. `NewMvccKey` = `codec.EncodeBytes(nil, key)`, nil for empty;
   `MvccKey.Raw` = `codec.DecodeBytes` and PANICS on decode error.
   Inferable: partially — panic-on-error is a choice.
