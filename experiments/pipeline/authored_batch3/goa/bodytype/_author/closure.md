# Closure — bodytype

`expr/http_body_types.go` — HTTP body derivation: header/param/cookie
subtraction, generated body type naming, bases/references merge, pkgpath
cleanup, type walking.

Symbols stubbed: `defaultRequestHeaderAttributes`, `httpRequestBody`,
`httpStreamingBody`, `httpResponseBody`, `httpErrorResponseBody`,
`buildHTTPResponseBody`, `generatedUserType`, `copyOpenAPITypeMeta`,
`concat`, `renameType`, `RemovePkgPath`, `removeAttributes`,
`removeAttribute`, `extendBodyAttribute`, `walk`, `walkrec`.

Tests removed: `expr/http_body_types_test.go` deleted.
