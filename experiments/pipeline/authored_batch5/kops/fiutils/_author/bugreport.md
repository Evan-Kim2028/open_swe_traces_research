# Bug report

Assorted fi utility helpers are broken: slice equality is wrong (order-insensitive compare
returns order-sensitive results and vice versa), `HashString` doesn't produce sha256 hex, the
IPv4/IPv6 predicates misclassify v4-mapped addresses and CIDRs, `ParseCIDRNotation` rejects the
`/N#hex` form, `CIDRSubnet` computes the wrong subnet, `SanitizeString` mangles allowed
characters or truncates the wrong end, `ExpandPath` doesn't expand `~/`, and the YAML helpers
don't go through the JSON-compatible yaml codec.

Expected: documented comparison semantics; sha256-hex hashing; v4-mapped addresses classified
as IPv4; `/20#1`-style notation parses; subnetting by index; `_`-replacement with tail
truncation at 200; `~/` → `$HOME`; sigs.k8s.io/yaml round-trip.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/utils/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
