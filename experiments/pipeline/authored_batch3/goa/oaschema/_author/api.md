# Exported API — oaschema

```go
func ToString(val any) string
func ToStringMap(val any) any
func ProjectExample(at *expr.AttributeExpr, val any) any
func (s *Schema) Dup() *Schema
func (s *Schema) Merge(other *Schema)
func MustGenerate(meta expr.MetaExpr) bool
func AdditionalPropertiesFromExpr(meta expr.MetaExpr) any
```

Unexported: `projectExample`, `projectObjectExample`,
`projectArrayExample`, `projectMapExample`, `exampleMap`,
`exampleSlice`, `duplicateJSONValue`, `duplicateJSONReflectValue`,
`Schema.createMergeItems`.

## Pre-existing callers

The OpenAPI generators call Dup before mutating shared schemas, Merge to
combine extension schemas, and ProjectExample to drop meta-hidden fields
from rendered examples.
