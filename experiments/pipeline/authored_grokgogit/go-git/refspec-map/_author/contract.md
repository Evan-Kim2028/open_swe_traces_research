# Contract — refspec-map

Hidden suite: `tests/hidden/config/refspec_map_bb_test.go`
(package `config`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Specs with zero or several `:` or an empty destination fail with `ErrRefSpecMalformedSeparator`; plain, forced, and delete (`:dst`) spellings validate. |
| TestDetail02 | 2 | no | SHAPE: mismatched or repeated `*` counts between source and destination fail with the wildcard sentinel; equal counts of 0 or 1 pass. |
| TestDetail03 | 3 | yes | A leading `+` sets the force flag and is not part of `Src()`. |
| TestDetail04 | 4 | yes | A spec beginning with `:` is a delete with an empty source. |
| TestDetail05 | 5 | yes | Non-wildcard `Match` is exact equality with the source — prefix, suffix, and superstring names all fail; a forced spec still matches its source. |
| TestDetail06 | 6 | partially | Wildcard `Match` requires the name to carry the source's `*`-split prefix and suffix and be at least their combined length — an empty star-slice still matches. |
| TestDetail07 | 7 | yes | Non-wildcard `Dst` returns the text after `:`, including under a force flag. |
| TestDetail08 | 8 | no | SHAPE: wildcard `Dst` copies the star-matched slice into the destination `*` — asserted on the committed `*bc` example and a plain `*` case. |
| TestDetail09 | 9 | yes | `MatchAny` is true when any member matches, false when none do or the list is empty. |
| TestDetail10 | 10 | doc | Malformed specs surface the two package sentinels — separator faults give the separator sentinel, wildcard faults the wildcard sentinel, and they are not aliased. |

Refusals/softening: line 1 does not pin validation of edge spellings the
rule leaves open (e.g. whether a delete's source emptiness is separately
checked). Line 2 asserts the sentinel and the count rule on clean
single-violation inputs; inputs violating both rules are not pinned. Line 6
asserts the committed prefix/suffix/length conditions only. Line 8 asserts
the committed example verbatim plus one parallel substitution; other
wildcard arities are left free.
