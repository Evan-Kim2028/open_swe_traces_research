# Contract — sigblock

Hidden suite: `tests/hidden/plumbing/object/sigblock_bb_test.go`
(package `object`, in-package — all targets are unexported helpers).
One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | `parseSignedBytes` on a mid-line `-----BEGIN PGP SIGNATURE-----` → -1; on a line-boundary marker → the exact byte offset of the line start. |
| TestDetail02 | 2 | partially | `-----BEGIN PGP MESSAGE-----` at a boundary → position + `signatureTypeOpenPGP`. |
| TestDetail03 | 3 | doc | Two markers (PGP then SSH) → position = the LAST marker's offset, type = `signatureTypeSSH`. |
| TestDetail04 | 4 | doc | `countSignatureBlocks` returns ≥2 for two `BEGIN PGP SIGNATURE` lines, exactly 1 for a single block. |
| TestDetail05 | 5 | doc | `isSignatureHeader` true only for `gpgsig ` and `gpgsig-sha256 ` (trailing space); false for `gpgsig`, `gpgsigfoo`, `gpgsig-x `, `gpgsig2 `. |
| TestDetail06 | 6 | no | SHAPE: `stripHeaderSignatures` drops a `gpgsig` header + its space-continuation, and a directly-following `gpgsig-sha256` + continuation; `object x`, `other keep`, and body lines survive. |
| TestDetail07 | 7 | partially | A `gpgsig`-looking line in the BODY (after the blank separator) is copied verbatim while the header `gpgsig` is stripped. |
| TestDetail08 | 8 | partially | `stripObjectSignatures` on a tag truncates the trailing inline `BEGIN PGP SIGNATURE` block (`INLINE` gone); on a commit the same bytes are kept. |
| TestDetail09 | 9 | no | SHAPE: an unterminated final body line is still copied through to the output — no truncation, no error. |
| TestDetail10 | 10 | partially | Destination object type is set from the requested `objType` (CommitObject), not inherited from the TagObject source. |

No refusals — every `no` line here has an observable shape (content survives / is dropped /
call terminates) that the suite asserts without pinning internals.
