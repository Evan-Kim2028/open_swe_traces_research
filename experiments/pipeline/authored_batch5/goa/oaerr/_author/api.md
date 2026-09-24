# Exported API — oaerr

```go
func ResponseContentType(response *expr.HTTPResponseExpr) string
func ErrorResponseExample(errorExpression *expr.ErrorExpr, body *expr.AttributeExpr, generator *expr.ExampleGenerator, values Values) (any, bool)
```

`setExampleField` is the package-private write-guard helper.

## Pre-existing callers

The v2/v3 response builders call `ResponseContentType` per response and
`ErrorResponseExample` when the response uses the built-in error result.
