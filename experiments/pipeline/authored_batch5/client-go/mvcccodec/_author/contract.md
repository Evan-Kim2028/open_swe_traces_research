# Contract (L2) — mvcccodec

`mvccLock.MarshalBinary` encodes all eight fields — `startTS`, `primary`,
`value`, `op`, `ttl`, `forUpdateTS`, `txnSize`, `minCommitTS` — and
`UnmarshalBinary` reads the same order; the shape committed is a
deterministic round-trip in which every field participates. `mvccValue`
marshals `valueType`, `startTS`, `commitTS`, `value` and round-trips the
same way. `marshalHelper` latches the first error and every later write
or read is a no-op; marshal and unmarshal return that latched error.
`WriteSlice` length-frames a slice so `ReadSlice` recovers it, reads
exactly the declared length, and rejects a declared length above the cap.
`writeFull` loops `Write` until all bytes are out. `NewMvccKey` encodes a
raw key via the bytes codec and yields an empty encoding for empty input;
`MvccKey.Raw` decodes back to the raw key and panics on malformed input.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `mvccLock` marshal/unmarshal round-trips all eight fields; encoding is deterministic and every field participates |
| `TestDetail02` | `mvccValue` marshal/unmarshal round-trips `valueType`/`startTS`/`commitTS`/`value` |
| `TestDetail03` | `marshalHelper` latches the first error; later writes and reads are no-ops |
| `TestDetail04` | `WriteSlice`/`ReadSlice` round-trip framed slices, read exactly the declared length, and reject oversized declarations |
| `TestDetail05` | `writeFull` loops until all bytes are written |
| `TestDetail06` | `NewMvccKey` encodes via the bytes codec, empty for empty input; `Raw` decodes and panics on malformed data |
