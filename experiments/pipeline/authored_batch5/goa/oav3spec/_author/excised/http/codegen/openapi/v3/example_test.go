// This file checks how OpenAPI 3 objects receive authored and replacement
// examples without changing the evaluated attribute.
package openapiv3

import (
	_ "testing"

	_ "github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
)

