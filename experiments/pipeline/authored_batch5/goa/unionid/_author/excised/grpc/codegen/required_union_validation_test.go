// This file checks the validation functions generated for gRPC server requests
// and client responses which contain a required OneOf.
package codegen

import (
	"strings"
	"testing"

	_ "github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/codegen/testutil"
	d "example.internal/apikit/v3/dsl"
)


// validationSection returns the generated validator named function.
func validationSection(t *testing.T, sections []*codegen.SectionTemplate, function string) string {
	t.Helper()
	for _, section := range sections {
		code := codegen.SectionCode(t, section)
		if strings.Contains(code, "func "+function+"(") {
			return code
		}
	}
	t.Errorf("missing generated validator %s", function)
	return ""
}

// requiredUnionValidationDSL creates branches whose values include scalars,
// messages, an empty message, a byte slice, a named string, and Any.
func requiredUnionValidationDSL() {
	token := d.Type("Token", d.String)
	detail := d.Type("Detail", func() {
		d.Field(1, "label", d.String)
		d.Required("label")
	})
	inactive := d.Type("Inactive", func() {})
	request := d.Type("RequestChoice", func() {
		d.OneOf("choice", func() {
			d.TypeName("RequestChoiceValue")
			d.Field(1, "number", d.Int, func() { d.Minimum(1) })
			d.Field(2, "detail", detail)
			d.Field(3, "inactive", inactive)
			d.Field(4, "blob", d.Bytes)
			d.Field(5, "token", token)
			d.Field(6, "metadata", d.Any)
		})
		d.Required("choice")
	})
	response := d.Type("ResponseChoice", func() {
		d.OneOf("choice", func() {
			d.TypeName("ResponseChoiceValue")
			d.Field(1, "number", d.Int, func() { d.Minimum(1) })
			d.Field(2, "detail", detail)
			d.Field(3, "inactive", inactive)
			d.Field(4, "blob", d.Bytes)
			d.Field(5, "token", token)
			d.Field(6, "metadata", d.Any)
		})
		d.Required("choice")
	})
	d.Service("UnionValidation", func() {
		d.Method("Exchange", func() {
			d.Payload(request)
			d.Result(response)
			d.GRPC(func() {})
		})
	})
}
