# Exported API — unionid

```go
type UnionDeclarationID struct{ /* private */ }
type UnionTypeID string

func NewUnionDeclarationID(attribute *expr.AttributeExpr) UnionDeclarationID
func NewUnionTypeID(union *expr.Union) UnionTypeID
```

Unexported writers: `writeUnionTypeID`, `writeUnionAttributeID`,
`writeUnionObjectID`, `writeUnionIDPart`.

## Pre-existing callers

NameScope uses UnionTypeID to decide whether two union attributes refer
to the same generated declaration, and codegen keys union branches by
the declaration ID.
