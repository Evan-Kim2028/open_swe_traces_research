# Contract — jwt-time-windows

Hidden suite: `tests/hidden/server/jwt_time_windows_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Both entry points are exercised: the per-range predicate directly (with
pre-seated instants) and the claims-level entry (which parses strings,
applies the locale, and re-seats onto now's date).

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | Nil claims yield `(false, 0)`. |
| TestDetail02 | 2 | yes | Non-nil claims with no time ranges yield `(true, 0)`. |
| TestDetail03 | 3 | yes | A range covering now only in the claims locale (`Pacific/Kiritimati`, UTC+14, no DST) matches; the same instant does not match when the process local zone has zero offset — the locale is loaded and `now` converted. |
| TestDetail04 | 4 | yes | A claims locale that cannot be loaded yields `(false, 0)` even when a range would otherwise match. |
| TestDetail05 | 5 | no | Range strings are parsed to second resolution with the committed clock layout — a 5-second window containing now matches with exactly the computed remaining, and an instant one second past it does not. (Shape: seconds are honoured; no parse internals pinned.) |
| TestDetail06 | 6 | yes | A clock range re-seats onto now's calendar date (same window matches on a different year/day), and a `now` carrying sub-second precision still counts a window starting at its clock second — nanoseconds are zeroed. |
| TestDetail07 | 7 | no | A non-wrapping range matches an interior instant and rejects instants equal to start, equal to end, or after end — endpoints are exclusive. (Asserted as behaviour of the committed predicate; the boolean formula is the derivable consequence.) |
| TestDetail08 | 8 | doc | A matching non-wrapping range reports remaining equal to end minus now. |
| TestDetail09 | 9 | doc | A range with start after end wraps midnight: it matches just after midnight, while a non-wrapping range does not match outside itself, and an empty start==end range matches nothing. |
| TestDetail10 | 10 | doc | In the after-midnight side of a wrapping range, end being after now suffices — remaining is end minus now. |
| TestDetail11 | 11 | doc | In the before-midnight side of a wrapping range, the effective end is one calendar day later — remaining spans midnight; a mid-morning instant does not match. |
| TestDetail12 | 12 | partially | With two matching ranges (2h and 5h remaining) and one non-match, the call reports the maximum remaining — 5h. |
| TestDetail13 | 13 | yes | With no matching range the call yields `(false, 0)`. |
| TestDetail14 | 14 | yes | An unparseable start or end anywhere in the list fails the whole call with `(false, 0)`, even alongside a range that matches. |

Refusals/softening: lines 5 and 7 are `Inferable: no` — asserted only
through the observable boundary behaviour the committed rules imply (seconds
honoured, exclusive endpoints), never through internal formulas or literal
strings.
