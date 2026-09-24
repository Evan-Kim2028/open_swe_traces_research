# Closure — atpred

`expr/attribute.go` — attribute predicates: requiredness, defaults,
tags, member lookup/deletion, and the unalias/walkAttribute helpers.

Symbols stubbed: `(*AttributeExpr).AllRequired`, `IsRequired`,
`IsRequiredNoDefault`, `IsPrimitivePointer`, `HasDefaultValue`,
`GetDefault`, `SetDefault`, `Find`, `Delete`, `HasTag`, `HasTagPrefix`,
`FieldTag`, `TaggedAttribute`, `walkAttribute`, `unalias`.

Tests removed: predicate tests trimmed from `expr/attribute_test.go`.
Disjoint from valrules which stubs the ValidationExpr methods in the
same file.
