# Closure — valrules

`expr/attribute.go` — ValidationExpr lifecycle: bound-conflict
validation, constraint-tightening merge, required-list set operations,
emptiness predicate, dup.

Symbols stubbed: `(*ValidationExpr).Validate`, `Merge`, `AddRequired`,
`RemoveRequired`, `HasRequiredOnly`, `Dup`.

Tests removed: validation tests trimmed from `expr/attribute_test.go`.
Disjoint from atpred which stubs the AttributeExpr predicate set in the
same file.
