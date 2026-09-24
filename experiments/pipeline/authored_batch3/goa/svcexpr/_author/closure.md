# Closure — svcexpr

`expr/server.go` — server/host expression validation, URI parsing and
substitution, scheme aggregation, default host/URI finalization.

Symbols stubbed: `(*ServerExpr).EvalName`, `(*ServerExpr).Validate`,
`(*ServerExpr).Finalize`, `(*ServerExpr).Schemes`, `(*HostExpr).Validate`,
`(*HostExpr).Finalize`, `(*HostExpr).EvalName`, `(*HostExpr).Attribute`,
`(*HostExpr).Schemes`, `(*HostExpr).HasHTTPScheme`,
`(*HostExpr).HasGRPCScheme`, `(*HostExpr).URIString`, `URIExpr.Params`,
`URIExpr.Scheme`.

Tests removed: `expr/server_test.go` deleted (pinned both Validate tables,
EvalName and the lazy attribute); `TestAPIExprSchemes` trimmed from
`expr/api_test.go` (pinned the sorted scheme union).
