# Exported API — atpred

```go
func (a *AttributeExpr) AllRequired() []string
func (a *AttributeExpr) IsRequired(attName string) bool
func (a *AttributeExpr) IsRequiredNoDefault(attName string) bool
func (a *AttributeExpr) IsPrimitivePointer(attName string, useDefault bool) bool
func (a *AttributeExpr) HasTag(tag string) bool
func (a *AttributeExpr) HasTagPrefix(prefix string) bool
func (a *AttributeExpr) FieldTag() (tag string, found bool)
func (a *AttributeExpr) HasDefaultValue(attName string) bool
func (a *AttributeExpr) GetDefault(attName string) any
func (a *AttributeExpr) SetDefault(def any)
func (a *AttributeExpr) Find(name string) *AttributeExpr
func (a *AttributeExpr) Delete(name string)
func TaggedAttribute(a *AttributeExpr, tag string) string
```

`walkAttribute` and `unalias` are unexported helpers backing Find/Delete
and pointer decisions.

## Pre-existing callers

Body computation, validation, and codegen call `IsRequired`,
`IsPrimitivePointer`, `Find`, and `Delete` on every mapped attribute;
security-scheme analysis calls `TaggedAttribute` for `security:*` keys.
