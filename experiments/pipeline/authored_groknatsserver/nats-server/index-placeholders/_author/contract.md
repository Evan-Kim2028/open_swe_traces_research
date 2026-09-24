# Contract — index-placeholders

Hidden suite: `tests/hidden/server/index_placeholders_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Error assertions use the mapping-error type and sentinel constants that
remain visible in the package; every error check also requires the error to
name the offending token.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Tokens of length 0 or 1 (`""`, `"a"`, `"$"`) return the no-transform kind, index list `[-1]`, scalar -1, empty string argument, nil error. |
| TestDetail02 | 2 | yes | `"$2"` and `"$10"` return the wildcard kind with that single index, scalar -1, empty string. |
| TestDetail03 | 3 | doc | `"$x"`, `"$1.5"`, `"$x1"`, `"$a2"` — `$` followed by a non-integer — return the same no-transform tuple with nil error, not a parse failure. |
| TestDetail04 | 4 | yes | `"{{}}"` (length 4), missing-open `"ab}}"`, missing-close `"{{abc"`, and short mustaches are not dispatched (no-transform, nil error); `"{{q}}"` (length > 4, both markers) does reach dispatch and errors. |
| TestDetail05 | 5 | yes | `{{wildcard()}}` is the bad-transform kind with a not-enough-args mapping error naming the token. |
| TestDetail06 | 6 | yes | `{{wildcard(3)}}` is the wildcard kind, indexes `[3]`, scalar -1. |
| TestDetail07 | 7 | yes | `{{wildcard(1,2)}}` is the bad-transform kind with a too-many-args mapping error naming the token. |
| TestDetail08 | 8 | doc | `{{partition(10)}}` is the partition kind, empty index list, scalar 10. |
| TestDetail09 | 9 | doc | `{{partition(10,1,2)}}` is the partition kind, indexes `[1,2]` in order, scalar 10. |
| TestDetail10 | 10 | yes | `{{partition(2147483648)}}` and `{{random(2147483648)}}` (one over the int32 ceiling) are the bad-transform kind with an invalid-argument mapping error naming the token. |
| TestDetail11 | 11 | yes | `splitfromleft`, `splitfromright`, `slicefromleft`, `slicefromright`, `left`, and `right` each dispatch to the matching kind constant — `{{name(3,2)}}` yields indexes `[3]`, scalar 2 — and each rejects a single argument as not-enough-args. |
| TestDetail12 | 12 | yes | `{{split(1,-)}}` is the split kind, indexes `[1]`, string argument `"-"`; `{{split(1)}}` errors as bad-transform. |
| TestDetail13 | 13 | no | `{{split(1,- -)}}` and `{{split(1,.)}}` — a delimiter containing a space or the subject separator — are the bad-transform kind with an invalid-argument mapping error naming the token. (The delimiter restrictions are committed; the error shape is asserted, not any literal message.) |
| TestDetail14 | 14 | partially | `{{random(7)}}` is the random kind, empty index list, scalar 7. `{{random()}}` and `{{random(1,2)}}` are the bad-transform kind carrying a mapping-destination error naming the token — the specific inner sentinel for wrong arity is not derivable and is not pinned. |
| TestDetail15 | 15 | yes | `{{nosuchfn(1)}}` is the bad-transform kind with an unknown-function mapping error naming the token. |
| TestDetail16 | 16 | yes | `{{Wildcard(3)}}`, `{{ wildcard(3)}}`, and `{{wildcard( 3 )}}` all parse to wildcard `[3]` — the name matching folds case on the first letter and tolerates internal whitespace; `{{WILDCARD(3)}}` matches no function and errors. |
| TestDetail17 | 17 | yes | Integer arguments parse after surrounding spaces are trimmed (`{{wildcard( 3 )}}` etc. — also exercised in the T16 cases). |
| TestDetail18 | 18 | yes | Every successful non-split mapping in the suite asserts the string argument is `""` (all `iphOK` calls pin it). |

Refusals/softening: line 13 is asserted only as the committed bad-transform
kind plus an invalid-argument mapping error naming the token — no literal
message. Line 14's inner error sentinel is deliberately unpinned: the
derivable part is "bad-transform plus a mapping error for the token", and the
concrete arity sentinel differs per arity. Line 16's case-folding claim is
asserted through the regex-visible behaviour (first letter fold + internal
whitespace); the all-caps form is asserted to fail as the derivable
consequence, not from an invented rule.
