# Exported API — attachsvc

```go
func (r *RootExpr) EvaluateAttachedServices(services []*ServiceExpr, types ...UserType) error
```

Evaluates a subset of an already-walked design: prepares, validates and
finalizes the selected services, their methods, the named types, and the
transport expressions mounted on them — using the OWNING root's API and
transport trees, not the package-level Root.

## Pre-existing callers

Used by generators/tests that run a second eval pass over services from a
design that was already built once.
