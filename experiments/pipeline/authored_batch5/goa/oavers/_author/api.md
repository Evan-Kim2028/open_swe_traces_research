# Exported API — oavers

The closure is called by the OpenAPI file planners in `v2`/`v3` and the
example generator:

```go
// Specs computes the OpenAPI documents to generate from the given API meta.
func Specs(meta expr.MetaExpr) ([]Spec, error)
```

`Spec{Version, Path}` and the `Version20/30/32` + `DefaultPath*` constants stay
compiled; the closure owns `Specs` plus the helpers `selectedVersions`,
`specPath`, `validatePathKeys`, `knownVersion`.

## Pre-existing callers

`http/codegen/openapi` file generation (`Files` callers) and every spec build
entry point ask `Specs` which documents to emit and where.
