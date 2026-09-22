# Bug report

Subnet math is broken: network overlap and containment checks return wrong results, splitting a
CIDR produces the wrong subnets, and `CIDRMap.Allocate` hands out the base range it was told to
skip or re-issues CIDRs already marked in use.

Expected: overlap is base-address containment; `BelongsTo` requires same family, longer-or-equal
child prefix, and masked-base equality; `SplitInto` enumerates sequential subnets
(`10.0.0.0/8` → `10.0.0.0/10`, `10.64.0.0/10`, ...); `Allocate` skips the first range and returns
the first unused candidate.

Reproduce with:

```
go test -count=1 ./pkg/util/subnet/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
