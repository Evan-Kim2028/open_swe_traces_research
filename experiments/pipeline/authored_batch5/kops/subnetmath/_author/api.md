# Exported API — subnetmath

Package `pkg/util/subnet` (importable as `example.internal/clustkit/pkg/util/subnet`).

IPv4/IPv6 CIDR arithmetic.

- `func Overlap(l, r *net.IPNet) bool` — true when either network contains the other's base
  address; false when either is nil.
- `func BelongsTo(parent, child *net.IPNet) bool` — child is inside parent: same address
  family, child prefix not shorter than parent's, and both masked to the parent mask agree.
- `func SplitInto(additionalBits uint, parent *net.IPNet) ([]*net.IPNet, error)` — splits a
  v4 network into `1<<additionalBits` sequential subnets; errors on non-v4 parents.
  `SplitInto1/2/4/8` are fixed wrappers (additionalBits 0–3).
- `type CIDRMap` — tracks used CIDRs:
  - `(*CIDRMap).MarkInUse(s string) error` — parse + record a used CIDR.
  - `(*CIDRMap).Allocate(from string, mask net.IPMask) (*net.IPNet, error)` — walks candidate
    subnets of size `mask` inside `from`, SKIPPING the first (base) range, and returns the
    first non-overlapping one (recorded as used); error when exhausted.
  - `incrementIP(ip, mask)` — bumps `ip` by one subnet-width of `mask` (v4 and v6 paths).
  - `cidrsOverlap(l, r)` — the internal overlap check (same as `Overlap` minus nil guards).
