package codegen

import (
	_ "testing"

	_ "example.internal/apikit/v3/codegen/testutil"
	_ "example.internal/apikit/v3/expr"

	_ "github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/codegen/codegentest"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)

