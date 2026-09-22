// This file renders complete Swagger 2.0 documents from prepared HTTP designs
// and compares the JSON and YAML output produced with run-owned example state.
package openapiv2_test

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/getkin/kin-openapi/openapi2"
	_ "github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen/testutil"
	_ "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
	openapiv2 "example.internal/apikit/v3/http/codegen/openapi/v2"
	"example.internal/apikit/v3/http/codegen/testdata"
)

func TestSections(t *testing.T) {
	var (
		goldenPath = filepath.Join("testdata", t.Name())
	)
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"empty", testdata.EmptyDSL},
		{"file-service", testdata.FileServiceDSL},
		{"valid", testdata.SimpleDSL},
		{"multiple-services", testdata.MultipleServicesDSL},
		{"multiple-views", testdata.MultipleViewsDSL},
		{"explicit-view", testdata.ExplicitViewDSL},
		{"released-response-collection-names", testdata.ReleasedResponseCollectionNamesDSL},
		{"security", testdata.SecurityDSL},
		{"server-host-with-variables", testdata.ServerHostWithVariablesDSL},
		{"with-spaces", testdata.WithSpacesDSL},
		{"with-map", testdata.WithMapDSL},
		{"with-any", testdata.WithAnyDSL},
		{"path-with-wildcards", testdata.PathWithWildcardDSL},
		{"path-with-multiple-wildcards", testdata.PathWithMultipleWildcardDSL},
		{"path-with-multiple-explicit-wildcards", testdata.PathWithMultipleExplicitWildcardDSL},
		{"headers", testdata.HeadersDSL},
		{"typename", testdata.TypenameDSL},
		{"not-generate-server", testdata.NotGenerateServerDSL},
		{"not-generate-host", testdata.NotGenerateHostDSL},
		{"not-generate-attribute", testdata.NotGenerateAttributeDSL},
		{"json-prefix", testdata.JSONPrefixDSL},
		{"json-indent", testdata.JSONIndentDSL},
		{"json-prefix-indent", testdata.JSONPrefixIndentDSL},
		{"additional-properties-type", testdata.AdditionalPropertiesTypeDSL},
		{"additional-properties-payload-result", testdata.AdditionalPropertiesPayloadResultDSL},
		{"additional-properties-embedded-payload-result", testdata.AdditionalPropertiesPayloadResultDSL},
		{"error-examples", testdata.ErrorExamplesDSL},
		{"shared-error-description", testdata.SharedErrorDescriptionDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			oFiles, err := openapiv2.Files(root, openapi.DefaultPath20)
			if err != nil {
				t.Fatalf("OpenAPI failed with %s", err)
			}
			for i, o := range oFiles {
				tname := fmt.Sprintf("file%d", i)
				s := o.SectionTemplates
				t.Run(tname, func(t *testing.T) {
					if len(s) != 1 {
						t.Fatalf("expected 1 section, got %d", len(s))
					}
					if s[0].Source == "" {
						t.Fatalf("empty section template")
					}
					if s[0].Data == nil {
						t.Fatalf("nil data")
					}
					var buf bytes.Buffer
					tmpl := template.Must(template.New("openapi").Funcs(s[0].FuncMap).Parse(s[0].Source))
					if err := tmpl.Execute(&buf, s[0].Data); err != nil {
						t.Fatalf("failed to render template: %s", err)
					}
					if filepath.Ext(o.Path) == ".json" {
						if err := validateSwagger(buf.Bytes()); err != nil {
							t.Errorf("invalid swagger: %s", err)
						}
					}

					golden := filepath.Join(goldenPath, fmt.Sprintf("%s_%s.golden", c.Name, tname))
					if filepath.Ext(o.Path) == ".json" {
						testutil.AssertJSON(t, golden, buf.Bytes())
					} else {
						testutil.AssertString(t, golden, buf.String())
					}
				})
			}
		})
	}
}




// validateSwagger asserts that the given bytes contain a valid Swagger spec.
func validateSwagger(b []byte) error {
	doc := &openapi2.T{}
	if err := doc.UnmarshalJSON(b); err != nil {
		return err
	}
	if doc.Swagger == "" {
		return errors.New("nil swagger")
	}
	return nil
}
