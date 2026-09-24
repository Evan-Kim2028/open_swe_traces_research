# Contract (L2) — zonespec

A zone spec is a comma-separated list of rules; each rule is `*` or a DNS name pattern, optionally negated. `ParseZoneSpec` resolves each rule into a `ZoneSpec`; `ParseZoneRules` splits the spec on commas honoring escapes. `MatchesExplicitly` reports whether a DNS name is claimed by a non-wildcard rule — matching is case-insensitive, respects the `.` suffix, and a negated rule excludes the name. An empty or all-wildcard spec never matches explicitly.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestEnsureDotSuffix` | names are normalized to a trailing dot |
| `TestParseZoneSpec` | comma-separated rules parse into zone specs |
| `TestParseZoneRules` | rule splitting handles escapes and empty entries |
| `TestMatchesExplicitly` | wildcard vs explicit-name matching, negation excludes |
