# Contract — inforefs

Hidden suite: `tests/hidden/plumbing/protocol/packp/inforefs_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | A line with no TAB fails the whole decode with `errors.Is(err, ErrInvalidInfoRefs)`. |
| TestDetail02 | 2 | doc | Hash fields of length 39, 41, 62, and 40-non-hex all fail the whole body (even after a valid first line); a 64-hex hash decodes and produces the sha256 reference. |
| TestDetail03 | 3 | doc | On a rejected body the receiver's `References` keeps its pre-existing slice — no partial append. |
| TestDetail04 | 4 | doc | Empty lines interleaved in the body are skipped silently; 2 refs decode. |
| TestDetail05 | 5 | doc | A `bad..name` line (invalid refname) is dropped while surrounding valid lines still decode — skip, not fatal. |
| TestDetail06 | 6 | doc | `refs/tags/v1^{}` decodes with the `^{}` suffix preserved in the name. |
| TestDetail07 | 7 | partially | A line with valid hash + TAB + empty name is skipped; the following valid line still decodes. |
| TestDetail08 | 8 | doc | A ~70 KB single line fails decode and leaves the receiver empty; a mid-stream reader error is returned unchanged (`errors.Is` the sentinel). |
| TestDetail09 | 9 | partially | `name\r\n` decodes with the name stripped to `name`; a name containing a second `\r` is skipped (name-rule failure) while the next line still decodes. |
| TestDetail10 | 10 | yes | `Encode` output equals `"<hash>\t<name>\n"` per reference in stored order — exact byte equality including the `^{}` name. |
| TestDetail11 | 11 | doc | An 8-hex-char hash field fails with `ErrInvalidInfoRefs` and produces no reference — no silent zero-padding. |

No softening needed — every line was `doc`/`yes`/`partially` and each assertion lands on the
committed text.
