# Exported API — rootval

```go
func (r *RootExpr) WalkSets(walk eval.SetWalker)
func (r *RootExpr) UserType(name string) UserType
func (r *RootExpr) Service(name string) *ServiceExpr
func (r *RootExpr) Error(name string) *ErrorExpr
func (r *RootExpr) Validate() error
func (r *RootExpr) Finalize()
func ValidateSharedErrorNames(roots ...*RootExpr) *eval.ValidationErrors
func (m MetaExpr) Dup() MetaExpr
func (m MetaExpr) Merge(src MetaExpr)
func (m MetaExpr) Last(key string) (string, bool)
```

## Pre-existing callers

`WalkSets` drives the whole eval engine — every DSL run walks these sets in
order. `UserType`/`Service`/`Error` are the design's public lookups.
`MetaExpr` helpers back every `Meta` call in the DSL.
