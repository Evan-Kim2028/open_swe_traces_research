# Closure — intercept

`expr/interceptor.go` — interceptor design validation:
`InterceptorExpr.validate` checks that declared read/write access matches the
method's payload/result/streaming shapes (object-only, streaming-compatible,
fields exist after base-type merge), and `EvalName` names the expression in
errors.

Symbols stubbed: `InterceptorExpr.EvalName`, `InterceptorExpr.validate`, `InterceptorExpr.validateAttributeAccess`.

Tests removed (reach the stubs, verified by excision):
- `expr/interceptor_test.go`: `TestInterceptorExpr_Validate`
- `expr/method_test.go`: `TestMethodExprValidateInterceptors`
