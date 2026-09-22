# API left after excision

```go
type RefSpec string
func (s RefSpec) Validate() error
func (s RefSpec) IsForceUpdate() bool
func (s RefSpec) IsDelete() bool
func (s RefSpec) IsExactSHA1() bool
func (s RefSpec) Src() string
func (s RefSpec) Match(n plumbing.ReferenceName) bool
func (s RefSpec) IsWildcard() bool
func (s RefSpec) Dst(n plumbing.ReferenceName) plumbing.ReferenceName
func (s RefSpec) Reverse() RefSpec
func MatchAny(l []RefSpec, n plumbing.ReferenceName) bool
```

`Src`, `IsForceUpdate`, `IsDelete`, `IsWildcard`, and `Reverse` still have their original bodies. `Match` / `Dst` / `Validate` panic.
