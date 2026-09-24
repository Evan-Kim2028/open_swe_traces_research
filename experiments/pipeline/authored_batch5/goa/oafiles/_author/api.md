# Exported API — oafiles

```go
func Files(spec any, meta expr.MetaExpr, section, path string) []*codegen.File
```

## Pre-existing callers

The v2/v3 spec builders call `Files` once per planned document; `toJSON` /
`toYAML` are template funcs bound into the section FuncMaps.
