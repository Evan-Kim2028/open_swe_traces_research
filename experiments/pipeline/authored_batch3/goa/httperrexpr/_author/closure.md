# Closure — httperrexpr

`expr/http_error.go` — HTTP error expression: parent-scoped error matching,
JSON-RPC reserved-code predicate, header-vs-error-type validation, finalize
defaults, dup.

Symbols stubbed: `(*HTTPErrorExpr).EvalName`, `(*HTTPErrorExpr).IsJSONRPC`,
`(*HTTPErrorExpr).Validate`, `jsonRPCErrorCodeReserved`,
`(*HTTPErrorExpr).Finalize`, `(*HTTPErrorExpr).mappedError`,
`(*HTTPErrorExpr).Dup`.

Tests removed: `expr/http_error_test.go` deleted (pins every header-mapping
error message); `TestJSONRPCErrorCodeAcceptsAllowedValues` and
`TestJSONRPCErrorCodeRejectsReservedValues` trimmed from
`expr/jsonrpc_error_contract_test.go` (pin the allowed/reserved code sets).
