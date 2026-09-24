# Contract — dns-alt-name-matches

Hidden suite: `tests/hidden/server/dns_alt_name_matches_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
SAN input is built with the same split-lower helper the call sites use;
candidate URLs are `*url.URL` values.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | A nil entry in the URL slice is skipped: nil alone yields no match, and a nil in front of a matching URL still matches. |
| TestDetail02 | 2 | yes | A URL whose hostname is uppercase still matches a lowercase SAN — the hostname is lowered before the label walk. |
| TestDetail03 | 3 | doc | Label counts must be equal: a 2-label SAN cannot match a 3-label hostname, and vice versa; equal counts with equal labels match. |
| TestDetail04 | 4 | doc | A leading `"*"` SAN label skips label 0 only: `*.a.b` matches `x.a.b` but not `x.b.b`. |
| TestDetail05 | 5 | doc | A `"*"` in a non-leading SAN label is a literal: `a.*` fails against `a.b` and `a.*.c` against `a.b.c`, but `a.*` matches a hostname whose second label is literally `*`. |
| TestDetail06 | 6 | doc | `*` spans at most one label: `*.a.b` does not match `x.y.a.b`. |
| TestDetail07 | 7 | yes | Non-wildcard labels compare by plain equality on already-lowered text: `a.b` rejects `a.c` and accepts `a.B`. |
| TestDetail08 | 8 | yes | Any single surviving URL is enough: a match behind two non-matching URLs returns true. |
| TestDetail09 | 9 | yes | No matching URL (or an empty slice) returns false. |
| TestDetail10 | 10 | partially | An empty hostname is not special-cased: it is one empty label, so a one-label `"*"` SAN matches it, a one-label literal SAN does not, and a two-label SAN fails on the count. |

Refusals/softening: none. Line 10 is asserted only through the equal-length
and label-walk consequences (the derivable part); nothing is asserted about
how the empty string itself is produced internally.
