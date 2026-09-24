# Contract (L2) — ratecat

- Search endpoints classify outside the core bucket: code search lands in a
  distinct bucket from general search, and general search classifies the
  same regardless of HTTP method.
- The GraphQL endpoint classifies into its own non-core bucket, distinct
  from the search buckets.
- The manifest-conversion and source-import endpoints are method-scoped
  categories: the committed method lands outside core, and the same path
  under a different method does not share that bucket.
- The code-scanning-upload, scim, and dependency-snapshot endpoint families
  classify outside core, keyed by their path suffix/prefix so different
  resource prefixes agree, and method-scoped where committed.
- The audit-log and repository dependency-sbom endpoint families classify
  outside core, suffix/prefix-keyed across resource prefixes.
- Everything else — including runner-registration endpoints — falls through
  to the core bucket.
- Classification is a pure function of method and path: repeated calls agree,
  no client state is consulted.

Note on shape: the category constants are the exported vocabulary, but which
endpoint binds to which non-core bucket is an endpoint convention — the
suite asserts non-core distinctness, method-scoping and suffix/prefix-keying
rather than pinning each path to a specific named bucket.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — code-search ≠ search, both ≠ core; search is method-insensitive |
| `TestDetail02` | 2 — graphql non-core and distinct from search buckets |
| `TestDetail03` | 3 — method-scoped non-core for manifest-conversions and import (shape) |
| `TestDetail04` | 4 — sarifs/scim/snapshots non-core; suffix-keyed, method-scoped (shape) |
| `TestDetail05` | 5 — audit-log/sbom non-core; suffix/prefix-keyed (shape) |
| `TestDetail06` | 6 — everything else, incl. runner registration → core |
| `TestDetail07` | 7 — pure function: repeated classification identical |
