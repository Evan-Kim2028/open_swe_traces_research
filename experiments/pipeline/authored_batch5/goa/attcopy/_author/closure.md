# Closure — attcopy

`expr/attribute_graph_copier.go` — deep copy of a connected attribute graph:
memoized node copy preserving sharing and cycles, copy→original resolution,
deep copy of mutable `any` payloads via reflection (defaults, validations,
examples), cyclic-value and unexported-mutable-field rejection.

Symbols stubbed: `NewAttributeGraphCopier`,
`AttributeGraphCopier.{Copy,Original,dataTypes,dataType}`,
`copyAttributeValidation`, `copyAttributeMeta`, `copyAttributeValue`,
`copyAttributeReflectValue`, `enterAttributeValue`,
`attributeTypeContainsReference`.

Tests removed: `expr/attribute_graph_copier_test.go` deleted (pins every
commitment). No other in-tree test reaches the stubs.
