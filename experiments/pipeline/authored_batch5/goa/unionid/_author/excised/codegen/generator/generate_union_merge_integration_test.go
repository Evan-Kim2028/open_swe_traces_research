// This file verifies that complete generation merges shared union
// declarations without losing their package-owned names.
package generator

import (
	_ "os"
	_ "path/filepath"
	_ "strings"
	_ "testing"

	_ "example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/dsl"
)

