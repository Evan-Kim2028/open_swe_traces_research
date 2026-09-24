# Exported API — oaext

```go
func ExtensionsFromExpr(mdata expr.MetaExpr) map[string]any
func ExtensionsFromMethod(method *expr.MethodExpr) map[string]any
```

`extensionsFromExprWithPrefix` is package-private.

## Pre-existing callers

Info/Path/Operation/Response/Tag marshal paths and the spec builders call
these to lift authored meta into `x-*` output keys.
