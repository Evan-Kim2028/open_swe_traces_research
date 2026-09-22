# Exported API — bodytype

```go
func RemovePkgPath(attr *AttributeExpr)
```

The remaining functions are unexported HTTP body computation helpers:
`httpRequestBody`, `httpStreamingBody`, `httpResponseBody`,
`httpErrorResponseBody`, `buildHTTPResponseBody`, `generatedUserType`,
`copyOpenAPITypeMeta`, `concat`, `renameType`, `removeAttributes`,
`removeAttribute`, `extendBodyAttribute`, `walk`, `walkrec`,
`defaultRequestHeaderAttributes`.

## Pre-existing callers

`HTTPEndpointExpr` Finalize computes request/streaming/response bodies
for every endpoint; OpenAPI codegen reads the computed body attributes
and their generated user types.
