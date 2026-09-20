# Closure — attachsvc

`expr/attached_service.go` — attached-service re-evaluation: membership and
ownership predicates over services/methods/types/transports, expression-set
assembly, prepare/validate/finalize driver.

Symbols stubbed: `(*RootExpr).EvaluateAttachedServices`,
`(*RootExpr).attachedServiceExpressions`, `collectHTTPExpressions`,
`collectGRPCExpressions`, `prepareExpressions`, `validateExpressions`,
`finalizeExpressions`.

Tests removed: `expr/attached_service_test.go` deleted (pins every
membership/ownership check and the all-before-finish pipeline).
