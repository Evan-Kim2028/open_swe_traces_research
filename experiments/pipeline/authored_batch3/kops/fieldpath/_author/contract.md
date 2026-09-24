# Contract (L2) — fieldpath

`FieldPath` is a parsed dotted path (`spec.containers[0].name`-style). `ParseFieldPath` builds the segment list, accepting array indices and rejecting malformed syntax. `String` round-trips to the canonical dotted form. `IsEmpty` is the zero path. `Matches` tests equality-or-wildcard segment by segment; `HasPrefixMatch` tests whether a shorter pattern is a prefix — a wildcard segment (`*`) matches any single segment, not a run.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestParseFieldPath` | parse, round-trip, wildcard matching, and prefix semantics |
