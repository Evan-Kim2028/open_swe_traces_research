# Closure — methodval

`expr/method.go` — method expression: payload/result validate pipeline,
security requirement resolution and tag predicates, interceptor merge,
finalize/inherit, streaming predicates.

Symbols stubbed: `(*MethodExpr).Prepare`, `(*MethodExpr).Validate`,
`(*MethodExpr).validateRequirements`, `isSecurityAttribute`,
`isStringType`, `(*MethodExpr).validateErrors`,
`(*MethodExpr).validateInterceptors`, `mergeInterceptors`, `hasTag`,
`hasTagPrefix`, `(*MethodExpr).Finalize`, `(*MethodExpr).IsStreaming`,
`(*MethodExpr).IsPayloadStreaming`, `(*MethodExpr).IsResultStreaming`,
`(*MethodExpr).HasMixedResults`, `copyReqs`.

Tests removed: `expr/method_test.go` deleted (pins requirement tags, error
inheritance, finalize ordering).
