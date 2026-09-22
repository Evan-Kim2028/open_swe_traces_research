# Exported API — attcopy

```go
// AttributeGraphCopier copies one connected attribute graph without merging
// distinct user types or breaking recursive links. Reuse one copier for
// every root that must share copied nodes.
type AttributeGraphCopier struct { /* opaque */ }

func NewAttributeGraphCopier() *AttributeGraphCopier

// Copy returns a deep copy of attribute. Shared and recursive nodes remain
// shared in the result.
func (c *AttributeGraphCopier) Copy(attribute *expr.AttributeExpr) *expr.AttributeExpr

// Original returns the input node copied to create attribute, or attribute
// unchanged when this copier did not create it.
func (c *AttributeGraphCopier) Original(attribute *expr.AttributeExpr) *expr.AttributeExpr
```

## Pre-existing callers

Generators snapshot design attributes they plan to mutate: `codegen/service`
uses the copier for retained plans and relocated declarations, and HTTP/gRPC
service data builders copy body and response attribute graphs before
attaching transport-specific metadata.
