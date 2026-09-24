// This file protects the released OpenAPI v2 function signatures and checks
// that their default files match files produced with the root's example
// generator and no replacement values.
package openapiv2_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	openapiv2 "example.internal/apikit/v3/http/codegen/openapi/v2"
)

var (
	_ func(*expr.RootExpr, *expr.HostExpr) (*openapiv2.V2, error) = openapiv2.NewV2
	_ func(*expr.RootExpr, string) ([]*codegen.File, error)       = openapiv2.Files

	facadeDSL = func() {
		dsl.API("facade", func() {
			dsl.Server("facade", func() {
				dsl.Host("localhost", func() {
					dsl.URI("https://goa.design")
				})
			})
		})
		dsl.Service("facade", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("message", dsl.String)
				})
				dsl.Result(func() {
					dsl.Attribute("answer", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.POST("/items")
				})
			})
		})
	}
)


// renderFiles runs each file template so the test compares the documents that
// users receive instead of comparing template implementation details.
func renderFiles(t *testing.T, files []*codegen.File) map[string]string {
	t.Helper()

	rendered := make(map[string]string, len(files))
	for _, file := range files {
		var buf bytes.Buffer
		for _, section := range file.SectionTemplates {
			tmpl, err := template.New("openapi").Funcs(section.FuncMap).Parse(section.Source)
			require.NoError(t, err)
			require.NoError(t, tmpl.Execute(&buf, section.Data))
		}
		rendered[file.Path] = buf.String()
	}
	return rendered
}
