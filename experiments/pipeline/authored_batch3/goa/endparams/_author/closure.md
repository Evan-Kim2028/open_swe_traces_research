# Closure — endparams

`expr/http_endpoint.go` — route/endpoint params: wildcard extraction,
path join, path/query/header/cookie validation.

Symbols stubbed: `(*HTTPEndpointExpr).PathParams`, `QueryParams`,
`validateParams`, `validateHeadersAndCookies`, `(*RouteExpr).Validate`,
`Params`, `FullPaths`, `IsAbsolute`.

Tests removed: route/endpoint validation tests trimmed from
`expr/http_endpoint_test.go`.
