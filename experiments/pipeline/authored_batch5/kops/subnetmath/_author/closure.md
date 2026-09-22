# Closure — subnetmath

Package: `pkg/util/subnet` (`example.internal/clustkit/pkg/util/subnet`).

Files: `pkg/util/subnet/subnet.go` (7 funcs), `pkg/util/subnet/cidrmap.go` (6 funcs) — 13 funcs.

Removed functions (bodies stubbed): `Overlap`, `BelongsTo`, `SplitInto1`, `SplitInto2`,
`SplitInto4`, `SplitInto8`, `SplitInto`, `CIDRMap.MarkInUse`, `incrementIP`, `duplicateIP`,
`CIDRMap.Allocate`, `CIDRMap.isInUse`, `cidrsOverlap`.

Exported entry point(s): `Overlap`/`BelongsTo`/`SplitInto*`/`CIDRMap.Allocate` — used by
networking-population code that carves pod/service CIDRs out of cluster ranges.

Test files removed in excision: `subnet_test.go`.
