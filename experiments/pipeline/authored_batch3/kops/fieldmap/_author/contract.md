# Contract (L2) — fieldmap

`ClusterField` wraps a field path and translates it between API versions: `HumanPath`/`PathInV1Alpha2` render the field in the v1alpha2 spelling, `InternalPath`/`PathInV1Alpha3` in the v1alpha3 spelling. `NewClusterField` normalizes the incoming path (accepting either version's spelling). `HumanPathForClusterField`/`InternalPathForClusterField` are the package-level shorthands. Fields with no version-specific rename map to themselves in both directions.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestClusterFieldHumanPath` | a renamed field translates to its user-facing path |
