# Contract — packenc

Hidden suite: `tests/hidden/plumbing/format/packfile/packenc_bb_test.go`
(package `packfile`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | The pack trailer equals the object-format hash of every preceding byte: SHA-1 body hash by default (returned hash matches trailer), SHA-256 body hash when the storer is configured SHA-256. |
| TestDetail02 | 2 | doc | `Encode` obtains the object list from the configured `ObjectSelector` — a supplied selector is consulted and the pack holds exactly its objects. |
| TestDetail03 | 3 | partially | A delta listed before its unwritten base emits the base first; the emitted delta entry's base offset points at the earlier base entry. |
| TestDetail04 | 4 | doc | A delta cycle (each entry delta'd on the other) terminates and produces a pack that parses to both original contents. |
| TestDetail05 | 5 | no | SHAPE: a fresh `ObjectToPack` is neither `IsWritten` nor `WantWrite`; `MarkWantWrite` sets `WantWrite` without `IsWritten`; after encode it reports `IsWritten`. Sentinel arithmetic unpinned. |
| TestDetail06 | 6 | partially | An OFS-delta entry's header carries a positive offset-VLQ distance landing exactly on the earlier base entry's offset. |
| TestDetail07 | 7 | no | SHAPE: all delta entries in one pack share one kind — OFS when `useRefDeltas` is false, REF when true. |
| TestDetail08 | 8 | partially | A REF-delta entry's base reference equals the base object's raw hash. |
| TestDetail09 | 9 | partially | Entry header bytes: `(type<<4)\|(size&0xF)` first byte with continuation bit, then 7-bit size groups LSB-first — checked literally for sizes 5 (`0x35`) and 300 (`0xBC 0x12`). |
| TestDetail10 | 10 | no | SHAPE: `Type`/`Hash`/`Size` answer from `Original` when present, then from saved metadata (`SaveOriginalMetadata` + `CleanOriginal`); with no original, `Type` falls back to the base's type (not the delta's). |
| TestDetail11 | 11 | no | SHAPE: `Type`/`Hash`/`Size` panic on an `ObjectToPack` with no source — no error return exists. |
| TestDetail12 | 12 | doc | `SetOriginal(nil)` keeps the previously resolved type/size/hash. |
| TestDetail13 | 13 | partially | A delta's `Depth` is `base.Depth+1` at link time; `BackToOriginal` resets it to 0 and clears delta-ness. |
| TestDetail14 | 14 | partially | The emitted pack parses end to end and each object inflates to its exact original content — per-entry zlib streams interleaved with the hash tee. |

Refusals/softening: line 5 asserted through the `IsWritten`/`WantWrite`
API only, never the sentinel values; line 6's "non-positive distance is an
error" half is unreachable via the public path (distances are always positive
in a real pack) and not asserted; line 10's exact fallback order beyond the
observable cases is not pinned.
