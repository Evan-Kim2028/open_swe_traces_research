# Contract — hfs-dot

Hidden suite: `tests/hidden/internal/pathutil/hfs_dot_bb_test.go`
(package `pathutil`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Positive and negative spellings of the dot-file shape round-trip the predicate as documented — folded ASCII equal to `"."+needle` matches, other spellings do not. |
| TestDetail02 | 2 | no | SHAPE: members of the ignored set placed before the dot, between needle letters, and after the last letter are all skipped — the match still succeeds. |
| TestDetail03 | 3 | no | SHAPE: every member of the committed ignored ranges is skipped inside a needle; representative code points just outside the ranges are not. |
| TestDetail04 | 4 | yes | With ignored code points consumed, the next real rune must be `'.'` — an ignored rune alone or a different rune fails. |
| TestDetail05 | 5 | yes | Each needle rune consumes the next non-ignored rune — interleaved ignored runes still match, a wrong real rune fails. |
| TestDetail06 | 6 | no | SHAPE: a non-ASCII rune in a needle position fails the match, including one that has an ASCII lowercase fold. |
| TestDetail07 | 7 | partially | ASCII uppercase in `part` folds and matches; a non-ASCII uppercase never folds into the needle. |
| TestDetail08 | 8 | yes | A non-ignored leftover after the needle fails; a trailing ignored rune is fine. |
| TestDetail09 | 9 | yes | Empty `part`, a lone dot, and needles that leave real runes unmatched all fail. |
| TestDetail10 | 10 | yes | `IsHFSDotGit(".GIT")` is true; `IsHFSDotGit(".gitmodules")` is false; the other committed needles match their plain spellings. |

Refusals/softening: line 1 is asserted through the documented positive and
negative shapes rather than an exhaustive predicate reimplementation. Line 3
probes every committed range member plus boundary outsiders; it does not pin
behaviour on code points the list is silent about inside the ranges. Line 7
asserts ASCII folding and the non-ASCII exclusion; the concrete fold function
is not pinned beyond its observable result.
