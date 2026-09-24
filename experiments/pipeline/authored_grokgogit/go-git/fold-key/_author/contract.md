# Contract — fold-key

Hidden suite: `tests/hidden/plumbing/format/config/fold_key_bb_test.go`
(package `config`, in-package — exercises the unexported `foldKey` and
`foldRune`). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | Across a table of pairs, `foldKey(a) == foldKey(b)` exactly when `strings.EqualFold(a, b)` — both directions checked. |
| TestDetail02 | 2 | doc | An all-lowercase-ASCII name returns unchanged and `testing.AllocsPerRun` reports zero allocations. |
| TestDetail03 | 3 | no | SHAPE: `foldRune` maps each probed rune to the smallest rune of its fold orbit, ASCII-letter-orbit results landing on the lowercase letter. |
| TestDetail04 | 4 | no | SHAPE: `foldRune` on ASCII capitals returns the lowercase letter; `foldKey("CORE")` is `"core"`. |
| TestDetail05 | 5 | doc | `foldKey("Core")` and `foldKey("core")` both equal `"core"`. |
| TestDetail06 | 6 | no | SHAPE: `foldKey` on the long-s character collides with `foldKey("s")` and `foldKey("S")`. |
| TestDetail07 | 7 | no | SHAPE: `foldKey` on the Kelvin sign collides with `foldKey("k")` and `foldKey("K")`. |
| TestDetail08 | 8 | no | SHAPE: names differing only in invalid UTF-8 bytes fold to the same key; a two-bad-byte name does not collide with a one-bad-byte name. |

Refusals/softening: line 3 is asserted only on runes whose orbit minimum is
visible from the committed rule (long s, Kelvin sign, ASCII capitals, a
non-letter); no arbitrary orbit table is reproduced. Line 8 asserts
collision behaviour, not the exact replacement bytes.
