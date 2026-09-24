# Details — subnetmath

1. `Overlap`/`cidrsOverlap` test `l.Contains(r.IP) || r.Contains(l.IP)` — base-address
   containment, so two disjoint same-family networks are false and a parent/child pair is
   true. `Overlap` additionally nil-guards. Inferable: yes.
2. `BelongsTo` requires equal address-family width (`childBits == parentBits`),
   `childOnes >= parentOnes`, and masked-base equality. `10.0.0.0/8` does NOT belong to
   `10.0.0.0/16` (child longer than parent is required). Inferable: yes.
3. `SplitInto` enumerates subnets in order — `SplitInto4(10.0.0.0/8)` → `10.0.0.0/10`,
   `10.64.0.0/10`, `10.128.0.0/10`, `10.192.0.0/10` — and errors on IPv6 parents.
   Inferable: yes.
4. `incrementIP` adds `1 << (bits - maskOnes)` — i.e. steps by the subnet size, not by 1 —
   with carry propagation into the high half for v6. Inferable: yes.
5. `Allocate` increments BEFORE the first candidate, so the base subnet of `from` is never
   returned (`Allocate("10.0.0.0/8", /24)` yields `10.0.1.0/24`, not `10.0.0.0/24`), marks the
   winner in use, and fails when no non-overlapping candidate remains. Inferable: partially —
   the skip-first rule is a policy choice (the code comments it).
6. `MarkInUse`/`Allocate` wrap parse errors as `error parsing CIDR %q: %v` /
   `error parsing network cidr %q: %v`. Inferable: partially — exact error text is arbitrary.
