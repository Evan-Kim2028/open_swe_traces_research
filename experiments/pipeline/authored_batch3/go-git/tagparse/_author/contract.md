# Contract — tagparse

Hidden suite: `tests/hidden/plumbing/object/tagparse_bb_test.go`
(package `object`, in-package). One `TestDetailNN` per DETAILS.md line.
Tag objects are built via `memory.NewStorage().NewEncodedObject()`.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | Wrong order (`type` first, `tag` before `type`) and missing `tag`/`object` headers all fail with `ErrMalformedTag`. |
| TestDetail02 | 2 | partially | A tag with no `tagger` line decodes to a zero `Tagger` (empty name/email, zero time) and re-encodes without a `tagger` line. |
| TestDetail03 | 3 | doc | A second `tag fake` header after `tag real` decodes with `Name == "real"` — dropped, not overriding, not erroring. |
| TestDetail04 | 4 | doc | An `x-custom-header` line decodes without error and is not stored. |
| TestDetail05 | 5 | doc | `gpgsig-sha256` with a ` cont1` continuation plus a second occurrence stores all of `part1`, `cont1`, `part2` in `SignatureSHA256`. |
| TestDetail06 | 6 | partially | A trailing `BEGIN PGP SIGNATURE` block after the message peels into `Signature` (contains `SIGDATA`) and out of `Message`. |
| TestDetail07 | 7 | doc | Encoded header order is object → type → tag → tagger → `gpgsig-sha256` → blank line — checked by position ordering. |
| TestDetail08 | 8 | doc | `Signature` appends verbatim after the message: `"body-no-trailing-newlineSIGTAIL"` (or suffix `SIGTAIL`) — no separator inserted. |
| TestDetail09 | 9 | no | SHAPE: a `Tagger` with only a Name still produces a `tagger` line on encode — not treated as zero. |
| TestDetail10 | 10 | doc | `EncodeWithoutSignature` on an unchanged decoded tag strips both `gpgsig-sha256` material and the inline signature while keeping the message; after mutating `Message`, the output re-encodes (contains `changed`, no sig header). |
| TestDetail11 | 11 | doc | Mutating `SignatureSHA256` alone still yields byte-identical `EncodeWithoutSignature` output to the unchanged tag — the field is excluded from the source-match. |
| TestDetail12 | 12 | no | SHAPE: a `gpgsig-sha256` line in tagger's position (no tagger present) still lands in `SignatureSHA256` — declined lines are re-served, not lost. |

No refusals — all `no` lines have observable shapes (value lands / is preserved / is not
treated as zero) asserted without pinning internals.
