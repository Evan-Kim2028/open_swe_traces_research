package openapi

import (
	_ "encoding/json"
	_ "maps"
	_ "strings"

	"example.internal/apikit/v3/expr"
)

// ExtensionsFromExpr generates openapi extensions from the given meta
// expression.
func ExtensionsFromExpr(mdata expr.MetaExpr) map[string]any {
	panic("excised: ExtensionsFromExpr")
}

// ExtensionsFromMethod returns the OpenAPI extensions authored as method
// metadata and advertises the method's idempotency contract when present.
func ExtensionsFromMethod(method *expr.MethodExpr) map[string]any {
	panic("excised: ExtensionsFromMethod")
}

// extensionsFromExprWithPrefix generates openapi extensions from
// the given meta expression with keys starting the given prefix.
func extensionsFromExprWithPrefix(mdata expr.MetaExpr, prefix string) map[string]any {
	panic("excised: extensionsFromExprWithPrefix")
}
