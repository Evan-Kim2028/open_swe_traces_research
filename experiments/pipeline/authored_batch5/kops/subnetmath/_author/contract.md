# Contract — subnetmath

CIDR arithmetic in `pkg/util/subnet`. Every commitment below is covered by
a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Overlap.** `Overlap` is base-address containment in either direction:
   parent/child pairs overlap, disjoint same-family networks do not, and
   nil arguments are guarded. Covered by `TestDetail01`.
2. **BelongsTo.** A child belongs to a parent only with equal address-family
   width, a prefix at least as long, and equal masked bases — a network
   belongs to itself, `/8` does not belong to `/16`, and v6/v4 mixes fail.
   Covered by `TestDetail02`.
3. **SplitInto ordering.** `SplitIntoN` splits into N subnets enumerated in
   order — `SplitInto4(10.0.0.0/8)` yields `10.0.0.0/10`, `10.64.0.0/10`,
   `10.128.0.0/10`, `10.192.0.0/10`; `SplitInto2` yields the two `/9`
   halves; `SplitInto1` yields the parent itself; IPv6 parents error.
   Covered by `TestDetail03`.
4. **incrementIP.** Steps by subnet size (`1 << (bits - ones)`), carrying
   across byte boundaries — `10.0.255.0/24` → `10.1.0.0` — and across the
   v6 half boundary (`/64` step lands on the boundary; `/65` on a full low
   half carries into the high half). Covered by `TestDetail04`.
5. **Allocate.** `Allocate` increments before testing the first candidate,
   so the base subnet of `from` is skipped; each returned winner is marked
   in use; when the range is exhausted it returns an error (and terminates).
   Covered by `TestDetail05`.
6. **Error shape.** `MarkInUse`/`Allocate` wrap CIDR parse failures; the
   error names the offending input. Exact text is not pinned. Covered by
   `TestDetail06`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes — containment, symmetry, nil-guard |
| TestDetail02 | 2 | yes — width, prefix length, masked base |
| TestDetail03 | 3 | yes — ordered enumeration, v6 error |
| TestDetail04 | 4 | yes — subnet-size steps and carry |
| TestDetail05 | 5 | partially — skip-first, mark-in-use, exhaustion error |
| TestDetail06 | 6 | partially — error names the offending input (shape) |
