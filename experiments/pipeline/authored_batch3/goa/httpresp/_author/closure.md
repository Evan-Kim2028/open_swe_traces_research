# Closure — httpresp

`expr/http_response.go` — HTTP response expression: prepare/validate/finalize
pipeline, body-allowed status predicate, unmapped-attribute mapping, dup.

Symbols stubbed: `(*HTTPResponseExpr).EvalName`, `(*HTTPResponseExpr).Prepare`,
`(*HTTPResponseExpr).Validate`, `(*HTTPResponseExpr).Finalize`,
`(*HTTPResponseExpr).Dup`, `(*HTTPResponseExpr).mapUnmappedAttrs`,
`bodyAllowedForStatus`.

Tests removed: `expr/http_response_test.go` deleted (pins every header/cookie
validation message and the text-content-type rule). Other tests exercise the
surface indirectly through eval and panic on the stubs.
