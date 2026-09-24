# Closure — jsonxform

Package: `pkg/jsonutils` (`example.internal/clustkit/pkg/jsonutils`).

Files: `pkg/jsonutils/transform.go` (11 funcs).

Removed functions (bodies stubbed): `NewTransformer`, `Transformer.AddStringTransform`,
`Transformer.AddObjectTransform`, `Transformer.AddSliceTransform`, `Transformer.Transform`,
`Transformer.visitAny`, `Transformer.visitMap`, `Transformer.visitSlice`, `SortSlice`,
`Transformer.visitPrimitive`, `Transformer.visitString`.

Exported entry point(s): `Transformer` — used by `kops toolbox dump`-style output
normalization and manifest cleanup paths; `SortSlice` gives deterministic ordering.

Test files removed in excision: `transform_test.go`.
