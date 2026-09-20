# Exported API — fieldpath

Package `util/pkg/reflectutils` (importable as `example.internal/clustkit/util/pkg/reflectutils`).

- `func ParseFieldPath(s string) (*FieldPath, error)` — parse `a.b[0].c`, `a[key]`, `a[*]` spellings into a `FieldPath`.
- `func (f *FieldPath) String() string` — render the path back to its canonical spelling.
- `func (p *FieldPath) IsEmpty() bool`, `Matches(r *FieldPath) bool`, `HasPrefixMatch(r *FieldPath) bool` — path predicates.
- `func (f *FieldPath) Extend(el FieldPathElement) *FieldPath` — intact helper.
- `type FieldPathElement{Type, token, number}` with `FieldPathElementTypeField|MapKey|ArrayIndex|WildcardIndex`.

Production callers: `util/pkg/reflectutils/access.go` (`SetFieldByPath`-style helpers), `walk.go` walkers.
