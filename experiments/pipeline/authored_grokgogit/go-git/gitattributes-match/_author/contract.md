# Contract — gitattributes-match

Hidden suite: `tests/hidden/plumbing/format/gitattributes/gitattributes_match_bb_test.go`
(package `gitattributes`, in-package). One `TestDetailNN` per DETAILS.md
line. Behaviour is probed through `ParsePattern(p, domain).Match(path)`.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | A path no longer than the domain never matches; a path one component longer can. |
| TestDetail02 | 2 | yes | A path missing the domain prefix, or carrying it reordered, fails; the domain prefix itself passes. |
| TestDetail03 | 3 | no | SHAPE: a one-segment pattern matches the last path component only — never a middle or first component. |
| TestDetail04 | 4 | partially | A multi-segment pattern matches only against the path under the domain, not a suffix elsewhere and not against the whole path when a domain exists. |
| TestDetail05 | 5 | no | SHAPE: an empty interior pattern segment is skipped without consuming a path component. |
| TestDetail06 | 6 | partially | A `**` last segment reached while path components remain succeeds for any remaining depth, including a lone `**` pattern. |
| TestDetail07 | 7 | no | SHAPE: `**` embedded inside a larger segment token never matches — not even text identical to the literal pattern. |
| TestDetail08 | 8 | partially | A middle `**` scans path components until the following segment matches, including the zero-component case; it does not swallow the segment the tail needs. |
| TestDetail09 | 9 | yes | Without a pending `**`, segments match path components in lockstep via single-component glob semantics (`*`, `?`). |
| TestDetail10 | 10 | yes | A path shorter than the remaining pattern fails — including when the unprocessed remainder is `**`. |
| TestDetail11 | 11 | no | SHAPE: a malformed character class in a segment makes the match return false — no error, no panic. |
| TestDetail12 | 12 | no | SHAPE: a plain single-segment pattern matching only a middle component fails. |

Refusals/softening: line 4 does not pin how the remaining pattern is
distributed over the remaining path — only where matching starts. Line 5's
assertion is limited to interior empty segments; a trailing empty segment on
an exhausted path collides with rule 10 and is deliberately not asserted.
Line 6's "succeeds immediately" is asserted only where the `**` segment is
reached before the path is exhausted (the exhausted-path case is asserted as
a failure under rule 10, which the annotation marks derivable). Line 8 does
not pin which candidate position is tried first.
