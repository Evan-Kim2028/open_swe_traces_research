# Exported API — oav2schema

```go
func BuildAttributeSchema(api *expr.APIExpr, at *expr.AttributeExpr, generator *expr.ExampleGenerator) *openapi.Schema
```

The only exported symbol. Everything else is package-private:
`schemaBuilder` (with its `definitions`/`definitionNames`/`values`
fields) and its methods, `initSchemaValidation`, `renamedResultType`.

## Pre-existing callers

`openapiv2` builder code (`builder.go`) drives schema construction for
paths, parameters, responses, and security definitions transitively
through the builder methods; `TestBuildAttributeSchema*` and
`TestAttributeTypeSchema*` call `BuildAttributeSchema` directly. Result
types are projected via `expr.Project` and examples flow through
`expr.ExampleGenerator` (`gen.At`/`Member`/`MapValue`/`ArrayElement`/
`UnionMember`/`UserTypeExampleIdentity`).
