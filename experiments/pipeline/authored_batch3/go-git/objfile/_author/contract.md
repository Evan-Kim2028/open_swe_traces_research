# Contract — objfile

Hidden suite: `tests/hidden/plumbing/format/objfile/objfile_bb_test.go`
(package `objfile`, in-package). One `TestDetailNN` per DETAILS.md line.
Inputs are hand-zlibbed byte strings; the stream is inflated directly where
committed bytes (not just errors) are the claim.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | zlib stream `"blob 5\x00hello"` → `Header()` = `(BlobObject, 5)`, body reads `"hello"`. |
| TestDetail02 | 2 | partially | A 36-byte header (`"blob "` + 30-digit size + NUL) fails — at construction or `Header()` — with `ErrHeaderTooLong` or `ErrHeader`. |
| TestDetail03 | 3 | partially | `"bogus 5\x00…"` (bad type) and `"blob xyz\x00…"` (non-numeric size) each fail at construction or `Header()`. |
| TestDetail04 | 4 | no | SHAPE: writer refuses `-3` with `ErrNegativeSize`; reader given `"blob -3\x00"` either refuses at construction or returns from `Header()` without panicking — reader's treatment unpinned. |
| TestDetail05 | 5 | partially | Over-budget unterminated header → `ErrHeaderTooLong`; stream ending before the NUL → a different failure that is NOT `ErrHeaderTooLong`. |
| TestDetail06 | 6 | no | `Read` before `Header` → `ErrHeaderNotRead`; `Hash` before `Header` → zero hash whose hex is 40 chars (format-sized, not absent). |
| TestDetail07 | 7 | partially | After reading the body, `Hash()` = sha1 of `"blob 5\x00hello"` — header bytes are hashed though never seen through `Read`. |
| TestDetail08 | 8 | doc | `WriteHeader(InvalidObject)` errors with zero bytes emitted; `WriteHeader(blob, -1)` → `ErrNegativeSize`. |
| TestDetail09 | 9 | no | SHAPE: `Write("abcde")` on a size-3 blob → `ErrOverflow`, and the inflated stream still contains the fitting `abc` — committed, not rolled back. |
| TestDetail10 | 10 | no | SHAPE: a write after the size is exhausted → `ErrOverflow`; committed `abc` remains in the stream. |
| TestDetail11 | 11 | no | SHAPE: `Write` before `WriteHeader` never returns nil — error or panic, never silent success. |
| TestDetail12 | 12 | doc | `Close` returns the identical error on repeat calls; `Reader.Close` does not call `Close` on the wrapped reader (close-tracker asserted). |

Refusals/softening: line 8's over-bound-writer clause is NOT asserted — unreachable through
the public API (max header is 9-char type + 19-digit size + 2 delimiters = 31 bytes < 32);
recorded here rather than faked. Line 4 pins only "writer refuses, reader doesn't crash".
