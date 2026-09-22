// This file verifies generated HTTP error body comments name the service
// errors that use each body type.
package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)


// renderHTTPSections writes all generated sections after the file header.
func renderHTTPSections(t *testing.T, file *codegen.File) string {
	t.Helper()
	var rendered strings.Builder
	for _, section := range file.SectionTemplates[1:] {
		require.NoError(t, section.Write(&rendered))
	}
	return rendered.String()
}
