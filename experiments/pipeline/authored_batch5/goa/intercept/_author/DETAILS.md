# Commitments — intercept

1. Payload read/write access requires an object payload; base-type attributes are merged into a duplicated payload before field checks. In-tree coverage: `TestInterceptorExpr_Validate`, `TestMethodExprValidateInterceptors` (trimmed). Inferable: partially — the error text and merge semantics are choices.
2. Result read/write access is rejected on streaming results and requires an object result. In-tree coverage: same. Inferable: partially.
3. Streaming-payload access requires the method to stream a payload object; streaming-result access requires a streaming result. In-tree coverage: same. Inferable: partially.
4. `validateAttributeAccess` reports each accessed field missing from the merged target attribute. In-tree coverage: same. Inferable: partially.
5. `EvalName` returns a stable "interceptor"-prefixed name used in diagnostics. In-tree coverage: error messages in trimmed tests. Inferable: no — exact spelling is arbitrary; verifier asserts the prefix shape.
