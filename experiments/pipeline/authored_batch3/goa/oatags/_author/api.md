# Exported API — oatags

```go
type Tag struct {
    Name, Summary, Description, Parent, Kind string
    ExternalDocs *ExternalDocs
    Extensions   map[string]any
}
func TagsFromExpr(mdata expr.MetaExpr, ver Version) []*Tag
func TagNamesFromExpr(mdata expr.MetaExpr) []string
func ExtensionsFromExpr(mdata expr.MetaExpr) map[string]any
func ExtensionsFromMethod(method *expr.MethodExpr) map[string]any
func MarshalJSON(v any, extensions map[string]any) ([]byte, error)
func MarshalYAML(v any, extensions map[string]any) (any, error)
```

Unexported: `parseTags`, `extensionsFromExprWithPrefix`.

## Pre-existing callers

Every OpenAPI generation reads tags and extensions from service/method
meta; all schema types marshal through MarshalJSON/MarshalYAML to merge
extension keys into emitted documents.
