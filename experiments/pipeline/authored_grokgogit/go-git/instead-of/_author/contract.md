# Contract — instead-of

Hidden suite: `tests/hidden/config/instead_of_bb_test.go`
(package `config`, in-package — exercises the unexported
`applyLongestInsteadOfMatch` helper alongside `URL.ApplyInsteadOf`).
One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | `ApplyInsteadOf` rewrites via that URL's own `InsteadOfs` only; a rule on a different `URL` value does not leak in. |
| TestDetail02 | 2 | yes | A remote URL carrying the `insteadOf` string as a prefix matches; a non-prefix occurrence does not. |
| TestDetail03 | 3 | doc | When two prefixes on the same URL both match, the longer wins regardless of slice order. |
| TestDetail04 | 4 | no | SHAPE: two `URL` values advertising the equal-length winning prefix resolve to the first one in slice order. |
| TestDetail05 | 5 | partially | The rewrite is `Name` plus the unmatched suffix of the remote URL — asserted on a concrete pair. |
| TestDetail06 | 6 | yes | No matching prefix returns the original URL from `ApplyInsteadOf` and `matched=false` from the helper. |
| TestDetail07 | 7 | no | SHAPE: across several `URL` values, the longest matching prefix wins even when it lives on a later element. |
| TestDetail08 | 8 | no | SHAPE: an empty `insteadOf` string — a zero-length prefix match — counts as no match on both the helper and `ApplyInsteadOf`. |

Refusals/softening: line 4 is exercised across two `URL` values because the
tie is only observable there; it does not pin the within-slice tie-break of
one URL. Lines 7–8 assert ordering and emptiness shape only — the iteration
mechanics are left free.
