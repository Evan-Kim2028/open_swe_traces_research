# Closure — httpsvc

`expr/http_service.go` — HTTP service expression: path computation (join,
clean, parent canonical), error lookup/inheritance, JSON-RPC route
creation and POST constraint, validate chain, finalize.

Symbols stubbed: all 21 methods of `*HTTPServiceExpr` in the file.

Tests removed: `expr/http_endpoint_test.go` deleted (pins parent-child
validation and canonical-endpoint behavior).
