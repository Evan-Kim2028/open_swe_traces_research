# Exported API — oav3types

```go
type EndpointBodies struct {
    RequestBody    *openapi.Schema
    ResponseBodies map[int][]*openapi.Schema
    SSEItemSchema  *openapi.Schema  // OpenAPI 3.2 only
}

// package-private driver used by the v3 spec builder:
func buildBodyTypes(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr, ver openapi.Version, generator *expr.ExampleGenerator, values openapi.Values) (map[string]map[string]*EndpointBodies, map[string]*openapi.Schema)
```

`EndpointBodies` is consumed by `builder.go` when assembling operations;
the second return value becomes `Components.Schemas`. Everything else —
`schemafier`, `hashAttribute`, `toRef`, `toStringMap`, hashing helpers —
is package-private.

## Pre-existing callers

`builder.go` calls `buildBodyTypes` once per document and reads
`EndpointBodies`/`sf.schemas` when building paths and components.
`expr.ExampleGenerator` (`At`/`Member`/`MapValue`/`ArrayElement`/
`UnionMember`/`UserTypeExampleIdentity`/`RequestBodyExampleIdentity`/
`ResponseBodyExampleIdentity`/`ErrorResponseBodyExampleIdentity`) is the
identity algebra the schemafier threads through recursion.
`openapi.ProjectResponseResult`, `expr.Project`,
`expr.DupAtt`, and `expr.ViewMetaKey` drive view projection.
