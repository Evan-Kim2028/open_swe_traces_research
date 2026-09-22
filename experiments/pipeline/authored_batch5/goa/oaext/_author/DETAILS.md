# Commitments — oaext

1. Both extension prefixes contribute; results merge into one map (nil when empty). In-tree coverage: `TestExtensions` (trimmed). Inferable: doc — ExtensionsFromExpr's comment covers both families.
2. Only meta keys whose remainder is a single-level `x-` name become extensions; nested `:` names and non-`x-` names are skipped. In-tree coverage: `TestExtensions` (trimmed). Inferable: partially — the x- and single-level filters are stated obliquely.
3. Each extension value is JSON-parsed so typed values (numbers, objects) emit natively; unparseable values stay raw strings. In-tree coverage: `TestExtensions` (trimmed). Inferable: partially — parse-with-fallback is a choice.
4. `ExtensionsFromMethod` adds `x-goa-idempotent: true` for idempotent methods, allocating the map when needed. In-tree coverage: `TestExtensionsFromMethod` (trimmed). Inferable: doc — its comment states the advertisement.
