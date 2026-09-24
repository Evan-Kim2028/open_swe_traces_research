# Commitments — oatags

1. parseTags reads "swagger:tag:<name>[:<field>]" and
   "openapi:tag:<name>[:<field>]" meta in sorted key order, grouping
   chunks by tag name; field "desc" sets Description, "url" and
   "url:desc" build ExternalDocs, and — only for OpenAPI 3.2 — summary,
   parent, kind. Other four-chunk keys attach extensions. In-tree
   coverage: deleted tests only. Inferable: no — the chunk layout and
   3.2 gating are details.
2. TagsFromExpr passes extras=true only for Version32; TagNamesFromExpr
   returns just the names. In-tree coverage: deleted tests only.
   Inferable: partially — the version gate is documented.
3. ExtensionsFromExpr merges swagger:extension:* and
   openapi:extension:* keys; ExtensionsFromMethod adds
   "x-goa-idempotent": true when the method is idempotent. In-tree
   coverage: deleted tests only. Inferable: no.
4. extensionsFromExprWithPrefix takes only meta keys under the prefix
   whose remainder is a single segment starting with "x-"; values are
   parsed as JSON when possible (numbers stay numeric) else kept as
   strings. In-tree coverage: deleted tests only. Inferable: no.
5. MarshalJSON encodes v, decodes it back into a map preserving numbers
   (UseNumber), merges extension keys over it, and re-encodes.
   MarshalYAML does the same through yaml round-trip, returning the
   merged map. In-tree coverage: deleted tests only. Inferable:
   partially — the round-trip trick is visible, UseNumber is not.
