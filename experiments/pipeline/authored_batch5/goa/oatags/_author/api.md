# Exported API — oatags

```go
func TagsFromExpr(mdata expr.MetaExpr, ver Version) []*Tag
func TagNamesFromExpr(mdata expr.MetaExpr) (tagNames []string)
func (t Tag) MarshalJSON() ([]byte, error)
func (t Tag) MarshalYAML() (any, error)
```

`Tag` stays compiled; `parseTags` is the package-private parser.

## Pre-existing callers

v2/v3 builders emit global `tags` arrays via `TagsFromExpr` and reference tag
names per operation via `TagNamesFromExpr`.
