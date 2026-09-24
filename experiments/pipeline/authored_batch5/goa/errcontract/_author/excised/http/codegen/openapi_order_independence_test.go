// This file verifies that HTTP and OpenAPI generation produce the same examples
// regardless of which one reads the design first.
package codegen

import (
	"bytes"
	"testing"
	"text/template"

	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)


// renderOpenAPI generates and renders all the OpenAPI specification files for
// the given root and returns their content indexed by file path. The call uses
// a fresh example generator so two identical designs yield identical documents.
func renderOpenAPI(t *testing.T, root *expr.RootExpr) map[string]string {
	t.Helper()
	plan, err := NewOpenAPIPlan(root, expr.NewExampleGenerator(root.API.RandomizerFactory))
	require.NoError(t, err)
	files := plan.Files()
	out := make(map[string]string, len(files))
	for _, f := range files {
		require.Len(t, f.SectionTemplates, 1)
		s := f.SectionTemplates[0]
		var buf bytes.Buffer
		tmpl := template.Must(template.New("openapi").Funcs(s.FuncMap).Parse(s.Source))
		require.NoError(t, tmpl.Execute(&buf, s.Data))
		out[f.Path] = buf.String()
	}
	return out
}
