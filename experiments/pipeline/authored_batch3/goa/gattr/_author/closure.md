# Closure — gattr

`expr/attribute_graph_copier.go` — graph-preserving attribute copier:
shared-node identity, original→copy map, recursive type fixup,
reflection-based deep copy of mutable leaf values.

Symbols stubbed: `NewAttributeGraphCopier`, `(*AttributeGraphCopier).Copy`,
`(*AttributeGraphCopier).Original`, and the recursive copy/deep-value
helpers.

Tests removed: pinning tests trimmed/deleted in expr package.
