# Exported API — gattr

```go
type AttributeGraphCopier struct{ /* private fields */ }

func NewAttributeGraphCopier() *AttributeGraphCopier
func (c *AttributeGraphCopier) Copy(att *AttributeExpr) *AttributeExpr
func (c *AttributeGraphCopier) Original(copy *AttributeExpr) *AttributeExpr
```

## Pre-existing callers

View projection and transport body computation copy attribute graphs;
the copier preserves shared subtrees (one original node maps to one
copy) and records original→copy links so callers can ask which design
attribute a computed attribute came from.
