# Exported API — gotag

```go
func GoNativeTypeName(t expr.DataType) string
func IsNilable(t expr.DataType) bool
func AttributeTags(att *expr.AttributeExpr) string
func AttributeTagsWithName(parent *expr.AttributeExpr, fieldName string, att *expr.AttributeExpr) string
```

`arrayElementIsPointer` and `goFieldIsPointer` are unexported pointer
decisions used by the type planner.

## Pre-existing callers

Go type rendering calls GoNativeTypeName for primitives, IsNilable for
pointer decisions, and AttributeTags(WithName) for struct field tags on
every generated struct field.
