// This file writes OpenAPI documents as JSON and YAML code generation files.
package openapi

import (
	_ "encoding/json"
	_ "path/filepath"
	"regexp"
	_ "strconv"
	_ "strings"
	_ "text/template"

	_ "gopkg.in/yaml.v3"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/expr"
)

var yamlDatePrefix = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}(?:$|[Tt ].*)`)

// Files returns the codegen files rendering the given OpenAPI document in
// JSON and YAML at gen/<path>.json and gen/<path>.yaml. section names the
// generated section templates and meta provides the JSON formatting options
// ("openapi:json:prefix" and "openapi:json:indent").
func Files(spec any, meta expr.MetaExpr, section, path string) []*codegen.File {
	panic("excised: Files")
}

// toJSON returns a template function encoding its argument in JSON, honoring
// the "openapi:json:prefix" and "openapi:json:indent" formatting meta.
func toJSON(meta expr.MetaExpr) func(any) string {
	panic("excised: toJSON")
}

// toYAML encodes its argument in YAML.
func toYAML(d any) string {
	panic("excised: toYAML")
}

// quoteDateShapedYAMLStrings keeps YAML readers from treating string examples
// as timestamps. Some readers reject date-shaped strings with invalid months
// or days before they can see that the OpenAPI schema declares a string.
func quoteDateShapedYAMLStrings(source string) string {
	panic("excised: quoteDateShapedYAMLStrings")
}
