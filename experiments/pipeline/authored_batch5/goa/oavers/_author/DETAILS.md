# Commitments — oavers

1. `openapi:versions` absent selects every supported version; present values select exactly the named ones, in fixed generation order (2.0, 3.0, 3.2) regardless of meta order. In-tree coverage: `TestSpecs` (trimmed). Inferable: doc — Specs' comment states the default and order.
2. Unknown version values are rejected. In-tree coverage: `TestSpecs` (trimmed). Inferable: doc — "errors on invalid version or path values"; exact message shape only.
3. `openapi:path:<version>` overrides the default output path; the override must be non-empty, relative to gen/, clean (no `..` escape), and extension-less. In-tree coverage: `TestSpecs` (trimmed). Inferable: doc — specPath's comment.
4. A path meta key whose version suffix is unknown errors instead of being silently ignored (typo fence). In-tree coverage: `TestSpecs` (trimmed). Inferable: doc — validatePathKeys' comment.
5. Two selected versions resolving to the same output path error. In-tree coverage: `TestSpecs` (trimmed). Inferable: partially — uniqueness is implied by file emission; the exact check is a choice.
