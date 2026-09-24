package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/codegen/testutil"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/testdata"
)


func TestEncodeMarshallingAndUnmarshalling(t *testing.T) {
	cases := []struct {
		Name           string
		DSL            func()
		SectionCount   int
		SectionsOffset int
	}{
		{"embedded-custom-pkg-type", testdata.EmbeddedCustomPkgTypeDSL, 2, 3},
		{"array-alias-extended", testdata.ArrayAliasExtendedDSL, 2, 3},
		{"extension-with-alias", testdata.ExtensionWithAliasDSL, 5, 4},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			plan := linkedHTTPPlanForRoot(t, root)
			fs := plan.ServerFiles()
			require.Len(t, fs, 2)
			sections := fs[1].SectionTemplates
			totalSectionsExpected := c.SectionsOffset + c.SectionCount
			require.Len(t, sections, totalSectionsExpected)
			for i := 0; i < c.SectionCount; i++ {
				code := codegen.SectionCode(t, sections[c.SectionsOffset+i])
				testutil.AssertGo(t, "testdata/golden/server_encode_marshal_"+c.Name+"_section"+string(rune('0'+i))+".go.golden", code)
			}
		})
	}
}
