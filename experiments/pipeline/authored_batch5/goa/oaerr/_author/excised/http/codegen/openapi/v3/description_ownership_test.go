// This file verifies that shared OpenAPI v3 components use the named Apikit type
// description while each response keeps its own error description.
package openapiv3

import (
	_ "testing"

	_ "github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)




